package clock

import (
	"sync"
	"sync/atomic"
	"time"
)

// Config 时钟配置
type Config struct {
	// ResetHour 每日重置小时（0-23），默认 5 表示凌晨5点
	ResetHour int
	// Timezone 时区名称，默认 "Asia/Shanghai"
	Timezone string
}

// DefaultConfig 返回默认配置
func DefaultConfig() Config {
	return Config{
		ResetHour: 5,
		Timezone:  "Asia/Shanghai",
	}
}

// GameClock 游戏时钟接口
type GameClock interface {
	// ===== 时间更新 =====

	// Tick 更新缓存时间（由外部定时调用）
	Tick()

	// ===== 时间偏移（调试用） =====

	// SetOffset 设置时间偏移量
	SetOffset(d time.Duration)

	// AddOffset 增加时间偏移量（可为负数）
	AddOffset(d time.Duration)

	// ResetOffset 重置偏移为 0
	ResetOffset()

	// Offset 获取当前偏移量
	Offset() time.Duration

	// ===== 基础时间获取 =====

	// Now 获取当前时间（缓存时间 + 偏移量）
	Now() time.Time

	// NowUnix 获取当前 Unix 时间戳（秒）
	NowUnix() int64

	// NowUnixMilli 获取当前 Unix 时间戳（毫秒）
	NowUnixMilli() int64

	// RealNow 获取真实系统时间（不含缓存和偏移）
	RealNow() time.Time

	// ===== 游戏日相关 =====

	// GameDay 获取指定时间的游戏日（考虑重置时间偏移）
	GameDay(t time.Time) time.Time

	// TodayGameDay 获取今天的游戏日
	TodayGameDay() time.Time

	// IsSameGameDay 判断两个时间是否在同一游戏日
	IsSameGameDay(t1, t2 time.Time) bool

	// NextResetTime 获取下次重置时间
	NextResetTime() time.Time

	// UntilNextReset 距离下次重置的时间
	UntilNextReset() time.Duration

	// ===== 日期计算 =====

	// StartOfDay 获取某天的开始时间（00:00:00）
	StartOfDay(t time.Time) time.Time

	// EndOfDay 获取某天的结束时间（23:59:59.999999999）
	EndOfDay(t time.Time) time.Time

	// StartOfWeek 获取某周的开始时间（周一 00:00:00）
	StartOfWeek(t time.Time) time.Time

	// StartOfMonth 获取某月的开始时间
	StartOfMonth(t time.Time) time.Time

	// ===== 日期属性 =====

	// DaysInMonth 获取指定年月的天数
	DaysInMonth(year int, month time.Month) int

	// DaysOfMonth 获取当年指定月份的天数（月份 1-12）
	DaysOfMonth(month int) int

	// DaysInMonthOf 获取指定时间所在月份的天数
	DaysInMonthOf(t time.Time) int

	// IsLeapYear 判断是否为闰年
	IsLeapYear(year int) bool

	// DayOfYear 获取一年中的第几天（1-366）
	DayOfYear(t time.Time) int

	// WeekOfYear 获取一年中的第几周（ISO 8601）
	WeekOfYear(t time.Time) int

	// ===== 时间计算 =====

	// AddDays 增加/减少天数
	AddDays(t time.Time, days int) time.Time

	// DaysBetween 计算两个时间之间的天数差（基于游戏日）
	DaysBetween(t1, t2 time.Time) int

	// ===== 时间转换 =====

	// FromUnix 从 Unix 时间戳创建 time.Time
	FromUnix(sec int64) time.Time

	// FromUnixMilli 从毫秒时间戳创建 time.Time
	FromUnixMilli(msec int64) time.Time
}

// gameClock GameClock 的默认实现
type gameClock struct {
	config   Config
	location *time.Location

	// 缓存时间（原子操作）
	cachedTime atomic.Value // time.Time

	// 时间偏移（用于调试）
	mu     sync.RWMutex
	offset time.Duration
}

// New 创建游戏时钟
func New(cfg Config) (GameClock, error) {
	if cfg.Timezone == "" {
		cfg.Timezone = "Asia/Shanghai"
	}
	if cfg.ResetHour < 0 || cfg.ResetHour > 23 {
		cfg.ResetHour = 5
	}

	loc, err := time.LoadLocation(cfg.Timezone)
	if err != nil {
		return nil, err
	}

	c := &gameClock{
		config:   cfg,
		location: loc,
	}
	c.cachedTime.Store(time.Now().In(loc))

	return c, nil
}

// MustNew 创建游戏时钟，失败时 panic
func MustNew(cfg Config) GameClock {
	c, err := New(cfg)
	if err != nil {
		panic(err)
	}
	return c
}

// ===== 时间更新 =====

func (c *gameClock) Tick() {
	c.cachedTime.Store(time.Now().In(c.location))
}

// ===== 时间偏移 =====

func (c *gameClock) SetOffset(d time.Duration) {
	c.mu.Lock()
	c.offset = d
	c.mu.Unlock()
}

func (c *gameClock) AddOffset(d time.Duration) {
	c.mu.Lock()
	c.offset += d
	c.mu.Unlock()
}

func (c *gameClock) ResetOffset() {
	c.mu.Lock()
	c.offset = 0
	c.mu.Unlock()
}

func (c *gameClock) Offset() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.offset
}

// ===== 基础时间获取 =====

func (c *gameClock) Now() time.Time {
	cached := c.cachedTime.Load().(time.Time)
	c.mu.RLock()
	offset := c.offset
	c.mu.RUnlock()
	return cached.Add(offset)
}

func (c *gameClock) NowUnix() int64 {
	return c.Now().Unix()
}

func (c *gameClock) NowUnixMilli() int64 {
	return c.Now().UnixMilli()
}

func (c *gameClock) RealNow() time.Time {
	return time.Now().In(c.location)
}

// ===== 游戏日相关 =====

func (c *gameClock) GameDay(t time.Time) time.Time {
	t = t.In(c.location)
	// 如果当前时间小于重置小时，则游戏日是前一天
	if t.Hour() < c.config.ResetHour {
		t = t.AddDate(0, 0, -1)
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, c.location)
}

func (c *gameClock) TodayGameDay() time.Time {
	return c.GameDay(c.Now())
}

func (c *gameClock) IsSameGameDay(t1, t2 time.Time) bool {
	d1 := c.GameDay(t1)
	d2 := c.GameDay(t2)
	return d1.Year() == d2.Year() && d1.YearDay() == d2.YearDay()
}

func (c *gameClock) NextResetTime() time.Time {
	now := c.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), c.config.ResetHour, 0, 0, 0, c.location)

	if now.Before(today) {
		return today
	}
	return today.AddDate(0, 0, 1)
}

func (c *gameClock) UntilNextReset() time.Duration {
	return c.NextResetTime().Sub(c.Now())
}

// ===== 日期计算 =====

func (c *gameClock) StartOfDay(t time.Time) time.Time {
	t = t.In(c.location)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, c.location)
}

func (c *gameClock) EndOfDay(t time.Time) time.Time {
	t = t.In(c.location)
	return time.Date(t.Year(), t.Month(), t.Day(), 23, 59, 59, 999999999, c.location)
}

func (c *gameClock) StartOfWeek(t time.Time) time.Time {
	t = t.In(c.location)
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7 // 周日视为第7天
	}
	// 回退到周一
	t = t.AddDate(0, 0, -(weekday - 1))
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, c.location)
}

func (c *gameClock) StartOfMonth(t time.Time) time.Time {
	t = t.In(c.location)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, c.location)
}

// ===== 日期属性 =====

func (c *gameClock) DaysInMonth(year int, month time.Month) int {
	// 下个月第0天 = 当月最后一天
	return time.Date(year, month+1, 0, 0, 0, 0, 0, c.location).Day()
}

func (c *gameClock) DaysOfMonth(month int) int {
	now := c.Now()
	return c.DaysInMonth(now.Year(), time.Month(month))
}

func (c *gameClock) DaysInMonthOf(t time.Time) int {
	t = t.In(c.location)
	return c.DaysInMonth(t.Year(), t.Month())
}

func (c *gameClock) IsLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

func (c *gameClock) DayOfYear(t time.Time) int {
	return t.In(c.location).YearDay()
}

func (c *gameClock) WeekOfYear(t time.Time) int {
	_, week := t.In(c.location).ISOWeek()
	return week
}

// ===== 时间计算 =====

func (c *gameClock) AddDays(t time.Time, days int) time.Time {
	return t.In(c.location).AddDate(0, 0, days)
}

func (c *gameClock) DaysBetween(t1, t2 time.Time) int {
	d1 := c.GameDay(t1)
	d2 := c.GameDay(t2)
	return int(d2.Sub(d1).Hours() / 24)
}

// ===== 时间转换 =====

func (c *gameClock) FromUnix(sec int64) time.Time {
	return time.Unix(sec, 0).In(c.location)
}

func (c *gameClock) FromUnixMilli(msec int64) time.Time {
	return time.UnixMilli(msec).In(c.location)
}
