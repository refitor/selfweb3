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

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/refitor/rslog"
	"github.com/urfave/negroni"
)

// global const
const (
	C_Store_Sys = "sys"
	C_date_time = "2006-01-02 15:04:05"

	C_Url_host = "https://selfrscrypto.refitor.com"
)

var vWorker *Worker

// flag
var (
	syskey  = flag.String("syskey", "", "--syskey=xxxxx")
	webPort = flag.String("port", "3157", "--port=3157")
	webPath = flag.String("webpath", "rsweb", "--webpath=rsweb")
	privKey = flag.String("privkey", "", "--privkey=xxxxxxxxxxxxxxxxxxxx")
	DBPath  = flag.String("dbpath", "./selfweb3.db", "--dbPath=./selfweb3.db")
	hostURL = flag.String("hosturl", "https://selfweb3.refitor.com", "--hosturl=https://example.com")
)

func Run(ctx context.Context, fs *embed.FS) {
	rslog.SetLevel("debug")

	// init
	vWorker = newWorker()
	vWorker.Init()
	defer vWorker.UnInit()

	// run
	router := rsweb.Init(*webPath, fs, RouterInit)
	go rsweb.Run(ctx, *webPort, func() http.Handler {
		n := negroni.New()
		n.Use(rsweb.NewCors(false, "http://localhost:5173", "http://localhost:9092", "http://localhost:3157", "https://*.refitor.com"))
		n.UseFunc(rsweb.NewGzip)
		n.Use(rsweb.NewRateLimite())
		n.UseFunc(rsweb.NewAPILog)
		n.UseHandlerFunc(router.ServeHTTP)
		return n
	})
}

type Meta struct {
	EncryptedPrivateKey string `json:"private"`
}

type Worker struct {
	meta        *Meta
	sysKey      string
	public      *ecdsa.PublicKey
	private     *ecdsa.PrivateKey
	WebPublic   *ecdsa.PublicKey
	selfPrivate *ecdsa.PrivateKey
}

func (p *Worker) Init() {
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
		user.WebauthnUser = wbuf
		// user.WebauthnUser = rscrypto.AesEncryptECB(wbuf, []byte(encryptKey))
		rslog.Debugf("WebauthnSaveToStore: %s, %s, %+v", key, encryptKey, user)
		return rsstore.SaveToDB(rsstore.Cache(), C_Store_User, key, user)
	}
	WebauthnGetFromStore = func(key, decryptKey string, ptrObject any) error {
		user := &User{}
		if err := rsstore.LoadFromDB(C_Store_User, key, user); err != nil {
			return err
		}
		wauthnBuf := user.WebauthnUser
		// wauthnBuf := rscrypto.AesDecryptECB([]byte(user.WebauthnUser), []byte(decryptKey))
		rslog.Debugf("WebauthnGetFromStore: %s, %s, %+v", key, decryptKey, user)
		return json.Unmarshal(wauthnBuf, ptrObject)
	}
	rsauth.InitEmail("smtp.126.com:465", "refitor@126.com", "xxxxxx")
}

func (p *Worker) UnInit() {
}

func newWorker() *Worker {
	// db
	FatalCheck(rsstore.InitSession())
	FatalCheck(rsstore.InitStore(*DBPath))
	FatalCheck(InitWebAuthn(*hostURL))

	rsstore.CreateDB(C_Store_Sys)
	rsstore.CreateDB(C_Store_User)

	s := new(Worker)
	private, ecdsaErr := crypto.GenerateKey()
	FatalCheck(ecdsaErr)
	s.private = private
	s.public = &private.PublicKey
	s.meta = &Meta{}
	s.sysKey = *syskey
	s.meta.EncryptedPrivateKey = hexutil.Encode(rscrypto.AesEncryptECB(crypto.FromECDSA(private), []byte(s.sysKey)))

	encryptedPrivateKey := *privKey
	if encryptedPrivateKey == "" {
		dbMeta := &Meta{}
		if err := rsstore.LoadFromDB(C_Store_Sys, "meta", dbMeta); err != nil {
			FatalCheck(rsstore.SaveToDB(nil, C_Store_Sys, "meta", s.meta))
			encryptedPrivateKey = s.meta.EncryptedPrivateKey
		} else {
			encryptedPrivateKey = dbMeta.EncryptedPrivateKey
		}
	}

	ebuf, _ := hexutil.Decode(encryptedPrivateKey)
	web2PrivateKey, err := crypto.ToECDSA(rscrypto.AesDecryptECB(ebuf, []byte(s.sysKey)))
	if err != nil {
		FatalCheck(fmt.Errorf("Init web2 server failed with invalid privateKey: %s", err.Error()))
	}
	s.selfPrivate = web2PrivateKey
	rslog.Debugf("init web2 server successed, address: %s, %s, %s", crypto.PubkeyToAddress(s.selfPrivate.PublicKey).Hex(), s.sysKey, encryptedPrivateKey)
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
