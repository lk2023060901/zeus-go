package logger

import "context"

// Level 表示日志等级。
type Level int

const (
	// LevelDebug 表示调试级日志。
	LevelDebug Level = iota
	// LevelInfo 表示信息级日志。
	LevelInfo
	// LevelWarn 表示警告级日志。
	LevelWarn
	// LevelError 表示错误级日志。
	LevelError
)

// Field 表示结构化日志字段。
type Field struct {
	Key   string
	Value any
}

// Logger 定义统一日志接口，兼容 slog 的 ctx 入口与无 ctx 入口。
type Logger interface {
	// With 返回附加字段后的 Logger，便于上下文透传。
	With(fields ...Field) Logger

	// WithGroup 开启字段分组（与 slog 对齐）。
	WithGroup(name string) Logger

	// Enabled 判断给定等级在当前上下文是否可输出（与 slog 对齐）。
	Enabled(ctx context.Context, level Level) bool

	// Log 按等级记录日志（与 slog 对齐，必须带 ctx）。
	Log(ctx context.Context, level Level, msg string, fields ...Field)

	// Debug 记录调试级日志（无 ctx）。
	Debug(msg string, fields ...Field)
	// Info 记录信息级日志（无 ctx）。
	Info(msg string, fields ...Field)
	// Warn 记录警告级日志（无 ctx）。
	Warn(msg string, fields ...Field)
	// Error 记录错误级日志（无 ctx）。
	Error(msg string, fields ...Field)

	// DebugContext 记录调试级日志（带 ctx）。
	DebugContext(ctx context.Context, msg string, fields ...Field)
	// InfoContext 记录信息级日志（带 ctx）。
	InfoContext(ctx context.Context, msg string, fields ...Field)
	// WarnContext 记录警告级日志（带 ctx）。
	WarnContext(ctx context.Context, msg string, fields ...Field)
	// ErrorContext 记录错误级日志（带 ctx）。
	ErrorContext(ctx context.Context, msg string, fields ...Field)

	// Sync 刷新缓冲并落盘（若实现需要）。
	Sync() error
}
