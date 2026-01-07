package shutdown

import (
	"context"
	"os"
	"os/signal"
	"sort"
	"sync"
	"syscall"
	"time"

	"github.com/aisgo/ais-go-pkg/logger"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

/* ========================================================================
 * Shutdown Manager - 优雅关停管理器
 * ========================================================================
 * 职责: 管理应用的优雅关停流程，支持优先级控制和超时管理
 * 特性:
 *   - 按优先级顺序执行关停钩子
 *   - 同优先级钩子并行执行
 *   - 全局超时控制
 *   - 信号监听 (SIGINT, SIGTERM, SIGQUIT)
 * ======================================================================== */

// ShutdownHook 关停钩子函数类型
type ShutdownHook func(ctx context.Context) error

// hookEntry 钩子条目，包含名称和优先级
type hookEntry struct {
	name     string
	hook     ShutdownHook
	priority int
}

// Manager 优雅关停管理器
type Manager struct {
	config  *Config
	logger  *logger.Logger
	timeout time.Duration
	hooks   []hookEntry
	mu      sync.RWMutex
	done    chan struct{}
	once    sync.Once
}

// ManagerParams 依赖参数
type ManagerParams struct {
	fx.In

	Logger *logger.Logger
	Config *Config
}

// NewManager 创建优雅关停管理器
func NewManager(p ManagerParams) *Manager {
	cfg := p.Config
	if cfg == nil {
		cfg = DefaultConfig()
	}

	return &Manager{
		config:  cfg,
		logger:  p.Logger,
		timeout: cfg.Timeout,
		hooks:   make([]hookEntry, 0),
		done:    make(chan struct{}),
	}
}

// RegisterHook 注册关停钩子（使用默认优先级）
func (m *Manager) RegisterHook(name string, hook ShutdownHook) {
	m.RegisterHookWithPriority(name, hook, PriorityNormal)
}

// RegisterHookWithPriority 注册带优先级的关停钩子
// priority: 优先级，数值越小越先执行
// 同优先级的钩子会并行执行
func (m *Manager) RegisterHookWithPriority(name string, hook ShutdownHook, priority int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.hooks = append(m.hooks, hookEntry{
		name:     name,
		hook:     hook,
		priority: priority,
	})

	m.logger.Info("Registered shutdown hook",
		zap.String("name", name),
		zap.Int("priority", priority),
	)
}

// Wait 阻塞等待关停信号
// 监听 SIGINT, SIGTERM, SIGQUIT 信号
func (m *Manager) Wait() {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	sig := <-sigChan
	m.logger.Info("Received shutdown signal", zap.String("signal", sig.String()))

	m.Shutdown(context.Background())
}

// Shutdown 执行优雅关停
// 可以直接调用，不依赖信号
func (m *Manager) Shutdown(ctx context.Context) {
	m.once.Do(func() {
		m.performShutdown(ctx)
		close(m.done)
	})
}

// Done 返回关停完成通道
// 可用于等待关停完成
func (m *Manager) Done() <-chan struct{} {
	return m.done
}

// IsShutdown 检查是否已经关停
func (m *Manager) IsShutdown() bool {
	select {
	case <-m.done:
		return true
	default:
		return false
	}
}

// performShutdown 执行实际的关停逻辑
func (m *Manager) performShutdown(ctx context.Context) {
	shutdownCtx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	// 复制钩子列表，避免在锁中执行
	m.mu.RLock()
	hooks := make([]hookEntry, len(m.hooks))
	copy(hooks, m.hooks)
	m.mu.RUnlock()

	// 按优先级排序（优先级小的先执行）
	sort.Slice(hooks, func(i, j int) bool {
		return hooks[i].priority < hooks[j].priority
	})

	m.logger.Info("Starting graceful shutdown",
		zap.Int("hooks", len(hooks)),
		zap.Duration("timeout", m.timeout),
	)

	// 按优先级分组执行
	groups := m.groupByPriority(hooks)
	var allResults []hookResult

	for _, group := range groups {
		if shutdownCtx.Err() != nil {
			m.logger.Warn("Shutdown timeout reached, skipping remaining hooks")
			break
		}

		m.logger.Info("Executing shutdown hooks",
			zap.Int("priority", group.priority),
			zap.Int("count", len(group.hooks)),
		)

		results := m.executeHookGroup(shutdownCtx, group.hooks)
		allResults = append(allResults, results...)
	}

	m.reportResults(allResults)

	if shutdownCtx.Err() == nil {
		m.logger.Info("Graceful shutdown completed successfully")
	} else {
		m.logger.Warn("Graceful shutdown completed with timeout")
	}
}

// hookGroup 钩子分组
type hookGroup struct {
	priority int
	hooks    []hookEntry
}

// groupByPriority 按优先级分组钩子
func (m *Manager) groupByPriority(hooks []hookEntry) []hookGroup {
	if len(hooks) == 0 {
		return nil
	}

	var groups []hookGroup
	currentPriority := hooks[0].priority
	currentGroup := hookGroup{priority: currentPriority}

	for _, h := range hooks {
		if h.priority != currentPriority {
			groups = append(groups, currentGroup)
			currentPriority = h.priority
			currentGroup = hookGroup{priority: currentPriority}
		}
		currentGroup.hooks = append(currentGroup.hooks, h)
	}
	groups = append(groups, currentGroup)

	return groups
}

// executeHookGroup 并行执行同一优先级的钩子
func (m *Manager) executeHookGroup(ctx context.Context, hooks []hookEntry) []hookResult {
	errChan := make(chan hookResult, len(hooks))
	var wg sync.WaitGroup

	for _, h := range hooks {
		wg.Add(1)
		go func(entry hookEntry) {
			defer wg.Done()

			start := time.Now()
			err := entry.hook(ctx)
			duration := time.Since(start)

			errChan <- hookResult{
				name:     entry.name,
				err:      err,
				duration: duration,
			}
		}(h)
	}

	// 等待所有钩子完成，或上下文超时
	results := make([]hookResult, 0, len(hooks))
	completedCount := 0

loop:
	for completedCount < len(hooks) {
		select {
		case result, ok := <-errChan:
			if !ok {
				break loop
			}
			results = append(results, result)
			completedCount++
		case <-ctx.Done():
			m.logger.Warn("Timeout waiting for hook group completion",
				zap.Int("completed", completedCount),
				zap.Int("total", len(hooks)),
			)
			break loop
		}
	}

	return results
}

// hookResult 钩子执行结果
type hookResult struct {
	name     string
	err      error
	duration time.Duration
}

// reportResults 报告关停结果
func (m *Manager) reportResults(results []hookResult) {
	successCount := 0
	for _, result := range results {
		if result.err != nil {
			m.logger.Error("Shutdown hook failed",
				zap.String("name", result.name),
				zap.Duration("duration", result.duration),
				zap.Error(result.err),
			)
		} else {
			m.logger.Info("Shutdown hook completed",
				zap.String("name", result.name),
				zap.Duration("duration", result.duration),
			)
			successCount++
		}
	}

	m.logger.Info("Shutdown summary",
		zap.Int("succeeded", successCount),
		zap.Int("total", len(results)),
	)
}

// WaitForShutdown 阻塞等待关停完成
func (m *Manager) WaitForShutdown() {
	<-m.done
}
