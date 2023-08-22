package wasm

import (
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"

	"selfweb3/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

const (
	c_name_user = "user"
	c_mode_web2 = "web2"
	c_mode_web3 = "web3"

	c_param_random  = "random"
	c_param_web2Key = "web2Key"

	c_param_web3Key    = "web3Key"
	c_param_recoverID  = "recoverID"
	c_param_web3Public = "web3Public"

	c_dapp_SelfVault      = "SelfVault"
	c_method_Web2Private  = "Web2Private"
	c_method_ResetTOTPKey = "ResetTOTPKey"
	c_method_ResetWeb2Key = "ResetWeb2Key"
)

type Web3User struct {
	QRCode     string
	Web2Data   string
	Web3Key    string
	RecoverID  string
	Web3Public string
}

type User struct {
	ID          string
	Web3Key     []byte
	Web2Key     []byte
	TOTPKey     []byte
	Web3Public  *ecdsa.PublicKey
	Web2Private *ecdsa.PrivateKey
}

func NewUser(userID, web2Key, web2Private string) (*User, error) {
	if web2Key == "" || web2Private == "" {
		return nil, errors.New("invalid web2Key or web2PrivateKey")
	}
	LogDebugf("before NewUser, %s, %s, %s", userID, web2Key, web2Private)

	// create user
	user := &User{
		ID:      userID,
		Web2Key: []byte(web2Key),
	}

	// decode web2 private key
	web2PrivateBuf, err := hexutil.Decode(web2Private)
	if err != nil {
		return nil, err
	}
	privateBuf := rscrypto.AesDecryptECB(web2PrivateBuf, []byte(web2Key))
	web2PrivateKey, err := crypto.ToECDSA(privateBuf[0:32])
	if err != nil {
		return nil, err
	}
	if len(privateBuf) > 32 {
		user.TOTPKey = privateBuf[32:]
	}
	user.Web2Private = web2PrivateKey
	LogDebugf("NewUser successed, user: %+v, %s", user, hexutil.Encode(crypto.FromECDSA(web2PrivateKey)))

	SetCache(userID, user, true)
	return user, nil
}

func GetUser(userID string) *User {
	if u := GetCache(userID, false, nil); u != nil {
		return u.(*User)
	}
	return nil
}

func (p *User) Load(web3Key, web3Public string) error {
	if web3Key == "" || web3Public == "" {
		return errors.New("invalid user web3Key or web3Public")
	}
	LogDebugf("before user load: %v, %v, %v", web3Key, web3Public, hexutil.Encode(crypto.FromECDSA(p.Web2Private)))

	// decrypt: contract.web3Key + web2Private => web3Key
	web3KeyBuf, err := hexutil.Decode(web3Key)
	if err != nil {
		return fmt.Errorf("Load failed, hexutil.Decode: %s", err)
	}
	keyBuf, err := ecies.ImportECDSA(p.Web2Private).Decrypt(web3KeyBuf, nil, nil)
	if err != nil {
		return fmt.Errorf("Load failed, ecies.ImportECDSA: %s", err)
	}
	p.Web3Key = keyBuf
	LogDebugf("decrypt web3Key successed: %s, %s", web3Key, p.Web3Key)

	// decrypt: contract.web3Public + web3Key => web3Public
	web3PublicBuf, err := hexutil.Decode(web3Public)
	if err != nil {
		return fmt.Errorf("Load failed, hexutil.Decode: %s", err)
	}
	publicKey, err := crypto.DecompressPubkey(rscrypto.AesDecryptECB(web3PublicBuf, p.Web3Key))
	if err != nil {
		return fmt.Errorf("Load failed, GetPublicKey: %s", err)
	}
	p.Web3Public = publicKey

	LogDebugf("LoadUser successed, user: %+v", *p)
	return nil
}

func (p *User) Register(recoverID string) (*Web3User, error) {
	if recoverID == "" {
		return nil, errors.New("invalid recoverID")
	}
	if len(p.Web3Key) > 0 || p.Web3Public != nil {
		return nil, fmt.Errorf("user registration again and again")
	}
	web3User := &Web3User{}

	// encrypt: web3Public + web3Key => contract.web3Public
	// encrypt: web3Key + web2Public => contract.web3Key
	web3Key := rscrypto.GetRandom(32, false)
	web3KeyBuf, err := ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(&p.Web2Private.PublicKey), []byte(web3Key), nil, nil)
	if err != nil {
		return nil, err
	}
	web3User.Web3Key = hexutil.Encode(web3KeyBuf)
	private, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	web3User.Web3Public = hexutil.Encode(rscrypto.AesEncryptECB(crypto.CompressPubkey(&private.PublicKey), []byte(web3Key)))

	// encrypt: dhKey + recoverID => contract.recoverID
	dhKey, err := rscrypto.GetDhKey(&private.PublicKey, p.Web2Private)
	if err != nil {
		return nil, err
	}
	web3User.RecoverID = hexutil.Encode(rscrypto.AesEncryptECB([]byte(recoverID), []byte(dhKey)))

	// update user
	p.Web3Key = []byte(web3Key)
	p.Web3Public = &private.PublicKey

	qrcode, err := p.InitTOTPKey()
	if err != nil {
		return nil, err
	}
	web2Data, err := Web2EncodePrivate(p)
	if err != nil {
		return nil, err
	}
	web3User.QRCode = qrcode
	web3User.Web2Data = web2Data
	LogDebugf("after encrypt: %+v, %+v, web3KeyBuf: %v, web2Private: %s", p, web3User, web3KeyBuf, hexutil.Encode(crypto.FromECDSA(p.Web2Private)))
	return web3User, nil
}

func (p *User) ResetWeb2Key(kind, random, web2Key string) (any, error) {
	if kind == "email" {
		SetCache("ResetWeb2Key", rscrypto.GetRandom(6, false), true)
		return "successed", nil
	}
	if web2Key == "" || web2Key == string(p.Web2Key) {
		return nil, errors.New("invalid input web2Key")
	}

	// random verify
	if cacheRandom := fmt.Sprintf("%v", GetCache("ResetWeb2Key", true, nil)); kind == "TOTP" && cacheRandom != "nil" && cacheRandom != "" {
		if randomBuf, err := hexutil.Decode(random); err != nil {
			return nil, err
		} else if random = string(rscrypto.AesDecryptECB(randomBuf, p.Web2Key)); random != cacheRandom {
			return nil, fmt.Errorf("invalid random, cache: %v, random: %s", cacheRandom, random)
		}
	}
	p.Web2Key = []byte(p.Web2Key)
	return Web2EncodePrivate(p)
}

func (p *User) ResetTOTPKey(recoverID, encryptedRecoverID string) (any, error) {
	LogDebugln(recoverID, encryptedRecoverID)

	// verify recoverID
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.Web2Private)
	if err != nil {
		return nil, err
	}
	if encryptedRecoverID != hexutil.Encode(rscrypto.AesEncryptECB([]byte(recoverID), []byte(dhKey))) {
		return nil, fmt.Errorf("invalid pushID for recovery, pushID: %s", recoverID)
	}

	qrcode, err := p.InitTOTPKey()
	if err != nil {
		return nil, err
	}
	web2Data, err := Web2EncodePrivate(p)
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

// {"method": "demo", "random": "", "params1": "", "param2": "", ...}
func (p *User) Handle(kind string, params ...string) (handleResult any, handleErr error) {
	if len(params) == 0 {
		return "successed", nil
	}
	paramMap := make(map[string]string, 0)
	if err := json.Unmarshal([]byte(params[0]), &paramMap); err != nil {
		return nil, err
	}
	LogDebugf("before Handle: %v, %v, %v", params, paramMap[c_param_web3Key], paramMap[c_param_web3Public])

	handleResult = "successed"
	switch paramMap["method"] {
	case c_dapp_SelfVault:
		handleErr = p.Load(paramMap[c_param_web3Key], paramMap[c_param_web3Public])
	case c_method_ResetTOTPKey:
		if err := p.Load(paramMap[c_param_web3Key], paramMap[c_param_web3Public]); err != nil {
			return nil, err
		}
		return p.ResetTOTPKey(params[len(params)-1], paramMap[c_param_recoverID])
	case c_method_ResetWeb2Key:
		return p.ResetWeb2Key(kind, paramMap[c_param_random], paramMap[c_param_web2Key])
	case c_method_Web2Private:
		return Web2EncodePrivate(p)
	}
	return
}

func (p *User) InitTOTPKey() (string, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.Web2Private)
	if err != nil {
		return "", err
	}
	tmpDhKey, err := rscrypto.GetDhKey(vWorker.web2NetPublic, vWorker.private)
	if err != nil {
		return "", err
	}
	LogDebugf("SetTOTPKey successed: %s, %s", dhKey, tmpDhKey)
	p.TOTPKey = rscrypto.AesEncryptECB([]byte(tmpDhKey), []byte(dhKey))
	return tmpDhKey, nil
}

func (p *User) GetTOTPKey() (string, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.Web2Private)
	if err != nil {
		return "", err
	}
	return string(rscrypto.AesDecryptECB(p.TOTPKey, []byte(dhKey))), nil
}
