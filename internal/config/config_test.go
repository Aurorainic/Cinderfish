package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	c, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 3000, c.Port)
	assert.Equal(t, "admin", c.RegistrationMode)
	assert.Equal(t, "none", c.FederationMode)
	assert.Equal(t, "local", c.Media.Backend)
}

func TestLoadFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
port = 4001
domain = "fish.example"
federation_mode = "allowlist"
`), 0o600))

	c, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, 4001, c.Port)
	assert.Equal(t, "fish.example", c.Domain)
	assert.Equal(t, "allowlist", c.FederationMode)
	assert.Equal(t, "admin", c.RegistrationMode, "file should not clobber defaults it does not set")
}

func TestEnvOverride(t *testing.T) {
	t.Setenv("CINDERFISH_PORT", "5555")
	t.Setenv("CINDERFISH_DOMAIN", "env.example")
	t.Setenv("CINDERFISH_REGISTRATION_MODE", "invite")

	c, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 5555, c.Port)
	assert.Equal(t, "env.example", c.Domain)
	assert.Equal(t, "invite", c.RegistrationMode)
}

func TestValidation(t *testing.T) {
	t.Setenv("CINDERFISH_REGISTRATION_MODE", "nonsense")
	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registration_mode")
}

func TestS3BackendValidation(t *testing.T) {
	t.Setenv("CINDERFISH_MEDIA_BACKEND", "s3")
	_, err := Load("")
	require.Error(t, err, "the s3 backend needs an endpoint and a bucket")

	t.Setenv("CINDERFISH_MEDIA_S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("CINDERFISH_MEDIA_S3_BUCKET", "cinderfish")
	c, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:9000", c.Media.S3.Endpoint)
	assert.Equal(t, "cinderfish", c.Media.S3.Bucket)
}

func TestRedisDefaultsAndFile(t *testing.T) {
	c, err := Load("")
	require.NoError(t, err)
	assert.False(t, c.Redis.Enabled, "redis must be opt-in")
	assert.Equal(t, "redis://127.0.0.1:6379/0", c.Redis.URL)
	assert.Equal(t, "cinderfish", c.Redis.Prefix)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
[redis]
enabled = true
url = "redis://redis.internal:6380/2"
prefix = "fish"
`), 0o600))

	c, err = Load(path)
	require.NoError(t, err)
	assert.True(t, c.Redis.Enabled)
	assert.Equal(t, "redis://redis.internal:6380/2", c.Redis.URL)
	assert.Equal(t, "fish", c.Redis.Prefix)
}

func TestRedisEnvOverrideAndValidation(t *testing.T) {
	t.Setenv("CINDERFISH_REDIS_ENABLED", "true")
	t.Setenv("CINDERFISH_REDIS_URL", "redis://127.0.0.1:6390/1")
	t.Setenv("CINDERFISH_REDIS_PREFIX", "test")

	c, err := Load("")
	require.NoError(t, err)
	assert.True(t, c.Redis.Enabled)
	assert.Equal(t, "redis://127.0.0.1:6390/1", c.Redis.URL)
	assert.Equal(t, "test", c.Redis.Prefix)

	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[redis]\nenabled = true\nurl = \"\"\n"), 0o600))
	t.Setenv("CINDERFISH_REDIS_ENABLED", "")
	t.Setenv("CINDERFISH_REDIS_URL", "")
	_, err = Load(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "redis.url")
}

func TestRedisBlankPrefixFallsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte("[redis]\nprefix = \"\"\n"), 0o600))

	c, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "cinderfish", c.Redis.Prefix)
}

// TestS3SettingsFromFileAndEnv pins the S3 block: the TOML fields parse, and the
// environment overrides (used by container deployments) replace them.
func TestS3SettingsFromFileAndEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	require.NoError(t, os.WriteFile(path, []byte(`
[media]
backend = "s3"

[media.s3]
endpoint = "minio.internal:9000"
bucket = "fish"
region = "us-east-1"
access_key = "file-ak"
secret_key = "file-sk"
use_ssl = true
public_base_url = "https://cdn.example/media"
ca_cert = "/etc/cinderfish/minio-ca.pem"
`), 0o600))

	c, err := Load(path)
	require.NoError(t, err)
	assert.Equal(t, "minio.internal:9000", c.Media.S3.Endpoint)
	assert.Equal(t, "fish", c.Media.S3.Bucket)
	assert.Equal(t, "/etc/cinderfish/minio-ca.pem", c.Media.S3.CACert)
	assert.True(t, c.Media.S3.UseSSL)

	t.Setenv("CINDERFISH_MEDIA_S3_ENDPOINT", "other.example:9001")
	t.Setenv("CINDERFISH_MEDIA_S3_CA_CERT", "/run/secrets/ca.pem")
	t.Setenv("CINDERFISH_MEDIA_S3_USE_SSL", "false")
	c, err = Load(path)
	require.NoError(t, err)
	assert.Equal(t, "other.example:9001", c.Media.S3.Endpoint)
	assert.Equal(t, "/run/secrets/ca.pem", c.Media.S3.CACert)
	assert.False(t, c.Media.S3.UseSSL)
}
