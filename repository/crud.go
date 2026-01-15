package repository

import (
	"context"
	"sync"

	"github.com/aisgo/ais-go-pkg/errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
)

/* ========================================================================
 * CRUD Repository Implementation - CRUD 操作实现
 * ========================================================================
 * 职责: 实现 CRUDRepository 接口
 *
 * 使用示例:
 *   // 1. 定义模型
 *   type User struct {
 *       repository.BaseModel
 *       Name  string `gorm:"column:name;type:varchar(100);not null"`
 *       Email string `gorm:"column:email;type:varchar(255);uniqueIndex"`
 *   }
 *
 *   // 2. 创建仓储
 *   repo := repository.NewRepository[User](db)
 *
 *   // 3. 基本 CRUD
 *   user := &User{Name: "Alice", Email: "alice@example.com"}
 *   err := repo.Create(ctx, user)
 *
 *   // 4. 查询
 *   foundUser, err := repo.FindByID(ctx, user.ID.String())
 *
 *   // 5. 更新
 *   user.Name = "Alice Updated"
 *   err = repo.Update(ctx, user)
 *
 *   // 6. 部分更新（防止批量赋值漏洞）
 *   err = repo.UpdateByID(ctx, user.ID.String(),
 *       map[string]any{"name": "New Name"},
 *       "name", "email") // 白名单字段
 *
 *   // 7. 删除（软删除）
 *   err = repo.Delete(ctx, user.ID.String())
 *
 *   // 8. 事务示例
 *   err = repo.Execute(ctx, func(txCtx context.Context) error {
 *       user1 := &User{Name: "User1"}
 *       if err := repo.Create(txCtx, user1); err != nil {
 *           return err
 *       }
 *
 *       user2 := &User{Name: "User2"}
 *       if err := repo.Create(txCtx, user2); err != nil {
 *           return err // 自动回滚
 *       }
 *
 *       return nil // 自动提交
 *   })
 *
 *   // 9. 分页查询
 *   page, err := repo.Paginate(ctx, repository.PageRequest{
 *       Page:     1,
 *       PageSize: 10,
 *   }, repository.WithCondition("age > ?", 18))
 * ======================================================================== */

const (
	// DefaultBatchSize 默认批量操作大小
	DefaultBatchSize = 100
)

// RepositoryImpl 仓储实现
type RepositoryImpl[T any] struct {
	db *gorm.DB

	// Schema 缓存（线程安全）
	schemaOnce sync.Once
	schema     *schema.Schema
	schemaErr  error
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

// withContext 返回带 context 的 DB (自动识别事务)
func (r *RepositoryImpl[T]) withContext(ctx context.Context) *gorm.DB {
	return getDBFromContext(ctx, r.db)
}

// getSchema 获取缓存的 Schema（线程安全）
func (r *RepositoryImpl[T]) getSchema() (*schema.Schema, error) {
	r.schemaOnce.Do(func() {
		stmt := &gorm.Statement{DB: r.db}
		r.schemaErr = stmt.Parse(r.newModelPtr())
		if r.schemaErr == nil {
			r.schema = stmt.Schema
		}
	})
	return r.schema, r.schemaErr
}

/* ========================================================================
 * Create 操作
 * ======================================================================== */

// Create 创建单条记录
func (r *RepositoryImpl[T]) Create(ctx context.Context, model *T) error {
	if model == nil {
		return errors.ErrInvalidArgument
	}

	return r.withContext(ctx).Create(model).Error
}

// CreateBatch 批量创建记录
func (r *RepositoryImpl[T]) CreateBatch(ctx context.Context, models []*T, batchSize int) error {
	if len(models) == 0 {
		return errors.ErrInvalidArgument
	}

	if batchSize <= 0 {
		batchSize = DefaultBatchSize
	}

	// 过滤 nil 模型
	validModels := make([]*T, 0, len(models))
	for _, m := range models {
		if m != nil {
			validModels = append(validModels, m)
		}
	}

	if len(validModels) == 0 {
		return nil
	}

	return r.withContext(ctx).CreateInBatches(validModels, batchSize).Error
}

/* ========================================================================
 * Update 操作
 * ======================================================================== */

// Update 更新记录（根据主键）
// 注意：使用 Save 会更新所有字段，包括零值字段。
func (r *RepositoryImpl[T]) Update(ctx context.Context, model *T) error {
	if model == nil {
		return errors.ErrInvalidArgument
	}

	result := r.withContext(ctx).Save(model)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// UpdateByID 根据 ID 更新指定字段
func (r *RepositoryImpl[T]) UpdateByID(ctx context.Context, id string, updates map[string]any, allowedFields ...string) error {
	if len(updates) == 0 {
		return errors.ErrInvalidArgument
	}

	// 过滤非法字段，防止注入/批量赋值漏洞
	filteredUpdates, err := r.filterUpdates(updates, allowedFields)
	if err != nil {
		return err
	}

	if len(filteredUpdates) == 0 {
		return errors.ErrInvalidArgument
	}

	model := r.newModelPtr()
	result := r.withContext(ctx).Model(model).Where("id = ?", id).Updates(filteredUpdates)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// filterUpdates 过滤掉 map 中非法的数据库列名，防止字段注入/批量赋值漏洞
func (r *RepositoryImpl[T]) filterUpdates(updates map[string]any, allowedFields []string) (map[string]any, error) {
	// 使用缓存的 Schema
	schema, err := r.getSchema()
	if err != nil {
		return nil, err
	}

	// 构建白名单 Set
	allowedSet := make(map[string]struct{})
	for _, f := range allowedFields {
		allowedSet[f] = struct{}{}
	}
	hasWhitelist := len(allowedSet) > 0

	filtered := make(map[string]any)
	for k, v := range updates {
		// 如果有白名单，检查字段是否在白名单中
		if hasWhitelist {
			if _, ok := allowedSet[k]; !ok {
				continue
			}
		}

		// 优先匹配数据库列名 (DB Name)
		if field, ok := schema.FieldsByDBName[k]; ok {
			if !field.PrimaryKey && field.Updatable {
				filtered[k] = v
			}
			continue
		}
		// 尝试匹配结构体字段名 (Struct Field Name)
		if field, ok := schema.FieldsByName[k]; ok {
			if !field.PrimaryKey && field.Updatable {
				filtered[field.DBName] = v
			}
			continue
		}
	}

	return filtered, nil
}

// UpsertBatch 批量更新或插入记录
// 注意：此方法使用 Upsert 语义（如果记录不存在则插入，存在则更新所有字段）。
// 对应 MySQL: INSERT ... ON DUPLICATE KEY UPDATE
// 对应 Postgres: INSERT ... ON CONFLICT DO UPDATE
func (r *RepositoryImpl[T]) UpsertBatch(ctx context.Context, models []*T) error {
	if len(models) == 0 {
		return errors.ErrInvalidArgument
	}

	// 过滤 nil 模型
	validModels := make([]*T, 0, len(models))
	for _, m := range models {
		if m != nil {
			validModels = append(validModels, m)
		}
	}

	if len(validModels) == 0 {
		return nil
	}

	// 使用 Upsert 实现高效批量更新
	if err := r.withContext(ctx).Clauses(clause.OnConflict{
		UpdateAll: true,
	}).Save(validModels).Error; err != nil {
		return err
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
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}

// DeleteBatch 批量软删除记录
func (r *RepositoryImpl[T]) DeleteBatch(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return errors.ErrInvalidArgument
	}

	model := r.newModelPtr()
	return r.withContext(ctx).Delete(model, "id IN ?", ids).Error
}

// HardDelete 硬删除记录（从数据库移除）
func (r *RepositoryImpl[T]) HardDelete(ctx context.Context, id string) error {
	model := r.newModelPtr()
	result := r.withContext(ctx).Unscoped().Delete(model, "id = ?", id)
	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}

	return nil
}
