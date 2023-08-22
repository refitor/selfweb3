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
	"sync"

	"selfweb3/backend/pkg/rsauth"
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
	rsauth.InitEmail("smtp.126.com:465", "refitor@126.com", "ZPHRFSXTEQUFNYLB")
}

type Worker struct {
	cache     sync.Map
	config    sync.Map
	memvar    sync.Map
	public    *ecdsa.PublicKey
	private   *ecdsa.PrivateKey
	NetPublic *ecdsa.PublicKey
}

func (p *Worker) Init() {
	rslog.SetLevel("info")

	// db
	FatalCheck(InitSession())
	FatalCheck(InitStore(*DBPath))
	FatalCheck(InitWebAuthn(*hostURL))

	CreateDB(C_Store_User)

	// user
	UserSaveToStore = func(key string, val any) error {
		return SaveToDB(C_Store_User, key, val)
	}
	UserGetFromStore = func(key string, ptrObject any) error {
		return LoadFromDB(C_Store_User, key, ptrObject)
	}

	// wabauthn
	// WebauthnSaveToStore = func(key string, val any) error {
	// 	user, err := CreateUser(key, val)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	return SaveToDB(C_Store_User, key, user)
	// }
	WebauthnGetFromStore = func(key string, ptrObject any) error {
		user := &User{}
		if err := LoadFromDB(C_Store_User, key, user); err != nil {
			return err
		}
		return json.Unmarshal(user.WebauthnUser, ptrObject)
	}

	rsauth.InitEmail("smtp.126.com:465", "refitor@gmail.com", "xxxxxxxxxxxxx")
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

func SetVar(key string, val interface{}, bForce bool) error {
	if !bForce {
		if _, ok := vWorker.memvar.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	vWorker.memvar.Store(key, val)
	return nil
}

// delete: beforeDelleteFunc return true
func GetVar(key string, bDelete bool, beforeDeleteFunc func(v interface{}) bool) interface{} {
	val, _ := vWorker.memvar.Load(key)
	if beforeDeleteFunc != nil {
		if beforeDeleteFunc(val) {
			vWorker.memvar.Delete(key)
		}
	} else if bDelete {
		vWorker.memvar.Delete(key)
	}
	return val
}

func SetConf(key, val string) {
	vWorker.config.Store(key, val)
}

func GetConf(key string) string {
	val, _ := vWorker.config.Load(key)
	return fmt.Sprintf("%v", val)
}

func GetCache(key any) any {
	v, _ := vWorker.cache.Load(key)
	return v
}

func SetCache(key, val any) {
	vWorker.cache.Store(key, val)
}

func FatalCheck(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func CreateDB(dbName string) error {
	return g_db.DBCreate(dbName)
}

func LoadFromDB(dbName, key string, ptrObject any) error {
	if buf, err := g_db.DBGet(dbName, key); err == nil {
		if ptrObject != nil {
			return json.Unmarshal(buf, ptrObject)
		} else {
			return nil
		}
	} else {
		return err
	}
}

func SaveToDB(dbName string, cacheKey, cacheValue any) (retErr error) {
	storeFunc := func(key, val any) error {
		abuf, err := json.Marshal(val)
		if err != nil {
			return err
		}
		if err := g_db.DBPut(dbName, fmt.Sprintf("%v", key), abuf); err != nil {
			return err
		}
		return nil
	}
	if cacheKey == "" {
		vWorker.cache.Range(func(key, value any) bool {
			if err := storeFunc(key, value); err != nil {
				rslog.Errorf("SaveToDB failed, dbName: %s, key: %v, val: %v", dbName, key, value)
				return false
			}
			return true
		})
	} else {
		return storeFunc(cacheKey, cacheValue)
	}
	return
}

func Str(data any) string {
	return fmt.Sprintf("%v", data)
}
