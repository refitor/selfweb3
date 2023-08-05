package server

import (
	"errors"
	"net/http"
)

func WebStatusCheck(r *http.Request) bool {
	return popFromSession(r) != ""
}

func pushToSession(w http.ResponseWriter, r *http.Request, cacheData interface{}) error {
	if session, err := vWorker.session.Get(r, c_Session_ID); session != nil {
		session.Values[c_Session_User] = cacheData
		if err := session.Save(r, w); err != nil {
			return err
		}
		return nil
	} else {
		return err
	}
}

func popFromSession(r *http.Request) interface{} {
	if session, _ := vWorker.session.Get(r, c_Session_ID); session != nil {
		if cacheData := session.Values[c_Session_User]; cacheData != nil {
			return cacheData
		}
	}
	return nil
}

func removeSession(w http.ResponseWriter, r *http.Request) error {
	if cacheData := popFromSession(r); cacheData != nil {
		if session, _ := vWorker.session.Get(r, c_Session_ID); session != nil {
			delete(session.Values, c_Session_User)
			if err := session.Save(r, w); err != nil {
				return err
			}
			return nil
		}
	}
	return errors.New("permission denied")
}
