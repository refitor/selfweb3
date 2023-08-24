package rsstore

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/sessions"
)

const (
	C_Session_ID   = "rs-session"
	C_Session_User = "rs-user"
)

var g_session *sessions.CookieStore

func InitSession() error {
	g_session = sessions.NewCookieStore([]byte(fmt.Sprintf("%v", time.Now().UnixNano())))
	g_session.MaxAge(3600)
	return nil
}

func PushToSession(w http.ResponseWriter, r *http.Request, key string, cacheData interface{}) error {
	if session, err := g_session.Get(r, C_Session_ID); session != nil {
		session.Values[key] = cacheData
		if err := session.Save(r, w); err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}

func PopFromSession(r *http.Request, key string) interface{} {
	if session, _ := g_session.Get(r, C_Session_ID); session != nil {
		if cacheData := session.Values[key]; cacheData != nil {
			return cacheData
		}
	}
	return nil
}

func RemoveSession(w http.ResponseWriter, r *http.Request, key string) error {
	if cacheData := PopFromSession(r, key); cacheData != nil {
		if session, _ := g_session.Get(r, C_Session_ID); session != nil {
			delete(session.Values, key)
			if err := session.Save(r, w); err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("permission denied")
}
