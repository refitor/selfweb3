package wasm

import (
	"fmt"
	"strings"

	"github.com/dgryski/dgoogauth"
)

func GetQrcode(user, secret string) string {
	return fmt.Sprintf("otpauth://totp/selfweb3:%s?secret=%s", user, secret)
}

func VerifyCode(secret, code string) (bool, error) {
	otpConfig := &dgoogauth.OTPConfig{
		Secret:      strings.TrimSpace(secret),
		WindowSize:  3,
		HotpCounter: 0,
		// UTC:         true,
	}
	return otpConfig.Authenticate(strings.TrimSpace(code))
}
