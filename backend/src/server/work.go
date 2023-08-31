package server

import (
	"context"
	"crypto/ecdsa"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"

	"selfweb3/backend/pkg/rsauth"
	"selfweb3/backend/pkg/rscrypto"
	"selfweb3/backend/pkg/rsstore"
	"selfweb3/backend/pkg/rsweb"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/refitor/rslog"
	"github.com/urfave/negroni"
)

// global const
const (
	C_date_time = "2006-01-02 15:04:05"

	C_Url_host = "https://selfrscrypto.refitor.com"
)

var vWorker *Worker

// flag
var (
	DBPath  = flag.String("dbpath", "./selfweb3.db", "--dbPath=./selfweb3.db")
	webPort = flag.String("port", "3157", "--port=3157")
	webPath = flag.String("webpath", "rsweb", "--webpath=rsweb")
	hostURL = flag.String("hosturl", "https://selfweb3.refitor.com", "--hosturl=https://example.com")
)

func Run(ctx context.Context, fs *embed.FS) {
	// init
	vWorker = newWorker()
	vWorker.Init()
	defer vWorker.UnInit()

	// run
	router := rsweb.Init(*webPath, fs, RouterInit)
	go rsweb.Run(ctx, *webPort, func() http.Handler {
		n := negroni.New()
		n.Use(rsweb.NewCors(false, "http://localhost:5173", "http://localhost:3157", "https://*.refitor.com"))
		n.UseFunc(rsweb.NewGzip)
		n.Use(rsweb.NewRateLimite())
		n.UseFunc(rsweb.NewAPILog)
		n.UseHandlerFunc(router.ServeHTTP)
		return n
	})
}

type Worker struct {
	public    *ecdsa.PublicKey
	private   *ecdsa.PrivateKey
	WebPublic *ecdsa.PublicKey
}

func (p *Worker) Init() {
	rslog.SetLevel("info")

	// db
	FatalCheck(rsstore.InitSession())
	FatalCheck(rsstore.InitStore(*DBPath))
	FatalCheck(InitWebAuthn(*hostURL))

	rsstore.CreateDB(C_Store_User)

	// user
	UserSaveToStore = func(key string, val any) error {
		return rsstore.SaveToDB(rsstore.Cache(), C_Store_User, key, val)
	}
	UserGetFromStore = func(key string, ptrObject any) error {
		return rsstore.LoadFromDB(C_Store_User, key, ptrObject)
	}

	// wabauthn
	WebauthnSaveToStore = func(key, encryptKey string, val any) error {
		user := &User{}
		if err := rsstore.LoadFromDB(C_Store_User, key, user); err != nil {
			return err
		}
		wbuf, err := json.Marshal(val)
		if err != nil {
			return err
		}
		user.WebauthnUser = rscrypto.AesDecryptECB(wbuf, []byte(encryptKey))
		return rsstore.SaveToDB(rsstore.Cache(), C_Store_User, key, user)
	}
	WebauthnGetFromStore = func(key, decryptKey string, ptrObject any) error {
		user := &User{}
		if err := rsstore.LoadFromDB(C_Store_User, key, user); err != nil {
			return err
		}
		return json.Unmarshal(rscrypto.AesDecryptECB([]byte(user.WebauthnUser), []byte(decryptKey)), ptrObject)
	}
	rsauth.InitEmail("smtp.126.com:465", "refitor@126.com", "xxxxxx")
}

func (p *Worker) UnInit() {
}

func newWorker() *Worker {
	s := new(Worker)
	private, ecdsaErr := crypto.GenerateKey()
	FatalCheck(ecdsaErr)
	s.private = private
	s.public = &private.PublicKey
	return s
}

func FatalCheck(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func Str(data any) string {
	return fmt.Sprintf("%v", data)
}
