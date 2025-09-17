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

func TestConfigService_SetReadFileResetToDefault(t *testing.T) {
	svc := NewConfigService()
	svc.SetConfigPath("nonexistent.yaml")
	svc.SetReadFile(nil)

	cfg, err := svc.Load()

	require.NoError(t, err)
	require.NotNil(t, cfg)
	assert.Equal(t, models.ConfigDefaults(), cfg)
}
