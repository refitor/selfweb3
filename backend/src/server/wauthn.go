package server

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"io"
	"strings"

	"github.com/duo-labs/webauthn/protocol"
	"github.com/duo-labs/webauthn/webauthn"
)

const (
	C_Store_WebauthnUser = "webauthn_user"
)

var wauthn *webauthn.WebAuthn
var loadUserFromStore func(string, string, any) error

func InitWebAuthn(rpOrigin string) error {
	rpID := strings.TrimPrefix(rpOrigin, "http://")
	rpID = strings.TrimPrefix(rpOrigin, "https://")

	w, err := webauthn.New(&webauthn.Config{
		RPDisplayName: "SelfWeb3",                 // Display Name for your site
		RPID:          rpID,                       // Generally the domain name for your site
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
	SessionData *webauthn.SessionData
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
func WauthnBeginRegister(username string, userLoad func(key any) (any, bool), userStore func(k, u any)) (interface{}, error, string) {
	// generate webauthn user
	if u, ok := userLoad(username); u != nil || ok {
		return nil, nil, "user has registered"
	}
	displayName := strings.Split(username, "@")[0]
	user := NewWebauthnUser(username, displayName)
	userStore(username, user)

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

	// response
	return options, nil, ""
}

func WauthnFinishRegister(username string, bufReader io.Reader, userLoad func(key any) (any, bool)) (interface{}, error, string) {
	// get webauthn user
	var user *webauthnUser
	if u, _ := userLoad(username); u == nil {
		return nil, errors.New("not found user: " + username), ""
	} else {
		user = u.(*webauthnUser)
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

	// response
	return "successed", nil, ""
}

func WauthnBeginLogin(username string, userLoad func(key any) (any, bool)) (interface{}, error, string) {
	// get webauthn user
	var user *webauthnUser
	if u, ok := userLoad(username); u != nil {
		user = u.(*webauthnUser)
	} else if ok {
		// load user to the cache
		user = &webauthnUser{}
		if err := loadUserFromStore(C_Store_WebauthnUser, username, user); err != nil {
			return nil, err, ""
		}
		if u, _ = userLoad(username); u != nil {
			user = u.(*webauthnUser)
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

	return options, nil, ""
}

func WauthnFinishLogin(username string, bufReader io.Reader, userLoad func(key any) (any, bool)) (interface{}, error, string) {
	// get webauthn user
	var user *webauthnUser
	if u, _ := userLoad(username); u == nil {
		return nil, errors.New("not found user: " + username), ""
	} else {
		user = u.(*webauthnUser)
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
