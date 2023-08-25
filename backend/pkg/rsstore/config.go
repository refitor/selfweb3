package rsstore

import (
	"fmt"
	"sync"
)

var vConfig sync.Map

func Config() *sync.Map {
	return &vConfig
}

func SetConf(key, val string) {
	vConfig.Store(key, val)
}

func GetConf(key string) string {
	val, _ := vConfig.Load(key)
	return fmt.Sprintf("%v", val)
}
