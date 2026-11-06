package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// TOTP parameters matching Misskey/Firefish's otpauth use: SHA-1, six
// digits and a thirty-second step. `window` one step in either direction
// absorbs clock drift between the server and the authenticator app.
const (
	TOTPDigits      = 6
	TOTPStep        = 30 * time.Second
	TOTPSecretBytes = 20
)

// totpWindow is the number of steps accepted on each side of now.
const totpWindow = 1

// GenerateTOTPSecret returns a fresh base32-encoded TOTP secret without
// padding, the shape authenticator apps expect.
func GenerateTOTPSecret() string {
	buf := make([]byte, TOTPSecretBytes)
	if _, err := rand.Read(buf); err != nil {
		panic("auth: crypto/rand failed: " + err.Error())
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf)
}

// TOTPCode returns the six-digit code for a secret at the given time.
func TOTPCode(secret string, t time.Time) (string, error) {
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return "", err
	}
	return totpCodeForCounter(key, uint64(t.Unix())/uint64(TOTPStep/time.Second)), nil
}

// ValidateTOTP reports whether token is the current code (or one step
// either side) for secret. Whitespace is ignored, matching otpauth.
func ValidateTOTP(secret, token string) bool {
	token = strings.ReplaceAll(strings.TrimSpace(token), " ", "")
	if len(token) != TOTPDigits {
		return false
	}
	key, err := decodeTOTPSecret(secret)
	if err != nil {
		return false
	}
	counter := uint64(time.Now().Unix()) / uint64(TOTPStep/time.Second)
	for delta := -totpWindow; delta <= totpWindow; delta++ {
		code := totpCodeForCounter(key, counter+uint64(delta))
		if subtle.ConstantTimeCompare([]byte(code), []byte(token)) == 1 {
			return true
		}
	}
	return false
}

// TOTPURL renders the otpauth:// URI an authenticator app can import.
func TOTPURL(secret, issuer, label string) string {
	v := url.Values{}
	v.Set("secret", secret)
	v.Set("issuer", issuer)
	v.Set("algorithm", "SHA1")
	v.Set("digits", fmt.Sprint(TOTPDigits))
	v.Set("period", fmt.Sprint(int(TOTPStep/time.Second)))
	return "otpauth://totp/" + url.PathEscape(issuer+":"+label) + "?" + v.Encode()
}

func decodeTOTPSecret(secret string) ([]byte, error) {
	s := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	if s == "" {
		return nil, fmt.Errorf("auth: empty TOTP secret")
	}
	if pad := len(s) % 8; pad != 0 {
		s += strings.Repeat("=", 8-pad)
	}
	key, err := base32.StdEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("auth: decode TOTP secret: %w", err)
	}
	return key, nil
}

func totpCodeForCounter(key []byte, counter uint64) string {
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)

	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)

	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	mod := uint32(1)
	for range TOTPDigits {
		mod *= 10
	}
	value %= mod
	return fmt.Sprintf("%0*d", TOTPDigits, value)
}
