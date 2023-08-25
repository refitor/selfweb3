package wasm

import (
	"encoding/json"
	"errors"
	"fmt"

	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

const (
	c_Error_Denied        = "permission denied"
	c_Error_InvalidParams = "invalid request params"
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

// @response publicKey
func WasmPublic(datas ...string) *Response {
	return wasmResponse(hexutil.Encode(crypto.CompressPubkey(vWorker.public)), "")
}

// @request userID unique user ID
// @request web2Key web2Key input by user
// @request web2NetPublic we2 network publicKey
// @request web2Datas web2 response params
// @response successed or error
func WasmInit(datas ...string) *Response {
	LogDebugf("WasmInit request: %v", datas)
	if len(datas) < 4 {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	userID, inputWeb2Key, web2NetPublic, web2Data := datas[0], datas[1], datas[2], datas[3]

	wd2 := &pkg.Web2Data{}
	if err := pkg.Web2DecodeEx(vWorker.private, web2NetPublic, web2Data, wd2); err != nil {
		return wasmResponse(nil, WebError(err, "invalid web2Params"))
	}
	web2Key, webAuthnKey, web2Private := wd2.Web2Key, wd2.WebAuthnKey, wd2.Web2Private
	if inputWeb2Key != "" {
		web2Key = inputWeb2Key
	}
	LogDebugf("-==============%s, %s, %+v", web2Key, web2Private, wd2)

	// parse web2NetPublic
	if public, err := rscrypto.GetPublicKey(web2NetPublic); err != nil {
		return wasmResponse(nil, WebError(err, "invalid web2NetPublic"))
	} else {
		vWorker.web2NetPublic = public
	}

	if u := GetUser(userID); u == nil {
		if _, err := NewUser(userID, web2Key, webAuthnKey, web2Private); err != nil {
			return wasmResponse(nil, WebError(err, "invalid web2Key or web2Private"))
		}
	}
	return wasmResponse("successed", "")
}

// @request userID unique user ID
// @request recoverID: Unique ID for social recovery, such as email address, mobile phone number, etc
// @response web3User or error
func WasmRegister(datas ...string) *Response {
	LogDebugf("WasmRegister request: %v", datas)
	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	userID, recoverID := datas[0], datas[1]

	// register user
	user := GetUser(userID)
	if user == nil {
		return wasmResponse(nil, "user register without init")
	}

	// init backendKey and recoverID
	web3User, err := user.Register(recoverID)
	if err != nil {
		return wasmResponse(nil, WebError(err, "encrypt web3 key or recoverID failed"))
	}
	return wasmResponse(web3User, "")
}

// @request userID: unique user ID
// @request authorizedID: ID used for dynamic authorization, such as email address, mobile phone number, etc
// @response successed or error
func WasmAuthorizeCode(datas ...string) *Response {
	LogDebugf("WasmAuthorizeCode request: %v", datas)
	if len(datas) < 1 || datas[0] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	userID, authorizedID := datas[0], datas[1]

	if u := GetUser(userID); u == nil {
		wasmResponse(nil, c_Error_Denied)
	}

	// encode authorized ID and code, provided by web2 service and connected with other third-party platforms
	web2Map := make(map[string]any, 0)
	web2Map[pkg.C_AuthorizeID] = authorizedID
	web2Map[pkg.C_AuthorizeCode] = rscrypto.GetRandom(6, true)
	web2Params, err := pkg.Web2Encode(vWorker.private, vWorker.web2NetPublic, web2Map)
	if err != nil {
		return wasmResponse(nil, WebError(err, "encode web2 params failed"))
	}

	// cache web2Params
	if err := SetCacheByTime("Authorize-"+userID, web2Map, true, 300, nil); err != nil {
		return wasmResponse(nil, WebError(err, ""))
	}
	return wasmResponse(web2Params, "")
}

// @request userID: wallet address
// @request params: Optional, specific business parameters for the next step of the process after dynamic authorization is successful
// @response business data or error
func WasmHandle(datas ...string) *Response {
	LogDebugf("WasmHandle request: %v", datas)
	if len(datas) < 1 || datas[0] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	userID := datas[0]
	LogDebugln(userID, datas[len(datas)-1])

	user := GetUser(userID)
	if user == nil {
		wasmResponse(nil, c_Error_Denied)
	}

	responseData, err := wasmHandle("", user, datas[len(datas)-1])
	if err != nil {
		return wasmResponse(nil, err.Error())
	}
	return wasmResponse(responseData, "")
}

// @request userID: wallet address
// @request kind: kind need verify for dynamic authorization， such as TOTP, email, etc.
// @request code: Verification code for dynamic authorization, such as TOTP, email, mobile phone verification codes, etc
// @request params: Optional, specific business parameters for the next step of the process after dynamic authorization is successful
// @response business data or error
func WasmVerify(datas ...string) *Response {
	LogDebugf("WasmVerify request: %v", datas)
	if len(datas) < 3 || datas[0] == "" || datas[1] == "" || datas[2] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	userID, code, kind := datas[0], datas[1], datas[2]
	LogDebugln(userID, code, kind, datas[len(datas)-1])

	user := GetUser(userID)
	if user == nil {
		wasmResponse(nil, c_Error_Denied)
	}

	// verify: TOTP, email
	var verifyErr error
	var responseData any
	switch kind {
	case "TOTP":
		if responseData, verifyErr = wasmHandle(kind, user, datas[len(datas)-1]); verifyErr != nil {
			break
		}
		if verifyErr = user.VerifyTOTP(code); verifyErr != nil {
			break
		}
	case "email":
		authorizeCache := GetCache("Authorize-"+userID, false, nil)
		if authorizeCache != nil {
			// verify code
			web2Map := authorizeCache.(map[string]any)
			recoverID := fmt.Sprintf("%v", web2Map[pkg.C_AuthorizeID])
			if code != fmt.Sprintf("%v", web2Map[pkg.C_AuthorizeCode]) {
				verifyErr = errors.New("dynamic authorization verify failed: email")
				break
			} else {
				GetCache("Authorize-"+userID, true, nil)
			}
			if responseData, verifyErr = wasmHandle(kind, user, datas[len(datas)-1], recoverID); verifyErr != nil {
				break
			}
		} else {
			verifyErr = errors.New("invalid authorized cache")
			break
		}
	default:
		return wasmResponse(nil, "unsuport authorize kind: "+kind)
	}
	if verifyErr != nil {
		return wasmResponse(nil, WebError(verifyErr, "dynamic authorization verify failed"))
	}
	return wasmResponse(responseData, "")
}

// {"method": "demo", "random": "", "params1": "", "param2": "", ...}
const (
	c_method_Web2Data       = "Web2Data"
	c_method_WebAuthnKey    = "WebAuthnKey"
	c_method_ResetTOTPKey   = "ResetTOTPKey"
	c_method_ResetWeb2Key   = "ResetWeb2Key"
	c_method_RelationVerify = "RelationVerify"
)

func wasmHandle(kind string, user *User, params ...string) (handleResult any, handleErr error) {
	if len(params) == 0 {
		return "successed", nil
	} else if len(params) > 0 && params[0] == "" {
		return "successed", nil
	}
	paramMap := make(map[string]string, 0)
	if err := json.Unmarshal([]byte(params[0]), &paramMap); err != nil {
		return nil, fmt.Errorf("wasmHandle failed with invalid params: %s", params[0])
	}
	LogDebugf("before Handle: %v, %v, %v", params, paramMap[c_param_web3Key], paramMap[c_param_web3Public])

	handleResult = "successed"
	switch paramMap["method"] {
	case c_method_RelationVerify:
		handleErr = user.Load(paramMap[c_param_web3Key], paramMap[c_param_web3Public])
	case c_method_ResetTOTPKey:
		if err := user.Load(paramMap[c_param_web3Key], paramMap[c_param_web3Public]); err != nil {
			return nil, err
		}
		return user.ResetTOTPKey(params[len(params)-1], paramMap[c_param_recoverID])
	case c_method_ResetWeb2Key:
		return user.ResetWeb2Key(params[len(params)-1], paramMap[c_param_random], paramMap[c_param_web2Key])
	case c_method_Web2Data:
		return Web2EncryptWeb2Data(user)
	case c_method_WebAuthnKey:
		if err := user.Load(paramMap[c_param_web3Key], paramMap[c_param_web3Public]); err != nil {
			return nil, err
		}
		LogDebugf("webAuthn: %+s", string(user.WebAuthnKey))
		return Web2DecryptWebAuthnKey(user)
	}
	return
}
