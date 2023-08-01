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
	"time"

	"selfweb3/common/rsauth"
	"selfweb3/common/rsweb"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/refitor/rslog"
)

// global const
const (
	c_name_user = "user"

	C_date_time = "2006-01-02 15:04:05"

	C_Url_host = "https://selfrscrypto.refitor.com"
)

var vserver *Server

// flag
var (
	DBPath  = flag.String("dbpath", "./selfweb3.db", "--dbPath=./selfweb3.db")
	webPort = flag.String("port", "3157", "--port=3157")
	webPath = flag.String("webpath", "rsweb", "--webpath=rsweb")
)

func Run(ctx context.Context, fs *embed.FS) {
	// init
	vserver = New()
	vserver.Init()
	defer vserver.UnInit()

	// run
	go rsweb.Run(ctx, *webPort, rsweb.Init(*webPath, fs, AuthInitRouter), false, "http://localhost:8000", "http://localhost:3157", "https://*.refitor.com")
}

type Server struct {
	db      *db_bolt
	cache   sync.Map
	config  sync.Map
	memvar  sync.Map
	public  *ecdsa.PublicKey
	private *ecdsa.PrivateKey
}

func (p *Server) Init() {
	rslog.SetLevel("info")

	// db
	db, err := boltDBInit(*DBPath)
	vserver.FatalCheck(err)
	vserver.db = db
	vserver.FatalCheck(vserver.db.DBCreate(c_name_user))

	rsauth.InitEmail("smtp.126.com:465", "refitor@126.com", "ZPHRFSXTEQUFNYLB")
}

func (p *Server) UnInit() {
}

func New() *Server {
	s := new(Server)
	private, ecdsaErr := crypto.GenerateKey()
	s.FatalCheck(ecdsaErr)
	s.private = private
	s.public = &private.PublicKey
	return s
}

func (p *Server) SetVar(key string, val interface{}, bForce bool) error {
	if !bForce {
		if _, ok := p.memvar.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	p.memvar.Store(key, val)
	return nil
}

// delete: beforeDelleteFunc return true
func (p *Server) GetVar(key string, bDelete bool, beforeDelleteFunc func(v interface{}) bool) interface{} {
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

func (p *Server) SetCache(key string, val interface{}, bForce bool) error {
	if !bForce {
		if _, ok := p.cache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	p.cache.Store(key, val)
	return nil
}

func (p *Server) SetCacheByTime(key string, val interface{}, bForce bool, timeout time.Duration, callback func(string) bool) error {
	if !bForce {
		if _, ok := p.cache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}

	p.cache.Store(key, val)

	if timeout > 0 {
		go p.autoClearByTimer(key, timeout, callback)
	}
	return nil
}

// timeUnit: second
func (p *Server) autoClearByTimer(key string, timeout time.Duration, callback func(string) bool) {
	timer := time.NewTimer(timeout * time.Second)
	for {
		select {
		case <-timer.C:
			if callback != nil {
				if callback(key) {
					timer.Stop()
					p.cache.Delete(key)
				} else {
					timer.Reset(timeout * time.Second)
					break
				}
			} else {
				timer.Stop()
				p.cache.Delete(key)
			}
			return
		}
	}
}

// delete: beforeDelleteFunc return true
func (p *Server) GetCache(key string, bDelete bool, beforeDelleteFunc func(v interface{}) bool) interface{} {
	val, _ := p.cache.Load(key)
	if beforeDelleteFunc != nil {
		if beforeDelleteFunc(val) {
			p.cache.Delete(key)
		}
	} else if bDelete {
		p.cache.Delete(key)
	}
	return val
}

func (p *Server) SetConf(key, val string) {
	p.config.Store(key, val)
}

func (p *Server) GetConf(key string) string {
	val, _ := p.config.Load(key)
	return fmt.Sprintf("%v", val)
}

func (p *Server) FatalCheck(err error) {
	if err != nil {
		log.Fatal(err.Error())
	}
}

func (p *Server) initFromDB() (retErr error) {
	p.db.DBRange("user", func(k string, v interface{}) bool {
		dbUser := &AuthUser{}
		if err := json.Unmarshal(v.([]byte), &dbUser); err != nil {
			retErr = err
			return false
		}

		p.SetCacheByTime(fmt.Sprintf("user-%v", k), dbUser, true, 900, func(s string) bool {
			rslog.Infof("before delete user at memory, authID: %v", k)
			if memUser, ok := vserver.GetCache(s, false, nil).(*AuthUser); ok && memUser != nil && time.Since(memUser.ActiveTime).Seconds() < 900 {
				return false
			} else {
				rslog.Infof("reset timer for user at memory, time-offset: %.2f", time.Since(memUser.ActiveTime).Seconds())
			}
			return true
		})
		return true
	})
	return nil
}

func (p *Server) SaveToDB() (retErr error) {
	p.cache.Range(func(key, value any) bool {
		abuf, err := json.Marshal(value)
		if err != nil {
			retErr = err
			return false
		}
		if err := vserver.db.DBPut(c_name_user, fmt.Sprintf("%v", key), abuf); err != nil {
			retErr = err
			return false
		}
		return true
	})
	return
}
