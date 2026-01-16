package app

import (
	"context"
	"errors"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/lk2023060901/zeus-go/pkg/logger"
	"github.com/lk2023060901/zeus-go/pkg/module"
	"github.com/lk2023060901/zeus-go/pkg/service"
)

var (
	errNilModule        = errors.New("app: module is nil")
	errNilService       = errors.New("app: service is nil")
	errRegisterLocked   = errors.New("app: register is locked after init")
	errApplicationAlive = errors.New("app: application already started")
	errConfigLocked     = errors.New("app: config is locked after init")
)

// Application 定义应用的生命周期与注册入口。
type Application interface {
	// Name 返回应用名称，用于标识当前应用实例。
	Name() string

	// RegisterModule 注册一个模块，供应用统一管理其生命周期。
	RegisterModule(m module.Module) error

	// RegisterService 注册一个服务，供应用统一管理其生命周期。
	RegisterService(s service.Service) error

	// Init 初始化应用及其所有模块、服务。
	Init(ctx context.Context) error

	// Start 启动应用及其所有模块、服务。
	Start(ctx context.Context) error

	// Run 启动应用并阻塞运行，直到收到退出信号或上下文取消。
	Run(ctx context.Context) error

	// Shutdown 触发应用的优雅关闭流程。
	Shutdown(ctx context.Context) error

	// Stop 停止应用及其所有模块、服务。
	Stop(ctx context.Context) error

	// Modules 返回已注册的模块列表。
	Modules() []module.Module

	// Services 返回已注册的服务列表。
	Services() []service.Service
}

// BaseApplication 提供 Application 的基础实现。
type BaseApplication struct {
	name string

	mu           sync.RWMutex
	modules      []module.Module
	services     []service.Service
	configPath   string
	initializing bool
	initialized  bool
	started      bool

	shutdownOnce sync.Once
	shutdownCh   chan struct{}
	shutdownErr  error
}

// NewBaseApplication 创建一个基础应用实例。
func NewBaseApplication(name string) *BaseApplication {
	return &BaseApplication{
		name:       name,
		shutdownCh: make(chan struct{}),
	}
}

// Name 返回应用名称，用于标识当前应用实例。
func (a *BaseApplication) Name() string {
	return a.name
}

// RegisterModule 注册一个模块，供应用统一管理其生命周期。
func (a *BaseApplication) RegisterModule(m module.Module) error {
	if m == nil {
		return errNilModule
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.initializing || a.initialized || a.started {
		return errRegisterLocked
	}
	a.modules = append(a.modules, m)
	return nil
}

// RegisterService 注册一个服务，供应用统一管理其生命周期。
func (a *BaseApplication) RegisterService(s service.Service) error {
	if s == nil {
		return errNilService
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.initializing || a.initialized || a.started {
		return errRegisterLocked
	}
	a.services = append(a.services, s)
	return nil
}

// SetConfigPath 设置应用配置文件路径，需在 Init 前调用。
func (a *BaseApplication) SetConfigPath(path string) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.initializing || a.initialized || a.started {
		return errConfigLocked
	}
	a.configPath = path
	return nil
}

// Init 初始化应用及其所有模块、服务。
func (a *BaseApplication) Init(ctx context.Context) error {
	a.mu.Lock()
	if a.initialized {
		a.mu.Unlock()
		return nil
	}
	if a.initializing {
		a.mu.Unlock()
		return errApplicationAlive
	}
	a.initializing = true
	modules := append([]module.Module(nil), a.modules...)
	services := append([]service.Service(nil), a.services...)
	configPath := a.configPath
	a.mu.Unlock()

	if err := a.initLogger(configPath); err != nil {
		a.mu.Lock()
		a.initializing = false
		a.mu.Unlock()
		return err
	}

	for _, m := range modules {
		if err := m.Init(ctx); err != nil {
			a.mu.Lock()
			a.initializing = false
			a.mu.Unlock()
			return err
		}
	}
	for _, s := range services {
		if err := s.Init(ctx); err != nil {
			a.mu.Lock()
			a.initializing = false
			a.mu.Unlock()
			return err
		}
	}

	a.mu.Lock()
	a.initializing = false
	a.initialized = true
	a.mu.Unlock()
	return nil
}

// Start 启动应用及其所有模块、服务。
func (a *BaseApplication) Start(ctx context.Context) error {
	if err := a.Init(ctx); err != nil {
		return err
	}

	a.mu.Lock()
	if a.started {
		a.mu.Unlock()
		return nil
	}
	modules := append([]module.Module(nil), a.modules...)
	services := append([]service.Service(nil), a.services...)
	a.mu.Unlock()

	var startedModules []module.Module
	for _, m := range modules {
		if err := m.Start(ctx); err != nil {
			a.rollbackModules(ctx, startedModules)
			return err
		}
		startedModules = append(startedModules, m)
	}

	var startedServices []service.Service
	for _, s := range services {
		if err := s.Start(ctx); err != nil {
			a.rollbackServices(ctx, startedServices)
			a.rollbackModules(ctx, startedModules)
			return err
		}
		startedServices = append(startedServices, s)
	}

	a.mu.Lock()
	a.started = true
	a.mu.Unlock()
	return nil
}

// Run 启动应用并阻塞运行，直到收到退出信号或上下文取消。
func (a *BaseApplication) Run(ctx context.Context) error {
	if err := a.Start(ctx); err != nil {
		return err
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	select {
	case <-ctx.Done():
		_ = a.Shutdown(context.Background())
		return ctx.Err()
	case <-sigCh:
		_ = a.Shutdown(context.Background())
		return a.shutdownError()
	case <-a.shutdownCh:
		return a.shutdownError()
	}
}

// Shutdown 触发应用的优雅关闭流程。
func (a *BaseApplication) Shutdown(ctx context.Context) error {
	a.shutdownOnce.Do(func() {
		err := a.Stop(ctx)
		a.mu.Lock()
		a.shutdownErr = err
		a.mu.Unlock()
		close(a.shutdownCh)
	})
	return a.shutdownError()
}

// Stop 停止应用及其所有模块、服务。
func (a *BaseApplication) Stop(ctx context.Context) error {
	a.mu.Lock()
	if !a.started {
		a.mu.Unlock()
		return nil
	}
	services := append([]service.Service(nil), a.services...)
	modules := append([]module.Module(nil), a.modules...)
	a.mu.Unlock()

	var stopErr error
	for i := len(services) - 1; i >= 0; i-- {
		if err := services[i].Stop(ctx); err != nil && stopErr == nil {
			stopErr = err
		}
	}
	for i := len(modules) - 1; i >= 0; i-- {
		if err := modules[i].Stop(ctx); err != nil && stopErr == nil {
			stopErr = err
		}
	}

	a.mu.Lock()
	a.started = false
	a.mu.Unlock()
	return stopErr
}

// Modules 返回已注册的模块列表。
func (a *BaseApplication) Modules() []module.Module {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]module.Module(nil), a.modules...)
}

// Services 返回已注册的服务列表。
func (a *BaseApplication) Services() []service.Service {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return append([]service.Service(nil), a.services...)
}

func (a *BaseApplication) rollbackModules(ctx context.Context, modules []module.Module) {
	for i := len(modules) - 1; i >= 0; i-- {
		_ = modules[i].Stop(ctx)
	}
}

func (a *BaseApplication) rollbackServices(ctx context.Context, services []service.Service) {
	for i := len(services) - 1; i >= 0; i-- {
		_ = services[i].Stop(ctx)
	}
}

func (a *BaseApplication) shutdownError() error {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.shutdownErr
}

func (a *BaseApplication) initLogger(path string) error {
	if path != "" {
		loaded, err := LoadConfigFromFile(path)
		if err != nil {
			return err
		}
		return a.initLoggerFromConfig(loaded)
	}
	return nil
}

func (a *BaseApplication) initLoggerFromConfig(cfg Config) error {
	if len(cfg.Loggers) == 0 {
		return nil
	}
	return logger.InitFromConfig(logger.Config{Loggers: cfg.Loggers})
}
