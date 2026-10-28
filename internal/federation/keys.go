// Package federation implements MilnFish's ActivityPub server and
// client: the WebFinger/Actor/Inbox/Outbox surface, HTTP Signatures, the
// three-mode federation policy, and the deliver queue.
//
// The authoritative contract is misskey-dev/misskey; the specific
// behaviours adopted here are recorded in
// docs/research/2026-10-28-m5-federation-source-calibration.md.
package federation

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
)

// KeySize is the RSA modulus size used for actor keypairs. 2048 bits is
// what Mastodon, Misskey and GoToSocial all ship and what every peer
// accepts; 4096 measurably slows the handshake on small instances
// without buying anything a personal instance needs.
const KeySize = 2048

// Keypair is an actor's RSA keypair in PEM form, the representation the
// users table stores and the actor document publishes.
type Keypair struct {
	PublicPEM  string
	PrivatePEM string
	private    *rsa.PrivateKey
}

// GenerateKeypair creates a new RSA keypair.
func GenerateKeypair() (*Keypair, error) {
	priv, err := rsa.GenerateKey(rand.Reader, KeySize)
	if err != nil {
		return nil, fmt.Errorf("federation: generate key: %w", err)
	}
	return encodeKeypair(priv)
}

func encodeKeypair(priv *rsa.PrivateKey) (*Keypair, error) {
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("federation: marshal private key: %w", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(&priv.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("federation: marshal public key: %w", err)
	}
	return &Keypair{
		PublicPEM:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})),
		PrivatePEM: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})),
		private:    priv,
	}, nil
}

// privateKey returns the parsed private key, parsing the PEM on first
// use so keypairs read back from the database behave like fresh ones.
func (k *Keypair) privateKey() (*rsa.PrivateKey, error) {
	if k.private != nil {
		return k.private, nil
	}
	priv, err := ParsePrivateKeyPEM(k.PrivatePEM)
	if err != nil {
		return nil, err
	}
	k.private = priv
	return priv, nil
}

// ParsePrivateKeyPEM parses a PKCS#8 or PKCS#1 RSA private key.
func ParsePrivateKeyPEM(s string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		return nil, fmt.Errorf("federation: private key is not PEM")
	}
	if k, err := x509.ParsePKCS8PrivateKey(block.Bytes); err == nil {
		priv, ok := k.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("federation: private key is not RSA")
		}
		return priv, nil
	}
	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("federation: parse private key: %w", err)
	}
	return priv, nil
}

// ParsePublicKeyPEM parses a PKIX (SPKI) or PKCS#1 RSA public key, which
// are the two forms peers publish in their actor documents.
func ParsePublicKeyPEM(s string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(s))
	if block == nil {
		// Some implementations publish the bare base64 body without the
		// PEM armour; re-wrap it before giving up.
		wrapped := "-----BEGIN PUBLIC KEY-----\n" + s + "\n-----END PUBLIC KEY-----\n"
		block, _ = pem.Decode([]byte(wrapped))
		if block == nil {
			return nil, fmt.Errorf("federation: public key is not PEM")
		}
	}
	if pub, err := x509.ParsePKIXPublicKey(block.Bytes); err == nil {
		rsaPub, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("federation: public key is not RSA")
		}
		return rsaPub, nil
	}
	pub, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("federation: parse public key: %w", err)
	}
	return pub, nil
}
