package ticker

import (
	"context"
	"sync"
	"time"

	"github.com/lk2023060901/zeus-go/pkg/conc"
)

// Handler 定时回调函数
type Handler func()

// Ticker 定时器接口
type Ticker interface {
	// Start 启动定时器（阻塞执行）
	Start(ctx context.Context) error
	// Stop 停止定时器
	Stop()
	// IsRunning 是否正在运行
	IsRunning() bool
	// Interval 获取间隔时间
	Interval() time.Duration
}

// ticker 定时器实现
type ticker struct {
	interval time.Duration
	handler  Handler

	mu       sync.RWMutex
	running  bool
	stopCh   chan struct{}
	stoppedC chan struct{}
}

// New 创建定时器
func New(interval time.Duration, handler Handler) Ticker {
	if interval <= 0 {
		interval = time.Second
	}
	return &ticker{
		interval: interval,
		handler:  handler,
	}
}

// Start 启动定时器（阻塞执行）
func (t *ticker) Start(ctx context.Context) error {
	t.mu.Lock()
	if t.running {
		t.mu.Unlock()
		return nil
	}
	t.running = true
	t.stopCh = make(chan struct{})
	t.stoppedC = make(chan struct{})
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.running = false
		close(t.stoppedC)
		t.mu.Unlock()
	}()

	tk := time.NewTicker(t.interval)
	defer tk.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-t.stopCh:
			return nil
		case <-tk.C:
			if t.handler != nil {
				t.handler()
			}
		}
	}
}

// Stop 停止定时器
func (t *ticker) Stop() {
	t.mu.RLock()
	if !t.running {
		t.mu.RUnlock()
		return
	}
	stopCh := t.stopCh
	stoppedC := t.stoppedC
	t.mu.RUnlock()

	close(stopCh)
	<-stoppedC
}

// IsRunning 是否正在运行
func (t *ticker) IsRunning() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.running
}

// Interval 获取间隔时间
func (t *ticker) Interval() time.Duration {
	return t.interval
}

// MultiTicker 多定时器管理器
type MultiTicker interface {
	// Add 添加定时器
	Add(name string, interval time.Duration, handler Handler)
	// Remove 移除定时器
	Remove(name string)
	// Start 启动所有定时器
	Start(ctx context.Context) error
	// Stop 停止所有定时器
	Stop()
	// Get 获取指定定时器
	Get(name string) Ticker
	// Names 获取所有定时器名称
	Names() []string
}

// multiTicker 多定时器实现
type multiTicker struct {
	mu      sync.RWMutex
	tickers map[string]Ticker
	running bool
	cancel  context.CancelFunc
	futures []*conc.Future[struct{}]
}

// NewMulti 创建多定时器管理器
func NewMulti() MultiTicker {
	return &multiTicker{
		tickers: make(map[string]Ticker),
	}
}

// Add 添加定时器
func (m *multiTicker) Add(name string, interval time.Duration, handler Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.tickers[name] = New(interval, handler)
}

// Remove 移除定时器
func (m *multiTicker) Remove(name string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if tk, ok := m.tickers[name]; ok {
		tk.Stop()
		delete(m.tickers, name)
	}
}

// Start 启动所有定时器
func (m *multiTicker) Start(ctx context.Context) error {
	m.mu.Lock()
	if m.running {
		m.mu.Unlock()
		return nil
	}
	m.running = true

	ctx, m.cancel = context.WithCancel(ctx)
	tickers := make([]Ticker, 0, len(m.tickers))
	for _, tk := range m.tickers {
		tickers = append(tickers, tk)
	}
	m.futures = make([]*conc.Future[struct{}], 0, len(tickers))
	m.mu.Unlock()

	// 使用 conc.Go 启动所有定时器
	for _, tk := range tickers {
		t := tk // capture for closure
		future := conc.Go(func() (struct{}, error) {
			err := t.Start(ctx)
			return struct{}{}, err
		})
		m.mu.Lock()
		m.futures = append(m.futures, future)
		m.mu.Unlock()
	}

	// 等待 context 取消
	<-ctx.Done()

	// 等待所有 futures 完成
	m.mu.RLock()
	futures := m.futures
	m.mu.RUnlock()
	_ = conc.BlockOnAll(futures...)

	m.mu.Lock()
	m.running = false
	m.futures = nil
	m.mu.Unlock()

	return ctx.Err()
}

// Stop 停止所有定时器
func (m *multiTicker) Stop() {
	m.mu.RLock()
	if !m.running || m.cancel == nil {
		m.mu.RUnlock()
		return
	}
	cancel := m.cancel
	futures := m.futures
	m.mu.RUnlock()

	cancel()
	_ = conc.BlockOnAll(futures...)
}

// Get 获取指定定时器
func (m *multiTicker) Get(name string) Ticker {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.tickers[name]
}

// Names 获取所有定时器名称
func (m *multiTicker) Names() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	names := make([]string, 0, len(m.tickers))
	for name := range m.tickers {
		names = append(names, name)
	}
	return names
}
