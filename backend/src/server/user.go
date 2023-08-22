package server

import (
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"selfweb3/backend/pkg/rsauth"
	"selfweb3/backend/pkg/rscrypto"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
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
	Web2Key            string `json:"Web2Key"`
	EncryptWeb2Private string `json:"Web2Private"`
}

type User struct {
	WebUser

	RecoverID    []byte            `json:"recoverID"`
	Web2Private  *ecdsa.PrivateKey `json:"-"`
	WebauthnUser json.RawMessage   `json:"WebauthnUser"`
}

func CreateUser(userID string, webAuthnUser any) (*User, error) {
	wuser := &WebUser{}

	wubuf, err := json.Marshal(webAuthnUser)
	if err != nil {
		return nil, err
	}
	private, err := crypto.GenerateKey()
	if err != nil {
		return nil, err
	}
	user := &User{
		WebUser:      *wuser,
		Web2Private:  private,
		WebauthnUser: wubuf,
	}
	user.WebUser.Web2Key = rscrypto.GetRandom(32, false)
	user.EncryptWeb2Private = hexutil.Encode(rscrypto.AesEncryptECB(crypto.FromECDSA(user.Web2Private), []byte(user.WebUser.Web2Key)))
	rslog.Debugf("CreateUser successed: %+v", user)

	// store
	if err := UserSaveToStore(userID, user); err != nil {
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

func (p *User) Web2EncodePrivate(dhNetKey string) string {
	web2Private := append(crypto.FromECDSA(p.Web2Private), []byte(dhNetKey)...)
	return hex.EncodeToString(rscrypto.AesEncryptECB(web2Private, []byte(p.Web2Key)))
}

func SendEmailToUser(email, content string) error {
	sendCh := make(chan struct{})
	if _, err := rsauth.PushByEmail(email, "dynamic authorization", "", content, func(err error) {
		if err != nil {
			rslog.Errorf("email send failed: %s", err.Error())
		}
		close(sendCh)
	}); err != nil {
		return err
	}
	<-sendCh
	return nil
}
