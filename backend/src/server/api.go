package server

import (
	"fmt"
	"net/http"

	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rscrypto"
	"selfweb3/backend/pkg/rsweb"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/refitor/rslog"
)

func RouterInit(router *httprouter.Router) {
	RouterHook(router, http.MethodGet, "/api/datas/load", webDatasLoad, UserVerify)
	RouterHook(router, http.MethodPost, "/api/datas/store", webDatasStore, UserVerify)
	RouterHook(router, http.MethodPost, "/api/datas/forward", webDatasForward, UserVerify)

	RouterHook(router, http.MethodPost, "/api/user/begin/login", WebAuthnBeginLogin, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/finish/login", WebAuthnFinishLogin, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/begin/register", WebAuthnBeginRegister, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/finish/register", WebAuthnFinishRegister, UserVerify)
}

func RouterHook(router *httprouter.Router, method string, path string, handleFunc http.HandlerFunc, hookFuncs ...func(w http.ResponseWriter, r *http.Request) bool) {
	router.HandlerFunc(method, path, func(w http.ResponseWriter, r *http.Request) {
		for _, hookFunc := range hookFuncs {
			if hookFunc != nil && !hookFunc(w, r) {
				rsweb.ResponseError(w, r, rsweb.C_Error_Denied)
				return
			}
		}
		handleFunc(w, r)
	})
}

func UserVerify(w http.ResponseWriter, r *http.Request) bool {
	userID := rsweb.WebParams(r).Get("userID")
	return userID != ""
}

// ====== datas ======
// POST: /api/datas/store
// params: userID, kind, recoverID, params
func webDatasStore(w http.ResponseWriter, r *http.Request) {
	user := &User{}
	userID := rsweb.WebParams(r).Get("userID")
	if err := LoadFromDB(C_Store_User, userID, user); err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}

	kind := rsweb.WebParams(r).Get("kind")
	switch kind {
	case "selfweb3.web2Data":
		recoverID := rsweb.WebParams(r).Get("recoverID")
		if recoverID != "" {
			user.RecoverID = rscrypto.AesEncryptECB([]byte(recoverID), []byte(user.Web2Key))
		} else if len(user.RecoverID) > 0 {
			recoverID = string(rscrypto.AesDecryptECB(user.RecoverID, []byte(user.Web2Key)))
		}
		user.EncryptWeb2Private = rsweb.WebParams(r).Get("params")
		if err := UserSaveToStore(userID, user); err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
		if err := SendEmailToUser(recoverID, fmt.Sprintf("[SelfWeb3] Hi, your selfweb3 account has been updated, please keep the web2 private key ciphertext safe: %s", user.EncryptWeb2Private)); err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
	default:
		rsweb.ResponseError(w, r, "unsupport store kind: "+kind)
	}
	rsweb.ResponseOk(w, r, "successed")
}

// GET: /api/datas/load
// params: userID, kind, public
func webDatasLoad(w http.ResponseWriter, r *http.Request) {
	user := &User{}
	userID := rsweb.WebParams(r).Get("userID")
	if err := LoadFromDB(C_Store_User, userID, user); err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}

	kind := rsweb.WebParams(r).Get("kind")
	webPublic := rsweb.WebParams(r).Get("public")
	switch kind {
	case "selfweb3.web2Data":
		web2Datas, err := pkg.Web2EncodeEx(vWorker.private, webPublic, user.WebUser)
		if err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
		rsweb.ResponseOk(w, r, map[string]any{
			pkg.C_Web2Datas:     web2Datas,
			pkg.C_Web2NetPublic: hexutil.Encode(crypto.CompressPubkey(vWorker.public)),
		})
	default:
		rsweb.ResponseError(w, r, "unsupport load kind: "+kind)
	}
}

// POST: /api/datas/load
// params: userID, kind, params
func webDatasForward(w http.ResponseWriter, r *http.Request) {
	webPublic := rsweb.WebParams(r).Get("public")
	kind := rsweb.WebParams(r).Get("kind")
	switch kind {
	case "email":
		web2Map, err := pkg.Web2DecodeEx(vWorker.private, webPublic, rsweb.WebParams(r).Get("params"))
		if err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
		if err := SendEmailToUser(Str(web2Map[pkg.C_AuthorizeID]), fmt.Sprintf("[SelfWeb3] code for dynamic authorization: %s", Str(web2Map[pkg.C_AuthorizeCode]))); err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
	default:
		rsweb.ResponseError(w, r, "unsupport forward kind: "+kind)
	}
	rsweb.ResponseOk(w, r, "successed")
}

// disable user session
// // ====== user ======
// func UserStatus(w http.ResponseWriter, r *http.Request) {
// 	rsweb.ResponseOk(w, r, PopFromSession(r, C_Session_User).(*User).WebUser)
// }

// func UserLogout(w http.ResponseWriter, r *http.Request) {
// 	userID := rsweb.WebParams(r).Get("userID")
// 	if userID == "" {
// 		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
// 		return
// 	}
// 	if user := PopFromSession(r, C_Session_User); user != nil && user.(*WebUser).Name != userID {
// 		rsweb.ResponseError(w, r, rsweb.C_Error_Denied)
// 		return
// 	}

// 	// remove from session
// 	RemoveSession(w, r, C_Session_User)

// 	// remove from cache
// 	WauthnClean(userID)

// 	rsweb.ResponseOk(w, r, "successed")
// }

// ====== webauthn ======
func WebAuthnBeginRegister(w http.ResponseWriter, r *http.Request) {
	userID := rsweb.WebParams(r).Get("userID")
	// displayName := rsweb.WebParams(r).Get("displayName")

	response, err, webErr := WauthnBeginRegister(userID)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}
	rsweb.ResponseOk(w, r, response)
}

func WebAuthnFinishRegister(w http.ResponseWriter, r *http.Request) {
	userID := rsweb.WebParams(r).Get("userID")

	wuser, err, webErr := WauthnFinishRegister(userID, r.Body, false)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}

	// create user
	if _, err := CreateUser(userID, wuser); err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}
	rsweb.ResponseOk(w, r, "successed")
}

func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	userID := rsweb.WebParams(r).Get("userID")

	response, err, webErr := WauthnBeginLogin(userID)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}
	rsweb.ResponseOk(w, r, response)
}

func WebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	userID := rsweb.WebParams(r).Get("userID")

	response, err, webErr := WauthnFinishLogin(userID, r.Body)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}
	rslog.Infof("WebAuthnFinishLogin response: %v", response)

	// // push user to session
	// PushToSession(w, r, C_Session_User, GetUser(userID))

	rsweb.ResponseOk(w, r, response)
}
