package logger

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnvEnabled(t *testing.T) {
	assert.True(t, envEnabled(""))

	t.Setenv("LOGGER_ENABLE_TEST", "true")
	assert.True(t, envEnabled("LOGGER_ENABLE_TEST"))

	t.Setenv("LOGGER_ENABLE_TEST", "0")
	assert.False(t, envEnabled("LOGGER_ENABLE_TEST"))
}

func TestParseLevel(t *testing.T) {
	level, err := parseLevel("")
	require.NoError(t, err)
	assert.Equal(t, LevelInfo, level)

	level, err = parseLevel("debug")
	require.NoError(t, err)
	assert.Equal(t, LevelDebug, level)

	level, err = parseLevel("info")
	require.NoError(t, err)
	assert.Equal(t, LevelInfo, level)

	level, err = parseLevel("warn")
	require.NoError(t, err)
	assert.Equal(t, LevelWarn, level)

	level, err = parseLevel("warning")
	require.NoError(t, err)
	assert.Equal(t, LevelWarn, level)

	level, err = parseLevel("error")
	require.NoError(t, err)
	assert.Equal(t, LevelError, level)

	_, err = parseLevel("invalid")
	assert.Error(t, err)
}

func TestResolveFilepath(t *testing.T) {
	assert.Equal(t, "", resolveFilepath(""))

	tempDir := t.TempDir()
	abs := filepath.Join(tempDir, "abs.log")
	assert.Equal(t, abs, resolveFilepath(abs))

	rel := "relative.log"
	exe, err := os.Executable()
	require.NoError(t, err)
	want := filepath.Join(filepath.Dir(exe), rel)
	assert.Equal(t, want, resolveFilepath(rel))
}

func TestInitFromConfigDisabled(t *testing.T) {
	resetRegistry()

	cfg := Config{
		Loggers: []NamedConfig{
			{
				Name:      "ws",
				EnableEnv: "LOGGER_ENABLE_TEST",
			},
		},
	}

	err := InitFromConfig(cfg)
	require.NoError(t, err)

	l := Get("ws")
	_, ok := l.(nopLogger)
	assert.True(t, ok)
}

func TestInitFromConfigEnabledMissingPath(t *testing.T) {
	resetRegistry()

	t.Setenv("LOGGER_ENABLE_TEST", "true")
	cfg := Config{
		Loggers: []NamedConfig{
			{
				Name:      "ws",
				EnableEnv: "LOGGER_ENABLE_TEST",
			},
		},
	}

	err := InitFromConfig(cfg)
	assert.Equal(t, errEmptyLogPath, err)
}

func TestInitFromConfigEnabled(t *testing.T) {
	resetRegistry()

	t.Setenv("LOGGER_ENABLE_TEST", "true")
	logPath := filepath.Join(t.TempDir(), "ws.log")
	cfg := Config{
		Loggers: []NamedConfig{
			{
				Name:      "ws",
				Filepath:  logPath,
				Level:     "info",
				EnableEnv: "LOGGER_ENABLE_TEST",
			},
		},
	}

	err := InitFromConfig(cfg)
	require.NoError(t, err)

	l := Get("ws")
	_, ok := l.(*ZapLogger)
	assert.True(t, ok)
}
