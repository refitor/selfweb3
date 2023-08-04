package server

import (
	"context"
	"crypto/ecdsa"
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"sync"

	"selfweb3/pkg/rsauth"
	"selfweb3/pkg/rsweb"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/refitor/rslog"
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
)

func Run(ctx context.Context, fs *embed.FS) {
	// init
	vWorker = newWorker()
	vWorker.Init()
	defer vWorker.UnInit()

	// run
	go rsweb.Run(ctx, *webPort, rsweb.Init(*webPath, fs, AuthInitRouter), false, "http://localhost:8000", "http://localhost:3157", "https://*.refitor.com")
}

type Worker struct {
	db      *db_bolt
	cache   sync.Map
	config  sync.Map
	memvar  sync.Map
	public  *ecdsa.PublicKey
	private *ecdsa.PrivateKey
}

func (p *Worker) Init() {
	rslog.SetLevel("info")

	// db
	db, err := boltDBInit(*DBPath)
	FatalCheck(err)
	vWorker.db = db
	FatalCheck(vWorker.db.DBCreate("default"))

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

func (p *Worker) SetVar(key string, val interface{}, bForce bool) error {
	if !bForce {
		if _, ok := p.memvar.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	p.memvar.Store(key, val)
	return nil
}

// delete: beforeDelleteFunc return true
func (p *Worker) GetVar(key string, bDelete bool, beforeDelleteFunc func(v interface{}) bool) interface{} {
	val, _ := p.memvar.Load(key)
	if beforeDelleteFunc != nil {
		if beforeDelleteFunc(val) {
			p.memvar.Delete(key)
		}
	} else if bDelete {
		p.memvar.Delete(key)
	}
	return val
}

func (p *Worker) SetConf(key, val string) {
	p.config.Store(key, val)
}

func (p *Worker) GetConf(key string) string {
	val, _ := p.config.Load(key)
	return fmt.Sprintf("%v", val)
}

func FatalCheck(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (p *Worker) SaveToDB(name string) (retErr error) {
	p.cache.Range(func(key, value any) bool {
		abuf, err := json.Marshal(value)
		if err != nil {
			retErr = err
			return false
		}
		if err := vWorker.db.DBPut(name, fmt.Sprintf("%v", key), abuf); err != nil {
			retErr = err
			return false
		}
		return true
	})
	return
}
