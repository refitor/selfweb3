package server

import (
	"net/http"

	"github.com/julienschmidt/httprouter"
)

func AuthInitRouter(router *httprouter.Router) {
	router.HandlerFunc(http.MethodGet, "/api/datas/load", webDatasLoad)
	router.HandlerFunc(http.MethodPost, "/api/datas/store", webDatasStore)
	router.HandlerFunc(http.MethodPost, "/api/user/recover", webUserRecover)
}

// Post: /api/datas/store
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webDatasStore(w http.ResponseWriter, r *http.Request) {

}

// Get: /api/datas/load
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webDatasLoad(w http.ResponseWriter, r *http.Request) {

}

// Get: /api/user/recover
// @request authID wallet address
// @response backendPublic backend public key
// @response walletPublic wallet account public key
func webUserRecover(w http.ResponseWriter, r *http.Request) {

}
