package postgres

import (
	"go.uber.org/fx"
)

/* ========================================================================
 * PostgreSQL Module
 * ========================================================================
 * 职责: 提供 PostgreSQL 依赖注入模块
 * ======================================================================== */

// Module PostgreSQL 模块
// 提供: *gorm.DB
var Module = fx.Module("postgres",
	fx.Provide(NewDB),
)
