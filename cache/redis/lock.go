package redis

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

/* ========================================================================
 * 分布式锁 - 基于 Redis 的 Redlock 简化实现
 * ========================================================================
 * 职责: 防止高并发下的资源竞争
 * 使用场景: 分布式系统中的并发控制
 * ======================================================================== */

var (
	ErrLockFailed   = errors.New("failed to acquire lock")
	ErrUnlockFailed = errors.New("failed to release lock")
)

// Lock 分布式锁
type Lock struct {
	client       *Client
	key          string
	value        string // 唯一标识，防止误删
	ttl          time.Duration
	defaultOpt   LockOption
	extendCtx    context.Context
	extendCancel context.CancelFunc
	mu           sync.Mutex // 保护 extendCtx 和 extendCancel
}

// LockOption 锁选项
type LockOption struct {
	TTL                time.Duration // 锁过期时间
	RetryTimes         int           // 重试次数
	RetryDelay         time.Duration // 重试间隔
	AutoExtend         bool          // 是否自动续期
	ExtendFactor       float64       // 续期触发因子（TTL 的多少比例时触发续期）
	MaxLifetime        time.Duration // 自动续期最大生命周期（<=0 使用默认值 TTL*10）
	IgnoreParentCancel bool          // 是否忽略父 context 的取消信号
}

// DefaultLockOption 默认锁选项
func DefaultLockOption() LockOption {
	return LockOption{
		TTL:          30 * time.Second,
		RetryTimes:   5,
		RetryDelay:   100 * time.Millisecond,
		AutoExtend:   false,
		ExtendFactor: 0.5, // TTL 的 50% 时续期
		MaxLifetime:  0,
	}
}

// NewLock 创建分布式锁
func (c *Client) NewLock(key string, opts ...LockOption) *Lock {
	opt := DefaultLockOption()
	if len(opts) > 0 {
		opt = opts[0]
	}

	return &Lock{
		client:     c,
		key:        "lock:" + key,
		value:      uuid.New().String(),
		ttl:        opt.TTL,
		defaultOpt: opt,
	}
}

// Acquire 获取锁
func (l *Lock) Acquire(ctx context.Context) error {
	return l.AcquireWithOption(ctx, l.defaultOpt)
}

// AcquireWithOption 带选项获取锁
func (l *Lock) AcquireWithOption(ctx context.Context, opt LockOption) error {
	if opt.TTL > 0 {
		l.ttl = opt.TTL
	}
	value := uuid.New().String()
	for i := 0; i < opt.RetryTimes; i++ {
		ok, err := l.client.SetNX(ctx, l.key, value, l.ttl)
		if err != nil {
			return err
		}
		if ok {
			l.mu.Lock()
			l.value = value
			l.mu.Unlock()
			// 如果开启自动续期，启动续期 goroutine
			if opt.AutoExtend {
				l.startAutoExtend(ctx, opt.ExtendFactor, opt.MaxLifetime, opt.IgnoreParentCancel)
			}
			return nil
		}

		// 等待重试
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opt.RetryDelay):
		}
	}

	return ErrLockFailed
}

// startAutoExtend 启动自动续期（线程安全）
func (l *Lock) startAutoExtend(parentCtx context.Context, extendFactor float64, maxLifetime time.Duration, ignoreParentCancel bool) {
	// 先停止旧的续期 goroutine（如果存在）
	l.stopAutoExtend()

	l.mu.Lock()
	defer l.mu.Unlock()

	if extendFactor <= 0 || extendFactor > 1 {
		extendFactor = DefaultLockOption().ExtendFactor
	}

	if maxLifetime <= 0 {
		maxLifetime = l.ttl * 10
	}

	// 默认继承父 context 的取消信号
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx := parentCtx
	if ignoreParentCancel {
		ctx = context.WithoutCancel(parentCtx)
	}
	ctx, cancel := context.WithCancel(ctx)
	l.extendCtx, l.extendCancel = ctx, cancel

	go l.autoExtendLoop(l.extendCtx, extendFactor, maxLifetime)
}

// stopAutoExtend 停止自动续期（线程安全）
func (l *Lock) stopAutoExtend() {
	l.mu.Lock()
	cancel := l.extendCancel
	l.extendCancel = nil
	l.extendCtx = nil
	l.mu.Unlock()

	if cancel != nil {
		cancel()
	}
}

// autoExtendLoop 自动续期循环
func (l *Lock) autoExtendLoop(ctx context.Context, extendFactor float64, maxLifetime time.Duration) {
	// 计算续期间隔
	interval := time.Duration(float64(l.ttl) * extendFactor)

	// 添加最大生命周期保护（防止无限续期导致 goroutine 泄漏）
	deadlineCtx, deadlineCancel := context.WithTimeout(ctx, maxLifetime)
	defer deadlineCancel()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-deadlineCtx.Done():
			// 超过最大生命周期或被取消
			return

		case <-ticker.C:
			// 尝试续期
			if !l.tryExtend(deadlineCtx) {
				// 续期失败，可能锁已丢失
				return
			}
		}
	}
}

// tryExtend 尝试续期，返回是否应继续
func (l *Lock) tryExtend(ctx context.Context) bool {
	for i := 0; i < 3; i++ {
		extendCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		err := l.Extend(extendCtx, l.ttl)
		cancel()

		if err == nil {
			return true
		}

		// 锁已丢失或上下文已取消
		if errors.Is(err, ErrLockFailed) || errors.Is(err, context.Canceled) {
			return false
		}

		// 临时错误，指数退避
		backoff := time.Duration(100*(1<<i)) * time.Millisecond
		select {
		case <-ctx.Done():
			return false
		case <-time.After(backoff):
			continue
		}
	}

	// 重试多次仍失败
	return false
}

// Release 释放锁
// 使用 Lua 脚本保证原子性：只有持有锁的人才能释放
func (l *Lock) Release(ctx context.Context) error {
	// 停止自动续期 goroutine
	l.stopAutoExtend()

	l.mu.Lock()
	value := l.value
	l.mu.Unlock()

	// Lua 脚本: 如果 value 匹配则删除
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.rdb.Eval(ctx, script, []string{l.key}, value).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrUnlockFailed
	}
	return nil
}

// Extend 延长锁时间
func (l *Lock) Extend(ctx context.Context, ttl time.Duration) error {
	l.mu.Lock()
	value := l.value
	l.mu.Unlock()

	// Lua 脚本: 如果 value 匹配则延长过期时间
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.rdb.Eval(ctx, script, []string{l.key}, value, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrLockFailed
	}
	return nil
}
