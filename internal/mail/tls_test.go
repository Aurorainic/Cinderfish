package mail_test

import (
	"bufio"
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Aurorainic/cinderfish/internal/mail"
)

// testTLSConfig builds a CA-trusted certificate for 127.0.0.1 and points Go's
// root pool at the CA through SSL_CERT_FILE.
func testTLSConfig(t *testing.T) *tls.Config {
	t.Helper()
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("ca key: %v", err)
	}
	caTmpl := &x509.Certificate{
		SerialNumber:          big.NewInt(1),
		Subject:               pkix.Name{CommonName: "test CA"},
		NotBefore:             time.Now().Add(-time.Hour),
		NotAfter:              time.Now().Add(time.Hour),
		IsCA:                  true,
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageDigitalSignature,
		BasicConstraintsValid: true,
	}
	caDER, err := x509.CreateCertificate(rand.Reader, caTmpl, caTmpl, &caKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("ca cert: %v", err)
	}
	caCert, err := x509.ParseCertificate(caDER)
	if err != nil {
		t.Fatalf("parse ca: %v", err)
	}

	leafKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatalf("leaf key: %v", err)
	}
	leafTmpl := &x509.Certificate{
		SerialNumber: big.NewInt(2),
		Subject:      pkix.Name{CommonName: "127.0.0.1"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	leafDER, err := x509.CreateCertificate(rand.Reader, leafTmpl, caCert, &leafKey.PublicKey, caKey)
	if err != nil {
		t.Fatalf("leaf cert: %v", err)
	}

	caPath := filepath.Join(t.TempDir(), "ca.pem")
	if err := os.WriteFile(caPath, pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: caDER}), 0o600); err != nil {
		t.Fatalf("write ca: %v", err)
	}
	t.Setenv("SSL_CERT_FILE", caPath)

	return &tls.Config{
		Certificates: []tls.Certificate{{
			Certificate: [][]byte{leafDER},
			PrivateKey:  leafKey,
		}},
	}
}

// serveSMTP runs one SMTP session over conn (plain or already TLS) and reports
// the DATA it received. Advertise STARTTLS and AUTH PLAIN as requested.
func serveSMTP(t *testing.T, conn net.Conn, cfg *tls.Config, startTLS, auth bool, msgs chan<- string) {
	t.Helper()
	defer conn.Close()
	var r *bufio.Reader
	var w *bufio.Writer
	reset := func(c net.Conn) {
		r = bufio.NewReader(c)
		w = bufio.NewWriter(c)
	}
	reset(conn)
	write := func(s string) { _, _ = w.WriteString(s); _ = w.Flush() }

	write("220 test ESMTP\r\n")
	var data strings.Builder
	inData := false
	for {
		line, err := r.ReadString('\n')
		if err != nil {
			return
		}
		if inData {
			if line == ".\r\n" {
				inData = false
				msgs <- data.String()
				write("250 ok\r\n")
				continue
			}
			data.WriteString(line)
			continue
		}
		upper := strings.ToUpper(line)
		switch {
		case strings.HasPrefix(upper, "EHLO"), strings.HasPrefix(upper, "HELO"):
			var b strings.Builder
			b.WriteString("250-test\r\n")
			if startTLS {
				b.WriteString("250-STARTTLS\r\n")
			}
			if auth {
				b.WriteString("250-AUTH PLAIN\r\n")
			}
			b.WriteString("250 HELP\r\n")
			write(b.String())
		case strings.HasPrefix(upper, "STARTTLS"):
			write("220 go ahead\r\n")
			tlsConn := tls.Server(conn, cfg)
			if err := tlsConn.Handshake(); err != nil {
				return
			}
			conn = tlsConn
			reset(conn)
		case strings.HasPrefix(upper, "AUTH"):
			write("235 authenticated\r\n")
		case strings.HasPrefix(upper, "MAIL FROM"), strings.HasPrefix(upper, "RCPT TO"):
			write("250 ok\r\n")
		case strings.HasPrefix(upper, "DATA"):
			inData = true
			write("354 go ahead\r\n")
		case strings.HasPrefix(upper, "QUIT"):
			write("221 bye\r\n")
			return
		default:
			write("250 ok\r\n")
		}
	}
}

// TestSMTPOverTLS covers both TLS modes and authentication: STARTTLS with a
// PLAIN login, and implicit TLS from the first byte.
func TestSMTPOverTLS(t *testing.T) {
	cfg := testTLSConfig(t)

	cases := []struct {
		name     string
		implicit bool
	}{
		{"starttls", false},
		{"implicit", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ln, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatalf("listen: %v", err)
			}
			t.Cleanup(func() { ln.Close() })
			if tc.implicit {
				ln = tls.NewListener(ln, cfg)
			}
			msgs := make(chan string, 1)
			go func() {
				conn, err := ln.Accept()
				if err != nil {
					return
				}
				serveSMTP(t, conn, cfg, !tc.implicit, true, msgs)
			}()

			host, port, _ := net.SplitHostPort(ln.Addr().String())
			var portNum int
			fmt.Sscanf(port, "%d", &portNum)
			mode := "starttls"
			if tc.implicit {
				mode = "tls"
			}
			sender := mail.New(mail.Settings{
				Host: host, Port: portNum, Username: "user", Password: "pass",
				From: "no-reply@fish.example", TLS: mode,
			})
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := sender.Send(ctx, mail.Message{
				To: "alice@fish.example", Subject: "hello", Body: "over tls\n",
			}); err != nil {
				t.Fatalf("send: %v", err)
			}
			select {
			case raw := <-msgs:
				if !strings.Contains(raw, "over tls") || !strings.Contains(raw, "To: alice@fish.example") {
					t.Errorf("message = %q", raw)
				}
			case <-time.After(3 * time.Second):
				t.Fatal("no message delivered")
			}
		})
	}
}
