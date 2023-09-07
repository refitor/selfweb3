package server

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"net/url"
	"strings"
	"sync"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
	"github.com/refitor/rslog"
)

var (
	wcache               sync.Map
	wauthn               *webauthn.WebAuthn
	WebauthnSaveToStore  func(key, encryptKey string, val any) error
	WebauthnGetFromStore func(key, decryptKey string, ptrObject any) error
)

func InitWebAuthn(rpOrigin string) error {
	u, _ := url.Parse(rpOrigin)
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "SelfWeb3",                 // Display Name for your site
		RPID:          u.Hostname(),               // Generally the domain name for your site
		RPOrigin:      rpOrigin,                   // The origin URL for WebAuthn requests
		RPIcon:        "https://duo.com/logo.png", // Optional icon URL for your site
	})

	// Mandatory verification by mobile phone or tablet cross-platform terminal device
	// w.Config.AuthenticatorSelection.AuthenticatorAttachment = protocol.CrossPlatform

	wauthn = w
	return err
}

// webauthnUser represents the user model
type webauthnUser struct {
	ID          uint64                `json:"id"`
	Name        string                `json:"name"`
	DisplayName string                `json:"displayName"`
	Credentials []webauthn.Credential `json:"credentials"`
	SessionData *webauthn.SessionData `json:"-"`
}

// NewUser creates and returns a new User
func NewWebauthnUser(name string, displayName string) *webauthnUser {

	user := &webauthnUser{}
	user.ID = randomUint64()
	user.Name = name
	user.DisplayName = displayName
	// user.credentials = []webauthn.Credential{}

	return user
}

func randomUint64() uint64 {
	buf := make([]byte, 8)
	rand.Read(buf)
	return binary.LittleEndian.Uint64(buf)
}

// WebAuthnID returns the user's ID
func (u webauthnUser) WebAuthnID() []byte {
	buf := make([]byte, binary.MaxVarintLen64)
	binary.PutUvarint(buf, uint64(u.ID))
	return buf
}

// WebAuthnName returns the user's username
func (u webauthnUser) WebAuthnName() string {
	return u.Name
}

// WebAuthnDisplayName returns the user's display name
func (u webauthnUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

// WebAuthnIcon is not (yet) implemented
func (u webauthnUser) WebAuthnIcon() string {
	return ""
}

// AddCredential associates the credential to the user
func (u *webauthnUser) AddCredential(cred webauthn.Credential) {
	u.Credentials = append(u.Credentials, cred)
}

// WebAuthnCredentials returns credentials owned by the user
func (u webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.Credentials
}

// CredentialExcludeList returns a CredentialDescriptor array filled
// with all the user's credentials
func (u webauthnUser) CredentialExcludeList() []protocol.CredentialDescriptor {

	credentialExcludeList := []protocol.CredentialDescriptor{}
	for _, cred := range u.Credentials {
		descriptor := protocol.CredentialDescriptor{
			Type:         protocol.PublicKeyCredentialType,
			CredentialID: cred.ID,
		}
		credentialExcludeList = append(credentialExcludeList, descriptor)
	}

	return credentialExcludeList
}

// webauthn handler
func WauthnBeginRegister(username string) (interface{}, error, string) {
	// generate webauthn user
	displayName := strings.Split(username, "@")[0]
	user := NewWebauthnUser(username, displayName)

	// begin registration
	registerOptions := func(credCreationOpts *protocol.PublicKeyCredentialCreationOptions) {
		credCreationOpts.CredentialExcludeList = user.CredentialExcludeList()
	}
	options, sessionData, err := wauthn.BeginRegistration(
		user,
		registerOptions,
	)
	if err != nil {
		return nil, err, ""
	}
	user.SessionData = sessionData

	// cache
	wcache.Store(username, user)

	// response
	return options, nil, ""
}

func WauthnFinishRegister(username, webAuthnKey string, bufReader io.Reader) (*webauthnUser, error, string) {
	// get webauthn user
	var user *webauthnUser
	if cacheUser, _ := wcache.Load(username); cacheUser != nil {
		user = cacheUser.(*webauthnUser)
	} else {
		return nil, errors.New("not found user: " + username), ""
	}

	// parse credential
	parsedResponse, err := protocol.ParseCredentialCreationResponseBody(bufReader)
	if err != nil {
		return nil, err, ""
	}

	// finish registration
	credential, err := wauthn.CreateCredential(user, *user.SessionData, parsedResponse)
	if err != nil {
		return nil, err, ""
	}
	user.AddCredential(*credential)
	user.SessionData = nil

	rslog.Debugf("WauthnFinishRegister add credential: %+v, webAuthnKey: %s", user, webAuthnKey)

	// store
	if WebauthnSaveToStore != nil && webAuthnKey != "" {
		WebauthnSaveToStore(username, webAuthnKey, user)
	}

	// response
	return user, nil, ""
}

func WauthnBeginLogin(username, webAuthnKey string) (interface{}, error, string) {
	// get webauthn user
	var user *webauthnUser
	if cacheUser, _ := wcache.Load(username); cacheUser != nil {
		user = cacheUser.(*webauthnUser)
	} else if WebauthnGetFromStore != nil {
		// load user from the storage
		user = &webauthnUser{}
		if err := WebauthnGetFromStore(username, webAuthnKey, user); err != nil {
			rslog.Errorf("GetStoreUser failed, username: %s, detail: %s", username, err.Error())
			return nil, nil, "invalid user"
		}
	}
	if user == nil {
		return nil, errors.New("not found user: " + username), ""
	}

	// begin login
	options, sessionData, err := wauthn.BeginLogin(user)
	if err != nil {
		return nil, err, ""
	}
	user.SessionData = sessionData

	// cache
	wcache.Store(username, user)

	return options, nil, ""
}

func WauthnFinishLogin(username string, bufReader io.Reader) (interface{}, error, string) {
	// get webauthn user
	var user *webauthnUser
	if cacheUser, _ := wcache.Load(username); cacheUser != nil {
		user = cacheUser.(*webauthnUser)
	} else {
		return nil, errors.New("not found user: " + username), ""
	}

	// parse credential
	parsedResponse, err := protocol.ParseCredentialRequestResponseBody(bufReader)
	if err != nil {
		return nil, err, ""
	}

	// finish login
	// in an actual implementation, we should perform additional checks on
	// the returned 'credential', i.e. check 'credential.Authenticator.CloneWarning'
	// and then increment the credentials counter
	_, err = wauthn.ValidateLogin(user, *user.SessionData, parsedResponse)
	if err != nil {
		return nil, err, ""
	}
	return "successed", nil, ""
}

func WauthnClean(username string) {
	if u, ok := wcache.Load(username); u != nil && ok {
		wcache.Delete(username)
	}
}
