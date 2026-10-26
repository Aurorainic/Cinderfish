package auth

import (
	"strings"
	"testing"
)

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("hunter2")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if !CheckPassword(hash, "hunter2") {
		t.Error("CheckPassword rejected the correct password")
	}
	if CheckPassword(hash, "hunter3") {
		t.Error("CheckPassword accepted a wrong password")
	}
	if CheckPassword("", "hunter2") {
		t.Error("CheckPassword accepted an empty hash")
	}
}

func TestHashPasswordRejectsEmpty(t *testing.T) {
	if _, err := HashPassword(""); err == nil {
		t.Error("HashPassword should reject an empty password")
	}
}

func TestTokenLengths(t *testing.T) {
	if got := len(GenerateNativeToken()); got != NativeTokenLength {
		t.Errorf("native token length = %d, want %d", got, NativeTokenLength)
	}
	if got := len(GenerateMiAuthToken()); got != MiAuthTokenLength {
		t.Errorf("miauth token length = %d, want %d", got, MiAuthTokenLength)
	}
	if got := len(GenerateOAuthToken()); got != OAuthTokenLength {
		t.Errorf("oauth token length = %d, want %d", got, OAuthTokenLength)
	}
	if got := len(GenerateGrantCode()); got != GrantCodeLength {
		t.Errorf("grant code length = %d, want %d", got, GrantCodeLength)
	}
}

func TestIsNativeUserToken(t *testing.T) {
	if !IsNativeUserToken(GenerateNativeToken()) {
		t.Error("a generated native token must be recognised")
	}
	if IsNativeUserToken(GenerateMiAuthToken()) {
		t.Error("a 32-char token must not be treated as native")
	}
	if IsNativeUserToken("") {
		t.Error("an empty token must not be native")
	}
}

func TestSecureRndstrUniqueness(t *testing.T) {
	seen := make(map[string]bool, 500)
	for range 500 {
		s := SecureRndstr(32)
		if len(s) != 32 {
			t.Fatalf("length = %d, want 32", len(s))
		}
		if seen[s] {
			t.Fatalf("duplicate token %q", s)
		}
		seen[s] = true
	}
}

func TestSecureRndstrLowerAlphabet(t *testing.T) {
	s := SecureRndstrLower(64)
	for _, r := range s {
		if !strings.ContainsRune(lowerChars, r) {
			t.Fatalf("unexpected character %q in lowercase token", r)
		}
	}
}

func TestHashTokenIsStableAndLowercase(t *testing.T) {
	h1 := HashToken("AbC")
	h2 := HashToken("AbC")
	if h1 != h2 {
		t.Error("HashToken must be stable")
	}
	if h1 != strings.ToLower(h1) {
		t.Error("HashToken must be lowercase")
	}
	if HashToken("AbC") == HashToken("abc") {
		t.Error("HashToken must distinguish case")
	}
}

func TestPKCEChallenge(t *testing.T) {
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	challenge := GenerateCodeChallenge(verifier)

	// The example from RFC 7636 appendix B.
	const want = "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if challenge != want {
		t.Errorf("challenge = %q, want %q", challenge, want)
	}
	if !VerifyCodeChallenge(verifier, challenge) {
		t.Error("VerifyCodeChallenge rejected a valid verifier")
	}
	if VerifyCodeChallenge("wrong-verifier", challenge) {
		t.Error("VerifyCodeChallenge accepted a wrong verifier")
	}
	if VerifyCodeChallenge("", challenge) {
		t.Error("VerifyCodeChallenge accepted an empty verifier")
	}
}

func TestValidateUsername(t *testing.T) {
	valid := []string{"a", "alice", "Alice_1", "user_2000", strings.Repeat("x", 20)}
	for _, u := range valid {
		if err := ValidateUsername(u); err != nil {
			t.Errorf("ValidateUsername(%q) = %v, want nil", u, err)
		}
	}

	invalid := []string{"", strings.Repeat("x", 21), "has space", "emoji🙂", "dash-name", "dot.name"}
	for _, u := range invalid {
		if err := ValidateUsername(u); err == nil {
			t.Errorf("ValidateUsername(%q) = nil, want an error", u)
		}
	}
}
