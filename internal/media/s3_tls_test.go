package media

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"io"
	"log"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestS3StoragePrivateCA proves the TLS path end to end: a self-signed CA is
// trusted through ca_cert, and Put/Get/Delete work against an HTTPS endpoint.
// Without the CA the same endpoint must be refused.
func TestS3StoragePrivateCA(t *testing.T) {
	caPEM, serverCert := testCA(t)
	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	if err := os.WriteFile(caPath, caPEM, 0o600); err != nil {
		t.Fatalf("write ca: %v", err)
	}

	srv := httptest.NewUnstartedServer(http.HandlerFunc(fakeS3))
	// The untrusted-client leg below is supposed to fail the handshake; keep
	// the server log quiet so a passing run is not full of warnings.
	srv.Config.ErrorLog = log.New(io.Discard, "", 0)
	srv.TLS = &tls.Config{Certificates: []tls.Certificate{serverCert}}
	srv.StartTLS()
	defer srv.Close()

	endpoint := strings.TrimPrefix(srv.URL, "https://")
	storage, err := NewS3Storage(S3Options{
		Endpoint: endpoint, Bucket: "bucket", Region: "us-east-1",
		AccessKey: "access", SecretKey: "secret", UseSSL: true, CACertPath: caPath,
	})
	if err != nil {
		t.Fatalf("storage: %v", err)
	}
	ctx := context.Background()
	body := []byte("hello over tls")
	if _, err := storage.Put(ctx, "ab/cd/key", bytes.NewReader(body), int64(len(body)), "text/plain"); err != nil {
		t.Fatalf("put: %v", err)
	}
	rc, err := storage.Get(ctx, "ab/cd/key")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	got, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !bytes.Equal(got, body) {
		t.Errorf("read %q, want %q", got, body)
	}
	if err := storage.Delete(ctx, "ab/cd/key"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := storage.Get(ctx, "ab/cd/key"); !errors.Is(err, ErrNotFound) {
		t.Errorf("get after delete = %v, want ErrNotFound", err)
	}

	// Without the private CA the certificate must not verify.
	untrusted, err := NewS3Storage(S3Options{
		Endpoint: endpoint, Bucket: "bucket", Region: "us-east-1",
		AccessKey: "access", SecretKey: "secret", UseSSL: true,
	})
	if err != nil {
		t.Fatalf("untrusted storage: %v", err)
	}
	if _, err := untrusted.Put(ctx, "x", bytes.NewReader(body), int64(len(body)), "text/plain"); err == nil {
		t.Fatal("put without the private CA succeeded, want a TLS error")
	}

	// A CA file without a certificate is rejected at construction.
	bad := filepath.Join(dir, "bad.pem")
	if err := os.WriteFile(bad, []byte("not a cert"), 0o600); err != nil {
		t.Fatalf("write bad ca: %v", err)
	}
	if _, err := NewS3Storage(S3Options{
		Endpoint: endpoint, Bucket: "bucket", AccessKey: "a", SecretKey: "s",
		UseSSL: true, CACertPath: bad,
	}); err == nil {
		t.Fatal("invalid CA file accepted")
	}
}

// fakeS3 is the minimal path-style S3 surface the storage implementation uses.
func fakeS3(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("location") != "" {
		w.Header().Set("Content-Type", "application/xml")
		fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/">us-east-1</LocationConstraint>`)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/bucket/")
	switch r.Method {
	case http.MethodPut:
		data, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		sum := md5.Sum(data)
		objects.Store(key, data)
		w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:])+`"`)
		w.WriteHeader(http.StatusOK)
	case http.MethodHead, http.MethodGet:
		stored, ok := objects.Load(key)
		if !ok {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprint(w, `<?xml version="1.0"?><Error><Code>NoSuchKey</Code><Message>missing</Message></Error>`)
			return
		}
		data := stored.([]byte)
		sum := md5.Sum(data)
		w.Header().Set("ETag", `"`+hex.EncodeToString(sum[:])+`"`)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		w.Header().Set("Content-Length", fmt.Sprint(len(data)))
		if r.Method == http.MethodGet {
			_, _ = w.Write(data)
		}
	case http.MethodDelete:
		objects.Delete(key)
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

var objects sync.Map

// testCA builds a one-off CA and a server certificate for 127.0.0.1.
func testCA(t *testing.T) ([]byte, tls.Certificate) {
	t.Helper()
	caKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	now := time.Now()
	caTemplate := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "cinderfish test CA"},
		NotBefore:             now.Add(-time.Hour),
		NotAfter:              now.Add(time.Hour),
		IsCA:                  true,
		BasicConstraintsValid: true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}

	leafKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	leafTemplate := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    now.Add(-time.Hour),
		NotAfter:     now.Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTemplate, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}
	caPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER})
	return caPEM, tls.Certificate{Certificate: [][]byte{leafDER}, PrivateKey: leafKey}
}
