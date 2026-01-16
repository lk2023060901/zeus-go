package service

import "context"

// Service 定义可插拔服务的生命周期。
type Service interface {
	// ID 返回服务的唯一标识。
	ID() string
	// Requires 返回该服务所依赖的服务 ID 列表。
	Requires() []string
	// Init 初始化服务。
	Init(ctx context.Context) error
	// Start 启动服务。
	Start(ctx context.Context) error
	// Stop 停止服务并释放资源。
	Stop(ctx context.Context) error
}
