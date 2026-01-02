package repository

import (
	"context"

	"ais.local/ais-go-pkg/errors"

	"gorm.io/gorm"
)

/* ========================================================================
 * Transaction Repository Implementation - 事务支持实现
 * ========================================================================
 * 职责: 实现 TransactionRepository 接口
 * ======================================================================== */

// Transaction 在事务中执行操作
// 如果 fn 返回错误，事务将回滚；否则提交
func (r *RepositoryImpl[T]) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	db := r.withContext(ctx)

	if err := db.Transaction(func(tx *gorm.DB) error {
		return fn(tx)
	}); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "transaction failed", err)
	}

	return nil
}

// WithTx 创建事务版本的仓储
// 返回的仓储实例使用传入的事务 DB
func (r *RepositoryImpl[T]) WithTx(tx *gorm.DB) Repository[T] {
	return &RepositoryImpl[T]{db: tx}
}

/* ========================================================================
 * 事务辅助方法
 * ======================================================================== */

// TransactionContext 事务上下文
// 用于在复杂业务场景中传递事务
type TransactionContext struct {
	tx *gorm.DB
}

// NewTransactionContext 创建事务上下文
func NewTransactionContext(tx *gorm.DB) *TransactionContext {
	return &TransactionContext{tx: tx}
}

// GetTx 获取事务 DB
func (tc *TransactionContext) GetTx() *gorm.DB {
	return tc.tx
}

// HasTx 检查是否有事务
func (tc *TransactionContext) HasTx() bool {
	return tc.tx != nil
}

// ExecInTransaction 在事务中执行操作（使用 TransactionContext）
func (r *RepositoryImpl[T]) ExecInTransaction(ctx context.Context, fn func(tc *TransactionContext) error) error {
	db := r.withContext(ctx)

	if err := db.Transaction(func(tx *gorm.DB) error {
		return fn(&TransactionContext{tx: tx})
	}); err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "transaction failed", err)
	}

	return nil
}

// WithTxContext 创建带事务上下文的仓储
// 如果 tc 有事务，使用事务 DB；否则使用普通 DB
func (r *RepositoryImpl[T]) WithTxContext(tc *TransactionContext) Repository[T] {
	if tc != nil && tc.HasTx() {
		return &RepositoryImpl[T]{db: tc.GetTx()}
	}
	return r
}
