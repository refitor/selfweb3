package pkg

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	C_SelfID        = "SelfID"
	C_Web2Data      = "Web2Data"
	C_Web2Address   = "Web2Address"
	C_AuthorizeID   = "AuthorizeID"
	C_AuthorizeCode = "AuthorizeCode"
	C_Web2NetPublic = "Web2NetPublic"
)

type Web2Data struct {
	TOTPKey     []byte
	Web2DHKey   []byte
	Web2Public  []byte
	Web3Public  []byte
	WebAuthnKey []byte
	SelfAddress string

	// TODO: web3 storage
	VerifyNonce int64
}

func Web2EncodeEx(priavateKey *ecdsa.PrivateKey, public string, data any) (string, error) {
	publicKey, err := rscrypto.GetPublicKey(public)
	if err != nil {
		return "", err
	}
	return Web2Encode(priavateKey, publicKey, data)
}

func Web2Encode(priavateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, data any) (string, error) {
	if publicKey == nil {
		return "", errors.New("invalid web2 network public key")
	}
	dataBuf, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	dhKey, err := rscrypto.GetDhKey(publicKey, priavateKey)
	if err != nil {
		return "", err
	}
	return hexutil.Encode(rscrypto.AesEncryptECB(dataBuf, []byte(dhKey))), nil
}

func Web2DecodeEx(privateKey *ecdsa.PrivateKey, public, data string, ptrObject any) error {
	publicKey, err := rscrypto.GetPublicKey(public)
	if err != nil {
		return err
	}
	return Web2Decode(privateKey, publicKey, data, ptrObject)
}

func Web2Decode(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, data string, ptrObject any) error {
	if publicKey == nil {
		return errors.New("invalid web2 network public key")
	}
	dataBuf, err := hexutil.Decode(data)
	if err != nil {
		return err
	}
	dhKey, err := rscrypto.GetDhKey(publicKey, privateKey)
	if err != nil {
		return err
	}

	if ptrObject == nil {
		return errors.New("invalid ptrObject for Web2Decode")
	}
	if err := json.Unmarshal(rscrypto.AesDecryptECB(dataBuf, []byte(dhKey)), ptrObject); err != nil {
		return err
	}
	return nil
}
