package repository

import (
	"context"
	"fmt"

	"ais.local/ais-go-pkg/errors"

	"gorm.io/gorm"
)

/* ========================================================================
 * CRUD Repository Implementation - CRUD 操作实现
 * ========================================================================
 * 职责: 实现 CRUDRepository 接口
 * ======================================================================== */

const (
	// DefaultBatchSize 默认批量操作大小
	DefaultBatchSize = 100
)

// RepositoryImpl 仓储实现
type RepositoryImpl[T any] struct {
	db *gorm.DB
}

// NewRepository 创建新的仓储实例
func NewRepository[T any](db *gorm.DB) Repository[T] {
	return &RepositoryImpl[T]{db: db}
}

// GetDB 获取底层 GORM DB 实例
func (r *RepositoryImpl[T]) GetDB() *gorm.DB {
	return r.db
}

// newModelPtr 创建新的模型指针
func (r *RepositoryImpl[T]) newModelPtr() *T {
	var model T
	return &model
}

// withContext 返回带 context 的 DB
func (r *RepositoryImpl[T]) withContext(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx)
}

/* ========================================================================
 * Create 操作
 * ======================================================================== */

// Create 创建单条记录
func (r *RepositoryImpl[T]) Create(ctx context.Context, model *T) error {
	if model == nil {
		return errors.New(errors.ErrCodeInvalidArgument, "model cannot be nil")
	}

	if err := r.withContext(ctx).Create(model).Error; err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to create record", err)
	}
	return nil
}

// CreateBatch 批量创建记录
func (r *RepositoryImpl[T]) CreateBatch(ctx context.Context, models []*T, batchSize int) error {
	if len(models) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "models cannot be empty")
	}

	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// 过滤 nil 模型
	var validModels []*T
	for _, m := range models {
		if m != nil {
			validModels = append(validModels, m)
		}
	}

	if len(validModels) == 0 {
		return nil
	}

	if err := r.withContext(ctx).CreateInBatches(validModels, batchSize).Error; err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to create records", err)
	}
	return nil
}

/* ========================================================================
 * Update 操作
 * ======================================================================== */

// Update 更新记录（根据主键）
func (r *RepositoryImpl[T]) Update(ctx context.Context, model *T) error {
	if model == nil {
		return errors.New(errors.ErrCodeInvalidArgument, "model cannot be nil")
	}

	result := r.withContext(ctx).Model(model).Updates(model)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to update record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}

// UpdateByID 根据 ID 更新指定字段
func (r *RepositoryImpl[T]) UpdateByID(ctx context.Context, id int64, updates map[string]any) error {
	if len(updates) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "updates cannot be empty")
	}

	model := r.newModelPtr()
	result := r.withContext(ctx).Model(model).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to update record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}

// UpdateBatch 批量更新记录
func (r *RepositoryImpl[T]) UpdateBatch(ctx context.Context, models []*T) error {
	if len(models) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "models cannot be empty")
	}

	// 使用事务执行批量更新
	return r.withContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i, model := range models {
			// 每处理一定数量后检查 context
			if i > 0 && i%100 == 0 {
				if ctx.Err() != nil {
					return ctx.Err()
				}
			}

			if model == nil {
				continue
			}

			if err := tx.Model(model).Updates(model).Error; err != nil {
				return errors.Wrap(errors.ErrCodeInternal, fmt.Sprintf("failed to update model at index %d", i), err)
			}
		}
		return nil
	})
}

/* ========================================================================
 * Delete 操作
 * ======================================================================== */

// Delete 软删除记录（设置 deleted_at）
func (r *RepositoryImpl[T]) Delete(ctx context.Context, id int64) error {
	model := r.newModelPtr()
	result := r.withContext(ctx).Delete(model, id)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to delete record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}

// DeleteBatch 批量软删除记录
func (r *RepositoryImpl[T]) DeleteBatch(ctx context.Context, ids []int64) error {
	if len(ids) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "ids cannot be empty")
	}

	model := r.newModelPtr()
	if err := r.withContext(ctx).Delete(model, ids).Error; err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to delete records", err)
	}

	return nil
}

// HardDelete 硬删除记录（从数据库移除）
func (r *RepositoryImpl[T]) HardDelete(ctx context.Context, id int64) error {
	model := r.newModelPtr()
	result := r.withContext(ctx).Unscoped().Delete(model, id)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to hard delete record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}
