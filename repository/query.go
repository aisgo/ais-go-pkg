package repository

import (
	"context"

	"ais.local/ais-go-pkg/errors"

	"gorm.io/gorm"
)

/* ========================================================================
 * Query Repository Implementation - 查询操作实现
 * ========================================================================
 * 职责: 实现 QueryRepository 接口
 * ======================================================================== */

// buildQuery 构建查询
func (r *RepositoryImpl[T]) buildQuery(ctx context.Context, opts *QueryOption) *gorm.DB {
	db := r.withContext(ctx)

	if opts == nil {
		return db
	}

	// 应用选择字段
	if len(opts.Select) > 0 {
		db = db.Select(opts.Select)
	}

	// 应用连接查询
	for _, join := range opts.Joins {
		db = db.Joins(join)
	}

	// 应用排序
	if opts.OrderBy != "" {
		db = db.Order(opts.OrderBy)
	}

	// 应用作用域
	for _, scope := range opts.Scopes {
		db = scope(db)
	}

	// 应用预加载
	for _, preload := range opts.Preloads {
		db = db.Preload(preload)
	}

	return db
}

/* ========================================================================
 * FindByID 操作
 * ======================================================================== */

// FindByID 根据 ID 查找记录
func (r *RepositoryImpl[T]) FindByID(ctx context.Context, id int64, opts ...Option) (*T, error) {
	opt := ApplyOptions(opts)
	model := r.newModelPtr()

	query := r.buildQuery(ctx, opt)
	if err := query.First(model, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "record not found")
		}
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to find record", err)
	}

	return model, nil
}

// FindByIDs 根据 ID 列表查找多条记录
func (r *RepositoryImpl[T]) FindByIDs(ctx context.Context, ids []int64, opts ...Option) ([]*T, error) {
	if len(ids) == 0 {
		return []*T{}, nil
	}

	opt := ApplyOptions(opts)
	var models []*T

	query := r.buildQuery(ctx, opt)
	if err := query.Find(&models, ids).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to find records", err)
	}

	return models, nil
}

/* ========================================================================
 * FindOne 操作
 * ======================================================================== */

// FindOne 查找单条记录（使用自定义条件）
func (r *RepositoryImpl[T]) FindOne(ctx context.Context, query string, args ...any) (*T, error) {
	return r.FindOneWithOpts(ctx, query, nil, args...)
}

// FindOneWithOpts 查找单条记录（带选项）
func (r *RepositoryImpl[T]) FindOneWithOpts(ctx context.Context, query string, opts []Option, args ...any) (*T, error) {
	var opt *QueryOption
	if len(opts) > 0 {
		opt = ApplyOptions(opts)
	}

	model := r.newModelPtr()
	db := r.buildQuery(ctx, opt)

	if err := db.Where(query, args...).First(model).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.New(errors.ErrCodeNotFound, "record not found")
		}
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to find record", err)
	}

	return model, nil
}

/* ========================================================================
 * FindByQuery 操作
 * ======================================================================== */

// FindByQuery 查找多条记录（使用自定义条件）
func (r *RepositoryImpl[T]) FindByQuery(ctx context.Context, query string, args ...any) ([]*T, error) {
	return r.FindByQueryWithOpts(ctx, query, nil, args...)
}

// FindByQueryWithOpts 查找多条记录（带选项）
func (r *RepositoryImpl[T]) FindByQueryWithOpts(ctx context.Context, query string, opts []Option, args ...any) ([]*T, error) {
	var opt *QueryOption
	if len(opts) > 0 {
		opt = ApplyOptions(opts)
	}

	var models []*T
	db := r.buildQuery(ctx, opt)

	if err := db.Where(query, args...).Find(&models).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to find records", err)
	}

	return models, nil
}

/* ========================================================================
 * Count/Exists 操作
 * ======================================================================== */

// Count 统计记录数
func (r *RepositoryImpl[T]) Count(ctx context.Context, query string, args ...any) (int64, error) {
	var count int64
	db := r.withContext(ctx)

	if err := db.Model(r.newModelPtr()).Where(query, args...).Count(&count).Error; err != nil {
		return 0, errors.Wrap(errors.ErrCodeInternal, "failed to count records", err)
	}

	return count, nil
}

// Exists 检查记录是否存在
func (r *RepositoryImpl[T]) Exists(ctx context.Context, query string, args ...any) (bool, error) {
	count, err := r.Count(ctx, query, args...)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}
