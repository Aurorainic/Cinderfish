package webhook

import (
	"context"
	"errors"
	"net"
	"testing"
)

func TestValidateURL(t *testing.T) {
	ctx := context.Background()
	for _, tt := range []struct {
		name string
		url  string
		ok   bool
	}{
		// The two public cases need DNS; this environment may have none,
		// in which case they resolve to a refusal and are skipped below.
		{"public https", "https://example.com/notify", dnsAvailable()},
		{"public http", "http://example.com/notify", dnsAvailable()},
		{"loopback", "http://127.0.0.1:9000/hook", false},
		{"loopback name", "http://localhost:9000/hook", false},
		{"private range", "http://192.168.1.10/hook", false},
		{"link-local", "http://169.254.169.254/latest/meta-data", false},
		{"not http", "gopher://hooks.example.com/x", false},
		{"no host", "http:///hook", false},
		{"garbage", "://not a url", false},
	} {
		if tt.url == "https://example.com/notify" || tt.url == "http://example.com/notify" {
			if !dnsAvailable() {
				continue
			}
		}
		err := ValidateURL(ctx, tt.url)
		if tt.ok && err != nil {
			t.Errorf("%s: ValidateURL(%q) = %v, want nil", tt.name, tt.url, err)
		}
		if !tt.ok && !errors.Is(err, ErrInvalidURL) {
			t.Errorf("%s: ValidateURL(%q) = %v, want ErrInvalidURL", tt.name, tt.url, err)
		}
	}
}

// dnsAvailable reports whether this environment can resolve public names
// at all; the positive ValidateURL cases are meaningless without it.
func dnsAvailable() bool {
	ips, err := net.DefaultResolver.LookupIPAddr(context.Background(), "example.com")
	return err == nil && len(ips) > 0
}
