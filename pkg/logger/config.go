package logger

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config 表示日志配置结构。
type Config struct {
	Loggers []NamedConfig `yaml:"loggers"`
}

// NamedConfig 表示单个具名日志配置。
type NamedConfig struct {
	Name       string `yaml:"name"`
	Filepath   string `yaml:"filepath"`
	Level      string `yaml:"level"`
	MaxSize    int    `yaml:"max_size"`
	MaxBackups int    `yaml:"max_backups"`
	MaxAge     int    `yaml:"max_age"`
	Compress   bool   `yaml:"compress"`
	EnableEnv  string `yaml:"enable_env"`
}

// InitFromConfig 根据配置创建并注册具名日志实例。
func InitFromConfig(cfg Config) error {
	for _, item := range cfg.Loggers {
		name := strings.TrimSpace(item.Name)
		if name == "" {
			return errEmptyLoggerName
		}

		if !envEnabled(item.EnableEnv) {
			if err := Register(name, Nop()); err != nil {
				return err
			}
			continue
		}

		level, err := parseLevel(item.Level)
		if err != nil {
			return err
		}

		path := resolveFilepath(item.Filepath)
		if path == "" {
			return errEmptyLogPath
		}

		l, err := NewZapLogger(ZapConfig{
			Filepath:   path,
			Level:      level,
			MaxSize:    item.MaxSize,
			MaxBackups: item.MaxBackups,
			MaxAge:     item.MaxAge,
			Compress:   item.Compress,
		})
		if err != nil {
			return err
		}

		if err := Register(name, l); err != nil {
			return err
		}
	}
	return nil
}

func envEnabled(key string) bool {
	if strings.TrimSpace(key) == "" {
		return true
	}
	raw := strings.TrimSpace(os.Getenv(key))
	if raw == "" {
		return false
	}
	switch strings.ToLower(raw) {
	case "1", "true", "t", "yes", "y", "on":
		return true
	default:
		return false
	}
}

func parseLevel(raw string) (Level, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "":
		return LevelInfo, nil
	case "debug":
		return LevelDebug, nil
	case "info":
		return LevelInfo, nil
	case "warn", "warning":
		return LevelWarn, nil
	case "error":
		return LevelError, nil
	default:
		return LevelInfo, fmt.Errorf("logger: invalid level %q", raw)
	}
}

func resolveFilepath(path string) string {
	if strings.TrimSpace(path) == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		return path
	}
	exe, err := os.Executable()
	if err != nil {
		return path
	}
	return filepath.Join(filepath.Dir(exe), path)
}
