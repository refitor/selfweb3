package rscrypto

import (
	"bytes"
	"crypto/aes"
	"crypto/ecdsa"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/crypto/ecies"
)

// aesKey: plain text
// publicKey: pem.memory
// privateKey: pem.memory
// signature: base64.EncodeToString

// =================== ECB ======================
func GenerateAesKey(data string) string {
	runesRandom := []rune(data)
	if len(runesRandom) < 32 {
		for i := 0; i < 32; i++ {
			data += "0"
		}
	}
	return data[:31]
}

func AesEncryptECB(origData []byte, key []byte) (encrypted []byte) {
	cipher, _ := aes.NewCipher(generateKey(key))
	length := (len(origData) + aes.BlockSize) / aes.BlockSize
	plain := make([]byte, length*aes.BlockSize)
	copy(plain, origData)
	pad := byte(len(plain) - len(origData))
	for i := len(origData); i < len(plain); i++ {
		plain[i] = pad
	}
	encrypted = make([]byte, len(plain))
	// 分组分块加密
	for bs, be := 0, cipher.BlockSize(); bs <= len(origData); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Encrypt(encrypted[bs:be], plain[bs:be])
	}
	return encrypted
}

func AesDecryptECB(encrypted []byte, key []byte) (decrypted []byte) {
	cipher, _ := aes.NewCipher(generateKey(key))
	decrypted = make([]byte, len(encrypted))
	//
	for bs, be := 0, cipher.BlockSize(); bs < len(encrypted); bs, be = bs+cipher.BlockSize(), be+cipher.BlockSize() {
		cipher.Decrypt(decrypted[bs:be], encrypted[bs:be])
	}

	trim := 0
	if len(decrypted) > 0 {
		trim = len(decrypted) - int(decrypted[len(decrypted)-1])
	}

	return decrypted[:trim]
}

func generateKey(key []byte) (genKey []byte) {
	genKey = make([]byte, 16)
	copy(genKey, key)
	for i := 16; i < len(key); {
		for j := 0; j < 16 && i < len(key); j, i = j+1, i+1 {
			genKey[j] ^= key[i]
		}
	}
	return genKey
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
