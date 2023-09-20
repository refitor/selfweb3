package wasm

import (
	"crypto/ecdsa"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
)

var (
	vWorker *Worker
)

type Worker struct {
	cache         sync.Map
	public        *ecdsa.PublicKey
	private       *ecdsa.PrivateKey
	web2NetPublic *ecdsa.PublicKey
}

func Init() *Worker {
	vWorker = newWorker()
	return vWorker
}

func newWorker() *Worker {
	// rslog.SetLevel("debug")
	// rslog.SetDepth(6)
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

func SetCache(key string, val interface{}, bForce bool) error {
	if !bForce {
		if _, ok := vWorker.cache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	vWorker.cache.Store(key, val)
	return nil
}

// delete: beforeDelleteFunc return true
func GetCache(key string, bDelete bool, beforeDelleteFunc func(v interface{}) bool) interface{} {
	val, _ := vWorker.cache.Load(key)
	if bDelete {
		if beforeDelleteFunc != nil && !beforeDelleteFunc(val) {
			return val
		}
		vWorker.cache.Delete(key)
	}
	return val
}

func SetCacheByTime(key string, val interface{}, bForce bool, timeout time.Duration, callback func(string) bool) error {
	if !bForce {
		if _, ok := vWorker.cache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	vWorker.cache.Store(key, val)

	if timeout > 0 {
		go autoClearByTimer(key, timeout, callback)
	}
	return nil
}

// timeUnit: second
func autoClearByTimer(key string, timeout time.Duration, callback func(string) bool) {
	timer := time.NewTimer(timeout * time.Second)
	for {
		select {
		case <-timer.C:
			if callback != nil {
				if callback(key) {
					timer.Stop()
					vWorker.cache.Delete(key)
				} else {
					timer.Reset(timeout * time.Second)
					break
				}
			} else {
				timer.Stop()
				vWorker.cache.Delete(key)
			}
			return
		}
	}
}

func Str(data any) string {
	return fmt.Sprintf("%v", data)
}

func WebError(err error, webErr string) string {
	logid := time.Now().UnixNano()
	if webErr == "" {
		webErr = "system processing exception"
	}
	if err != nil {
		LogDebugf("%v-%s", logid, err.Error())
	}
	return fmt.Sprintf("%v-%s", logid, webErr)
}

func LogDebugln(datas ...interface{}) {
	// rslog.Debug(datas...)
}

func LogDebugf(format string, datas ...interface{}) {
	// rslog.Debugf(format, datas...)
}
