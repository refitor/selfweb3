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

const (
	// need user registration
	c_error_user_invalid = "user.invalid"
)

var (
	wcache               sync.Map
	wauthn               *webauthn.WebAuthn
	WebauthnSaveToStore  func(key string, val any) error
	WebauthnGetFromStore func(key string, ptrObject any) error
)

func InitWebAuthn(rpOrigin string) error {
	u, _ := url.Parse(rpOrigin)
	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "SelfWeb3",                 // Display Name for your site
		RPID:          u.Hostname(),               // Generally the domain name for your site
		RPOrigin:      rpOrigin,                   // The origin URL for WebAuthn requests
		RPIcon:        "https://duo.com/logo.png", // Optional icon URL for your site
	})
	wauthn = w
	return err
}

// webauthnUser represents the user model
type webauthnUser struct {
	id          uint64
	name        string
	displayName string
	credentials []webauthn.Credential
	SessionData *webauthn.SessionData `json:"-"`
}

// NewUser creates and returns a new User
func NewWebauthnUser(name string, displayName string) *webauthnUser {

	user := &webauthnUser{}
	user.id = randomUint64()
	user.name = name
	user.displayName = displayName
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
	binary.PutUvarint(buf, uint64(u.id))
	return buf
}

// WebAuthnName returns the user's username
func (u webauthnUser) WebAuthnName() string {
	return u.name
}

// WebAuthnDisplayName returns the user's display name
func (u webauthnUser) WebAuthnDisplayName() string {
	return u.displayName
}

// WebAuthnIcon is not (yet) implemented
func (u webauthnUser) WebAuthnIcon() string {
	return ""
}

// AddCredential associates the credential to the user
func (u *webauthnUser) AddCredential(cred webauthn.Credential) {
	u.credentials = append(u.credentials, cred)
}

// WebAuthnCredentials returns credentials owned by the user
func (u webauthnUser) WebAuthnCredentials() []webauthn.Credential {
	return u.credentials
}

// CredentialExcludeList returns a CredentialDescriptor array filled
// with all the user's credentials
func (u webauthnUser) CredentialExcludeList() []protocol.CredentialDescriptor {

	credentialExcludeList := []protocol.CredentialDescriptor{}
	for _, cred := range u.credentials {
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
	storeUser := &webauthnUser{}
	if err := WebauthnGetFromStore(username, storeUser); storeUser != nil && err == nil {
		return nil, nil, "user registration again and again"
	}

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

func WauthnFinishRegister(username string, bufReader io.Reader, bSave bool) (*webauthnUser, error, string) {
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

	// store
	if bSave {
		WebauthnSaveToStore(username, user)
	}

	// response
	return user, nil, ""
}

func WauthnBeginLogin(username string) (interface{}, error, string) {
	// get webauthn user
	var user *webauthnUser
	if cacheUser, _ := wcache.Load(username); cacheUser != nil {
		user = cacheUser.(*webauthnUser)
	} else {
		// load user from the storage
		user = &webauthnUser{}
		if err := WebauthnGetFromStore(username, user); err != nil {
			rslog.Errorf("GetStoreUser failed, username: %s, detail: %s", username, err.Error())
			return nil, nil, c_error_user_invalid
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
