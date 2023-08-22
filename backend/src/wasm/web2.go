package wasm

import (
	"crypto/ecdsa"
	"selfweb3/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func Web2Init() (*ecdsa.PrivateKey, error) {
	private, err := crypto.GenerateKey()
	LogDebugf("Web2Init successed, private: %s", hexutil.Encode(crypto.FromECDSA(private)))
	return private, err
}

func Web2EncodePrivate(user *User) (string, error) {
	web2Private := append(crypto.FromECDSA(user.Web2Private), user.TOTPKey...)
	return hexutil.Encode(rscrypto.AesEncryptECB(web2Private, user.Web2Key)), nil
}
