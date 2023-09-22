package server

import (
	"fmt"
	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rsauth"
	"selfweb3/backend/pkg/rscrypto"
	"selfweb3/backend/pkg/rsstore"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/refitor/rslog"
	// uuid "github.com/satori/go.uuid"
	// "github.com/twmb/murmur3"
)

const (
	C_Store_User                     = "user"
	c_cache_webAuthnLogin            = "webAuthnLogin"
	c_cache_webAuthnLogin_bindWallet = "webAuthnLogin_bindWallet"
)

var (
	UserSaveToStore  func(key string, val any) error
	UserGetFromStore func(key string, ptrObject any) error
)

type User struct {
	pkg.Web2Data

	Web2Key      []byte
	Web2Nonce    int64
	RecoverID    []byte
	WebauthnUser []byte
}

func CreateUser(userID string) (*User, error) {
	// // generate userID
	// uid := uuid.NewV1()
	// hash := murmur3.Sum32([]byte(uid.String()))
	// selfID := fmt.Sprintf("%v", hash)

	user := &User{}
	user.Web2Key = []byte(rscrypto.GetRandom(32, false))
	webAuthnKey, err := rscrypto.GetDhKey(vWorker.WebPublic, vWorker.private)
	if err != nil {
		return nil, err
	}
	user.Web2Nonce = time.Now().UnixNano()
	user.Web2Data.WebAuthnKey = []byte(webAuthnKey)
	user.Web2Data.VerifyNonce = time.Now().UnixNano()
	user.Web2Data.Web2Public = crypto.CompressPubkey(&vWorker.selfPrivate.PublicKey)
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

func UserStoreWeb2Data(userID, recoverID, encryptWeb2Data string) error {
	rslog.Infof("before store user web2Data: %s, %s, %s", userID, recoverID, encryptWeb2Data)

	// load user
	user := &User{}
	if err := rsstore.LoadFromDB(C_Store_User, userID, user); err != nil {
		return err
	}
	if recoverID != "" {
		user.RecoverID = rscrypto.AesEncryptECB([]byte(recoverID), []byte(user.Web2Key))
	} else if len(user.RecoverID) > 0 {
		recoverID = string(rscrypto.AesDecryptECB(user.RecoverID, []byte(user.Web2Key)))
	}

	// parse encryptWeb2Data
	if err := pkg.Web2Decode(vWorker.private, vWorker.WebPublic, encryptWeb2Data, &user.Web2Data); err != nil {
		return err
	}

	rslog.Debugf("before store user: %+v", user)

	// store and send notifications
	if err := UserSaveToStore(userID, user); err != nil {
		return err
	}
	utokens := []rune(user.SelfAddress)
	selfAddress := fmt.Sprintf("%s...%s", string(utokens[:4]), string(utokens[len(utokens)-4:]))
	if err := SendEmailToUser("selfweb3 notifications", recoverID, fmt.Sprintf("[SelfWeb3] Hi. Your web3 account %s has been successfully registered. Welcome to SelfWeb3 to start exploring your own web3 ecosystem.", selfAddress)); err != nil {
		return err
	}
	rslog.Infof("store user web2Data successed: %+v", user.Web2Data)
	return nil
}

func UserLoadWeb2Data(userID, webPublic, params string) (any, error) {
	rslog.Infof("before load user web2Data: %s, %s, %s", userID, webPublic, params)

	// parse webPublic
	if webPublic != "" {
		publicKey, err := rscrypto.GetPublicKey(webPublic)
		if err != nil {
			return nil, err
		}
		vWorker.WebPublic = publicKey
	}

	// load user
	user := &User{}
	if err := rsstore.LoadFromDB(C_Store_User, userID, user); err != nil {
		if params == "initWeb2" {
			u, err := CreateUser(userID)
			if err != nil {
				return nil, err
			}
			user = u
		} else {
			return nil, err
		}
	}

	// encrypt web2Data
	web2Data, err := pkg.Web2EncodeEx(vWorker.private, webPublic, &user.Web2Data)
	if err != nil {
		return nil, err
	}
	rslog.Infof("load user web2Data successed: %s", web2Data)
	return web2Data, nil
}

func SendEmailToUser(title, email, content string) error {
	sendCh := make(chan struct{})
	if _, err := rsauth.PushByEmail(email, title, "", content, func(err error) {
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

func UserBindWallet(oldWallet, newWallet string) error {
	// load user
	user := &User{}
	if err := rsstore.LoadFromDB(C_Store_User, oldWallet, user); err != nil {
		return err
	}

	// cache
	afterWebAuthnLoginBind := func(key, val any) {
		if err := UserSaveToStore(Str(key), val.([]any)[1]); err != nil {
			rslog.Errorf("BindWallet store failed, key: %s, UserSaveToStore: %s", key, err.Error())
		} else {
			if err := rsstore.Store().DBDel(C_Store_User, Str(val.([]any)[2])); err != nil {
				rslog.Errorf("BindWallet delete failed, key: %s, DBDel: %s", key, err.Error())
				// TODO: 采集异常情况，根据策略处理，比如定时，重启或者管理员手动
			}
		}
	}
	cacheValue := []any{
		afterWebAuthnLoginBind,
		user,
		oldWallet,
	}
	cacheKey := fmt.Sprintf("%s%s", c_cache_webAuthnLogin, newWallet)
	rsstore.SetCacheByTime(cacheKey, cacheValue, true, 300, func(key, val any) bool {
		rslog.Errorf("BindWallet failed with 300s timeout, key: %s, val: %s", key, val)
		return true
	})
	return nil
}

func UserAfterWebAuthnLogin() {
	rsstore.Cache().Range(func(key, value any) bool {
		if strings.HasPrefix(Str(key), c_cache_webAuthnLogin) {
			if valList, ok := value.([]any); ok && len(valList) >= 1 {
				callback := valList[0].(func(any, any))
				callback(strings.TrimPrefix(Str(key), c_cache_webAuthnLogin), value)
				return false
			}
		}
		return true
	})
}
