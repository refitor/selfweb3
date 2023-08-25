package rsstore

import (
	"fmt"
	"sync"
	"time"
)

var vCache sync.Map

func Cache() *sync.Map {
	return &vCache
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

// delete: beforeDelleteFunc return true
func GetCache(key string, bDelete bool, beforeDelleteFunc func(v interface{}) bool) interface{} {
	val, _ := vCache.Load(key)
	if bDelete {
		if beforeDelleteFunc != nil && !beforeDelleteFunc(val) {
			return val
		}
		vCache.Delete(key)
	}
	return val
}

func SetCacheByTime(key string, val interface{}, bForce bool, timeout time.Duration, callback func(key, val any) bool) error {
	if !bForce {
		if _, ok := vCache.Load(key); ok {
			return fmt.Errorf("cache data already exists, key: %v", key)
		}
	}
	vCache.Store(key, val)

	if timeout > 0 {
		go autoClearByTimer(key, val, timeout, callback)
	}
	return nil
}

// timeUnit: second
func autoClearByTimer(key, val any, timeout time.Duration, callback func(key, val any) bool) {
	timer := time.NewTimer(timeout * time.Second)
	for {
		select {
		case <-timer.C:
			if callback != nil {
				if callback(key, val) {
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
