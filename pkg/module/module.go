package module

import "context"

// Module 定义可插拔模块的生命周期。
type Module interface {
	// ID 返回模块的唯一标识。
	ID() string
	// Version 返回模块版本信息。
	Version() string
	// Requires 返回该模块所依赖的模块 ID 列表。
	Requires() []string
	// Init 初始化模块。
	Init(ctx context.Context) error
	// Start 启动模块。
	Start(ctx context.Context) error
	// Stop 停止模块并释放资源。
	Stop(ctx context.Context) error
}
