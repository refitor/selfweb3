package wasm

import (
	"errors"
	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/crypto"
)

type SelfData struct {
	pkg.Web2Data
}

func Web2DataInit(user *User, selfKey []byte) *pkg.Web2Data {
	wd := &pkg.Web2Data{
		Web2DHKey: user.SelfData.Web2DHKey,
		// VerifyNonce = webNonce + selfNonce
		VerifyNonce: user.SelfData.VerifyNonce,
		SelfAddress: user.SelfData.SelfAddress,
		Web3Public:  crypto.CompressPubkey(user.Web3Public),
		WebAuthnKey: rscrypto.AesEncryptECB(user.SelfData.WebAuthnKey, selfKey),
	}
	return wd
}

func Web2EncryptWeb2Data(user *User) (string, error) {
	return pkg.Web2Encode(vWorker.private, vWorker.web2NetPublic, user.SelfData)
}

func Web2DecryptWebAuthnKey(user *User, selfKey []byte) (string, error) {
	if len(selfKey) == 0 || len(user.SelfData.WebAuthnKey) == 0 {
		return "", errors.New("WebAuthnKey failed with invalid selfKey or webAuthnKey")
	}
	wd2 := &pkg.Web2Data{
		WebAuthnKey: rscrypto.AesDecryptECB(user.SelfData.WebAuthnKey, selfKey),
	}
	LogDebugf("Web2DecryptWebAuthnKey: %s", wd2.WebAuthnKey)
	return pkg.Web2Encode(vWorker.private, vWorker.web2NetPublic, wd2)
}

// func encodeWeb2Private(user *User) string {
// 	web2Private := append(crypto.FromECDSA(user.Web2Private), user.TOTPKey...)
// 	return hexutil.Encode(rscrypto.AesEncryptECB(web2Private, user.Web2Key))
// }
