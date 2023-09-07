package wasm

import (
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"

	"selfweb3/backend/pkg/rscrypto"

	"github.com/dgryski/dgoogauth"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

const (
	c_param_random     = "random"
	c_param_web2Key    = "web2Key"
	c_param_web3Key    = "web3Key"
	c_param_recoverID  = "recoverID"
	c_param_web3Public = "web3Public"
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
	WebAuthnKey []byte
	Web3Public  *ecdsa.PublicKey
	Web2Private *ecdsa.PrivateKey
}

func NewUser(userID, web2Key, webAuthnKey, web2Private string) (*User, error) {
	if web2Key == "" {
		return nil, errors.New("invalid web2Key or web2PrivateKey")
	}
	LogDebugf("before NewUser, %s, %s, %s", userID, web2Key, web2Private)

	// create user
	user := &User{
		ID:          userID,
		Web2Key:     []byte(web2Key),
		WebAuthnKey: []byte(webAuthnKey),
	}
	if web2Private == "" {
		private, err := Web2Init()
		if err != nil {
			return nil, err
		}
		user.Web2Private = private
	} else {
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
		LogDebugf("NewUser successed, user: %+v, %s, %v", user, hexutil.Encode(crypto.FromECDSA(web2PrivateKey)), user.TOTPKey)
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

func (p *User) Load(web3Key, web3Public string) error {
	LogDebugf("before user load: %v, %v, %v", web3Key, web3Public, hexutil.Encode(crypto.FromECDSA(p.Web2Private)))

	if web3Public != "" {
		// decrypt: contract.web3Key + web2Private => web3Key
		web3PublicBuf, err := hexutil.Decode(web3Public)
		if err != nil {
			return fmt.Errorf("Load failed, hexutil.Decode: %s", err)
		}
		LogDebugln(web3PublicBuf)
		publicKeyBuf, err := ecies.ImportECDSA(p.Web2Private).Decrypt(web3PublicBuf, nil, nil)
		if err != nil {
			return fmt.Errorf("Load failed, ecies.ImportECDSA: %s", err)
		}
		publicKey, err := crypto.DecompressPubkey(publicKeyBuf)
		if err != nil {
			return fmt.Errorf("Load failed, GetPublicKey: %s", err)
		}
		p.Web3Public = publicKey
	}

	if web3Key != "" {
		// decrypt: contract.web3Public + web3Key => web3Public
		web3KeyBuf, err := hexutil.Decode(web3Key)
		if err != nil {
			return fmt.Errorf("Load failed, hexutil.Decode: %s", err)
		}
		dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.Web2Private)
		if err != nil {
			return err
		}
		p.Web3Key = rscrypto.AesDecryptECB(web3KeyBuf, []byte(dhKey))
		LogDebugf("decrypt web3Key successed: %s, %v, %v", web3Key, p.Web3Key, p.WebAuthnKey)
	}

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
	private, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	web3KeyBuf, err := ecies.Encrypt(rand.Reader, ecies.ImportECDSAPublic(&p.Web2Private.PublicKey), crypto.CompressPubkey(&private.PublicKey), nil, nil)
	if err != nil {
		return nil, err
	}
	web3User.Web3Public = hexutil.Encode(web3KeyBuf)

	web3Key := rscrypto.GetRandom(32, false)
	dhKey, err := rscrypto.GetDhKey(&private.PublicKey, p.Web2Private)
	if err != nil {
		return nil, err
	}
	web3User.Web3Key = hexutil.Encode(rscrypto.AesEncryptECB([]byte(web3Key), []byte(dhKey)))

	// encrypt: dhKey + recoverID => contract.recoverID
	sig, err := crypto.Sign(crypto.Keccak256([]byte(recoverID)), private)
	if err != nil {
		return nil, err
	}
	web3User.RecoverID = hexutil.Encode(sig)

	// update user
	p.Web3Key = []byte(web3Key)
	p.Web3Public = &private.PublicKey
	p.WebAuthnKey = []byte(hexutil.Encode(rscrypto.AesEncryptECB(p.WebAuthnKey, p.Web3Key)))

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
	LogDebugf("after encrypt: %+v, %+v, web3KeyBuf: %v, web2Private: %s", p, web3User, web3KeyBuf, hexutil.Encode(crypto.FromECDSA(p.Web2Private)))
	return web3User, nil
}

func (p *User) ResetWeb2Key(kind, random, web2Key string) (any, error) {
	if kind == "email" {
		emailRandom := rscrypto.GetRandom(6, false)
		SetCache("ResetWeb2Key", rscrypto.GetRandom(6, false), true)
		return emailRandom, nil
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
	return Web2EncryptWeb2Data(p)
}

func (p *User) ResetTOTPKey(recoverID, encryptedRecoverID string) (any, error) {
	LogDebugln(recoverID, encryptedRecoverID)

	// verify recoverID
	recoverIDBuf, err := hexutil.Decode(encryptedRecoverID)
	if err != nil {
		return nil, err
	}
	signPublic, err := crypto.SigToPub(crypto.Keccak256([]byte(recoverID)), recoverIDBuf)
	if err != nil {
		return nil, err
	}
	if !signPublic.Equal(p.Web3Public) {
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
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.Web2Private)
	if err != nil {
		return "", err
	}
	tmpDhKey, err := rscrypto.GetDhKey(vWorker.web2NetPublic, vWorker.private)
	if err != nil {
		return "", err
	}
	LogDebugf("InitTOTPKey successed: %s, %s", dhKey, tmpDhKey)
	p.TOTPKey = rscrypto.AesEncryptECB([]byte(tmpDhKey), []byte(dhKey))
	return tmpDhKey, nil
}

func (p *User) GetTOTPKey() (string, error) {
	dhKey, err := rscrypto.GetDhKey(p.Web3Public, p.Web2Private)
	if err != nil {
		return "", err
	}
	LogDebugf("GetTOTPKey successed: %v, %s, %s", p.TOTPKey, string(rscrypto.AesDecryptECB(p.TOTPKey, []byte(dhKey))), dhKey)
	return string(rscrypto.AesDecryptECB(p.TOTPKey, []byte(dhKey))), nil
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
