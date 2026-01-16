package logger

import (
	"errors"
	"sync"
)

var (
	errEmptyLoggerName  = errors.New("logger: name is empty")
	errNilLogger        = errors.New("logger: logger is nil")
	errLoggerRegistered = errors.New("logger: name already registered")
)

var (
	registryMu     sync.RWMutex
	registryByName = make(map[string]Logger)
)

// Register 注册具名 Logger。
func Register(name string, l Logger) error {
	if name == "" {
		return errEmptyLoggerName
	}
	if l == nil {
		return errNilLogger
	}
	registryMu.Lock()
	defer registryMu.Unlock()
	if _, exists := registryByName[name]; exists {
		return errLoggerRegistered
	}
	registryByName[name] = l
	return nil
}

// Get 按名称获取 Logger，若不存在返回 Nop。
func Get(name string) Logger {
	registryMu.RLock()
	l := registryByName[name]
	registryMu.RUnlock()
	if l == nil {
		return Nop()
	}
	return l
}

// Names 返回已注册的 Logger 名称列表。
func Names() []string {
	registryMu.RLock()
	defer registryMu.RUnlock()
	names := make([]string, 0, len(registryByName))
	for name := range registryByName {
		names = append(names, name)
	}
	return names
}
