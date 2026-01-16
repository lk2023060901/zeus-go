package app

import (
	"os"

	"github.com/lk2023060901/zeus-go/pkg/logger"
	"gopkg.in/yaml.v3"
)

// Config 表示应用配置结构。
type Config struct {
	// Loggers 表示日志配置段。
	Loggers []logger.NamedConfig `yaml:"loggers"`
}

// LoadConfigFromFile 从 YAML 文件加载应用配置。
func LoadConfigFromFile(path string) (Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := yaml.Unmarshal(raw, &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
