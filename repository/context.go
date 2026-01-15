package repository

import (
	"context"

	"gorm.io/gorm"
)

/* ========================================================================
 * Transaction Context Helper
 * ========================================================================
 * 职责: 处理 Context 中的事务传递
 * ======================================================================== */

type ctxTxKey struct{}

// getDBFromContext 尝试从 context 中获取事务 DB
// 如果 context 中存在事务，返回事务 DB；否则返回原始 DB
// 始终会将 context 绑定到返回的 DB 实例
func getDBFromContext(ctx context.Context, originalDB *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(ctxTxKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return originalDB.WithContext(ctx)
}
