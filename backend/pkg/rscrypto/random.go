package rscrypto

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strconv"
	"time"
)

func GetRandomInt(n int) (int64, error) {
	if v, err := strconv.ParseInt(GetRandom(n, true), 10, 64); err != nil {
		return 0, err
	} else {
		return v, nil
	}
}

func getRandomInt(max *big.Int) (int, error) {
	if max == nil {
		max = big.NewInt(time.Now().UnixNano())
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
		index, err := getRandomInt(max)
		if err != nil {
			return ""
		}
		buffer[i] = alphanum[index]
	}
	return string(buffer)
}
