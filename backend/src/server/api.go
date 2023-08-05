package server

import (
	"net/http"

	"selfweb3/pkg/rsweb"

	"github.com/julienschmidt/httprouter"

	// _ "github.com/koesie10/webauthn/attestation"
	"github.com/koesie10/webauthn/webauthn"
)

var wauthn = initWebAuthn()

func initWebAuthn() *webauthn.WebAuthn {
	wconfig := &webauthn.Config{
		RelyingPartyName:   "webauthn-demo",
		Debug:              true,
		AuthenticatorStore: storage,
	}
	w, err := webauthn.New(wconfig)
	FatalCheck(err)
	return w
}

func RouterInit(router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, "/api/datas/load", webDatasLoad)
	router.HandlerFunc(http.MethodPost, "/api/datas/store", webDatasStore)
	router.HandlerFunc(http.MethodPost, "/api/user/recover", webUserRecover)

	router.HandlerFunc(http.MethodPost, "/api/user/begin/login", WebAuthnBeginLogin)
	router.HandlerFunc(http.MethodPost, "/api/user/finish/login", WebAuthnFinishLogin)
	router.HandlerFunc(http.MethodPost, "/api/user/begin/register", WebAuthnBeginRegister)
	router.HandlerFunc(http.MethodPost, "/api/user/finish/register", WebAuthnFinishRegister)
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

// Post: /api/datas/store
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnBeginRegister(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	name := rsweb.WebParams(r).Get("name")
	// displayName := rsweb.WebParams(r).Get("displayName")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[authID]
	if !ok {
		u = &User{
			Name:           name,
			Authenticators: make(map[string]*Authenticator),
		}
		storage.users[name] = u
	}
	sess, _ := vWorker.session.Get(r, c_Session_ID)

	wauthn.StartRegistration(r, w, u, webauthn.WrapMap(sess.Values))
}

// Post: /api/datas/store
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnFinishRegister(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[authID]
	if !ok {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	sess, _ := vWorker.session.Get(r, c_Session_ID)

	wauthn.FinishRegistration(r, w, u, webauthn.WrapMap(sess.Values))
}

// Get: /api/datas/load
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[authID]
	sess, _ := vWorker.session.Get(r, c_Session_ID)
	if ok {
		wauthn.StartLogin(r, w, u, webauthn.WrapMap(sess.Values))
	} else {
		wauthn.StartLogin(r, w, nil, webauthn.WrapMap(sess.Values))
	}
}

// Get: /api/user/recover
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[authID]
	var authenticator webauthn.Authenticator
	sess, _ := vWorker.session.Get(r, c_Session_ID)
	if ok {
		authenticator = wauthn.FinishLogin(r, w, u, webauthn.WrapMap(sess.Values))
	} else {
		authenticator = wauthn.FinishLogin(r, w, nil, webauthn.WrapMap(sess.Values))
	}

	if authenticator == nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
		return
	} else if _, ok := authenticator.(*Authenticator); ok {
		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
	} else {
		rsweb.ResponseOk(w, r, "successed")
	}
}
