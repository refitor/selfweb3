package wasm

// import (
// 	"encoding/json"
// 	"errors"
// 	"selfweb3/pkg/rsweb"

// 	"github.com/go-webauthn/webauthn/protocol"
// 	"github.com/go-webauthn/webauthn/webauthn"
// 	"github.com/refitor/rslog"
// )

// const (
// 	c_name_webauthn = "webauthn"
// )

// var wauthn = initWebAuthn()

// func initWebAuthn() *webauthn.WebAuthn {
// 	wconfig := &webauthn.Config{
// 		RPDisplayName: "selfweb3.RP",                                                     // Display Name for your site
// 		RPID:          "localhost",                                                       // Generally the FQDN for your site
// 		RPOrigins:     []string{"https://selfweb3.refitor.com", "http://localhost:5173"}, // The origin URLs allowed for WebAuthn requests
// 	}
// 	w, err := webauthn.New(wconfig)
// 	FatalCheck(err)
// 	return w
// }

// func WebAuthnBeginRegister(datas ...string) *Response {
// 	if len(datas) < 1 || datas[0] == "" {
// 		return wasmResponse(nil, c_Error_InvalidParams)
// 	}
// 	authID, name, displayName := datas[0], datas[1], datas[2]

// 	// user
// 	user, err := GetWebAuthnUser(authID)
// 	if user != nil && err == nil {
// 		return wasmResponse(nil, rsweb.WebError(err, "user has registered"))
// 	}
// 	wuser, err := AddWebAuthnUser(authID, name, displayName)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
// 	}

// 	// registration
// 	opts := make([]webauthn.RegistrationOption, 0)
// 	opts = append(opts, webauthn.WithConveyancePreference(protocol.PreferDirectAttestation))
// 	// opts = append(opts, webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{AuthenticatorAttachment: protocol.Platform}))
// 	opts = append(opts, webauthn.WithAuthenticatorSelection(protocol.AuthenticatorSelection{AuthenticatorAttachment: protocol.CrossPlatform}))
// 	options, session, err := wauthn.BeginRegistration(wuser, opts...)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
// 	}
// 	wuser.sess = session
// 	rslog.Debugf("WebAuthnBeginRegister: %+v", options)

// 	// response
// 	return wasmResponse(options, "")
// }

// func WebAuthnFinishRegister(datas ...string) *Response {
// 	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
// 		return wasmResponse(nil, c_Error_InvalidParams)
// 	}
// 	authID, registerParams := datas[0], datas[1]

// 	// user
// 	wuser, err := GetWebAuthnUser(authID)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
// 	}

// 	// parse registerParams
// 	collectedClientData := &protocol.CollectedClientData{}
// 	if err = json.Unmarshal([]byte(registerParams), &collectedClientData); err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
// 	}

// 	// // debug
// 	// collectedClientData.Challenge = wuser.sess.Challenge

// 	pcaData := new(protocol.ParsedCredentialCreationData)
// 	pcaData.Response.CollectedClientData = *collectedClientData
// 	// rslog.Debugf("WebAuthnFinishRegister: %+v, 222: %s, 333: %v", pcaData, registerParams, wuser.sess.Challenge)

// 	// // finish registration
// 	// cred, err := wauthn.CreateCredential(wuser, *wuser.sess, pcaData)
// 	// if err != nil {
// 	// 	return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
// 	// }

// 	cred, err := webauthn.MakeNewCredential(pcaData)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
// 	}
// 	rslog.Debugf("wuser.credential: %+v", cred)

// 	// save user
// 	SaveToStoreage(wuser, true)

// 	// response
// 	return wasmResponse("successed", "")
// }

// func WebAuthnBeginLogin(datas ...string) *Response {
// 	if len(datas) < 1 || datas[0] == "" {
// 		return wasmResponse(nil, c_Error_InvalidParams)
// 	}
// 	authID := datas[0]

// 	// user
// 	wuser, err := GetWebAuthnUser(authID)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
// 	}
// 	rslog.Debugf("WebAuthnBeginLogin111: %+v", wuser)

// 	// login
// 	opts := make([]webauthn.LoginOption, 0)
// 	opts = append(opts, webauthn.WithUserVerification(protocol.VerificationRequired))
// 	credAssertion, session, err := wauthn.BeginLogin(wuser, opts...)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
// 	}
// 	wuser.sess = session
// 	rslog.Debugf("WebAuthnBeginLogin222: %+v, session: %+v", credAssertion, session)

// 	// response
// 	return wasmResponse(credAssertion, "")
// }

// func WebAuthnFinishLogin(datas ...string) *Response {
// 	if len(datas) < 2 || datas[0] == "" || datas[1] == "" {
// 		return wasmResponse(nil, c_Error_InvalidParams)
// 	}
// 	authID, loginParams := datas[0], datas[1]

// 	rslog.Debugf("WebAuthnFinishLogin: %+v", loginParams)

// 	// user
// 	wuser, err := GetWebAuthnUser(authID)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
// 	}
// 	rslog.Debugf("WebAuthnFinishLogin: %+v", wuser)

// 	// parse loginParams
// 	collectedClientData := &protocol.CollectedClientData{}
// 	if err = json.Unmarshal([]byte(loginParams), &collectedClientData); err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_InvalidParams))
// 	}
// 	pcaData := new(protocol.ParsedCredentialAssertionData)
// 	pcaData.Response.CollectedClientData = *collectedClientData
// 	rslog.Debugf("WebAuthnFinishLogin: %+v, %+v", wuser, pcaData)

// 	// finish login
// 	cred, err := wauthn.ValidateLogin(wuser, *wuser.sess, pcaData)
// 	if err != nil {
// 		return wasmResponse(nil, rsweb.WebError(err, c_Error_SystenmExeception))
// 	}
// 	rslog.Debugf("wuser.credential: %+v", cred)

// 	// remove user at session or not

// 	// response
// 	return wasmResponse("successed", "")
// }

// // webAuthn user
// type WebauthnUser struct {
// 	ID          string
// 	Name        string
// 	DisplayName string

// 	sess *webauthn.SessionData `json:"-"`
// }

// // WebAuthnID provides the user handle of the user account. A user handle is an opaque byte sequence with a maximum
// // size of 64 bytes, and is not meant to be displayed to the user.
// //
// // To ensure secure operation, authentication and authorization decisions MUST be made on the basis of this id
// // member, not the displayName nor name members. See Section 6.1 of [RFC8266].
// //
// // It's recommended this value is completely random and uses the entire 64 bytes.
// //
// // Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dom-publickeycredentialuserentity-id)
// func (p *WebauthnUser) WebAuthnID() []byte {
// 	return []byte(p.ID)
// }

// // WebAuthnName provides the name attribute of the user account during registration and is a human-palatable name for the user
// // account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party SHOULD let the user
// // choose this, and SHOULD NOT restrict the choice more than necessary.
// //
// // Specification: §5.4.3. User Account Parameters for Credential Generation (https://w3c.github.io/webauthn/#dictdef-publickeycredentialuserentity)
// func (p *WebauthnUser) WebAuthnName() string {
// 	return p.Name
// }

// // WebAuthnDisplayName provides the name attribute of the user account during registration and is a human-palatable
// // name for the user account, intended only for display. For example, "Alex Müller" or "田中倫". The Relying Party
// // SHOULD let the user choose this, and SHOULD NOT restrict the choice more than necessary.
// //
// // Specification: §5.4.3. User Account Parameters for Credential Generation (https://www.w3.org/TR/webauthn/#dom-publickeycredentialuserentity-displayname)
// func (p *WebauthnUser) WebAuthnDisplayName() string {
// 	return p.Name
// }

// // WebAuthnCredentials provides the list of Credential objects owned by the user.
// func (p *WebauthnUser) WebAuthnCredentials() []webauthn.Credential {
// 	ret := make([]webauthn.Credential, 0)
// 	ret = append(ret, webauthn.Credential{
// 		ID: []byte("credential" + p.ID),
// 	})
// 	return ret
// }

// // WebAuthnIcon is a deprecated option.
// // Deprecated: this has been removed from the specification recommendation. Suggest a blank string.
// func (p *WebauthnUser) WebAuthnIcon() string {
// 	return "https://pics.com/avatar.png"
// }

// // webAuthn.Register走storage
// func GetWebAuthnUser(id string) (*WebauthnUser, error) {
// 	if user, ok := vStore[id]; ok {
// 		return user.(*WebauthnUser), nil
// 	} else {
// 		return nil, errors.New("user has not registered")
// 	}
// }

// func AddWebAuthnUser(id, name, displayName string) (*WebauthnUser, error) {
// 	if u, err := GetWebAuthnUser(id); u != nil && err == nil {
// 		return nil, errors.New("user has registered")
// 	}

// 	wuser := &WebauthnUser{
// 		ID:          id,
// 		Name:        name,
// 		DisplayName: displayName,
// 	}
// 	vStore[id] = wuser

// 	return wuser, nil
// }
