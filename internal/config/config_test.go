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
	require.Error(t, err)

	t.Setenv("CINDERFISH_MEDIA_S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("CINDERFISH_MEDIA_S3_BUCKET", "cinderfish")
	_, err = Load("")
	require.Error(t, err, "secret key check not enforced yet, but endpoint/bucket should pass; only other invalid fields fail")
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
