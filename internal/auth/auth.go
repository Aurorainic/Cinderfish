// Package auth implements Cinderfish's account authentication: password
// hashing, token issuance and lookup, and the OAuth 2.0 PKCE helpers.
//
// Behaviour follows misskey-dev/misskey: bcrypt with cost 8, 16-char
// native user tokens, 32-char MiAuth tokens and 128-char OAuth tokens.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const (
	// BcryptCost matches Misskey's bcrypt.genSalt(8).
	BcryptCost = 8

	// NativeTokenLength is the length of a user's own API token.
	NativeTokenLength = 16
	// MiAuthTokenLength is the length of a MiAuth-issued access token.
	MiAuthTokenLength = 32
	// OAuthTokenLength is the length of an OAuth 2.0 access token.
	OAuthTokenLength = 128
	// GrantCodeLength is the length of an OAuth 2.0 authorization code.
	GrantCodeLength = 128
)

const lowerChars = "0123456789abcdefghijklmnopqrstuvwxyz"
const upperChars = lowerChars + "ABCDEFGHIJKLMNOPQRSTUVWXYZ"

// HashPassword returns a bcrypt hash of the given password.
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("auth: password must not be empty")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("auth: hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword reports whether password matches the stored bcrypt hash.
func CheckPassword(hash, password string) bool {
	if hash == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// SecureRndstr returns a random string of the given length drawn from
// the alphanumeric alphabet, mirroring Misskey's secureRndstr.
func SecureRndstr(length int) string {
	return secureRndstr(length, upperChars)
}

// SecureRndstrLower is SecureRndstr restricted to lowercase characters,
// used for invitation codes.
func SecureRndstrLower(length int) string {
	return secureRndstr(length, lowerChars)
}

func secureRndstr(length int, chars string) string {
	n := len(chars)
	out := make([]byte, length)
	for i := range out {
		var b [1]byte
		for {
			if _, err := rand.Read(b[:]); err != nil {
				panic("auth: crypto/rand failed: " + err.Error())
			}
			// Reject values in the biased tail so the distribution is
			// uniform over the alphabet.
			limit := byte(256 - (256 % n))
			if b[0] < limit {
				out[i] = chars[int(b[0])%n]
				break
			}
		}
	}
	return string(out)
}

// GenerateNativeToken returns a user's own API token (16 chars).
func GenerateNativeToken() string {
	return SecureRndstr(NativeTokenLength)
}

// GenerateMiAuthToken returns a MiAuth access token (32 chars).
func GenerateMiAuthToken() string {
	return SecureRndstr(MiAuthTokenLength)
}

// GenerateOAuthToken returns an OAuth 2.0 access token (128 chars).
func GenerateOAuthToken() string {
	return SecureRndstr(OAuthTokenLength)
}

// GenerateGrantCode returns an OAuth 2.0 authorization code (128 chars).
func GenerateGrantCode() string {
	return SecureRndstr(GrantCodeLength)
}

// IsNativeUserToken reports whether the token has the shape of a native
// user token. Misskey decides this purely by length (16).
func IsNativeUserToken(token string) bool {
	return len(token) == NativeTokenLength
}

// HashToken returns the lowercase hex SHA-256 of a token, used as the
// lookup key for access tokens.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

// GenerateCodeChallenge computes the PKCE S256 challenge for a verifier.
func GenerateCodeChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// VerifyCodeChallenge checks a PKCE verifier against a stored challenge.
func VerifyCodeChallenge(verifier, challenge string) bool {
	if verifier == "" || challenge == "" {
		return false
	}
	return subtleEqual(GenerateCodeChallenge(verifier), challenge)
}

func subtleEqual(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range len(a) {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

// ValidateUsername checks a local username against Misskey's schema:
// /^\w{1,20}$/.
func ValidateUsername(username string) error {
	if username == "" {
		return errors.New("auth: username must not be empty")
	}
	if len(username) > 20 {
		return errors.New("auth: username must be at most 20 characters")
	}
	for _, r := range username {
		if !isWordRune(r) {
			return fmt.Errorf("auth: username contains an invalid character %q", r)
		}
	}
	return nil
}

func isWordRune(r rune) bool {
	switch {
	case r >= 'a' && r <= 'z':
		return true
	case r >= 'A' && r <= 'Z':
		return true
	case r >= '0' && r <= '9':
		return true
	case r == '_':
		return true
	default:
		return false
	}
}

// NormalizeUsername lowercases a username for uniqueness checks.
func NormalizeUsername(username string) string {
	return strings.ToLower(username)
}
