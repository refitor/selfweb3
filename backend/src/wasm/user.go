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

	SelfKey3     string
	RecoverID3   string
	SelfPrivate3 string

	// init by NewUser
	Web2PublicKey *ecdsa.PublicKey
	Web3PublicKey *ecdsa.PublicKey

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
		user.Web2PublicKey = web2Public
	}
	if len(sd.Web3Public) > 0 {
		web3Public, err := crypto.DecompressPubkey(sd.Web3Public)
		if err != nil {
			return nil, err
		}
		user.Web3PublicKey = web3Public
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

func (p *User) Load() ([]byte, *ecdsa.PrivateKey, error) {
	LogDebugf("before user load: %v, %v, %v", p.RecoverID3, p.SelfPrivate3, p.SelfKey3)

	if p.SelfKey3 != "" {
		// decrypt: contract.selfKey + wallet + web2DHKey => selfKey
		selfKeyBuf, err := hexutil.Decode(p.SelfKey3)
		if err != nil {
			return nil, nil, fmt.Errorf("Load failed, hexutil.Decode: %s", err)
		}
		decryptSelfKey := rscrypto.AesDecryptECB(selfKeyBuf, []byte(p.SelfData.Web2DHKey))
		LogDebugf("decrypt selfKey successed: %s, %v, %v", p.SelfKey3, p.SelfData.Web2DHKey, string(decryptSelfKey))

		// decrypt: contract.selfPrivate + selfKey => selfPrivate
		if p.SelfPrivate3 != "" {
			selfPrivateBuf, err := hexutil.Decode(p.SelfPrivate3)
			if err != nil {
				return nil, nil, fmt.Errorf("Load failed, hexutil.Decode: %s", err)
			}
			LogDebugln(selfPrivateBuf)
			privateKey, err := crypto.ToECDSA(rscrypto.AesDecryptECB(selfPrivateBuf, decryptSelfKey))
			if err != nil {
				return nil, nil, fmt.Errorf("Load failed, ecies.ToECDSA: %s", err.Error())
			}
			LogDebugf("user Load successed, privateKey: %+v, user: %+v", privateKey, p)
			return decryptSelfKey, privateKey, nil
		}
	}
	return nil, nil, errors.New("user Load failed with invalid params")
}

func (p *User) Register(recoverID string) (*Web3User, error) {
	if recoverID == "" {
		return nil, errors.New("invalid recoverID")
	}
	wd2 := &pkg.Web2Data{}

	// init web3User
	web3User, selfKey, selfPrivate, err := p.InitWeb3User(recoverID, wd2)
	if err != nil {
		return nil, err
	}
	qrcode, totpKey, err := p.InitTOTPKey(selfPrivate)
	if err != nil {
		return nil, err
	}
	wd2.TOTPKey = totpKey

	// init selfData
	p.SelfData.Web2Data = *Web2DataInit(wd2, p, []byte(selfKey))
	web2Data, err := Web2EncryptWeb2Data(&p.SelfData.Web2Data)
	if err != nil {
		return nil, err
	}
	web3User.QRCode = qrcode
	web3User.Web2Data = web2Data
	LogDebugf("after encrypt: user: %+v, web2Data: %+v, web3User: %+v, web2DHKey: %s", p, web2Data, web3User, p.SelfData.Web2DHKey)
	return web3User, nil
}

func (p *User) InitWeb3User(recoverID string, wd2 *pkg.Web2Data) (*Web3User, string, *ecdsa.PrivateKey, error) {
	// web3Public
	web3Private, err := crypto.GenerateKey()
	if err != nil {
		return nil, "", nil, err
	}
	p.Web3PublicKey = &web3Private.PublicKey
	wd2.Web3Public = crypto.CompressPubkey(p.Web3PublicKey)
	LogDebugf("after generate web3Public: %+v", crypto.PubkeyToAddress(*p.Web3PublicKey).Hex())

	// selfKey + web2DHKey + wallet => contract.selfKey
	web2DHKey, err := rscrypto.GetDhKey(p.Web2PublicKey, web3Private)
	if err != nil {
		return nil, "", nil, err
	}
	wd2.Web2DHKey = []byte(web2DHKey)
	selfKey := rscrypto.GetRandom(32, false)
	selfKey3 := hexutil.Encode(rscrypto.AesEncryptECB([]byte(selfKey), []byte(web2DHKey)))
	LogDebugf("after encrypt selfKey: selfKey: %s, web2DHKey: %s, selfKey3: %s", selfKey, web2DHKey, selfKey3)
	p.SelfKey3 = selfKey3

	// selfPrivate + selfKey => contract.selfPrivate
	selfPrivate, err := crypto.GenerateKey()
	if err != nil {
		return nil, "", nil, err
	}
	wd2.SelfAddress = crypto.PubkeyToAddress(selfPrivate.PublicKey).Hex()
	selfPrivateKey3 := hexutil.Encode(rscrypto.AesEncryptECB(crypto.FromECDSA(selfPrivate), []byte(selfKey)))
	LogDebugf("after encrypt selfPrivate: wd2.SelfAddress: %s, selfKey: %s, selfPrivateKey3: %s", wd2.SelfAddress, selfKey, selfPrivateKey3)
	p.SelfPrivate3 = selfPrivateKey3

	// recoverID + web3Private => contract.recoverID
	sig, err := crypto.Sign(rscrypto.EthHash([]byte(recoverID)).Bytes(), web3Private)
	if err != nil {
		return nil, "", nil, err
	}
	recoverID3 := hexutil.Encode(sig)
	LogDebugf("after generate recoverID signature, recoverID: %s, recoverID3: %s", recoverID, recoverID3)
	p.RecoverID3 = recoverID3

	return &Web3User{
		// QRCode:      qrcode,
		SelfKey: selfKey3,
		// Web2Data:    web2Data,
		RecoverID:   recoverID3,
		SelfPrivate: selfPrivateKey3,
		SelfAddress: wd2.SelfAddress,
	}, selfKey, selfPrivate, nil
}

func (p *User) ResetSelfKey(kind, random, web2Key string, selfPrivate *ecdsa.PrivateKey) (any, error) {
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

func (p *User) ResetTOTPKey(recoverID, encryptedRecoverID string, selfPrivate *ecdsa.PrivateKey) (any, error) {
	// verify recoverID
	recoverIDBuf, err := hexutil.Decode(encryptedRecoverID)
	if err != nil {
		return nil, err
	}
	if !crypto.VerifySignature(crypto.CompressPubkey(p.Web3PublicKey), rscrypto.EthHash([]byte(recoverID)).Bytes(), recoverIDBuf[:len(recoverIDBuf)-1]) {
		return nil, fmt.Errorf("invalid pushID for recovery, pushID: %s", recoverID)
	}

	qrcode, totpKey, err := p.InitTOTPKey(selfPrivate)
	if err != nil {
		return nil, err
	}
	p.SelfData.TOTPKey = totpKey
	web2Data, err := Web2EncryptWeb2Data(&p.SelfData.Web2Data)
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

func (p *User) InitTOTPKey(selfPrivate *ecdsa.PrivateKey) (string, []byte, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3PublicKey, selfPrivate)
	if err != nil {
		return "", nil, err
	}
	tmpDhKey, err := rscrypto.GetDhKey(vWorker.web2NetPublic, vWorker.private)
	if err != nil {
		return "", nil, err
	}
	LogDebugf("InitTOTPKey successed: %s, %s", dhKey, tmpDhKey)
	totpKey := rscrypto.AesEncryptECB([]byte(tmpDhKey), []byte(dhKey))
	return tmpDhKey, totpKey, nil
}

func (p *User) GetTOTPKey(selfPrivate *ecdsa.PrivateKey) (string, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3PublicKey, selfPrivate)
	if err != nil {
		return "", err
	}
	totpKey := string(rscrypto.AesDecryptECB(p.SelfData.TOTPKey, []byte(dhKey)))
	LogDebugf("GetTOTPKey successed: %v, %s, %s", p.SelfData.TOTPKey, totpKey, dhKey)
	return totpKey, nil
}

func (p *User) VerifyTOTP(code string) error {
	_, selfPrivate, err := p.Load()
	if err != nil {
		return err
	}
	secret, err := p.GetTOTPKey(selfPrivate)
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

func (p *User) HandleAssociatedVerify(action, relateTimes string, selfPrivate *ecdsa.PrivateKey) (any, error) {
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
		result, err := p.PackSigAndMerkleLeaves(action, selfPrivate)
		p.RelateVerifyParams = make([][]byte, 0)
		return result, err
	}
	return nil, nil
}

func (p *User) PackSigAndMerkleLeaves(action string, selfPrivate *ecdsa.PrivateKey) (any, error) {
	LogDebugf("before PackSigAndMerkleLeaves, user: %+v, action: %s", p, action)
	message := fmt.Sprintf("%d", time.Now().UnixNano())
	msgHash := rscrypto.EthHash([]byte(message))
	sig, err := crypto.Sign(msgHash.Bytes(), selfPrivate)
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
		SelfMsg   string   `json:"selfMsg"`
		Message   string   `json:"message"`
		Signature string   `json:"signature"`
		SelfAuth  [][]byte `json:"selfAuth"`
	}{
		message,
		ethMsg,
		hexutil.Encode(sig),
		p.RelateVerifyParams,
	}
	return output, nil
}
