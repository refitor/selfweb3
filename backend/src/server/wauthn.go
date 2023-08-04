package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"selfweb3/pkg/rsweb"
	"sync"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	c_name_webauthn = "webauthn"
)

var vStore sync.Map
var wauthn = initWebAuthn()

func initWebAuthn() *webauthn.WebAuthn {
	wconfig := &webauthn.Config{
		RPDisplayName: "Go Webauthn",                               // Display Name for your site
		RPID:          "go-webauthn.local",                         // Generally the FQDN for your site
		RPOrigins:     []string{"https://login.go-webauthn.local"}, // The origin URLs allowed for WebAuthn requests
	}
	w, err := webauthn.New(wconfig)
	FatalCheck(err)
	return w
}

func WebAuthnBeginRegister(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	name := rsweb.WebParams(r).Get("name")
	displayName := rsweb.WebParams(r).Get("displayName")
	if authID == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// user
	// user, err := GetWebAuthnUser(authID)
	// if user != nil && err == nil {
	// 	rsweb.ResponseError(w, r, rsweb.WebError(err, "user has registered"))
	// 	return
	// }
	wuser, err := AddWebAuthnUser(authID, name, displayName)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, rsweb.C_Error_InvalidParams))
		return
	}

	// registration
	options, session, err := wauthn.BeginRegistration(wuser)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, rsweb.C_Error_InvalidParams))
		return
	}
	wuser.sess = session

	// response
	rsweb.ResponseOk(w, r, options)
}

func WebAuthnFinishRegister(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	registerParams := rsweb.WebParams(r).Get("registerParams")
	if authID == "" || registerParams == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// user
	wuser, err := GetWebAuthnUser(authID)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// parse registerParams
	parsedResponse := &protocol.ParsedCredentialCreationData{}
	if err := json.Unmarshal([]byte(registerParams), &parsedResponse); err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
		return
	}

	// finish registration
	cred, err := wauthn.CreateCredential(wuser, *wuser.sess, parsedResponse)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
		return
	}
	wuser.credential = cred

	// save user
	// SaveToStoreage(wuser, true)

	// response
	rsweb.ResponseOk(w, r, "successed")
}

func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	name := rsweb.WebParams(r).Get("name")
	if authID == "" || name == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// user
	wuser, err := GetWebAuthnUser(authID)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// login
	credAssertion, session, err := wauthn.BeginLogin(wuser)
	wuser.sess = session

	// response
	rsweb.ResponseOk(w, r, credAssertion)
}

func WebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	authID := rsweb.WebParams(r).Get("authID")
	loginParams := rsweb.WebParams(r).Get("loginParams")
	if authID == "" || loginParams == "" {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// user
	wuser, err := GetWebAuthnUser(authID)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
		return
	}

	// parse loginParams
	credAssertionData := &protocol.ParsedCredentialAssertionData{}
	if err := json.Unmarshal([]byte(loginParams), &credAssertionData); err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
		return
	}

	// finish login
	cred, err := wauthn.ValidateLogin(wuser, *wuser.sess, credAssertionData)
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.C_Error_SystenmExeception)
		return
	}
	wuser.credential = cred

	// remove user at session or not

	// response
	rsweb.ResponseOk(w, r, "successed")
}

// webAuthn user
type WebauthnUser struct {
	ID          string
	Name        string
	DisplayName string

	sess                *webauthn.SessionData         `json:"-"`
	credential          *webauthn.Credential          `json:"-"`
	credentialAssertion *protocol.CredentialAssertion `json:"-"`
}

// WebAuthnID provides the user handle of the user account. A user handle is an opaque byte sequence with a maximum
// size of 64 bytes, and is not meant to be displayed to the user.
//
// To ensure secure operation, authentication and authorization decisions MUST be made on the basis of this id
// member, not the displayName nor name members. See Section 6.1 of [RFC8266].
//
// It's recommended this value is completely random and uses the entire 64 bytes.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dom-publickeycredentialuserentity-id)
func (p *WebauthnUser) WebAuthnID() []byte {
	return []byte(p.ID)
}

// WebAuthnName provides the name attribute of the user account during registration and is a human-palatable name for the user
// account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party SHOULD let the user
// choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dictdef-publickeycredentialuserentity)
func (p *WebauthnUser) WebAuthnName() string {
	return p.Name
}

// WebAuthnDisplayName provides the name attribute of the user account during registration and is a human-palatable
// name for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party
// SHOULD let the user choose this, and SHOULD NOT restrict the choice more than necessary.
//
// Specification: §5.4.3. User Account Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialuserentity-displayname)
func (p *WebauthnUser) WebAuthnDisplayName() string {
	return p.Name
}

// WebAuthnCredentials provides the list of Credential objects owned by the user.
func (p *WebauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return []webauthn.Credential{}
}

// WebAuthnIcon is a deprecated option.
// Deprecated: this has been removed from the specification recommendation. Suggest a blank string.
func (p *WebauthnUser) WebAuthnIcon() string {
	return "https://pics.com/avatar.png"
}

// webAuthn.Register走storage
func GetWebAuthnUser(id string) (*WebauthnUser, error) {
	if user, ok := vStore.Load(id); ok {
		return user.(*WebauthnUser), nil
	} else {
		return nil, errors.New("user has not registered")
	}
}

func AddWebAuthnUser(id, name, displayName string) (*WebauthnUser, error) {
	// if u, err := GetWebAuthnUser(id); u != nil && err == nil {
	// 	return nil, errors.New("user has registered")
	// }
	vStore.Delete(id)

	wuser := &WebauthnUser{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
	}
	vStore.Store(id, wuser)

	return wuser, nil
}
