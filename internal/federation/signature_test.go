package federation

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// testKeypair returns a keypair shared by the signature tests. Generating
// RSA keys is slow enough that doing it once per test file matters.
func testKeypair(t *testing.T) *Keypair {
	t.Helper()
	k, err := GenerateKeypair()
	if err != nil {
		t.Fatalf("GenerateKeypair: %v", err)
	}
	return k
}

// signedRequest builds and signs a request the way the deliverer does.
func signedRequest(t *testing.T, k *Keypair, keyID, method, target string, body []byte) *http.Request {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, target, rdr)
	if err != nil {
		t.Fatalf("new request: %v", err)
	}
	// httptest.NewRequest derives Host from the URL; mirror that here so
	// the signed host matches what the verifying server will see.
	req.Host = req.URL.Host
	if err := SignRequest(req, body, keyID, privOf(t, k)); err != nil {
		t.Fatalf("sign: %v", err)
	}
	return req
}

func privOf(t *testing.T, k *Keypair) *rsa.PrivateKey {
	t.Helper()
	priv, err := ParsePrivateKeyPEM(k.PrivatePEM)
	if err != nil {
		t.Fatalf("parse private key: %v", err)
	}
	return priv
}

func publicKeyResolver(k *Keypair) func(string) (*rsa.PublicKey, error) {
	return func(keyID string) (*rsa.PublicKey, error) {
		return ParsePublicKeyPEM(k.PublicPEM)
	}
}

// serverRequest converts a client request into what the server side sees:
// the Host header is present and the body is the supplied bytes.
func serverRequest(t *testing.T, req *http.Request, body []byte) *http.Request {
	t.Helper()
	got := httptest.NewRequest(req.Method, req.URL.String(), bytes.NewReader(body))
	got.Host = req.URL.Host
	got.Header.Set("Host", req.Host)
	for k, v := range req.Header {
		got.Header[k] = v
	}
	return got
}

// resign replaces the Signature header after a test mutated the request.
func resign(t *testing.T, req *http.Request, k *Keypair, keyID string, headers []string) {
	t.Helper()
	s, err := signingString(req, headers)
	if err != nil {
		t.Fatalf("signing string: %v", err)
	}
	sum := sha256.Sum256([]byte(s))
	sig, err := rsa.SignPKCS1v15(rand.Reader, privOf(t, k), crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	req.Header.Set("Signature", `keyId="`+keyID+`",algorithm="rsa-sha256",headers="`+
		strings.Join(headers, " ")+`",signature="`+base64.StdEncoding.EncodeToString(sig)+`"`)
}

func TestSignAndVerifyPost(t *testing.T) {
	k := testKeypair(t)
	body := []byte(`{"type":"Create","actor":"https://a.example/users/1"}`)
	req := signedRequest(t, k, "https://a.example/users/1#main-key", http.MethodPost, "https://b.example/inbox", body)

	if got := req.Header.Get("Signature"); got == "" {
		t.Fatal("no Signature header was set")
	}
	if got, want := req.Header.Get("Digest"), DigestHeader(body); got != want {
		t.Errorf("Digest = %q, want %q", got, want)
	}
	if got := req.Header.Get("Content-Type"); got != "application/activity+json" {
		t.Errorf("Content-Type = %q", got)
	}

	srv := serverRequest(t, req, body)
	keyID, err := VerifyRequest(srv, VerifyOptions{
		Host:      "b.example",
		Body:      body,
		PublicKey: publicKeyResolver(k),
	})
	if err != nil {
		t.Fatalf("VerifyRequest: %v", err)
	}
	if keyID != "https://a.example/users/1#main-key" {
		t.Errorf("keyID = %q", keyID)
	}
}

func TestSignAndVerifyGet(t *testing.T) {
	k := testKeypair(t)
	req := signedRequest(t, k, "https://a.example/users/1#main-key", http.MethodGet,
		"https://b.example/users/1?x=1", nil)

	// A GET carries no Digest and must not require one.
	if got := req.Header.Get("Digest"); got != "" {
		t.Errorf("GET set a Digest header: %q", got)
	}
	srv := serverRequest(t, req, nil)
	if _, err := VerifyRequest(srv, VerifyOptions{
		Host:      "b.example",
		PublicKey: publicKeyResolver(k),
	}); err != nil {
		t.Fatalf("VerifyRequest: %v", err)
	}
}

func TestVerifyRejectsTamperedBody(t *testing.T) {
	k := testKeypair(t)
	body := []byte(`{"type":"Create"}`)
	req := signedRequest(t, k, "https://a.example/users/1#main-key", http.MethodPost, "https://b.example/inbox", body)

	tampered := append([]byte{}, body...)
	tampered[0] = '['
	srv := serverRequest(t, req, tampered)
	// The Digest header still describes the original body, so the check
	// must fail before the RSA signature is even considered.
	if _, err := VerifyRequest(srv, VerifyOptions{
		Host:      "b.example",
		Body:      tampered,
		PublicKey: publicKeyResolver(k),
	}); !errors.Is(err, ErrDigestMismatch) {
		t.Fatalf("err = %v, want ErrDigestMismatch", err)
	}
}

func TestVerifyRejectsWrongHost(t *testing.T) {
	k := testKeypair(t)
	body := []byte("{}")
	req := signedRequest(t, k, "https://a.example/users/1#main-key", http.MethodPost, "https://b.example/inbox", body)
	srv := serverRequest(t, req, body)

	// A signature produced for b.example must not verify at c.example,
	// which is what stops cross-instance replay.
	if _, err := VerifyRequest(srv, VerifyOptions{
		Host:      "c.example",
		Body:      body,
		PublicKey: publicKeyResolver(k),
	}); !errors.Is(err, ErrHostMismatch) {
		t.Fatalf("err = %v, want ErrHostMismatch", err)
	}
}

func TestVerifyRejectsMissingSignature(t *testing.T) {
	k := testKeypair(t)
	srv := httptest.NewRequest(http.MethodPost, "https://b.example/inbox", bytes.NewReader(nil))
	srv.Host = "b.example"
	srv.Header.Set("Host", "b.example")

	_, err := VerifyRequest(srv, VerifyOptions{Host: "b.example", PublicKey: publicKeyResolver(k)})
	if !errors.Is(err, ErrNoSignature) {
		t.Fatalf("err = %v, want ErrNoSignature", err)
	}
}

func TestVerifyRejectsDigestNotCovered(t *testing.T) {
	k := testKeypair(t)
	body := []byte("{}")
	req, err := http.NewRequest(http.MethodPost, "https://b.example/inbox", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Host = "b.example"
	req.Header.Set("Digest", DigestHeader(body))
	req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	// A POST signature that leaves the body unbound is what Misskey
	// rejects, so sign only (request-target), date and host.
	resign(t, req, k, "https://a.example/users/1#main-key",
		[]string{"(request-target)", "date", "host"})

	srv := serverRequest(t, req, body)
	if _, err := VerifyRequest(srv, VerifyOptions{
		Host: "b.example", Body: body, PublicKey: publicKeyResolver(k),
	}); !errors.Is(err, ErrMissingHeader) {
		t.Fatalf("err = %v, want ErrMissingHeader", err)
	}
}

func TestVerifyRejectsStaleSignature(t *testing.T) {
	k := testKeypair(t)
	body := []byte("{}")
	req := signedRequest(t, k, "https://a.example/users/1#main-key", http.MethodPost, "https://b.example/inbox", body)
	req.Header.Set("Date", time.Now().Add(-time.Hour).UTC().Format(http.TimeFormat))
	resign(t, req, k, "https://a.example/users/1#main-key", signedHeadersPost)

	srv := serverRequest(t, req, body)
	if _, err := VerifyRequest(srv, VerifyOptions{
		Host: "b.example", Body: body, PublicKey: publicKeyResolver(k),
	}); !errors.Is(err, ErrStaleSignature) {
		t.Fatalf("err = %v, want ErrStaleSignature", err)
	}
}

func TestVerifyAcceptsAuthorizationHeader(t *testing.T) {
	k := testKeypair(t)
	body := []byte("{}")
	req := signedRequest(t, k, "https://a.example/users/1#main-key", http.MethodPost, "https://b.example/inbox", body)
	sig := req.Header.Get("Signature")
	req.Header.Set("Authorization", "Signature "+sig)
	req.Header.Del("Signature")

	srv := serverRequest(t, req, body)
	if _, err := VerifyRequest(srv, VerifyOptions{
		Host: "b.example", Body: body, PublicKey: publicKeyResolver(k),
	}); err != nil {
		t.Fatalf("VerifyRequest with Authorization header: %v", err)
	}
}

func TestParseSignatureHeader(t *testing.T) {
	p, err := ParseSignatureHeader(`keyId="https://a.example/u#main-key",algorithm="rsa-sha256",headers="(request-target) host date",signature="AA=="`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.KeyID != "https://a.example/u#main-key" {
		t.Errorf("keyId = %q", p.KeyID)
	}
	want := []string{"(request-target)", "host", "date"}
	if len(p.Headers) != len(want) {
		t.Fatalf("headers = %v", p.Headers)
	}
	for i := range want {
		if p.Headers[i] != want[i] {
			t.Errorf("headers[%d] = %q, want %q", i, p.Headers[i], want[i])
		}
	}
}

func TestParseSignatureHeaderDefaults(t *testing.T) {
	// A header with no algorithm and no headers list defaults to
	// rsa-sha256 over date only, per the draft.
	p, err := ParseSignatureHeader(`keyId="k",signature="AA=="`)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if p.Algorithm != "rsa-sha256" {
		t.Errorf("algorithm = %q", p.Algorithm)
	}
	if len(p.Headers) != 1 || p.Headers[0] != "date" {
		t.Errorf("headers = %v", p.Headers)
	}
}
