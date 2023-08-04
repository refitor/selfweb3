package wasm

import (
	"fmt"
	"sync"
	"time"
)

var (
	vCache sync.Map
	vStore sync.Map
)

func SaveToStoreage(data any, bFormat bool) error {
	return nil
}

func LoadFromStorage() ([]byte, error) {
	return nil, nil
}

func SetCache(key string, val interface{}, bForce bool) error {
	if !bForce {
		if _, ok := vCache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	vCache.Store(key, val)
	return nil
}

func SetCacheByTime(key string, val interface{}, bForce bool, timeout time.Duration, callback func(string) bool) error {
	if !bForce {
		if _, ok := vCache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}

	vCache.Store(key, val)

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
					vCache.Delete(key)
				} else {
					timer.Reset(timeout * time.Second)
					break
				}
			} else {
				timer.Stop()
				vCache.Delete(key)
			}
			return
		}
	}
}

// delete: beforeDelleteFunc return true
func GetCache(key string, bDelete bool, beforeDelleteFunc func(v interface{}) bool) interface{} {
	val, _ := vCache.Load(key)
	if beforeDelleteFunc != nil {
		if beforeDelleteFunc(val) {
			vCache.Delete(key)
		}
	} else if bDelete {
		vCache.Delete(key)
	}
	return val
}
