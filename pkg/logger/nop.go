package logger

import "context"

// Nop 返回一个不会输出任何日志的 Logger。
func Nop() Logger {
	return nop
}

type nopLogger struct{}

func (nopLogger) With(_ ...Field) Logger {
	return nop
}

func (nopLogger) WithGroup(_ string) Logger {
	return nop
}

func (nopLogger) Enabled(_ context.Context, _ Level) bool {
	return false
}

func (nopLogger) Log(_ context.Context, _ Level, _ string, _ ...Field) {}

func (nopLogger) Debug(_ string, _ ...Field) {}

func (nopLogger) Info(_ string, _ ...Field) {}

func (nopLogger) Warn(_ string, _ ...Field) {}

func (nopLogger) Error(_ string, _ ...Field) {}

func (nopLogger) DebugContext(_ context.Context, _ string, _ ...Field) {}

func (nopLogger) InfoContext(_ context.Context, _ string, _ ...Field) {}

func (nopLogger) WarnContext(_ context.Context, _ string, _ ...Field) {}

func (nopLogger) ErrorContext(_ context.Context, _ string, _ ...Field) {}

func (nopLogger) Sync() error {
	return nil
}

var nop Logger = nopLogger{}
