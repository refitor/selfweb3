package rscrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

// aesKey: plain text
// publicKey: pem.memory
// privateKey: pem.memory
// signature: base64.EncodeToString

// =================== ECB ======================
func AesEncryptECB(origData []byte, key []byte) []byte {
	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	//Create a new GCM - https://en.wikipedia.org/wiki/Galois/Counter_Mode
	//https://golang.org/pkg/crypto/cipher/#NewGCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	//Create a nonce. Nonce should be from GCM
	nonce := make([]byte, aesGCM.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}

	//Encrypt the data using aesGCM.Seal
	//Since we don't want to save the nonce somewhere else in this case, we add it as a prefix to the encrypted data. The first nonce argument in Seal is the prefix.
	return aesGCM.Seal(nonce, nonce, origData, nil)
}

func AesDecryptECB(encrypted []byte, key []byte) []byte {
	//Create a new Cipher Block from the key
	block, err := aes.NewCipher(key)
	if err != nil {
		panic(err.Error())
	}

	//Create a new GCM
	aesGCM, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}

	//Get the nonce size
	nonceSize := aesGCM.NonceSize()

	//Extract the nonce from the encrypted data
	nonce, ciphertext := encrypted[:nonceSize], encrypted[nonceSize:]

	//Decrypt the data
	decrypted, err := aesGCM.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return decrypted
}

// =================== ECB ======================

// =================== ecdsa ======================
func GetPublicKey(key string) (*ecdsa.PublicKey, error) {
	pubBuf, err := hexutil.Decode(key)
	if err != nil {
		return nil, fmt.Errorf("GetPublicKey failed at hexutil.Decode, detail: %s", err.Error())
	}
	publicKey, err := crypto.DecompressPubkey(pubBuf)
	if err != nil {
		return nil, fmt.Errorf("GetPublicKey failed at crypto.DecompressPubkey, detail: %s", err.Error())
	}
	return publicKey, nil
}

func GetDhKey(publicKey *ecdsa.PublicKey, privateKey *ecdsa.PrivateKey) (string, error) {
	if publicKey == nil {
		return "", errors.New("GetDhKey failed with invalid public key")
	}
	if privateKey == nil {
		return "", errors.New("GetDhKey failed with invalid private key")
	}

	skLen := 32
	prv := ecies.ImportECDSA(privateKey)
	pub := ecies.ImportECDSAPublic(publicKey)
	if prv.PublicKey.Curve != pub.Curve {
		return "", ecies.ErrInvalidCurve
	}
	if skLen > ecies.MaxSharedKeyLength(pub) {
		return "", ecies.ErrSharedKeyTooBig
	}

	x, _ := pub.Curve.ScalarMult(pub.X, pub.Y, prv.D.Bytes())
	if x == nil {
		return "", ecies.ErrSharedKeyIsPointAtInfinity
	}

	sk := make([]byte, skLen)
	skBytes := x.Bytes()
	copy(sk[len(sk)-len(skBytes):], skBytes)

	var buf bytes.Buffer
	binary.Write(&buf, binary.BigEndian, x.Int64())
	return strings.ToUpper(base32.StdEncoding.EncodeToString(HmacSha1(buf.Bytes(), nil))), nil
}

func HmacSha1(key, data []byte) []byte {
	h := hmac.New(sha1.New, key)
	if total := len(data); total > 0 {
		h.Write(data)
	}
	return h.Sum(nil)
}

// =================== ecdsa ======================

func EthHash(data []byte) common.Hash {
	// msgHash := crypto.Keccak256Hash(data).Bytes()
	return crypto.Keccak256Hash([]byte(fmt.Sprintf("\x19Ethereum Signed Message:\n%d%s", len(data), string(data))))
}
