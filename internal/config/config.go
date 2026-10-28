// Package config loads CinderFish configuration from a TOML file with
// CINDERFISH_ environment variable overrides.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Host             string     `toml:"host"`
	Port             int        `toml:"port"`
	Domain           string     `toml:"domain"`
	DatabaseURL      string     `toml:"database_url"`
	RegistrationMode string     `toml:"registration_mode"`
	FederationMode   string     `toml:"federation_mode"`
	Federation       Federation `toml:"federation"`
	Media            Media      `toml:"media"`
}

// Federation groups the ActivityPub settings.
type Federation struct {
	// PublicURL overrides the scheme://host derived from domain and
	// port. Set it when the instance sits behind a reverse proxy.
	PublicURL string `toml:"public_url"`

	// AllowlistHosts seeds federation_allowlist on first start; after
	// that the table is authoritative.
	AllowlistHosts []string `toml:"allowlist_hosts"`
}

type Media struct {
	Backend   string     `toml:"backend"`
	LocalPath string     `toml:"local_path"`
	S3        S3Settings `toml:"s3"`
}

type S3Settings struct {
	Endpoint      string `toml:"endpoint"`
	Bucket        string `toml:"bucket"`
	Region        string `toml:"region"`
	AccessKey     string `toml:"access_key"`
	SecretKey     string `toml:"secret_key"`
	UseSSL        bool   `toml:"use_ssl"`
	PublicBaseURL string `toml:"public_base_url"`
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

var validRegistration = map[string]bool{"open": true, "invite": true, "admin": true}
var validFederation = map[string]bool{"public": true, "allowlist": true, "none": true}
var validMediaBackend = map[string]bool{"local": true, "s3": true}

// Load reads the TOML file at path (optional; defaults apply otherwise),
// then applies CINDERFISH_* environment overrides and validates the result.
func Load(path string) (*Config, error) {
	c := &Config{
		Host:             "127.0.0.1",
		Port:             3000,
		Domain:           "example.com",
		DatabaseURL:      "postgres://cinderfish:cinderfish@localhost:5432/cinderfish",
		RegistrationMode: "admin",
		FederationMode:   "none",
		Media:            Media{Backend: "local", LocalPath: "./data/files"},
	}

	if path != "" {
		if _, err := toml.DecodeFile(path, c); err != nil {
			return nil, fmt.Errorf("config: decode %s: %w", path, err)
		}
	}
	applyEnv(c)
	if err := c.validate(); err != nil {
		return nil, err
	}
	return c, nil
}

func applyEnv(c *Config) {
	setStr("CINDERFISH_HOST", &c.Host)
	setStr("CINDERFISH_DOMAIN", &c.Domain)
	setStr("CINDERFISH_DATABASE_URL", &c.DatabaseURL)
	setStr("CINDERFISH_REGISTRATION_MODE", &c.RegistrationMode)
	setStr("CINDERFISH_FEDERATION_MODE", &c.FederationMode)
	setStr("CINDERFISH_MEDIA_BACKEND", &c.Media.Backend)
	setStr("CINDERFISH_MEDIA_LOCAL_PATH", &c.Media.LocalPath)
	setInt("CINDERFISH_PORT", &c.Port)
}

func setStr(key string, dst *string) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		*dst = v
	}
}

func setInt(key string, dst *int) {
	if v, ok := os.LookupEnv(key); ok {
		var n int
		if _, err := fmt.Sscanf(strings.TrimSpace(v), "%d", &n); err == nil {
			*dst = n
		}
	}
}

func (c *Config) validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("config: invalid port %d", c.Port)
	}
	if strings.TrimSpace(c.Domain) == "" {
		return fmt.Errorf("config: domain is required")
	}
	if strings.TrimSpace(c.DatabaseURL) == "" {
		return fmt.Errorf("config: database_url is required")
	}
	if !validRegistration[c.RegistrationMode] {
		return fmt.Errorf("config: registration_mode must be open, invite or admin (got %q)", c.RegistrationMode)
	}
	if !validFederation[c.FederationMode] {
		return fmt.Errorf("config: federation_mode must be public, allowlist or none (got %q)", c.FederationMode)
	}
	if !validMediaBackend[c.Media.Backend] {
		return fmt.Errorf("config: media.backend must be local or s3 (got %q)", c.Media.Backend)
	}
	if c.Media.Backend == "s3" {
		s := c.Media.S3
		if s.Endpoint == "" || s.Bucket == "" {
			return fmt.Errorf("config: media.s3.endpoint and media.s3.bucket are required for the s3 backend")
		}
	}
	return nil
}
