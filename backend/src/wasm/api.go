package wasm

import (
	"encoding/hex"
	"errors"
	"fmt"

	"selfweb3/pkg/rsauth"
	"selfweb3/pkg/rscrypto"
	"selfweb3/pkg/rsweb"

	"github.com/ethereum/go-ethereum/crypto"
)

const (
	c_Data_Pass               = "pass"
	c_Data_Reload             = "reload"
	c_Data_Pending            = "pending"
	c_Data_Success            = "successed"
	c_Error_Denied            = "permission denied"
	c_Error_InvalidParams     = "invalid request params"
	c_Error_SystenmExeception = "system processing exception"
)

type Response struct {
	Data  interface{}
	Error string
}

func wasmResponse(data any, err string) *Response {
	wasmResp := new(Response)
	wasmResp.Data = data
	wasmResp.Error = err
	return wasmResp
}

// @request authID wallet address
// @response backendPublic backend public key
// @response web3Public wallet account public key
func Load(datas ...string) *Response {
	if len(datas) < 3 || datas[0] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, web3Public, backendKey := datas[0], datas[1], datas[2]

	// cache backendKey
	_, err := LoadAuthUser(authID, web3Public, backendKey)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, ""))
	}
	return wasmResponse(hex.EncodeToString(crypto.CompressPubkey(vWorker.public)), "")
}

// @request authID: wallet address
// @request recoverID: push channel used to recover the backend (email phone number, etc.)
// @response recoverID: recovery ID encrypted by backend key
// @response qrcode: QR code for Google Authenticator scanning to add account
// @response backendKey: encrypted by backend public key
func Register(datas ...string) *Response {
	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, recoverID := datas[0], datas[1]

	// register user
	auser, err := LoadAuthUser(authID, "", "")
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, ""))
	}

	// init backendKey and recoverID
	encryptedRecoverID, encryptedBackendKey, err := auser.Init(authID, recoverID, nil)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, "encrypt backendKey or recoverID failed"))
	}

	vWorker := make(map[string]string, 0)
	vWorker["recoverID"] = encryptedRecoverID
	vWorker["backendKey"] = encryptedBackendKey
	vWorker["qrcode"], _ = auser.GetQrcode(authID)
	return wasmResponse(vWorker, "")
}

// Post: /api/user/recover
// @request authID: wallet address
// @request pushID: push channel used to recover the backend (email phone number, etc.)
// @response successed or error
func Recover(datas ...string) *Response {
	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, pushID := datas[0], datas[1]

	// send verify code
	code := rscrypto.GetRandom(6, true)
	if err := SetCacheByTime("recoveryCode-"+authID, code, true, 300, nil); err != nil {
		return wasmResponse(nil, rsweb.WebError(err, ""))
	}

	// sendCh := make(chan struct{})
	// if _, err := rsauth.PushByEmail(pushID, "dynamic authorization", "", fmt.Sprintf("[SelfCrypto] code for dynamic authorization: %s", code), func(err error) {
	// 	if err != nil {
	// 		rslog.Errorf("email send failed: %s", err.Error())
	// 	}
	// 	close(sendCh)
	// }); err != nil {
	// 	return wasmResponse(nil, rsweb.WebError(err, ""))
	// }
	// <-sendCh

	// rslog.Debugf("push code to recoverID successed, recoverID: %s, code: %s", pushID, code)
	SetCache("pushID-"+authID, pushID, true)

	return wasmResponse(code, "")
}

// @request authID: wallet address
// @request code: code input for dynamic authorization
// @request kind: kind need verify for dynamic authorization
// @request params: params need verify for dynamic authorization
// @response for email verify qrcode for dynamic authorization verify
// @response for dynamic authorization content encrypted or decrypted by backend key
func Auth(datas ...string) *Response {
	if len(datas) < 5 || datas[0] == "" || datas[1] == "" || datas[2] == "" || datas[3] == "" || datas[4] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, code, kind := datas[0], datas[1], datas[2]
	authParams1, authParams2 := datas[3], datas[4]

	// verify user
	auser, err := LoadAuthUser(authID, "", "")
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, ""))
	}
	if auser.SelfPrivate == nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Data_Reload))
	}

	// verify by google
	var verifyErr error
	var responseData interface{}
	vWorker := make(map[string]interface{}, 0)
	switch kind {
	case "google":
		secret, err := auser.getKey(auser.Web3Public, nil)
		if err != nil {
			verifyErr = err
			break
		}
		if ok, err := rsauth.NewGoogleAuth().VerifyCode(secret, code); err != nil {
			verifyErr = err
			break
		} else if !ok {
			verifyErr = errors.New("selfCrypto google verify failed")
			break
		}
		responseData, err = auser.HandleCrypto(authParams1, authParams2)
		if err != nil {
			verifyErr = err
			break
		}
	case "email":
		memCode := GetCache("recoveryCode-"+authID, false, nil)
		if code != fmt.Sprintf("%v", memCode) {
			verifyErr = errors.New("selfCrypto email verify failed")
			break
		} else {
			GetCache(authID, true, nil)
		}

		pushID := fmt.Sprintf("%v", GetCache("pushID-"+authID, true, nil))
		resetUser, err := auser.Reset(authID, pushID, authParams1, authParams2)
		if err != nil {
			return wasmResponse(nil, rsweb.WebError(err, ""))
		}
		vWorker["qrcode"], _ = resetUser.GetQrcode(authID)
		responseData = vWorker
	default:
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	if verifyErr != nil {
		return wasmResponse(nil, rsweb.WebError(verifyErr, "selfCrypto verify failed"))
	}

	return wasmResponse(responseData, "")
}
