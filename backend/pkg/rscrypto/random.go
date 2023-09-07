package rscrypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"time"
)

func GetRandomInt(max *big.Int) (int, error) {
	if max == nil {
		seed := "0123456789"
		alphanum := seed + fmt.Sprintf("%v", time.Now().UnixNano())
		max = big.NewInt(int64(len(alphanum)))
	}
	vrand, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	return int(vrand.Int64()), nil
}

func GetRandom(n int, isNO bool) string {
	seed := "0123456789"
	if !isNO {
		seed = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"
	}
	alphanum := seed + fmt.Sprintf("%v", time.Now().UnixNano())
	buffer := make([]byte, n)
	max := big.NewInt(int64(len(alphanum)))

	for i := 0; i < n; i++ {
		index, err := GetRandomInt(max)
		if err != nil {
			return ""
		}
		buffer[i] = alphanum[index]
	}
	return string(buffer)
}
