package server

import (
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"path/filepath"
	"runtime"
	"selfweb3/common/rsauth"
	"selfweb3/common/rsweb"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/refitor/rslog"
)

func AuthInitRouter(router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, "/api/user/load", webLoad)
	router.HandlerFunc(http.MethodPost, "/api/user/regist", webRegist)
	router.HandlerFunc(http.MethodPost, "/api/user/verify", webAuth)
	router.HandlerFunc(http.MethodPost, "/api/user/recover", webRecover)
}

// Get: /api/user/load
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webLoad(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	backendKey := rsweb.WebParams(r).Get("backendKey")
	walletPublic := rsweb.WebParams(r).Get("walletPublic")
	if authID == "" || backendKey == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// cache backendKey
	_, err := GetAuthUser(authID, walletPublic, backendKey)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	rsweb.ResponseOk(w, r, hexutil.Encode(crypto.FromECDSAPub(vserver.public))[4:])
}

// Post: /api/user/regist
// @request authID: wallet address
// @request recoverID: push channel used to recover the backend (email phone number, etc.)
// @response recoverID: recovery ID encrypted by backend key
// @response qrcode: QR code for Google Authenticator scanning to add account
// @response backendKey: encrypted by backend public key
func webRegist(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	recoverID := rsweb.WebParams(r).Get("recoverID")
	if authID == "" || recoverID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// register user
	auser, err := GetAuthUser(authID, "", "")
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// init backendKey and recoverID
	encryptedRecoverID, encryptedBackendKey, err := auser.Init(authID, recoverID, nil)
	if err != nil {
		rsweb.ResponseError(w, r, WebError(err, "encrypt backendKey or recoverID failed"))
		return
	}

	retMap := make(map[string]string, 0)
	retMap["recoverID"] = encryptedRecoverID
	retMap["backendKey"] = encryptedBackendKey
	retMap["qrcode"], _ = auser.GetQrcode(authID)
	rsweb.ResponseOk(w, r, retMap)
}

// Post: /api/user/recover
// @request authID: wallet address
// @request pushID: push channel used to recover the backend (email phone number, etc.)
// @response successed or error
func webRecover(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	pushID := rsweb.WebParams(r).Get("pushID")
	if authID == "" || pushID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// send verify code
	code := GetRandom(6, true)
	sendCh := make(chan struct{})
	if err := vserver.SetCacheByTime("recoveryCode-"+authID, code, true, 300, nil); err != nil {
		rsweb.ResponseError(w, r, WebError(PackError(err), ""))
		return
	}
	if _, err := rsauth.PushByEmail(pushID, "dynamic authorization", "", fmt.Sprintf("[selfweb3] code for dynamic authorization: %s", code), func(err error) {
		if err != nil {
			rslog.Errorf("email send failed: %s", err.Error())
		}
		close(sendCh)
	}); err != nil {
		rsweb.ResponseError(w, r, WebError(PackError(err), ""))
		return
	}
	<-sendCh

	// rslog.Debugf("push code to recoverID successed, recoverID: %s, code: %s", pushID, code)
	vserver.SetCache("pushID-"+authID, pushID, true)
	rsweb.ResponseOk(w, r, "successed")
}

// Post: /api/user/verify
// @request authID: wallet address
// @request code: code input for dynamic authorization
// @request kind: kind need verify for dynamic authorization
// @request authParams1: params need verify for dynamic authorization
// @request authParams2: params need verify for dynamic authorization
// @response for email verify qrcode for dynamic authorization verify
// @response for dynamic authorization content encrypted or decrypted by backend key
func webAuth(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	code := rsweb.WebParams(r).Get("code")
	kind := rsweb.WebParams(r).Get("kind")
	authParams1 := rsweb.WebParams(r).Get("authParams1")
	authParams2 := rsweb.WebParams(r).Get("authParams2")
	if authID == "" || code == "" || kind == "" || authParams1 == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// verify user
	auser, err := GetAuthUser(authID, "", "")
	if err != nil {
		rsweb.ResponseError(w, r, WebError(err, ""))
		return
	}
	if auser.SelfPrivate == nil {
		rsweb.ResponseError(w, r, WebError(err, ""))
		return
	}

	// verify by google
	var verifyErr error
	var responseData interface{}
	retMap := make(map[string]interface{}, 0)
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
			verifyErr = errors.New("selfweb3 google verify failed")
			break
		}
		responseData, err = auser.HandleCrypto(authParams1, authParams2)
		if err != nil {
			verifyErr = err
			break
		}
	case "email":
		memCode := vserver.GetCache("recoveryCode-"+authID, false, nil)
		if code != fmt.Sprintf("%v", memCode) {
			verifyErr = errors.New("selfweb3 email verify failed")
			break
		} else {
			vserver.GetCache(authID, true, nil)
		}

		pushID := fmt.Sprintf("%v", vserver.GetCache("pushID-"+authID, true, nil))
		resetUser, err := auser.Reset(authID, pushID, authParams1) //, authParams2)
		if err != nil {
			rsweb.ResponseError(w, r, WebError(err, ""))
			return
		}
		retMap["qrcode"], _ = resetUser.GetQrcode(authID)
		responseData = retMap
	default:
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	if verifyErr != nil {
		rsweb.ResponseError(w, r, WebError(PackError(verifyErr), "selfweb3 verify failed"))
		return
	}

	rsweb.ResponseOk(w, r, responseData)
}

// ===================================help function=====================================
func GetPublic(privateKey []byte) ([]byte, error) {
	privBlock, _ := pem.Decode(privateKey)
	if privBlock == nil {
		return nil, errors.New("private key error")
	}
	private, err := x509.ParseECPrivateKey(privBlock.Bytes)
	if err != nil {
		return nil, err
	}

	pubBuf, err := x509.MarshalPKIXPublicKey(&private.PublicKey)
	if err != nil {
		return nil, err
	} else {
		pubBuf = pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubBuf})
	}
	return pubBuf, nil
}

func GetRandomInt(max *big.Int) (int, error) {
	if max == nil {
		seed := "0123456789"
		alphanum := seed + fmt.Sprintf("%v", time.Now().UnixNano())
		max = big.NewInt(int64(len(alphanum)))
	}
	vrand, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return int(vrand.Int64()), nil
}

func GetRandom(n int, isNO bool) string {
	seed := "0123456789"
	if !isNO {
		seed = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}
	alphanum := seed + fmt.Sprintf("%v", time.Now().UnixNano())
	buffer := make([]byte, n)
	max := big.NewInt(int64(len(alphanum)))

	for i := 0; i < n; i++ {
		index, err := GetRandomInt(max)
		if err != nil {
			return ""
		}

		buffer[i] = alphanum[index]
	}
	return string(buffer)
}

func PackError(err error) error {
	if err != nil {
		_, file, line, _ := runtime.Caller(2)
		return fmt.Errorf("%s.%v===>%s", filepath.Base(file), line, err.Error())
	}
	return nil
}

func WebError(err error, webErr string) string {
	logid := time.Now().UnixNano()
	if webErr == "" {
		webErr = "system processing exception"
	}
	if err != nil {
		rslog.Errorf("%v-%s", logid, err.Error())
	}
	return fmt.Sprintf("%v-%s", logid, webErr)
}
