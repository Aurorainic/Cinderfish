package federation

import (
	"context"
	"testing"
)

func TestHostMatchesSuffix(t *testing.T) {
	// Misskey's rule: an entry matches itself and its subdomains, and a
	// lookalike domain that merely ends with the entry does not match.
	cases := []struct {
		host, entry string
		want        bool
	}{
		{"example.com", "example.com", true},
		{"sub.example.com", "example.com", true},
		{"deep.sub.example.com", "example.com", true},
		{"notexample.com", "example.com", false},
		{"example.com.evil.test", "example.com", false},
		{"Example.COM", "example.com", true},
		{"example.com", "", false},
	}
	for _, c := range cases {
		if got := HostMatchesSuffix(c.host, c.entry); got != c.want {
			t.Errorf("HostMatchesSuffix(%q, %q) = %v, want %v", c.host, c.entry, got, c.want)
		}
	}
}

func TestParseMode(t *testing.T) {
	for _, s := range []string{"public", "allowlist", "none"} {
		if _, err := ParseMode(s); err != nil {
			t.Errorf("ParseMode(%q): %v", s, err)
		}
	}
	if _, err := ParseMode("federated"); err == nil {
		t.Error("ParseMode accepted an unknown mode")
	}
}

func TestModeMisskeyMapping(t *testing.T) {
	cases := map[Mode]string{ModePublic: "all", ModeAllowlist: "specified", ModeNone: "none"}
	for m, want := range cases {
		if got := m.MisskeyMode(); got != want {
			t.Errorf("%s.MisskeyMode() = %q, want %q", m, got, want)
		}
	}
}

func TestHostOf(t *testing.T) {
	cases := map[string]string{
		"https://a.example/users/1": "a.example",
		"https://A.Example:443/x":   "a.example",
		"http://localhost:3000/":    "localhost",
	}
	for in, want := range cases {
		got, err := HostOf(in)
		if err != nil {
			t.Errorf("HostOf(%q): %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("HostOf(%q) = %q, want %q", in, got, want)
		}
	}
	if _, err := HostOf("/relative/path"); err == nil {
		t.Error("HostOf accepted a relative URL")
	}
}

// policyDB is implemented in policy_db_test.go behind the test-database
// build tag, so the unit tests above run without PostgreSQL.

func TestPolicyWithoutDatabaseRefusesEverything(t *testing.T) {
	p := NewPolicy(nil, "self.example")
	ctx := context.Background()

	// A nil database means "no instance_meta row"; the safe reading of
	// that is mode none, never mode public.
	if p.IsAllowed(ctx, "other.example") {
		t.Error("allowed a remote host with no database")
	}
	if !p.IsAllowed(ctx, "self.example") {
		t.Error("refused the instance's own host")
	}
	if err := p.CheckAllowed(ctx, "other.example"); err == nil {
		t.Error("CheckAllowed allowed a host with no database")
	}
}
