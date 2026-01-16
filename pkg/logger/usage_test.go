package logger

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZapLoggerUsage(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "app.log")
	l, err := NewZapLogger(ZapConfig{
		Filepath: logPath,
		Level:    LevelDebug,
		MaxSize:  1,
	})
	require.NoError(t, err)

	l.Info("hello", Field{Key: "k", Value: "v"})
	l.WithGroup("ws").Debug("ping", Field{Key: "id", Value: 1})
	require.NoError(t, l.Sync())

	records := readLogRecords(t, logPath)
	assert.True(t, hasRecord(records, "hello", "k", "v"))
	assert.True(t, hasRecord(records, "ping", "ws.id", float64(1)))
}

func TestLoggerEnabled(t *testing.T) {
	logPath := filepath.Join(t.TempDir(), "level.log")
	l, err := NewZapLogger(ZapConfig{
		Filepath: logPath,
		Level:    LevelInfo,
		MaxSize:  1,
	})
	require.NoError(t, err)

	assert.False(t, l.Enabled(context.Background(), LevelDebug))
	assert.True(t, l.Enabled(context.Background(), LevelInfo))
	assert.True(t, l.Enabled(context.Background(), LevelWarn))
}

func TestRegistryUsage(t *testing.T) {
	resetRegistry()

	t.Setenv("WS_LOG_ENABLE", "true")
	logPath := filepath.Join(t.TempDir(), "ws.log")
	cfg := Config{
		Loggers: []NamedConfig{
			{
				Name:      "ws",
				Filepath:  logPath,
				Level:     "info",
				EnableEnv: "WS_LOG_ENABLE",
			},
		},
	}
	require.NoError(t, InitFromConfig(cfg))

	log := Get("ws")
	log.Info("ready", Field{Key: "port", Value: 8080})
	require.NoError(t, log.Sync())

	records := readLogRecords(t, logPath)
	assert.True(t, hasRecord(records, "ready", "port", float64(8080)))
}

func TestRegistryUsageDisabled(t *testing.T) {
	resetRegistry()

	logPath := filepath.Join(t.TempDir(), "disabled.log")
	cfg := Config{
		Loggers: []NamedConfig{
			{
				Name:      "ws",
				Filepath:  logPath,
				Level:     "info",
				EnableEnv: "WS_LOG_ENABLE",
			},
		},
	}
	require.NoError(t, InitFromConfig(cfg))

	log := Get("ws")
	log.Info("no_output")
	require.NoError(t, log.Sync())

	_, err := os.Stat(logPath)
	assert.Error(t, err)
}

func readLogRecords(t *testing.T, path string) []map[string]any {
	data, err := os.ReadFile(path)
	require.NoError(t, err)

	lines := strings.Split(string(data), "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &record))
		records = append(records, record)
	}
	require.NotEmpty(t, records)
	return records
}

func hasRecord(records []map[string]any, msg string, key string, val any) bool {
	for _, record := range records {
		if record["msg"] != msg {
			continue
		}
		if record[key] == val {
			return true
		}
	}
	return false
}
