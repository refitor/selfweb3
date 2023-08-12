package server

import (
	"encoding/json"
	"time"

	"github.com/refitor/rslog"
)

const (
	C_Store_User = "user"
)

var (
	UserSaveToStore  func(key string, val any) error
	UserGetFromStore func(key string, ptrObject any) error
)

type WebUser struct {
	ID   string `json:"id"`
	Name string `json:"username"`

	ActiveTime time.Time `json:"activeTime"`
}

type User struct {
	WebUser
	WebauthnUser json.RawMessage

	// Web3Public  *ecdsa.PublicKey
	// SelfPrivate *ecdsa.PrivateKey
}

func CreateUser(username string, webAuthnUser any) (*User, error) {
	wuser := &WebUser{
		Name: username,
	}

	wubuf, err := json.Marshal(webAuthnUser)
	if err != nil {
		return nil, err
	}
	user := &User{
		WebUser:      *wuser,
		WebauthnUser: wubuf,
	}

	// store
	if err := UserSaveToStore(username, user); err != nil {
		return nil, err
	}
	return user, nil
}

func GetUser(username string) *User {
	user := &User{}
	if err := UserGetFromStore(username, user); err != nil {
		rslog.Errorf("UserGetFromStore failed: %s, %s", username, err.Error())
		return nil
	}
	return user
}
