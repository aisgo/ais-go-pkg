package shutdown

import (
	"go.uber.org/fx"
)

/* ========================================================================
 * Shutdown FX Module - 优雅关停 FX 模块
 * ========================================================================
 * 职责: 提供 FX 依赖注入支持
 * ======================================================================== */

// Module FX 模块
var Module = fx.Module("shutdown",
	fx.Provide(
		NewManager,
		func() *Config {
			return DefaultConfig()
		},
	),
)

// ManagerResult Manager 返回结果
type ManagerResult struct {
	fx.Out

	Manager *Manager
}

// ProvideManager 提供 Manager（用于 FX）
func ProvideManager(p ManagerParams) ManagerResult {
	return ManagerResult{
		Manager: NewManager(p),
	}
}
