package federation

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Errors returned by signature verification. The inbox handler maps all
// of them to the same 401 so peers cannot distinguish the reasons, but
// tests and logs can.
var (
	ErrNoSignature      = errors.New("federation: request has no Signature header")
	ErrMalformedSig     = errors.New("federation: malformed Signature header")
	ErrMissingHeader    = errors.New("federation: signature does not cover a required header")
	ErrHostMismatch     = errors.New("federation: Host header does not match this instance")
	ErrDigestMismatch   = errors.New("federation: digest does not match the body")
	ErrBadDigest        = errors.New("federation: malformed or unsupported Digest header")
	ErrUnsupportedAlgo  = errors.New("federation: unsupported signature algorithm")
	ErrSignatureInvalid = errors.New("federation: signature does not verify")
	ErrStaleSignature   = errors.New("federation: signature date is outside the accepted window")
)

// signedHeadersPost is the header set Misskey signs for outbound POSTs,
// in the order it lists them (ApRequestService.createSignedPost).
var signedHeadersPost = []string{"(request-target)", "date", "host", "digest"}

// signedHeadersGet is the header set for outbound GETs.
var signedHeadersGet = []string{"(request-target)", "date", "host"}

// maxSignatureAge bounds how stale a signed request may be, and how far
// into the future its Date may sit. Peers with badly skewed clocks are
// rare enough that rejecting them is better than accepting replays.
const maxSignatureAge = 5 * time.Minute

// SignatureParams is a parsed Signature header.
type SignatureParams struct {
	KeyID     string
	Algorithm string
	Headers   []string
	Signature []byte
}

// DigestHeader returns the Digest header value for body.
func DigestHeader(body []byte) string {
	sum := sha256.Sum256(body)
	return "SHA-256=" + base64.StdEncoding.EncodeToString(sum[:])
}

// SignRequest signs req in place for an outbound request, setting the
// Date, Digest (for POST) and Signature headers. keyID is the actor's
// key id, the URL ending in #main-key.
//
// The signed header list matches Misskey exactly: (request-target) date
// host digest for POST, (request-target) date host for GET.
func SignRequest(req *http.Request, body []byte, keyID string, priv *rsa.PrivateKey) error {
	if req.Header.Get("Date") == "" {
		req.Header.Set("Date", time.Now().UTC().Format(http.TimeFormat))
	}

	headers := signedHeadersGet
	if req.Method == http.MethodPost {
		headers = signedHeadersPost
		if body != nil {
			req.Header.Set("Digest", DigestHeader(body))
		}
		if req.Header.Get("Content-Type") == "" {
			req.Header.Set("Content-Type", "application/activity+json")
		}
	}

	signingString, err := signingString(req, headers)
	if err != nil {
		return err
	}
	sum := sha256.Sum256([]byte(signingString))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		return fmt.Errorf("federation: sign: %w", err)
	}

	req.Header.Set("Signature", fmt.Sprintf(
		`keyId="%s",algorithm="rsa-sha256",headers="%s",signature="%s"`,
		keyID, strings.Join(headers, " "), base64.StdEncoding.EncodeToString(sig),
	))
	return nil
}

// signingString builds the draft-cavage signing string for the given
// header names, in order.
func signingString(req *http.Request, headers []string) (string, error) {
	lines := make([]string, 0, len(headers))
	for _, name := range headers {
		switch name {
		case "(request-target)":
			target := req.URL.EscapedPath()
			if target == "" {
				target = "/"
			}
			if req.URL.RawQuery != "" {
				target += "?" + req.URL.RawQuery
			}
			lines = append(lines, "(request-target): "+strings.ToLower(req.Method)+" "+target)
		case "host":
			// The Host header is what the peer will see, which for an
			// outbound request is URL.Host unless it was set explicitly.
			host := req.Host
			if host == "" {
				host = req.URL.Host
			}
			lines = append(lines, "host: "+host)
		case "date":
			lines = append(lines, "date: "+req.Header.Get("Date"))
		case "digest":
			lines = append(lines, "digest: "+req.Header.Get("Digest"))
		default:
			lines = append(lines, name+": "+req.Header.Get(name))
		}
	}
	return strings.Join(lines, "\n"), nil
}

// ParseSignatureHeader parses a Signature or Authorization: Signature
// header value into its parameters.
func ParseSignatureHeader(v string) (*SignatureParams, error) {
	params := map[string]string{}
	for _, part := range splitSignatureParams(v) {
		eq := strings.Index(part, "=")
		if eq < 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(part[:eq]))
		val := strings.TrimSpace(part[eq+1:])
		val = strings.Trim(val, `"`)
		params[key] = val
	}

	p := &SignatureParams{
		KeyID:     params["keyid"],
		Algorithm: params["algorithm"],
	}
	if p.KeyID == "" || params["signature"] == "" {
		return nil, ErrMalformedSig
	}
	if h, ok := params["headers"]; ok && h != "" {
		p.Headers = strings.Fields(strings.ToLower(h))
	} else {
		// The draft says a missing headers parameter means the Date
		// header only.
		p.Headers = []string{"date"}
	}
	if p.Algorithm == "" {
		p.Algorithm = "rsa-sha256"
	}
	sig, err := base64.StdEncoding.DecodeString(params["signature"])
	if err != nil {
		return nil, ErrMalformedSig
	}
	p.Signature = sig
	return p, nil
}

// splitSignatureParams splits a signature header on commas, respecting
// quoted values that may themselves contain commas.
func splitSignatureParams(v string) []string {
	var out []string
	var cur strings.Builder
	inQuote := false
	for _, r := range v {
		switch {
		case r == '"':
			inQuote = !inQuote
			cur.WriteRune(r)
		case r == ',' && !inQuote:
			out = append(out, strings.TrimSpace(cur.String()))
			cur.Reset()
		default:
			cur.WriteRune(r)
		}
	}
	if s := strings.TrimSpace(cur.String()); s != "" {
		out = append(out, s)
	}
	return out
}

// VerifyOptions carries what the inbox needs to check a request.
type VerifyOptions struct {
	// Host is this instance's hostname; the signed Host header must
	// equal it, which is what binds the signature to this server.
	Host string
	// Body is the raw request body, used for the Digest check.
	Body []byte
	// PublicKey resolves a keyId to the peer's RSA public key.
	PublicKey func(keyID string) (*rsa.PublicKey, error)
	// Now overrides the clock in tests.
	Now func() time.Time
}

// VerifyRequest checks the Digest and HTTP Signature of an inbound
// request and returns the keyId that signed it.
//
// The checks and their order follow Misskey's inbox handler: a missing
// or malformed signature is rejected, the Host and Digest headers must be
// covered by the signature, the Digest must match the body, and only then
// is the RSA signature verified.
func VerifyRequest(req *http.Request, opts VerifyOptions) (string, error) {
	header := req.Header.Get("Signature")
	if header == "" {
		// Misskey's parser also accepts the Authorization form.
		if auth := req.Header.Get("Authorization"); len(auth) > len("Signature ") &&
			strings.EqualFold(auth[:len("Signature ")], "Signature ") {
			header = auth[len("Signature "):]
		}
	}
	if header == "" {
		return "", ErrNoSignature
	}

	params, err := ParseSignatureHeader(header)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(params.Algorithm, "rsa-sha256") {
		return "", ErrUnsupportedAlgo
	}

	covered := map[string]bool{}
	for _, h := range params.Headers {
		covered[h] = true
	}
	if !covered["host"] {
		return "", fmt.Errorf("%w: host", ErrMissingHeader)
	}
	if !covered["date"] {
		return "", fmt.Errorf("%w: date", ErrMissingHeader)
	}
	if req.Method == http.MethodPost && !covered["digest"] {
		return "", fmt.Errorf("%w: digest", ErrMissingHeader)
	}

	// The signed host must be this instance: otherwise a signature
	// captured by a peer could be replayed against us.
	if opts.Host != "" {
		realHost := req.Header.Get("Host")
		if realHost == "" {
			realHost = req.Host
		}
		if !strings.EqualFold(realHost, opts.Host) {
			return "", ErrHostMismatch
		}
	}

	if req.Method == http.MethodPost {
		if err := verifyDigest(req.Header.Get("Digest"), opts.Body); err != nil {
			return "", err
		}
	}

	now := time.Now
	if opts.Now != nil {
		now = opts.Now
	}

	// The signature must cover the Date header, so the request carries one;
	// a request without it has an empty date line in the signing string and
	// would otherwise escape the freshness window entirely.
	dateStr := req.Header.Get("Date")
	if dateStr == "" {
		return "", ErrStaleSignature
	}
	if date, err := http.ParseTime(dateStr); err != nil {
		return "", ErrStaleSignature
	} else {
		delta := now().Sub(date)
		if delta > maxSignatureAge || delta < -maxSignatureAge {
			return "", ErrStaleSignature
		}
	}

	// Rebuild the signing string from what the server actually received.
	lines := make([]string, 0, len(params.Headers))
	for _, name := range params.Headers {
		switch name {
		case "(request-target)":
			target := req.URL.EscapedPath()
			if target == "" {
				target = "/"
			}
			if req.URL.RawQuery != "" {
				target += "?" + req.URL.RawQuery
			}
			lines = append(lines, "(request-target): "+strings.ToLower(req.Method)+" "+target)
		case "host":
			host := req.Header.Get("Host")
			if host == "" {
				host = req.Host
			}
			lines = append(lines, "host: "+host)
		default:
			lines = append(lines, name+": "+req.Header.Get(name))
		}
	}
	signingString := strings.Join(lines, "\n")

	pub, err := opts.PublicKey(params.KeyID)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256([]byte(signingString))
	if err := rsa.VerifyPKCS1v15(pub, crypto.SHA256, sum[:], params.Signature); err != nil {
		return "", ErrSignatureInvalid
	}
	return params.KeyID, nil
}

// verifyDigest checks a Digest header against body.
func verifyDigest(header string, body []byte) error {
	if header == "" {
		return ErrBadDigest
	}
	algo, val, ok := strings.Cut(header, "=")
	if !ok {
		return ErrBadDigest
	}
	if !strings.EqualFold(strings.TrimSpace(algo), "SHA-256") {
		return ErrBadDigest
	}
	sum := sha256.Sum256(body)
	if base64.StdEncoding.EncodeToString(sum[:]) != strings.TrimSpace(val) {
		return ErrDigestMismatch
	}
	return nil
}
