package clock

import (
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	tests := []struct {
		name    string
		cfg     Config
		wantErr bool
	}{
		{
			name:    "default config",
			cfg:     DefaultConfig(),
			wantErr: false,
		},
		{
			name:    "empty timezone uses default",
			cfg:     Config{ResetHour: 5, Timezone: ""},
			wantErr: false,
		},
		{
			name:    "invalid timezone",
			cfg:     Config{ResetHour: 5, Timezone: "Invalid/Zone"},
			wantErr: true,
		},
		{
			name:    "invalid reset hour gets corrected",
			cfg:     Config{ResetHour: 25, Timezone: "Asia/Shanghai"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := New(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("New() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTick(t *testing.T) {
	c := MustNew(DefaultConfig())

	t1 := c.Now()
	time.Sleep(10 * time.Millisecond)
	t2 := c.Now()

	// 未调用 Tick，时间应该相同
	if !t1.Equal(t2) {
		t.Errorf("Now() should return cached time, got t1=%v, t2=%v", t1, t2)
	}

	// 调用 Tick 后时间应该更新
	time.Sleep(10 * time.Millisecond)
	c.Tick()
	t3 := c.Now()

	if t3.Before(t1) || t3.Equal(t1) {
		t.Errorf("After Tick(), Now() should return updated time")
	}
}

func TestOffset(t *testing.T) {
	c := MustNew(DefaultConfig())
	c.Tick()

	// 初始偏移为 0
	if c.Offset() != 0 {
		t.Errorf("Initial offset should be 0, got %v", c.Offset())
	}

	// 设置偏移
	c.SetOffset(time.Hour)
	if c.Offset() != time.Hour {
		t.Errorf("Offset should be 1h, got %v", c.Offset())
	}

	// 增加偏移
	c.AddOffset(30 * time.Minute)
	if c.Offset() != 90*time.Minute {
		t.Errorf("Offset should be 90m, got %v", c.Offset())
	}

	// 减少偏移
	c.AddOffset(-20 * time.Minute)
	if c.Offset() != 70*time.Minute {
		t.Errorf("Offset should be 70m, got %v", c.Offset())
	}

	// 重置偏移
	c.ResetOffset()
	if c.Offset() != 0 {
		t.Errorf("Offset should be 0 after reset, got %v", c.Offset())
	}
}

func TestNowWithOffset(t *testing.T) {
	c := MustNew(DefaultConfig())
	c.Tick()

	base := c.Now()
	c.AddOffset(24 * time.Hour)
	withOffset := c.Now()

	diff := withOffset.Sub(base)
	if diff != 24*time.Hour {
		t.Errorf("Now() with 24h offset should differ by 24h, got %v", diff)
	}
}

func TestRealNow(t *testing.T) {
	c := MustNew(DefaultConfig())
	c.AddOffset(24 * time.Hour)

	real := c.RealNow()
	now := time.Now()

	// RealNow 应该接近系统时间（差距不超过 1 秒）
	if real.Sub(now).Abs() > time.Second {
		t.Errorf("RealNow() should be close to system time")
	}
}

func TestGameDay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ResetHour = 5
	c := MustNew(cfg)

	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "before reset hour",
			input:    time.Date(2026, 1, 6, 4, 30, 0, 0, loc),
			expected: time.Date(2026, 1, 5, 0, 0, 0, 0, loc),
		},
		{
			name:     "after reset hour",
			input:    time.Date(2026, 1, 6, 10, 30, 0, 0, loc),
			expected: time.Date(2026, 1, 6, 0, 0, 0, 0, loc),
		},
		{
			name:     "exactly at reset hour",
			input:    time.Date(2026, 1, 6, 5, 0, 0, 0, loc),
			expected: time.Date(2026, 1, 6, 0, 0, 0, 0, loc),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.GameDay(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("GameDay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsSameGameDay(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ResetHour = 5
	c := MustNew(cfg)

	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected bool
	}{
		{
			name:     "same calendar day, both after reset",
			t1:       time.Date(2026, 1, 6, 10, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 6, 23, 0, 0, 0, loc),
			expected: true,
		},
		{
			name:     "different calendar day, same game day",
			t1:       time.Date(2026, 1, 6, 23, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 7, 4, 0, 0, 0, loc),
			expected: true,
		},
		{
			name:     "different game days",
			t1:       time.Date(2026, 1, 6, 4, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 6, 6, 0, 0, 0, loc),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.IsSameGameDay(tt.t1, tt.t2)
			if result != tt.expected {
				t.Errorf("IsSameGameDay() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestNextResetTime(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ResetHour = 5
	c, _ := New(cfg)

	loc, _ := time.LoadLocation("Asia/Shanghai")

	// 设置时间为 2026-01-06 10:00
	gc := c.(*gameClock)
	gc.cachedTime.Store(time.Date(2026, 1, 6, 10, 0, 0, 0, loc))

	next := c.NextResetTime()
	expected := time.Date(2026, 1, 7, 5, 0, 0, 0, loc)
	if !next.Equal(expected) {
		t.Errorf("NextResetTime() = %v, want %v", next, expected)
	}

	// 设置时间为 2026-01-06 04:00（重置前）
	gc.cachedTime.Store(time.Date(2026, 1, 6, 4, 0, 0, 0, loc))
	next = c.NextResetTime()
	expected = time.Date(2026, 1, 6, 5, 0, 0, 0, loc)
	if !next.Equal(expected) {
		t.Errorf("NextResetTime() before reset = %v, want %v", next, expected)
	}
}

func TestStartOfDay(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	input := time.Date(2026, 1, 6, 14, 30, 45, 123456789, loc)
	result := c.StartOfDay(input)
	expected := time.Date(2026, 1, 6, 0, 0, 0, 0, loc)

	if !result.Equal(expected) {
		t.Errorf("StartOfDay() = %v, want %v", result, expected)
	}
}

func TestEndOfDay(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	input := time.Date(2026, 1, 6, 14, 30, 45, 0, loc)
	result := c.EndOfDay(input)
	expected := time.Date(2026, 1, 6, 23, 59, 59, 999999999, loc)

	if !result.Equal(expected) {
		t.Errorf("EndOfDay() = %v, want %v", result, expected)
	}
}

func TestStartOfWeek(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name     string
		input    time.Time
		expected time.Time
	}{
		{
			name:     "wednesday",
			input:    time.Date(2026, 1, 7, 14, 0, 0, 0, loc), // Wednesday
			expected: time.Date(2026, 1, 5, 0, 0, 0, 0, loc),  // Monday
		},
		{
			name:     "monday",
			input:    time.Date(2026, 1, 5, 10, 0, 0, 0, loc), // Monday
			expected: time.Date(2026, 1, 5, 0, 0, 0, 0, loc),  // Monday
		},
		{
			name:     "sunday",
			input:    time.Date(2026, 1, 11, 10, 0, 0, 0, loc), // Sunday
			expected: time.Date(2026, 1, 5, 0, 0, 0, 0, loc),   // Monday
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.StartOfWeek(tt.input)
			if !result.Equal(tt.expected) {
				t.Errorf("StartOfWeek() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestStartOfMonth(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	input := time.Date(2026, 1, 15, 14, 30, 0, 0, loc)
	result := c.StartOfMonth(input)
	expected := time.Date(2026, 1, 1, 0, 0, 0, 0, loc)

	if !result.Equal(expected) {
		t.Errorf("StartOfMonth() = %v, want %v", result, expected)
	}
}

func TestDaysInMonth(t *testing.T) {
	c := MustNew(DefaultConfig())

	tests := []struct {
		year     int
		month    time.Month
		expected int
	}{
		{2026, time.January, 31},
		{2026, time.February, 28},
		{2024, time.February, 29}, // 闰年
		{2026, time.April, 30},
		{2026, time.December, 31},
	}

	for _, tt := range tests {
		t.Run(tt.month.String(), func(t *testing.T) {
			result := c.DaysInMonth(tt.year, tt.month)
			if result != tt.expected {
				t.Errorf("DaysInMonth(%d, %s) = %d, want %d", tt.year, tt.month, result, tt.expected)
			}
		})
	}
}

func TestDaysOfMonth(t *testing.T) {
	cfg := DefaultConfig()
	c, _ := New(cfg)

	loc, _ := time.LoadLocation("Asia/Shanghai")
	gc := c.(*gameClock)
	// 设置时间为 2026 年
	gc.cachedTime.Store(time.Date(2026, 6, 15, 10, 0, 0, 0, loc))

	tests := []struct {
		month    int
		expected int
	}{
		{1, 31},  // January
		{2, 28},  // February (2026 非闰年)
		{4, 30},  // April
		{12, 31}, // December
	}

	for _, tt := range tests {
		result := c.DaysOfMonth(tt.month)
		if result != tt.expected {
			t.Errorf("DaysOfMonth(%d) = %d, want %d", tt.month, result, tt.expected)
		}
	}

	// 测试闰年
	gc.cachedTime.Store(time.Date(2024, 6, 15, 10, 0, 0, 0, loc))
	if days := c.DaysOfMonth(2); days != 29 {
		t.Errorf("DaysOfMonth(2) in 2024 = %d, want 29", days)
	}
}

func TestDaysInMonthOf(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		time     time.Time
		expected int
	}{
		{time.Date(2026, 1, 15, 0, 0, 0, 0, loc), 31},
		{time.Date(2026, 2, 10, 0, 0, 0, 0, loc), 28},
		{time.Date(2024, 2, 10, 0, 0, 0, 0, loc), 29}, // 闰年
		{time.Date(2026, 4, 1, 0, 0, 0, 0, loc), 30},
	}

	for _, tt := range tests {
		result := c.DaysInMonthOf(tt.time)
		if result != tt.expected {
			t.Errorf("DaysInMonthOf(%v) = %d, want %d", tt.time, result, tt.expected)
		}
	}
}

func TestIsLeapYear(t *testing.T) {
	c := MustNew(DefaultConfig())

	tests := []struct {
		year     int
		expected bool
	}{
		{2024, true},  // 能被4整除
		{2025, false}, // 不能被4整除
		{2000, true},  // 能被400整除
		{1900, false}, // 能被100整除但不能被400整除
		{2026, false},
	}

	for _, tt := range tests {
		t.Run(string(rune(tt.year)), func(t *testing.T) {
			result := c.IsLeapYear(tt.year)
			if result != tt.expected {
				t.Errorf("IsLeapYear(%d) = %v, want %v", tt.year, result, tt.expected)
			}
		})
	}
}

func TestDayOfYear(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		date     time.Time
		expected int
	}{
		{time.Date(2026, 1, 1, 0, 0, 0, 0, loc), 1},
		{time.Date(2026, 2, 1, 0, 0, 0, 0, loc), 32},
		{time.Date(2026, 12, 31, 0, 0, 0, 0, loc), 365},
		{time.Date(2024, 12, 31, 0, 0, 0, 0, loc), 366}, // 闰年
	}

	for _, tt := range tests {
		t.Run(tt.date.Format("2006-01-02"), func(t *testing.T) {
			result := c.DayOfYear(tt.date)
			if result != tt.expected {
				t.Errorf("DayOfYear() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestWeekOfYear(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	// 2026-01-05 是周一，第2周
	input := time.Date(2026, 1, 5, 0, 0, 0, 0, loc)
	result := c.WeekOfYear(input)
	if result != 2 {
		t.Errorf("WeekOfYear() = %d, want 2", result)
	}
}

func TestAddDays(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	base := time.Date(2026, 1, 6, 10, 0, 0, 0, loc)

	tests := []struct {
		days     int
		expected time.Time
	}{
		{1, time.Date(2026, 1, 7, 10, 0, 0, 0, loc)},
		{-1, time.Date(2026, 1, 5, 10, 0, 0, 0, loc)},
		{30, time.Date(2026, 2, 5, 10, 0, 0, 0, loc)},
	}

	for _, tt := range tests {
		result := c.AddDays(base, tt.days)
		if !result.Equal(tt.expected) {
			t.Errorf("AddDays(%d) = %v, want %v", tt.days, result, tt.expected)
		}
	}
}

func TestDaysBetween(t *testing.T) {
	cfg := DefaultConfig()
	cfg.ResetHour = 5
	c := MustNew(cfg)

	loc, _ := time.LoadLocation("Asia/Shanghai")

	tests := []struct {
		name     string
		t1       time.Time
		t2       time.Time
		expected int
	}{
		{
			name:     "same game day",
			t1:       time.Date(2026, 1, 6, 10, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 6, 20, 0, 0, 0, loc),
			expected: 0,
		},
		{
			name:     "one game day apart",
			t1:       time.Date(2026, 1, 6, 10, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 7, 10, 0, 0, 0, loc),
			expected: 1,
		},
		{
			name:     "cross midnight same game day",
			t1:       time.Date(2026, 1, 6, 23, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 7, 4, 0, 0, 0, loc),
			expected: 0,
		},
		{
			name:     "negative days",
			t1:       time.Date(2026, 1, 10, 10, 0, 0, 0, loc),
			t2:       time.Date(2026, 1, 6, 10, 0, 0, 0, loc),
			expected: -4,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.DaysBetween(tt.t1, tt.t2)
			if result != tt.expected {
				t.Errorf("DaysBetween() = %d, want %d", result, tt.expected)
			}
		})
	}
}

func TestFromUnix(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	// 2026-01-06 10:00:00 +0800 的 Unix 时间戳
	expected := time.Date(2026, 1, 6, 10, 0, 0, 0, loc)
	unix := expected.Unix()

	result := c.FromUnix(unix)
	if !result.Equal(expected) {
		t.Errorf("FromUnix() = %v, want %v", result, expected)
	}
}

func TestFromUnixMilli(t *testing.T) {
	c := MustNew(DefaultConfig())
	loc, _ := time.LoadLocation("Asia/Shanghai")

	expected := time.Date(2026, 1, 6, 10, 0, 0, 123000000, loc)
	milli := expected.UnixMilli()

	result := c.FromUnixMilli(milli)
	if !result.Equal(expected) {
		t.Errorf("FromUnixMilli() = %v, want %v", result, expected)
	}
}
