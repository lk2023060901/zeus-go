package ticker

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tk := New(100*time.Millisecond, func() {})

	if tk.Interval() != 100*time.Millisecond {
		t.Errorf("Interval() = %v, want %v", tk.Interval(), 100*time.Millisecond)
	}
	if tk.IsRunning() {
		t.Error("IsRunning() = true, want false")
	}
}

func TestNewWithInvalidInterval(t *testing.T) {
	tk := New(0, func() {})
	if tk.Interval() != time.Second {
		t.Errorf("Interval() = %v, want %v (default)", tk.Interval(), time.Second)
	}

	tk = New(-1*time.Second, func() {})
	if tk.Interval() != time.Second {
		t.Errorf("Interval() = %v, want %v (default)", tk.Interval(), time.Second)
	}
}

func TestTickerStartStop(t *testing.T) {
	var count int64
	tk := New(10*time.Millisecond, func() {
		atomic.AddInt64(&count, 1)
	})

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- tk.Start(ctx)
	}()

	// 等待启动
	time.Sleep(20 * time.Millisecond)
	if !tk.IsRunning() {
		t.Error("IsRunning() = false after Start, want true")
	}

	// 等待几次回调
	time.Sleep(50 * time.Millisecond)

	// 停止
	cancel()
	<-done

	if tk.IsRunning() {
		t.Error("IsRunning() = true after Stop, want false")
	}

	finalCount := atomic.LoadInt64(&count)
	if finalCount < 3 {
		t.Errorf("Handler called %d times, want at least 3", finalCount)
	}
}

func TestTickerStopMethod(t *testing.T) {
	var count int64
	tk := New(10*time.Millisecond, func() {
		atomic.AddInt64(&count, 1)
	})

	ctx := context.Background()
	done := make(chan error, 1)

	go func() {
		done <- tk.Start(ctx)
	}()

	// 等待启动
	time.Sleep(20 * time.Millisecond)

	// 使用 Stop() 方法停止
	tk.Stop()
	<-done

	if tk.IsRunning() {
		t.Error("IsRunning() = true after Stop(), want false")
	}
}

func TestTickerDoubleStart(t *testing.T) {
	tk := New(100*time.Millisecond, func() {})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- tk.Start(ctx)
	}()

	// 等待启动
	time.Sleep(20 * time.Millisecond)

	// 再次启动应该立即返回
	err := tk.Start(ctx)
	if err != nil {
		t.Errorf("Second Start() returned error: %v", err)
	}

	cancel()
	<-done
}

func TestTickerStopNotRunning(t *testing.T) {
	tk := New(100*time.Millisecond, func() {})

	// 停止未运行的 ticker 不应该 panic
	tk.Stop()
}

func TestMultiTicker(t *testing.T) {
	var count1, count2 int64

	mt := NewMulti()
	mt.Add("ticker1", 10*time.Millisecond, func() {
		atomic.AddInt64(&count1, 1)
	})
	mt.Add("ticker2", 20*time.Millisecond, func() {
		atomic.AddInt64(&count2, 1)
	})

	// 检查 Names
	names := mt.Names()
	if len(names) != 2 {
		t.Errorf("Names() returned %d items, want 2", len(names))
	}

	// 检查 Get
	if mt.Get("ticker1") == nil {
		t.Error("Get(ticker1) = nil, want non-nil")
	}
	if mt.Get("nonexistent") != nil {
		t.Error("Get(nonexistent) != nil, want nil")
	}

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)

	go func() {
		done <- mt.Start(ctx)
	}()

	// 等待运行
	time.Sleep(60 * time.Millisecond)

	cancel()
	<-done

	c1 := atomic.LoadInt64(&count1)
	c2 := atomic.LoadInt64(&count2)

	if c1 < 3 {
		t.Errorf("ticker1 called %d times, want at least 3", c1)
	}
	if c2 < 1 {
		t.Errorf("ticker2 called %d times, want at least 1", c2)
	}
}

func TestMultiTickerRemove(t *testing.T) {
	mt := NewMulti()
	mt.Add("ticker1", 100*time.Millisecond, func() {})
	mt.Add("ticker2", 100*time.Millisecond, func() {})

	if len(mt.Names()) != 2 {
		t.Error("Expected 2 tickers after Add")
	}

	mt.Remove("ticker1")
	if len(mt.Names()) != 1 {
		t.Error("Expected 1 ticker after Remove")
	}

	if mt.Get("ticker1") != nil {
		t.Error("ticker1 should be nil after Remove")
	}
}

func TestMultiTickerDoubleStart(t *testing.T) {
	mt := NewMulti()
	mt.Add("ticker1", 100*time.Millisecond, func() {})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- mt.Start(ctx)
	}()

	// 等待启动
	time.Sleep(20 * time.Millisecond)

	// 再次启动应该立即返回
	err := mt.Start(ctx)
	if err != nil {
		t.Errorf("Second Start() returned error: %v", err)
	}

	cancel()
	<-done
}

func TestMultiTickerStopMethod(t *testing.T) {
	mt := NewMulti()
	mt.Add("ticker1", 10*time.Millisecond, func() {})

	ctx := context.Background()
	done := make(chan error, 1)

	go func() {
		done <- mt.Start(ctx)
	}()

	// 等待启动
	time.Sleep(20 * time.Millisecond)

	// 使用 Stop() 方法停止
	mt.Stop()
	<-done
}

func TestMultiTickerStopNotRunning(t *testing.T) {
	mt := NewMulti()
	mt.Add("ticker1", 100*time.Millisecond, func() {})

	// 停止未运行的 multi ticker 不应该 panic
	mt.Stop()
}
