package pkg

import (
	"crypto/ecdsa"
	"encoding/json"
	"errors"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

const (
	C_Web2Key       = "Web2Key"
	C_Web2Datas     = "Web2Datas"
	C_Web2Private   = "Web2Private"
	C_Web2NetPublic = "Web2NetPublic"

	C_AuthorizeID   = "AuthorizeID"
	C_AuthorizeCode = "AuthorizeCode"
)

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

func Web2DecodeEx(privateKey *ecdsa.PrivateKey, public, data string) (map[string]any, error) {
	publicKey, err := rscrypto.GetPublicKey(public)
	if err != nil {
		return nil, err
	}
	return Web2Decode(privateKey, publicKey, data)
}

func Web2Decode(privateKey *ecdsa.PrivateKey, publicKey *ecdsa.PublicKey, data string) (map[string]any, error) {
	if publicKey == nil {
		return nil, errors.New("invalid web2 network public key")
	}
	dataBuf, err := hexutil.Decode(data)
	if err != nil {
		return nil, err
	}
	dhKey, err := rscrypto.GetDhKey(publicKey, privateKey)
	if err != nil {
		return nil, err
	}

	dataMap := make(map[string]any)
	if err := json.Unmarshal(rscrypto.AesDecryptECB(dataBuf, []byte(dhKey)), &dataMap); err != nil {
		return nil, err
	}
	return dataMap, nil
}
