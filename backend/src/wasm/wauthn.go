package wasm

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"selfweb3/pkg/rsweb"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
)

const (
	c_name_webauthn = "webauthn"
)

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

func WebAuthnBeginRegister(datas ...string) *Response {
	if len(datas) < 1 || datas[0] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, name, displayName := datas[0], datas[1], datas[2]

	// user
	user, err := GetWebAuthnUser(authID)
	if user != nil && err == nil {
		return wasmResponse(nil, rsweb.WebError(err, "user has registered"))
	}
	wuser, err := AddWebAuthnUser(authID, name, displayName)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
	}

	// registration
	options, session, err := wauthn.BeginRegistration(wuser)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
	}
	wuser.sess = session

	// response
	return wasmResponse(options, "")
}

func WebAuthnFinishRegister(datas ...string) *Response {
	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, registerParams := datas[0], datas[1]

	// user
	wuser, err := GetWebAuthnUser(authID)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
	}

	// build http.Request
	webauthnReq, err := http.NewRequest(http.MethodPost, "", ioutil.NopCloser(bytes.NewBufferString(registerParams)))
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
	}

	// finish registration
	cred, err := wauthn.FinishRegistration(wuser, *wuser.sess, webauthnReq)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
	}
	wuser.credential = cred

	// save user
	SaveToStoreage(wuser, true)

	// response
	return wasmResponse("successed", "")
}

func WebAuthnBeginLogin(datas ...string) *Response {
	if len(datas) < 1 || datas[0] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID := datas[0]

	// user
	wuser, err := GetWebAuthnUser(authID)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, ""))
	}

	// login
	credAssertion, session, err := wauthn.BeginLogin(wuser)
	wuser.sess = session

	// response
	return wasmResponse(credAssertion, "")
}

func WebAuthnFinishLogin(datas ...string) *Response {
	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
		return wasmResponse(nil, c_Error_InvalidParams)
	}
	authID, loginParams := datas[0], datas[1]

	// user
	wuser, err := GetWebAuthnUser(authID)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
	}

	// parse loginParams
	credAssertionData := &protocol.ParsedCredentialAssertionData{}
	if err := json.Unmarshal([]byte(loginParams), &credAssertionData); err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
	}

	// finish login
	cred, err := wauthn.ValidateLogin(wuser, *wuser.sess, credAssertionData)
	if err != nil {
		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
	}
	wuser.credential = cred

	// remove user at session or not

	// response
	return wasmResponse("successed", "")
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
	if u, err := GetWebAuthnUser(id); u != nil && err == nil {
		return nil, errors.New("user has registered")
	}

	wuser := &WebauthnUser{
		ID:          id,
		Name:        name,
		DisplayName: displayName,
	}
	vStore.Store(id, wuser)

	return wuser, nil
}
