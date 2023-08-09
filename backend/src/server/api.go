package server

import (
	"net/http"

	"selfweb3/pkg/rsweb"

	"github.com/julienschmidt/httprouter"
	"github.com/refitor/rslog"

	// _ "github.com/koesie10/webauthn/attestation"
	"github.com/koesie10/webauthn/webauthn"
)

var wauthn = initWebAuthn()
var sessValues = make(map[interface{}]interface{}, 0)

func initWebAuthn() *webauthn.WebAuthn {
	wconfig := &webauthn.Config{
		RelyingPartyName: "webauthn-demo",
		Debug:            true,
		// RelyingPartyID:            "localhost",
		AuthenticatorStore: storage,
		// SessionKeyPrefixChallenge: "challenge",
		// RelyingPartyOrigin:        "https://selfweb3.refitor.com",
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
	// authID := rsweb.WebParams(r).Get("id")
	name := rsweb.WebParams(r).Get("name")
	// displayName := rsweb.WebParams(r).Get("displayName")
	if name == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[name]
	if !ok {
		u = &User{
			Name:           name,
			Authenticators: make(map[string]*Authenticator),
		}
		storage.users[name] = u
	}
	// sess, _ := vWorker.session.Get(r, c_Session_ID)
	wrapSession := webauthn.WrapMap(sessValues)
	wauthn.StartRegistration(r, w, u, wrapSession)

	key1 := wauthn.Config.SessionKeyPrefixChallenge + ".register"
	key2 := wauthn.Config.SessionKeyPrefixUserID + ".register"
	sessValues[key1], _ = wrapSession.Get(key1)
	sessValues[key2], _ = wrapSession.Get(key2)
	rslog.Debugf("11111111111: %+v", sessValues)
}

// Post: /api/datas/store
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnFinishRegister(w http.ResponseWriter, r *http.Request) {
	// authID := rsweb.WebParams(r).Get("id")
	name := rsweb.WebParams(r).Get("name")
	// displayName := rsweb.WebParams(r).Get("displayName")
	if name == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[name]
	if !ok {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	// sess, _ := vWorker.session.Get(r, c_Session_ID)

	rslog.Debugf("11111111111++++++++++++: %+v", sessValues)
	wauthn.FinishRegistration(r, w, u, webauthn.WrapMap(sessValues))
}

// Get: /api/datas/load
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	name := rsweb.WebParams(r).Get("name")
	if name == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	u, ok := storage.users[name]
	// sess, err := vWorker.session.Get(r, c_Session_ID)
	if ok {
		wrapSession := webauthn.WrapMap(sessValues)
		wauthn.StartLogin(r, w, u, wrapSession)

		key1 := wauthn.Config.SessionKeyPrefixChallenge + ".login"
		sessValues[key1], _ = wrapSession.Get(key1)
		rslog.Debugf("2222222222222: %+v", sessValues)
	} else {
		wauthn.StartLogin(r, w, nil, webauthn.WrapMap(sessValues))
	}
}

// Get: /api/user/recover
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func WebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	// authID := rsweb.WebParams(r).Get("id")
	name := rsweb.WebParams(r).Get("name")
	if name == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}
	rslog.Debugf("2222222222222++++++++++++: %+v", sessValues)

	u, ok := storage.users[name]
	var authenticator webauthn.Authenticator
	// sess, _ := vWorker.session.Get(r, c_Session_ID)
	if ok {
		authenticator = wauthn.FinishLogin(r, w, u, webauthn.WrapMap(sessValues))
	} else {
		authenticator = wauthn.FinishLogin(r, w, nil, webauthn.WrapMap(sessValues))
	}
	rslog.Debug(authenticator)
	rsweb.ResponseOk(w, r, "successed")

	// if authenticator != nil {
	// 	if authr, ok := authenticator.(*Authenticator); ok {
	// 		rsweb.ResponseOk(w, r, authr.User)
	// 	} else {
	// 		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
	// 	}
	// }
}
