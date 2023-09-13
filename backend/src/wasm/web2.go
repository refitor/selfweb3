package wasm

import (
	"crypto/ecdsa"
	"errors"
	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

func Web2Init() (*ecdsa.PrivateKey, error) {
	return crypto.GenerateKey()
}

func Web2EncryptWeb2Data(user *User) (string, error) {
	wd2 := &pkg.Web2Data{
		Nonce:       user.Web2Data.Nonce,
		WebAuthnKey: string(user.WebAuthnKey),
		Web2Private: encodeWeb2Private(user),
	}
	return pkg.Web2Encode(vWorker.private, vWorker.web2NetPublic, wd2)
}

func Web2DecryptWebAuthnKey(user *User) (string, error) {
	if len(user.Web3Key) == 0 || len(user.WebAuthnKey) == 0 {
		return "", errors.New("WebAuthnKey failed with invalid web3Key or webAuthnKey")
	}
	webAuthnBuf, err := hexutil.Decode(string(user.WebAuthnKey))
	if err != nil {
		return "", err
	}
	wd2 := &pkg.Web2Data{
		WebAuthnKey: string(rscrypto.AesDecryptECB(webAuthnBuf, user.Web3Key)),
	}
	LogDebugf("Web2DecryptWebAuthnKey: %s", wd2.WebAuthnKey)
	return pkg.Web2Encode(vWorker.private, vWorker.web2NetPublic, wd2)
}

func encodeWeb2Private(user *User) string {
	web2Private := append(crypto.FromECDSA(user.Web2Private), user.TOTPKey...)
	return hexutil.Encode(rscrypto.AesEncryptECB(web2Private, user.Web2Key))
}
