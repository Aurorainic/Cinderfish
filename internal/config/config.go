// Package config loads CinderFish configuration from a TOML file with
// CINDERFISH_ environment variable overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Host             string `toml:"host"`
	Port             int    `toml:"port"`
	Domain           string `toml:"domain"`
	DatabaseURL      string `toml:"database_url"`
	RegistrationMode string `toml:"registration_mode"`
	FederationMode   string `toml:"federation_mode"`
	// WebDir points at the built frontend (index.html + assets). An
	// empty or missing directory leaves the server API-only.
	WebDir     string     `toml:"web_dir"`
	Federation Federation `toml:"federation"`
	Media      Media      `toml:"media"`
	Redis      Redis      `toml:"redis"`
	Mail       Mail       `toml:"mail"`
	// SigninRateLimit caps password attempts per source IP per hour, the
	// cap Misskey's signin limiter uses. Zero means the default of 10;
	// a negative value disables the cap (tests, sealed networks).
	SigninRateLimit int `toml:"signin_rate_limit"`
	// DatabaseMaxConns caps the pgxpool size. Zero means the default of 10;
	// raise it only when the PostgreSQL server can take the connections.
	DatabaseMaxConns int `toml:"database_max_conns"`
}

// Mail configures outbound email. An empty Host means the server logs the
// messages instead of delivering them: verification links still work when
// copied out of the log, so a fresh instance can be set up before SMTP is
// reachable, and tests never touch the network.
type Mail struct {
	Host     string `toml:"host"`
	Port     int    `toml:"port"`
	Username string `toml:"username"`
	Password string `toml:"password"`
	// From is the envelope sender. It defaults to no-reply@<domain>.
	From string `toml:"from"`
	// TLS selects the transport: "starttls" (default, submission port),
	// "tls" (implicit TLS, port 465) or "none" (plain text, tests only).
	TLS string `toml:"tls"`
}

// Enabled reports whether a real SMTP host is configured.
func (m Mail) Enabled() bool { return strings.TrimSpace(m.Host) != "" }

// Redis enables the optional Redis backend. PostgreSQL stays the system
// of record; Redis carries cross-process Streaming events and delivery
// wakeups, and is disabled by default.
type Redis struct {
	Enabled bool   `toml:"enabled"`
	URL     string `toml:"url"`
	// Prefix namespaces every key and channel this instance uses, so one
	// Redis database can be shared by several instances.
	Prefix string `toml:"prefix"`
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
	// CACert is a PEM file the S3 client trusts in addition to the system
	// roots; self-hosted MinIO and private S3 gateways usually need it.
	CACert string `toml:"ca_cert"`
}

func (c *Config) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}

var validRegistration = map[string]bool{"open": true, "invite": true, "admin": true}
var validFederation = map[string]bool{"public": true, "allowlist": true, "none": true}
var validMediaBackend = map[string]bool{"local": true, "s3": true}
var validMailTLS = map[string]bool{"starttls": true, "tls": true, "none": true}

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
		WebDir:           "./web/dist",
		Media:            Media{Backend: "local", LocalPath: "./data/files"},
		Redis:            Redis{Enabled: false, URL: "redis://127.0.0.1:6379/0", Prefix: "cinderfish"},
		Mail:             Mail{Port: 587, TLS: "starttls"},
	}

	if path != "" {
		if _, err := toml.DecodeFile(path, c); err != nil {
			return nil, fmt.Errorf("config: decode %s: %w", path, err)
		}
	}
	applyEnv(c)
	if strings.TrimSpace(c.Redis.Prefix) == "" {
		c.Redis.Prefix = "cinderfish"
	}
	if c.Mail.TLS == "" {
		c.Mail.TLS = "starttls"
	}
	if c.Mail.Port == 0 {
		if c.Mail.TLS == "tls" {
			c.Mail.Port = 465
		} else {
			c.Mail.Port = 587
		}
	}
	if strings.TrimSpace(c.Mail.From) == "" {
		c.Mail.From = "no-reply@" + strings.TrimSpace(c.Domain)
	}
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
	setStr("CINDERFISH_MEDIA_S3_ENDPOINT", &c.Media.S3.Endpoint)
	setStr("CINDERFISH_MEDIA_S3_BUCKET", &c.Media.S3.Bucket)
	setStr("CINDERFISH_MEDIA_S3_REGION", &c.Media.S3.Region)
	setStr("CINDERFISH_MEDIA_S3_ACCESS_KEY", &c.Media.S3.AccessKey)
	setStr("CINDERFISH_MEDIA_S3_SECRET_KEY", &c.Media.S3.SecretKey)
	setBool("CINDERFISH_MEDIA_S3_USE_SSL", &c.Media.S3.UseSSL)
	setStr("CINDERFISH_MEDIA_S3_PUBLIC_BASE_URL", &c.Media.S3.PublicBaseURL)
	setStr("CINDERFISH_MEDIA_S3_CA_CERT", &c.Media.S3.CACert)
	setStr("CINDERFISH_WEB_DIR", &c.WebDir)
	setStr("CINDERFISH_REDIS_URL", &c.Redis.URL)
	setStr("CINDERFISH_REDIS_PREFIX", &c.Redis.Prefix)
	setBool("CINDERFISH_REDIS_ENABLED", &c.Redis.Enabled)
	setStr("CINDERFISH_MAIL_HOST", &c.Mail.Host)
	setStr("CINDERFISH_MAIL_USERNAME", &c.Mail.Username)
	setStr("CINDERFISH_MAIL_PASSWORD", &c.Mail.Password)
	setStr("CINDERFISH_MAIL_FROM", &c.Mail.From)
	setStr("CINDERFISH_MAIL_TLS", &c.Mail.TLS)
	setInt("CINDERFISH_MAIL_PORT", &c.Mail.Port)
	setInt("CINDERFISH_PORT", &c.Port)
	setInt("CINDERFISH_DATABASE_MAX_CONNS", &c.DatabaseMaxConns)
}

func setStr(key string, dst *string) {
	if v, ok := os.LookupEnv(key); ok && strings.TrimSpace(v) != "" {
		*dst = v
	}
}

func setBool(key string, dst *bool) {
	if v, ok := os.LookupEnv(key); ok {
		if parsed, err := strconv.ParseBool(strings.TrimSpace(v)); err == nil {
			*dst = parsed
		}
	}
}

func setInt(key string, dst *int) {
	if v, ok := os.LookupEnv(key); ok {
		if n, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
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
	if c.Redis.Enabled && strings.TrimSpace(c.Redis.URL) == "" {
		return fmt.Errorf("config: redis.url is required when redis is enabled")
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
	if !validMailTLS[c.Mail.TLS] {
		return fmt.Errorf("config: mail.tls must be starttls, tls or none (got %q)", c.Mail.TLS)
	}
	if c.Mail.Enabled() && (c.Mail.Port <= 0 || c.Mail.Port > 65535) {
		return fmt.Errorf("config: invalid mail port %d", c.Mail.Port)
	}
	return nil
}
