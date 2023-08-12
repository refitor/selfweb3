package server

import (
	"net/http"

	"selfweb3/pkg/rsweb"

	"github.com/julienschmidt/httprouter"
)

func RouterInit(router *httprouter.Router) {
	RouterHook(router, http.MethodGet, "/api/datas/load", webDatasLoad, nil)
	RouterHook(router, http.MethodPost, "/api/datas/store", webDatasStore, nil)

	// RouterHook(router, http.MethodPost, "/api/user/status", UserStatus, UserSessionCheck)
	// RouterHook(router, http.MethodPost, "/api/user/logout", UserLogout, UserSessionCheck)
	// // RouterHook(router, http.MethodPost, "/api/user/recovery", UserRecovery, UserSessionCheck)
	// // RouterHook(router, http.MethodPost, "/api/user/auth", UserAuth, UserSessionCheck) // totp, email......

	RouterHook(router, http.MethodPost, "/api/user/begin/login", WebAuthnBeginLogin, nil)
	RouterHook(router, http.MethodPost, "/api/user/finish/login", WebAuthnFinishLogin, nil)
	RouterHook(router, http.MethodPost, "/api/user/begin/register", WebAuthnBeginRegister, nil)
	RouterHook(router, http.MethodPost, "/api/user/finish/register", WebAuthnFinishRegister, nil)
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

// ====== datas ======
// Post: /api/datas/store
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webDatasStore(w http.ResponseWriter, r *http.Request) {

}

// Get: /api/datas/load
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webDatasLoad(w http.ResponseWriter, r *http.Request) {

}

// ====== user ======
func UserStatus(w http.ResponseWriter, r *http.Request) {
	rsweb.ResponseOk(w, r, PopFromSession(r, C_Session_User).(*User).WebUser)
}

func UserLogout(w http.ResponseWriter, r *http.Request) {
	username := rsweb.WebParams(r).Get("username")
	if username == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	if user := PopFromSession(r, C_Session_User); user != nil && user.(*WebUser).Name != username {
		rsweb.ResponseError(w, r, rsweb.C_Error_Denied)
		return
	}

	// remove from session
	RemoveSession(w, r, C_Session_User)

	// remove from cache
	WauthnClean(username)

	rsweb.ResponseOk(w, r, "successed")
}

// ====== webauthn ======
func WebAuthnBeginRegister(w http.ResponseWriter, r *http.Request) {
	username := rsweb.WebParams(r).Get("username")
	// displayName := rsweb.WebParams(r).Get("displayName")
	if username == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	response, err, webErr := WauthnBeginRegister(username)
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
	username := rsweb.WebParams(r).Get("username")
	if username == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	wuser, err, webErr := WauthnFinishRegister(username, r.Body, false)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}

	// create user
	if _, err := CreateUser(username, wuser); err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}
	rsweb.ResponseOk(w, r, "successed")
}

func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	username := rsweb.WebParams(r).Get("username")
	if username == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	response, err, webErr := WauthnBeginLogin(username)
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
	username := rsweb.WebParams(r).Get("username")
	if username == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	response, err, webErr := WauthnFinishLogin(username, r.Body)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}

	// push user to session
	PushToSession(w, r, C_Session_User, GetUser(username))

	rsweb.ResponseOk(w, r, response)
}
