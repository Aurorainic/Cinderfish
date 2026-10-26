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
	t.Setenv("MILNFISH_PORT", "5555")
	t.Setenv("MILNFISH_DOMAIN", "env.example")
	t.Setenv("MILNFISH_REGISTRATION_MODE", "invite")

	c, err := Load("")
	require.NoError(t, err)
	assert.Equal(t, 5555, c.Port)
	assert.Equal(t, "env.example", c.Domain)
	assert.Equal(t, "invite", c.RegistrationMode)
}

func TestValidation(t *testing.T) {
	t.Setenv("MILNFISH_REGISTRATION_MODE", "nonsense")
	_, err := Load("")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "registration_mode")
}

func TestS3BackendValidation(t *testing.T) {
	t.Setenv("MILNFISH_MEDIA_BACKEND", "s3")
	_, err := Load("")
	require.Error(t, err)

	t.Setenv("MILNFISH_MEDIA_S3_ENDPOINT", "http://localhost:9000")
	t.Setenv("MILNFISH_MEDIA_S3_BUCKET", "milnfish")
	_, err = Load("")
	require.Error(t, err, "secret key check not enforced yet, but endpoint/bucket should pass; only other invalid fields fail")
}
