package services

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"cc-dailyuse-bar/src/models"
)

func newTestConfigService(reader func(string) ([]byte, error)) *ConfigService {
	svc := NewConfigService()
	svc.SetConfigPath("config.yaml")
	svc.SetReadFile(reader)
	return svc
}

func TestConfigService_GetConfigPath(t *testing.T) {
	svc := NewConfigService()
	path := svc.GetConfigPath()

	assert.NotEmpty(t, path)
	assert.Contains(t, path, "cc-dailyuse-bar")
	assert.Contains(t, path, "config.yaml")
	assert.True(t, filepath.IsAbs(path))
}

func TestConfigService_LoadDefaultsWhenFileMissing(t *testing.T) {
	svc := newTestConfigService(func(string) ([]byte, error) {
		return nil, os.ErrNotExist
	})

	cfg, err := svc.Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, models.ConfigDefaults(), cfg)
}

func TestConfigService_LoadPropagatesReadError(t *testing.T) {
	expectedErr := errors.New("permission denied")
	svc := newTestConfigService(func(string) ([]byte, error) {
		return nil, expectedErr
	})

	cfg, err := svc.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Equal(t, expectedErr, err)
}

func TestConfigService_LoadInvalidYAML(t *testing.T) {
	svc := newTestConfigService(func(string) ([]byte, error) {
		return []byte("not: [valid"), nil
	})

	cfg, err := svc.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "yaml")
}

func TestConfigService_LoadInvalidConfig(t *testing.T) {
	svc := newTestConfigService(func(string) ([]byte, error) {
		return []byte(`ccusage_path: "ccusage"
update_interval: -10
yellow_threshold: 5.0
red_threshold: 10.0
debug_level: "INFO"
cache_window: 10
cmd_timeout: 5`), nil
	})

	cfg, err := svc.Load()

	require.Error(t, err)
	assert.Nil(t, cfg)
	assert.Contains(t, err.Error(), "update_interval")
}

func TestConfigService_LoadValidConfig(t *testing.T) {
	svc := newTestConfigService(func(string) ([]byte, error) {
		return []byte(`ccusage_path: "/custom/ccusage"
update_interval: 60
yellow_threshold: 7.5
red_threshold: 15.0
debug_level: "DEBUG"
cache_window: 25
cmd_timeout: 12`), nil
	})

	cfg, err := svc.Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)

	assert.Equal(t, "/custom/ccusage", cfg.CCUsagePath)
	assert.Equal(t, 60, cfg.UpdateInterval)
	assert.Equal(t, 7.5, cfg.YellowThreshold)
	assert.Equal(t, 15.0, cfg.RedThreshold)
	assert.Equal(t, "DEBUG", cfg.DebugLevel)
	assert.Equal(t, 25, cfg.CacheWindow)
	assert.Equal(t, 12, cfg.CmdTimeout)
}

func TestConfigService_Validate(t *testing.T) {
	svc := NewConfigService()
	base := models.ConfigDefaults()

	testCases := []struct {
		name     string
		mutate   func(*models.Config)
		wantErr  bool
		errToken string
	}{
		{
			name:   "valid defaults",
			mutate: func(*models.Config) {},
		},
		{
			name:     "empty ccusage path",
			mutate:   func(c *models.Config) { c.CCUsagePath = "" },
			wantErr:  true,
			errToken: "ccusage_path",
		},
		{
			name:     "update interval out of range",
			mutate:   func(c *models.Config) { c.UpdateInterval = 301 },
			wantErr:  true,
			errToken: "update_interval",
		},
		{
			name: "red threshold below yellow",
			mutate: func(c *models.Config) {
				c.YellowThreshold = 10
				c.RedThreshold = 5
			},
			wantErr:  true,
			errToken: "red_threshold",
		},
		{
			name:     "invalid debug level",
			mutate:   func(c *models.Config) { c.DebugLevel = "TRACE" },
			wantErr:  true,
			errToken: "debug_level",
		},
		{
			name:     "cache window out of range",
			mutate:   func(c *models.Config) { c.CacheWindow = 0 },
			wantErr:  true,
			errToken: "cache_window",
		},
		{
			name:     "cmd timeout out of range",
			mutate:   func(c *models.Config) { c.CmdTimeout = 0 },
			wantErr:  true,
			errToken: "cmd_timeout",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := *base
			tc.mutate(&cfg)

			err := svc.Validate(&cfg)
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errToken)
				return
			}
			assert.NoError(t, err)
		})
	}
}

// Pins the nil-config guard added in commit a495947. Without the guard,
// Validate would dereference nil and panic instead of returning a typed
// validation error.
func TestConfigService_ValidateNil(t *testing.T) {
	svc := NewConfigService()

	err := svc.Validate(nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "VALIDATION_ERROR")
	assert.Contains(t, err.Error(), "nil")
}

func TestConfigService_SetReadFileResetToDefault(t *testing.T) {
	svc := NewConfigService()
	svc.SetConfigPath("nonexistent.yaml")
	svc.SetReadFile(nil)

	cfg, err := svc.Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, models.ConfigDefaults(), cfg)
}

func TestConfigService_EnsureConfigDir(t *testing.T) {
	svc := NewConfigService()
	configPath := filepath.FromSlash("/home/user/.config/cc-dailyuse-bar/config.yaml")
	svc.SetConfigPath(configPath)

	var capturedPath string
	var capturedMode os.FileMode

	svc.SetMkdirAll(func(path string, mode os.FileMode) error {
		capturedPath = path
		capturedMode = mode
		return nil
	})

	err := svc.EnsureConfigDir()
	require.NoError(t, err)
	assert.Equal(t, filepath.Dir(configPath), capturedPath)
	assert.Equal(t, os.FileMode(0755), capturedMode)
}

func TestConfigService_EnsureConfigDirError(t *testing.T) {
	svc := NewConfigService()
	expectedErr := errors.New("mkdir failed")
	svc.SetMkdirAll(func(path string, mode os.FileMode) error {
		return expectedErr
	})

	err := svc.EnsureConfigDir()
	require.Error(t, err)
	assert.ErrorIs(t, err, expectedErr)
}

func TestConfigService_Save(t *testing.T) {
	svc := NewConfigService()
	svc.SetConfigPath("/test/config.yaml")

	// Mock MkdirAll
	svc.SetMkdirAll(func(path string, mode os.FileMode) error {
		return nil
	})

	// Mock WriteFile
	var capturedData []byte
	var capturedPath string
	svc.SetWriteFile(func(path string, data []byte, mode os.FileMode) error {
		capturedPath = path
		capturedData = data
		return nil
	})

	cfg := models.ConfigDefaults()
	cfg.YellowThreshold = 12.34

	err := svc.Save(cfg)
	require.NoError(t, err)

	assert.Equal(t, "/test/config.yaml", capturedPath)
	assert.Contains(t, string(capturedData), "yellow_threshold: 12.34")
}

func TestConfigService_SaveValidationFailed(t *testing.T) {
	svc := NewConfigService()
	cfg := models.ConfigDefaults()
	cfg.CCUsagePath = "" // Invalid

	err := svc.Save(cfg)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "ccusage_path")
}

func TestConfigService_SaveNil(t *testing.T) {
	svc := NewConfigService()

	require.NotPanics(t, func() {
		err := svc.Save(nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "config is nil")
	})
}

func TestConfigService_SaveMkdirFailed(t *testing.T) {
	svc := NewConfigService()
	svc.SetMkdirAll(func(path string, mode os.FileMode) error {
		return errors.New("mkdir error")
	})

	err := svc.Save(models.ConfigDefaults())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "mkdir error")
}

func TestConfigService_SaveWriteFailed(t *testing.T) {
	svc := NewConfigService()
	svc.SetMkdirAll(func(path string, mode os.FileMode) error {
		return nil
	})
	svc.SetWriteFile(func(path string, data []byte, mode os.FileMode) error {
		return errors.New("write error")
	})

	err := svc.Save(models.ConfigDefaults())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "write error")
}
