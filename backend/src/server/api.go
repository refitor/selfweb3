package server

import (
	"fmt"
	"net/http"

	"selfweb3/pkg/rsweb"

	"github.com/julienschmidt/httprouter"
)

func RouterInit(router *httprouter.Router) {
	RouterHook(router, http.MethodGet, "/api/datas/load", webDatasLoad, nil)
	RouterHook(router, http.MethodPost, "/api/datas/store", webDatasStore, nil)
	RouterHook(router, http.MethodPost, "/api/user/recover", webUserRecover, nil)

	RouterHook(router, http.MethodPost, "/api/user/logout", WebAuthnLogout, UserSessionCheck)
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

// Get: /api/user/recover
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webUserRecover(w http.ResponseWriter, r *http.Request) {

}

func WebAuthnBeginRegister(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("id")
	username := rsweb.WebParams(r).Get("username")
	// displayName := rsweb.WebParams(r).Get("displayName")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	if username == "" {
		username = authID
	}

	response, err, webErr := WauthnBeginRegister(authID, func(key any) (any, bool) {
		return GetCache(C_Store_WebauthnUser, key)
	}, func(k, u any) {
		SetCache(k, u)
	})
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
	authID := rsweb.WebParams(r).Get("id")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	response, err, webErr := WauthnFinishRegister(authID, r.Body, func(key any) (any, bool) {
		return GetCache(C_Store_WebauthnUser, key)
	})
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

func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("id")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	response, err, webErr := WauthnBeginLogin(authID, func(key any) (any, bool) {
		return GetCache(C_Store_WebauthnUser, key)
	})
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
	authID := rsweb.WebParams(r).Get("id")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	response, err, webErr := WauthnFinishLogin(authID, r.Body, func(key any) (any, bool) {
		return GetCache(C_Store_WebauthnUser, key)
	})
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}

	// push user to session
	PushToSession(w, r, C_Session_User, authID)

	rsweb.ResponseOk(w, r, response)
}

func WebAuthnLogout(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("id")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	if authID != fmt.Sprintf("%v", PopFromSession(r, C_Session_User)) {
		rsweb.ResponseError(w, r, rsweb.C_Error_Denied)
		return
	}

	// remove from session
	RemoveSession(w, r, C_Session_User)

	// remove from cache
	vWorker.cache.Delete(authID)

	// store
	if err := SaveToDB(C_Store_WebauthnUser, authID); err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}

	rsweb.ResponseOk(w, r, "successed")
}
