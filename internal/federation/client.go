package federation

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

// Errors returned by outbound resolution.
var (
	ErrRemoteStatus   = errors.New("federation: remote returned an error status")
	ErrNotActivityPub = errors.New("federation: remote is not serving ActivityPub")
	ErrIDMismatch     = errors.New("federation: object id does not match the requested URL")
)

// UserAgent identifies this instance to peers.
const UserAgent = "Cinderfish/0.1.1"

// Client performs signed outbound requests and remote resolution.
//
// Every outbound call goes through the policy first, so switching the
// instance to none or allowlist stops resolution immediately rather than
// at the next restart.
type Client struct {
	policy   *Policy
	renderer *Renderer
	http     *http.Client
	// KeySource returns the signing key for one local user. Deliveries pass
	// the activity's author: a peer checks that the signature's key belongs
	// to the actor it names, so signing someone else's activity with the
	// instance key gets it rejected. An empty userID asks for the instance
	// key, which is what plain object fetches use.
	KeySource func(ctx context.Context, userID string) (keyID string, priv *rsa.PrivateKey, err error)
}

// NewClient returns a client for this instance.
func NewClient(policy *Policy, renderer *Renderer, keySource func(context.Context, string) (string, *rsa.PrivateKey, error)) *Client {
	return &Client{
		policy:    policy,
		renderer:  renderer,
		KeySource: keySource,
		http: &http.Client{
			Timeout: 15 * time.Second,
			// ActivityPub identity is the URL, so following redirects is
			// normal; the limit stops a redirect loop.
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return errors.New("too many redirects")
				}
				return nil
			},
		},
	}
}

// SetHTTPClient replaces the transport, for tests.
func (c *Client) SetHTTPClient(h *http.Client) { c.http = h }

// WebFinger is a decoded JRD document.
type WebFinger struct {
	Subject string          `json:"subject"`
	Links   []WebFingerLink `json:"links"`
}

// WebFingerLink is one link entry.
type WebFingerLink struct {
	Rel      string `json:"rel"`
	Type     string `json:"type"`
	Href     string `json:"href"`
	Template string `json:"template"`
}

// Self returns the ActivityPub actor URL from the self link.
func (w *WebFinger) Self() string {
	for _, l := range w.Links {
		if l.Rel == "self" && (strings.Contains(l.Type, "activity+json") || strings.Contains(l.Type, "ld+json")) {
			return l.Href
		}
	}
	return ""
}

// ResolveWebFinger looks up acct (user@host) and returns its JRD.
//
// The query shape copies Misskey's WebfingerService: a bare hostname
// gets /.well-known/webfinger?resource=acct:<user>@<host>.
func (c *Client) ResolveWebFinger(ctx context.Context, acct string) (*WebFinger, error) {
	user, host, ok := SplitAcct(acct)
	if !ok {
		return nil, fmt.Errorf("federation: %q is not an acct", acct)
	}
	if err := c.policy.CheckAllowed(ctx, host); err != nil {
		return nil, err
	}

	target := "https://" + host + "/.well-known/webfinger?resource=" + url.QueryEscape("acct:"+user+"@"+host)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/jrd+json, application/json")
	req.Header.Set("User-Agent", UserAgent)

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: webfinger %s returned %d", ErrRemoteStatus, acct, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("federation: read webfinger: %w", err)
	}
	var wf WebFinger
	if err := json.Unmarshal(body, &wf); err != nil {
		return nil, fmt.Errorf("federation: decode webfinger: %w", err)
	}
	return &wf, nil
}

// FetchObject GETs an ActivityPub object, signing the request when a key
// source is configured. When objectID is non-empty the returned object's
// id must match it, which is what stops a peer from serving someone
// else's object at its own URL.
func (c *Client) FetchObject(ctx context.Context, target, objectID string) (Object, error) {
	if err := c.policy.CheckURLAllowed(ctx, target); err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", ContentTypeActivity+", "+ContentTypeLD)
	req.Header.Set("User-Agent", UserAgent)
	if err := c.sign(ctx, req, nil, ""); err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: GET %s returned %d", ErrRemoteStatus, target, resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return nil, fmt.Errorf("federation: read object: %w", err)
	}

	// Some servers answer an AP fetch with HTML that carries an
	// alternate link; treat a non-JSON body as "not ActivityPub" rather
	// than trying to parse it.
	if ct := resp.Header.Get("Content-Type"); ct != "" && !isAPContentType(ct) {
		return nil, fmt.Errorf("%w: GET %s served %s", ErrNotActivityPub, target, ct)
	}

	obj, err := DecodeObject(body)
	if err != nil {
		return nil, err
	}
	if objectID != "" {
		if got := obj.String("id"); got != "" && got != objectID {
			return nil, fmt.Errorf("%w: asked for %s, got %s", ErrIDMismatch, objectID, got)
		}
	}
	return obj, nil
}

// FetchMedia downloads a public media object with a strict response-size
// limit. Each redirect is checked against the live federation policy.
func (c *Client) FetchMedia(ctx context.Context, target string, maxBytes int64) (string, []byte, string, error) {
	if maxBytes <= 0 {
		return "", nil, "", fmt.Errorf("federation: invalid media size limit")
	}
	if err := c.policy.CheckURLAllowed(ctx, target); err != nil {
		return "", nil, "", err
	}
	client := *c.http
	checkRedirect := client.CheckRedirect
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 5 {
			return errors.New("too many redirects")
		}
		if err := c.policy.CheckURLAllowed(req.Context(), req.URL.String()); err != nil {
			return err
		}
		if checkRedirect != nil {
			return checkRedirect(req, via)
		}
		return nil
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target, nil)
	if err != nil {
		return "", nil, "", err
	}
	req.Header.Set("User-Agent", UserAgent)
	resp, err := client.Do(req)
	if err != nil {
		return "", nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", nil, "", fmt.Errorf("%w: media returned %d", ErrRemoteStatus, resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return "", nil, "", err
	}
	if int64(len(data)) > maxBytes {
		return "", nil, "", fmt.Errorf("federation: media exceeds size limit")
	}
	parsed, _ := url.Parse(target)
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" {
		name = "remote-media"
	}
	contentType, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type"))
	return name, data, contentType, nil
}

// PostActivity signs and POSTs an activity to inboxURL. asUserID names the
// local account the activity speaks for; it picks the signing key.
func (c *Client) PostActivity(ctx context.Context, inboxURL string, activity Object, asUserID string) error {
	if err := c.policy.CheckURLAllowed(ctx, inboxURL); err != nil {
		return err
	}
	body, err := json.Marshal(activity)
	if err != nil {
		return fmt.Errorf("federation: encode activity: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, inboxURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	if err := c.sign(ctx, req, body, asUserID); err != nil {
		return err
	}

	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<16))

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("%w: POST %s returned %d", ErrRemoteStatus, inboxURL, resp.StatusCode)
	}
	return nil
}

func (c *Client) sign(ctx context.Context, req *http.Request, body []byte, asUserID string) error {
	if c.KeySource == nil {
		return nil
	}
	keyID, priv, err := c.KeySource(ctx, asUserID)
	if err != nil {
		return fmt.Errorf("federation: signing key: %w", err)
	}
	return SignRequest(req, body, keyID, priv)
}

// do performs the request, naming the host rather than the full URL in
// transport errors.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		host := ""
		if req.URL != nil {
			host = req.URL.Host
		}
		return nil, fmt.Errorf("federation: %s %s: %w", req.Method, host, err)
	}
	return resp, nil
}

// isAPContentType reports whether ct is one of the ActivityPub media
// types Misskey accepts.
func isAPContentType(ct string) bool {
	ct = strings.ToLower(ct)
	return strings.HasPrefix(ct, "application/activity+json") ||
		strings.HasPrefix(ct, "application/ld+json")
}

// SplitAcct splits "user@host" into its parts.
func SplitAcct(acct string) (user, host string, ok bool) {
	acct = strings.TrimPrefix(strings.TrimSpace(acct), "@")
	user, host, ok = strings.Cut(acct, "@")
	if !ok || user == "" || host == "" {
		return "", "", false
	}
	return user, strings.ToLower(host), true
}
