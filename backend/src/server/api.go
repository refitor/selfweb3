package server

import (
	"fmt"
	"net/http"

	"selfweb3/backend/pkg"
	"selfweb3/backend/pkg/rsstore"
	"selfweb3/backend/pkg/rsweb"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/julienschmidt/httprouter"
	"github.com/refitor/rslog"
)

const (
	c_datas_kind_web2Data     = "web2Data"
	c_datas_kind_bindWallet   = "bindWallet"
	c_datas_kind_relateVerify = "relateVerify"
)

func RouterInit(router *httprouter.Router) {
	RouterHook(router, http.MethodGet, "/api/datas/load", webDatasLoad, UserVerify)
	RouterHook(router, http.MethodPost, "/api/datas/store", webDatasStore, UserVerify)
	RouterHook(router, http.MethodPost, "/api/datas/forward", webDatasForward, UserVerify)

	RouterHook(router, http.MethodPost, "/api/user/auth/verify", UserAuthVerify, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/begin/login", WebAuthnBeginLogin, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/finish/login", WebAuthnFinishLogin, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/begin/register", WebAuthnBeginRegister, UserVerify)
	RouterHook(router, http.MethodPost, "/api/user/finish/register", WebAuthnFinishRegister, UserVerify)
}

func RouterHook(router *httprouter.Router, method string, path string, handleFunc http.HandlerFunc, hookFuncs ...func(w http.ResponseWriter, r *http.Request) bool) {
	router.HandlerFunc(method, path, func(w http.ResponseWriter, r *http.Request) {
		for _, hookFunc := range hookFuncs {
			if hookFunc != nil && !hookFunc(w, r) {
				rsweb.ResponseError(w, r, rsweb.C_Error_Denied)
				return
			}
		}
		handleFunc(w, r)
	})
}

func UserVerify(w http.ResponseWriter, r *http.Request) bool {
	userID := rsweb.WebParams(r).Get("userID")
	return userID != ""
}

// ====== datas ======
// POST: /api/datas/store
func webDatasStore(w http.ResponseWriter, r *http.Request) {
	kind := rsweb.WebParams(r).Get("kind")
	var handleErr error
	switch kind {
	case c_datas_kind_web2Data:
		handleErr = UserStoreWeb2Data(rsweb.WebParams(r).Get("userID"), rsweb.WebParams(r).Get("recoverID"), rsweb.WebParams(r).Get("params"))
	case c_datas_kind_bindWallet:
		handleErr = UserBindWallet(rsweb.WebParams(r).Get("oldWallet"), rsweb.WebParams(r).Get("newWallet"))
	default:
		rsweb.ResponseError(w, r, "unsupport store kind: "+kind)
	}
	if handleErr != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(handleErr, ""))
		return
	}
	rsweb.ResponseOk(w, r, "successed")
}

// GET: /api/datas/load
func webDatasLoad(w http.ResponseWriter, r *http.Request) {
	kind := rsweb.WebParams(r).Get("kind")
	switch kind {
	case c_datas_kind_web2Data:
		web2Data, err := UserLoadWeb2Data(rsweb.WebParams(r).Get("userID"), rsweb.WebParams(r).Get("public"), rsweb.WebParams(r).Get("params"))
		if err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
		rsweb.ResponseOk(w, r, map[string]any{
			pkg.C_Web2Data:      web2Data,
			pkg.C_Web2NetPublic: hexutil.Encode(crypto.CompressPubkey(vWorker.public)),
			pkg.C_Web2Address:   crypto.PubkeyToAddress(vWorker.selfPrivate.PublicKey).Hex(),
		})
	default:
		rsweb.ResponseError(w, r, "unsupport load kind: "+kind)
	}
}

// POST: /api/datas/load
func webDatasForward(w http.ResponseWriter, r *http.Request) {
	kind := rsweb.WebParams(r).Get("kind")
	switch kind {
	case "email":
		web2Map := make(map[string]string, 0)
		if err := pkg.Web2DecodeEx(vWorker.private, rsweb.WebParams(r).Get("public"), rsweb.WebParams(r).Get("params"), &web2Map); err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
		if err := SendEmailToUser("selfweb3 notifications", Str(web2Map[pkg.C_AuthorizeID]), fmt.Sprintf("[SelfWeb3] code for dynamic authorization: %s", Str(web2Map[pkg.C_AuthorizeCode]))); err != nil {
			rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
			return
		}
	default:
		rsweb.ResponseError(w, r, "unsupport forward kind: "+kind)
	}
	rsweb.ResponseOk(w, r, "successed")
}

func UserAuthVerify(w http.ResponseWriter, r *http.Request) {
	rsweb.ResponseError(w, r, "unsupport UserAuthVerify")
}

// disable user session
// // ====== user ======
// func UserStatus(w http.ResponseWriter, r *http.Request) {
// 	rsweb.ResponseOk(w, r, PopFromSession(r, C_Session_User).(*User).WebUser)
// }

// func UserLogout(w http.ResponseWriter, r *http.Request) {
// 	userID := rsweb.WebParams(r).Get("userID")
// 	if userID == "" {
// 		rsweb.ResponseError(w, r, rsweb.C_Error_InvalidParams)
// 		return
// 	}
// 	if user := PopFromSession(r, C_Session_User); user != nil && user.(*WebUser).Name != userID {
// 		rsweb.ResponseError(w, r, rsweb.C_Error_Denied)
// 		return
// 	}

// 	// remove from session
// 	RemoveSession(w, r, C_Session_User)

// 	// remove from cache
// 	WauthnClean(userID)

// 	rsweb.ResponseOk(w, r, "successed")
// }

// ====== webauthn ======
func WebAuthnBeginRegister(w http.ResponseWriter, r *http.Request) {
	userID := rsweb.WebParams(r).Get("userID")
	// displayName := rsweb.WebParams(r).Get("displayName")

	response, err, webErr := WauthnBeginRegister(userID)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}
	rsweb.ResponseOk(w, r, response)
}

func WebAuthnFinishRegister(w http.ResponseWriter, r *http.Request) {
	user := &User{}
	userID := rsweb.WebParams(r).Get("userID")
	if err := rsstore.LoadFromDB(C_Store_User, userID, user); err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}

	_, err, webErr := WauthnFinishRegister(userID, string(user.Web2Data.WebAuthnKey), r.Body)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}
	rsweb.ResponseOk(w, r, "successed")
}

func WebAuthnBeginLogin(w http.ResponseWriter, r *http.Request) {
	web2Data := &pkg.Web2Data{}
	userID := rsweb.WebParams(r).Get("userID")
	rslog.Infof("WebAuthnBeginLogin: %s", rsweb.WebParams(r).Get("webAuthnKey"))
	// if err := pkg.Web2Decode(vWorker.private, vWorker.WebPublic, rsweb.WebParams(r).Get("webAuthnKey"), web2Data); err != nil {
	// 	rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
	// 	return
	// }
	// rslog.Infof("WebAuthnBeginLogin parse web2Data successed: %+v", web2Data)

	response, err, webErr := WauthnBeginLogin(userID, Str(web2Data.WebAuthnKey))
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, webErr))
		return
	}
	rsweb.ResponseOk(w, r, response)
}

func WebAuthnFinishLogin(w http.ResponseWriter, r *http.Request) {
	userID := rsweb.WebParams(r).Get("userID")

	response, err, webErr := WauthnFinishLogin(userID, r.Body)
	if webErr != "" {
		rsweb.ResponseError(w, r, webErr)
		return
	}
	if err != nil {
		rsweb.ResponseError(w, r, rsweb.WebError(err, ""))
		return
	}
	rslog.Infof("WebAuthnFinishLogin response: %v", response)

	// // push user to session
	// PushToSession(w, r, C_Session_User, GetUser(userID))

	// callback after webAuthnLogin
	UserAfterWebAuthnLogin()

	rsweb.ResponseOk(w, r, response)
}

func UserSessionCheck(w http.ResponseWriter, r *http.Request) bool {
	return rsstore.PopFromSession(r, rsstore.C_Session_User) != ""
}
