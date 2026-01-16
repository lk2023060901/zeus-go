package logger

import (
	"context"
	"errors"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var errEmptyLogPath = errors.New("logger: log file path is empty")

// ZapConfig 定义基于 zap 与 lumberjack 的日志配置。
type ZapConfig struct {
	// Filepath 表示日志文件路径。
	Filepath string
	// Level 表示日志输出等级。
	Level Level
	// MaxSize 表示单个日志文件的最大大小，单位为 MB。
	MaxSize int
	// MaxBackups 表示保留的旧日志文件数量。
	MaxBackups int
	// MaxAge 表示保留旧日志文件的最大天数。
	MaxAge int
	// Compress 表示是否压缩旧日志文件。
	Compress bool
}

// ZapLogger 提供基于 zap 的 Logger 实现。
type ZapLogger struct {
	base  *zap.Logger
	group string
}

// NewZapLogger 创建一个基于 zap 与 lumberjack 的 Logger 实例。
func NewZapLogger(cfg ZapConfig) (*ZapLogger, error) {
	if cfg.Filepath == "" {
		return nil, errEmptyLogPath
	}
	if cfg.MaxSize <= 0 {
		cfg.MaxSize = 100
	}
	level := toZapLevel(cfg.Level)

	writer := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Filepath,
		MaxSize:    cfg.MaxSize,
		MaxBackups: cfg.MaxBackups,
		MaxAge:     cfg.MaxAge,
		Compress:   cfg.Compress,
	})

	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	encoder := zapcore.NewJSONEncoder(encoderCfg)

	core := zapcore.NewCore(encoder, writer, level)
	base := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	return &ZapLogger{base: base}, nil
}

// With 返回附加字段后的 Logger，便于上下文透传。
func (l *ZapLogger) With(fields ...Field) Logger {
	if len(fields) == 0 {
		return l
	}
	return &ZapLogger{
		base:  l.base.With(toZapFields(l.group, fields)...),
		group: l.group,
	}
}

// WithGroup 开启字段分组（与 slog 对齐）。
func (l *ZapLogger) WithGroup(name string) Logger {
	if name == "" {
		return l
	}
	group := name
	if l.group != "" {
		group = l.group + "." + name
	}
	return &ZapLogger{
		base:  l.base,
		group: group,
	}
}

// Enabled 判断给定等级在当前上下文是否可输出（与 slog 对齐）。
func (l *ZapLogger) Enabled(_ context.Context, level Level) bool {
	return l.base.Core().Enabled(toZapLevel(level))
}

// Log 按等级记录日志（与 slog 对齐，必须带 ctx）。
func (l *ZapLogger) Log(ctx context.Context, level Level, msg string, fields ...Field) {
	if !l.Enabled(ctx, level) {
		return
	}
	l.base.Log(toZapLevel(level), msg, toZapFields(l.group, fields)...)
}

// Debug 记录调试级日志（无 ctx）。
func (l *ZapLogger) Debug(msg string, fields ...Field) {
	l.DebugContext(context.Background(), msg, fields...)
}

// Info 记录信息级日志（无 ctx）。
func (l *ZapLogger) Info(msg string, fields ...Field) {
	l.InfoContext(context.Background(), msg, fields...)
}

// Warn 记录警告级日志（无 ctx）。
func (l *ZapLogger) Warn(msg string, fields ...Field) {
	l.WarnContext(context.Background(), msg, fields...)
}

// Error 记录错误级日志（无 ctx）。
func (l *ZapLogger) Error(msg string, fields ...Field) {
	l.ErrorContext(context.Background(), msg, fields...)
}

// DebugContext 记录调试级日志（带 ctx）。
func (l *ZapLogger) DebugContext(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, LevelDebug, msg, fields...)
}

// InfoContext 记录信息级日志（带 ctx）。
func (l *ZapLogger) InfoContext(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, LevelInfo, msg, fields...)
}

// WarnContext 记录警告级日志（带 ctx）。
func (l *ZapLogger) WarnContext(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, LevelWarn, msg, fields...)
}

// ErrorContext 记录错误级日志（带 ctx）。
func (l *ZapLogger) ErrorContext(ctx context.Context, msg string, fields ...Field) {
	l.Log(ctx, LevelError, msg, fields...)
}

// Sync 刷新缓冲并落盘（若实现需要）。
func (l *ZapLogger) Sync() error {
	return l.base.Sync()
}

func toZapLevel(level Level) zapcore.Level {
	switch level {
	case LevelDebug:
		return zapcore.DebugLevel
	case LevelInfo:
		return zapcore.InfoLevel
	case LevelWarn:
		return zapcore.WarnLevel
	case LevelError:
		return zapcore.ErrorLevel
	default:
		return zapcore.InfoLevel
	}
}

func toZapFields(group string, fields []Field) []zap.Field {
	if len(fields) == 0 {
		return nil
	}
	zapFields := make([]zap.Field, 0, len(fields))
	for _, field := range fields {
		key := field.Key
		if group != "" {
			key = group + "." + key
		}
		if err, ok := field.Value.(error); ok {
			zapFields = append(zapFields, zap.NamedError(key, err))
			continue
		}
		zapFields = append(zapFields, zap.Any(key, field.Value))
	}
	return zapFields
}

var _ Logger = (*ZapLogger)(nil)
