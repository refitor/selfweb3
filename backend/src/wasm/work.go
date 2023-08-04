package wasm

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/refitor/rslog"
)

// global const
const (
	C_date_time = "2006-01-02 15:04:05"
)

var (
	vWorker *Worker
)

type Worker struct {
	config  sync.Map
	memvar  sync.Map
	public  *ecdsa.PublicKey
	private *ecdsa.PrivateKey
}

func Init() *Worker {
	vWorker = newWorker()
	rslog.SetLevel("debug")
	return vWorker
}

func UnInit() {
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
