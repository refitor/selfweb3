package wasm

import (
	"errors"
	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rscrypto"
)

type SelfData struct {
	pkg.Web2Data
}

func Web2DataInit(wd2 *pkg.Web2Data, user *User, selfKey []byte) *pkg.Web2Data {
	wd2.WebAuthnKey = rscrypto.AesEncryptECB(user.SelfData.WebAuthnKey, selfKey)
	wd2.Web2Public = user.SelfData.Web2Public
	return wd2
}

func Web2EncryptWeb2Data(wd2 *pkg.Web2Data) (string, error) {
	return pkg.Web2Encode(vWorker.private, vWorker.web2NetPublic, wd2)
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
