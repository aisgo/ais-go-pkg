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
	stopChan     chan struct{} // 用于停止续期 goroutine
	stopOnce     sync.Once     // 确保 stopChan 只关闭一次
	extendCtx    context.Context
	extendCancel context.CancelFunc
	mu           sync.Mutex // 保护 extendCtx 和 extendCancel
}

// LockOption 锁选项
type LockOption struct {
	TTL          time.Duration // 锁过期时间
	RetryTimes   int           // 重试次数
	RetryDelay   time.Duration // 重试间隔
	AutoExtend   bool          // 是否自动续期
	ExtendFactor float64       // 续期触发因子（TTL 的多少比例时触发续期）
}

// DefaultLockOption 默认锁选项
func DefaultLockOption() LockOption {
	return LockOption{
		TTL:          30 * time.Second,
		RetryTimes:   5,
		RetryDelay:   100 * time.Millisecond,
		AutoExtend:   false,
		ExtendFactor: 0.5, // TTL 的 50% 时续期
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
		stopChan:   make(chan struct{}),
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
	for i := 0; i < opt.RetryTimes; i++ {
		ok, err := l.client.SetNX(ctx, l.key, l.value, l.ttl)
		if err != nil {
			return err
		}
		if ok {
			// 如果开启自动续期，启动续期 goroutine
			if opt.AutoExtend {
				l.mu.Lock()
				l.extendCtx, l.extendCancel = context.WithCancel(context.Background())
				go l.autoExtendLoop(l.extendCtx, opt.ExtendFactor)
				l.mu.Unlock()
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

// autoExtendLoop 自动续期循环
func (l *Lock) autoExtendLoop(ctx context.Context, extendFactor float64) {
	// 计算续期间隔
	interval := time.Duration(float64(l.ttl) * extendFactor)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-l.stopChan:
			return
		case <-ticker.C:
			extendCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := l.Extend(extendCtx, l.ttl)
			cancel()

			if err != nil {
				if errors.Is(err, ErrLockFailed) {
					// 锁已过期或被其他进程夺取，停止续期
					return
				}
				// 网络抖动或其他 Redis 错误，记录日志并继续下一次重试
				// 不直接 return，防止在短暂网络故障下锁意外失效
			}
		}
	}
}

// Release 释放锁
// 使用 Lua 脚本保证原子性：只有持有锁的人才能释放
func (l *Lock) Release(ctx context.Context) error {
	// 停止自动续期 goroutine
	l.mu.Lock()
	if l.extendCancel != nil {
		l.extendCancel()
	}
	l.mu.Unlock()

	// 停止自动续期 (确保只关闭一次)
	l.stopOnce.Do(func() {
		close(l.stopChan)
	})

	// Lua 脚本: 如果 value 匹配则删除
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`

	result, err := l.client.rdb.Eval(ctx, script, []string{l.key}, l.value).Int64()
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
	// Lua 脚本: 如果 value 匹配则延长过期时间
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("PEXPIRE", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	result, err := l.client.rdb.Eval(ctx, script, []string{l.key}, l.value, ttl.Milliseconds()).Int64()
	if err != nil {
		return err
	}
	if result == 0 {
		return ErrLockFailed
	}
	return nil
}
