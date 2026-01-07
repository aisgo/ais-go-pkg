package repository

import (
	"context"

	"github.com/aisgo/ais-go-pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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
// 注意：使用 Save 会更新所有字段，包括零值字段。
func (r *RepositoryImpl[T]) Update(ctx context.Context, model *T) error {
	if model == nil {
		return errors.New(errors.ErrCodeInvalidArgument, "model cannot be nil")
	}

	result := r.withContext(ctx).Save(model)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to update record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}

// UpdateByID 根据 ID 更新指定字段
func (r *RepositoryImpl[T]) UpdateByID(ctx context.Context, id string, updates map[string]any) error {
	if len(updates) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "updates cannot be empty")
	}

	// 过滤非法字段，防止注入/批量赋值漏洞
	filteredUpdates, err := r.filterUpdates(updates)
	if err != nil {
		return err
	}

	if len(filteredUpdates) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "no valid fields to update")
	}

	model := r.newModelPtr()
	result := r.withContext(ctx).Model(model).Where("id = ?", id).Updates(filteredUpdates)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to update record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}

// filterUpdates 过滤掉 map 中非法的数据库列名，防止字段注入/批量赋值漏洞
func (r *RepositoryImpl[T]) filterUpdates(updates map[string]any) (map[string]any, error) {
	// 解析模型 Schema 以获取合法字段
	stmt := &gorm.Statement{DB: r.db}
	if err := stmt.Parse(r.newModelPtr()); err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to parse model schema", err)
	}

	filtered := make(map[string]any)
	for k, v := range updates {
		// 优先匹配数据库列名 (DB Name)
		if field, ok := stmt.Schema.FieldsByDBName[k]; ok {
			if !field.PrimaryKey && field.Updatable {
				filtered[k] = v
			}
			continue
		}
		// 尝试匹配结构体字段名 (Struct Field Name)
		if field, ok := stmt.Schema.FieldsByName[k]; ok {
			if !field.PrimaryKey && field.Updatable {
				filtered[field.DBName] = v
			}
			continue
		}
	}

	return filtered, nil
}

// UpdateBatch 批量更新记录
// 注意：此方法使用 Upsert 语义（如果记录不存在则插入，存在则更新所有字段）。
func (r *RepositoryImpl[T]) UpdateBatch(ctx context.Context, models []*T) error {
	if len(models) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "models cannot be empty")
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

	// 使用 Upsert 实现高效批量更新
	// MySQL: INSERT ... ON DUPLICATE KEY UPDATE
	// Postgres: INSERT ... ON CONFLICT DO UPDATE
	if err := r.withContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Save(validModels).Error; err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to batch update records", err)
	}

	return nil
}

/* ========================================================================
 * Delete 操作
 * ======================================================================== */

// Delete 软删除记录（设置 deleted_at）
func (r *RepositoryImpl[T]) Delete(ctx context.Context, id string) error {
	model := r.newModelPtr()
	result := r.withContext(ctx).Delete(model, "id = ?", id)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to delete record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}

// DeleteBatch 批量软删除记录
func (r *RepositoryImpl[T]) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errors.New(errors.ErrCodeInvalidArgument, "ids cannot be empty")
	}

	model := r.newModelPtr()
	if err := r.withContext(ctx).Delete(model, "id IN ?", ids).Error; err != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to delete records", err)
	}

	return nil
}

// HardDelete 硬删除记录（从数据库移除）
func (r *RepositoryImpl[T]) HardDelete(ctx context.Context, id string) error {
	model := r.newModelPtr()
	result := r.withContext(ctx).Unscoped().Delete(model, "id = ?", id)
	if result.Error != nil {
		return errors.Wrap(errors.ErrCodeInternal, "failed to hard delete record", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New(errors.ErrCodeNotFound, "record not found")
	}

	return nil
}
