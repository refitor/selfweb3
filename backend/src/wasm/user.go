package wasm

import (
	"crypto/ecdsa"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/dgryski/dgoogauth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	c_param_random      = "random"
	c_param_selfPass    = "selfPass"
	c_param_selfKey     = "selfKey"
	c_param_recoverID   = "recoverID"
	c_param_selfPrivate = "selfPrivate"
	c_param_relateTimes = "relateTimes"

	c_verify_action        = "action"
	c_verify_action_query  = "query"
	c_verify_action_update = "update"
)

type Web3User struct {
	QRCode      string
	Web2Data    string
	SelfAddress string

	SelfKey     string
	RecoverID   string
	SelfPrivate string
}

type User struct {
	SelfData *SelfData

	RecoverID   string
	Web2Public  *ecdsa.PublicKey
	Web3Public  *ecdsa.PublicKey
	SelfPrivate *ecdsa.PrivateKey

	RelateTimes        int
	RelateVerifyParams [][]byte
}

func NewUser(userID string, sd *pkg.Web2Data) (*User, error) {
	if sd == nil {
		return nil, errors.New("invalid selfData")
	}
	LogDebugf("before NewUser, %s, %+v", userID, sd)

	// create user
	user := &User{
		SelfData: &SelfData{},
	}
	user.SelfData.Web2Data = *sd
	if len(sd.Web2Public) > 0 {
		web2Public, err := crypto.DecompressPubkey(sd.Web2Public)
		if err != nil {
			return nil, err
		}
		user.Web2Public = web2Public
	}
	if len(sd.Web3Public) > 0 {
		web3Public, err := crypto.DecompressPubkey(sd.Web3Public)
		if err != nil {
			return nil, err
		}
		user.Web3Public = web3Public
		LogDebugf("NewUser successed, user: %+v, %s, %v", user)
	}

	// cache user
	SetCache(userID, user, true)
	return user, nil
}

func GetUser(userID string) *User {
	if u := GetCache(userID, false, nil); u != nil {
		return u.(*User)
	}
	return nil
}

func (p *User) Load(recoverID, selfPrivate, selfKey string) ([]byte, error) {
	LogDebugf("before user load: %v, %v, %v", recoverID, selfPrivate, selfKey)

	if selfPrivate != "" && p.SelfPrivate == nil {
		// decrypt: contract.web3Key + web2Private => web3Key
		selfPrivateBuf, err := hexutil.Decode(selfPrivate)
		if err != nil {
			return nil, fmt.Errorf("Load failed, hexutil.Decode: %s", err)
		}
		LogDebugln(selfPrivateBuf)
		privateKey, err := crypto.ToECDSA(rscrypto.AesDecryptECB(selfPrivateBuf, p.SelfData.Web2DHKey))
		if err != nil {
			return nil, fmt.Errorf("Load failed, ecies.ToECDSA: %s", err.Error())
		}
		p.SelfPrivate = privateKey
	}

	var decryptSelfKey []byte
	if selfKey != "" {
		// decrypt: contract.web3Public + web3Key => web3Public
		selfKeyBuf, err := hexutil.Decode(selfKey)
		if err != nil {
			return nil, fmt.Errorf("Load failed, hexutil.Decode: %s", err)
		}
		selfDHKey, err := rscrypto.GetDhKey(p.Web3Public, p.SelfPrivate)
		if err != nil {
			return nil, err
		}
		decryptSelfKey = rscrypto.AesDecryptECB(selfKeyBuf, []byte(selfDHKey))

		LogDebugf("decrypt web3Key successed: %s, %v, %v", selfKey, selfDHKey, string(decryptSelfKey))
	}
	if recoverID != "" && p.RecoverID == "" {
		p.RecoverID = recoverID
	}
	LogDebugf("LoadUser successed, user: %+v", *p)
	return decryptSelfKey, nil
}

func (p *User) Register(recoverID string) (*Web3User, error) {
	if recoverID == "" {
		return nil, errors.New("invalid recoverID")
	}
	if p.Web3Public != nil && p.SelfPrivate != nil {
		return nil, fmt.Errorf("user registration again and again")
	}
	web3User := &Web3User{}

	// web3Public
	web3Private, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	p.Web3Public = &web3Private.PublicKey

	// selfParivate
	// TODO: selfPrivate + web2Key + selfPass => contract.SelfKey
	selfPrivate, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	p.SelfPrivate = selfPrivate
	web2DHKey, err := rscrypto.GetDhKey(p.Web2Public, web3Private)
	if err != nil {
		return nil, err
	}
	p.SelfData.Web2DHKey = []byte(web2DHKey)
	p.SelfData.SelfAddress = crypto.PubkeyToAddress(selfPrivate.PublicKey).Hex()
	LogDebugf("before encrypt selfPrivate: %v, %v, %v", vWorker.web2NetPublic, web3Private, web2DHKey)
	selfPrivateBuf := rscrypto.AesEncryptECB(crypto.FromECDSA(selfPrivate), []byte(web2DHKey))
	web3User.SelfPrivate = hexutil.Encode(selfPrivateBuf)

	// selfKey
	selfKey := rscrypto.GetRandom(32, false)
	selfDHKey, err := rscrypto.GetDhKey(p.Web3Public, p.SelfPrivate)
	if err != nil {
		return nil, err
	}
	web3User.SelfKey = hexutil.Encode(rscrypto.AesEncryptECB([]byte(selfKey), []byte(selfDHKey)))

	// recoverID
	sig, err := crypto.Sign(rscrypto.EthHash([]byte(recoverID)).Bytes(), web3Private)
	if err != nil {
		return nil, err
	}
	web3User.RecoverID = hexutil.Encode(sig)

	// init selfData
	p.SelfData.Web2Data = *Web2DataInit(p, []byte(selfKey))
	qrcode, err := p.InitTOTPKey()
	if err != nil {
		return nil, err
	}
	web2Data, err := Web2EncryptWeb2Data(p)
	if err != nil {
		return nil, err
	}
	web3User.QRCode = qrcode
	web3User.Web2Data = web2Data
	web3User.SelfAddress = p.SelfData.SelfAddress
	LogDebugf("after encrypt: user: %+v, web2Data: %+v, web3User: %+v, web2DHKey: %s", p, web2Data, web3User, p.SelfData.Web2DHKey)
	return web3User, nil
}

func (p *User) ResetSelfPass(kind, random, web2Key string) (any, error) {
	// if kind == "email" {
	// 	emailRandom := rscrypto.GetRandom(6, false)
	// 	SetCache("ResetWeb2Key", rscrypto.GetRandom(6, false), true)
	// 	return emailRandom, nil
	// }
	// if web2Key == "" || web2Key == string(p.Web2Key) {
	// 	return nil, errors.New("invalid input web2Key")
	// }

	// // random verify
	// if cacheRandom := fmt.Sprintf("%v", GetCache("ResetWeb2Key", true, nil)); kind == "TOTP" && cacheRandom != "nil" && cacheRandom != "" {
	// 	if randomBuf, err := hexutil.Decode(random); err != nil {
	// 		return nil, err
	// 	} else if random = string(rscrypto.AesDecryptECB(randomBuf, p.Web2Key)); random != cacheRandom {
	// 		return nil, fmt.Errorf("invalid random, cache: %v, random: %s", cacheRandom, random)
	// 	}
	// }
	// p.Web2Key = []byte(p.Web2Key)
	// return Web2EncryptWeb2Data(p)
	return nil, nil
}

func (p *User) ResetTOTPKey(recoverID, encryptedRecoverID string) (any, error) {
	LogDebugln(recoverID, encryptedRecoverID)

	// verify recoverID
	recoverIDBuf, err := hexutil.Decode(encryptedRecoverID)
	if err != nil {
		return nil, err
	}
	if !crypto.VerifySignature(crypto.CompressPubkey(p.Web3Public), rscrypto.EthHash([]byte(recoverID)).Bytes(), recoverIDBuf[:len(recoverIDBuf)-1]) {
		return nil, fmt.Errorf("invalid pushID for recovery, pushID: %s", recoverID)
	}

	qrcode, err := p.InitTOTPKey()
	if err != nil {
		return nil, err
	}
	web2Data, err := Web2EncryptWeb2Data(p)
	if err != nil {
		return nil, err
	}
	web3User := &Web3User{
		QRCode:   qrcode,
		Web2Data: web2Data,
	}
	LogDebugf("ResetTOTPKey web3User: %+v", web3User)
	return web3User, nil
}

func (p *User) InitTOTPKey() (string, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.SelfPrivate)
	if err != nil {
		return "", err
	}
	tmpDhKey, err := rscrypto.GetDhKey(vWorker.web2NetPublic, vWorker.private)
	if err != nil {
		return "", err
	}
	LogDebugf("InitTOTPKey successed: %s, %s", dhKey, tmpDhKey)
	p.SelfData.TOTPKey = rscrypto.AesEncryptECB([]byte(tmpDhKey), []byte(dhKey))
	return tmpDhKey, nil
}

func (p *User) GetTOTPKey() (string, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.SelfPrivate)
	if err != nil {
		return "", err
	}
	totpKey := string(rscrypto.AesDecryptECB(p.SelfData.TOTPKey, []byte(dhKey)))
	LogDebugf("GetTOTPKey successed: %v, %s, %s", p.SelfData.TOTPKey, totpKey, dhKey)
	return totpKey, nil
}

func (p *User) VerifyTOTP(code string) error {
	secret, err := p.GetTOTPKey()
	if err != nil {
		return err
	}

	otpConfig := &dgoogauth.OTPConfig{
		Secret:      strings.TrimSpace(secret),
		WindowSize:  3,
		HotpCounter: 0,
		// UTC:         true,
	}
	if ok, err := otpConfig.Authenticate(strings.TrimSpace(code)); err != nil {
		return err
	} else if !ok {
		return errors.New("dynamic authorization verify failed: TOTP")
	}
	return nil
}

func (p *User) HandleAssociatedVerify(action, relateTimes string) (any, error) {
	// handle relate times
	if rtimes, err := strconv.Atoi(relateTimes); err == nil {
		if rtimes > 0 && p.RelateTimes == 0 {
			p.RelateTimes = rtimes
		}
	} else {
		return nil, nil
	}
	if p.RelateTimes > 0 {
		p.RelateTimes -= 1
		p.SelfData.VerifyNonce += 1
		p.RelateVerifyParams = append(p.RelateVerifyParams, rscrypto.AesEncryptECB([]byte(fmt.Sprintf("%d", p.SelfData.VerifyNonce)), p.SelfData.Web2DHKey))
	}

	// Automatically trigger association verification parameter collection
	if p.RelateTimes == 0 {
		result, err := p.PackSigAndMerkleLeaves(action)
		p.RelateVerifyParams = make([][]byte, 0)
		return result, err
	}
	return nil, nil
}

func (p *User) PackSigAndMerkleLeaves(action string) (any, error) {
	LogDebugf("before PackSigAndMerkleLeaves, user: %+v, action: %s", p, action)
	message := fmt.Sprintf("%d", time.Now().UnixNano())
	msgHash := rscrypto.EthHash([]byte(message))
	sig, err := crypto.Sign(msgHash.Bytes(), p.SelfPrivate)
	if err != nil {
		return nil, err
	}

	// debug logic
	// verifyResult := crypto.VerifySignature(crypto.CompressPubkey(&p.Web2Private.PublicKey), msgHash.Bytes(), sig[:len(sig)-1])
	// verifyPublic, _ := crypto.Ecrecover(msgHash.Bytes(), sig)
	// verifySuccessed := bytes.Compare(crypto.FromECDSAPub(&p.Web2Private.PublicKey), verifyPublic) == 0
	// LogDebugf("sig: %s, %v, %d, verify: %v, err: %+v, result: %v", hexutil.Encode(sig), sig, len(sig), verifySuccessed, err, verifyResult)

	sig[64] = uint8(int(sig[64])) + 27 // Yes add 27, weird Ethereum quirk
	ethMsg := fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len([]byte(message)), message)
	output := struct {
		SelfMsg            string   `json:"selfMsg"`
		Message            string   `json:"message"`
		Signature          string   `json:"signature"`
		SelfAuth 		   [][]byte `json:"selfAuth"`
	}{
		message,
		ethMsg,
		hexutil.Encode(sig),
		p.RelateVerifyParams,
	}
	return output, nil
}
