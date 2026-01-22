package repository

import (
	"context"
	"database/sql"
	"math"

	"github.com/aisgo/ais-go-pkg/errors"

	"gorm.io/gorm"
)

/* ========================================================================
 * Page Repository Implementation - 分页查询实现
 * ========================================================================
 * 职责: 实现 PageRepository 接口
 * ======================================================================== */

// FindPage 分页查询
func (r *RepositoryImpl[T]) FindPage(ctx context.Context, page, pageSize int, query string, args ...any) (*PageResult[T], error) {
	return r.FindPageWithOpts(ctx, page, pageSize, query, nil, args...)
}

// FindPageWithOpts 分页查询（带选项）
func (r *RepositoryImpl[T]) FindPageWithOpts(ctx context.Context, page, pageSize int, query string, opts []Option, args ...any) (*PageResult[T], error) {
	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 1000 {
		pageSize = 1000 // 限制最大页大小
	}

	var opt *QueryOption
	if len(opts) > 0 {
		opt = ApplyOptions(opts)
	}

	return r.findPageWithSnapshot(ctx, opt, page, pageSize, func(db *gorm.DB) *gorm.DB {
		if query != "" {
			return db.Where(query, args...)
		}
		return db
	})
}

// FindPageByModel 根据模型条件分页查询
// 用于复杂的 WHERE 条件场景
func (r *RepositoryImpl[T]) FindPageByModel(ctx context.Context, page, pageSize int, model any, opts ...Option) (*PageResult[T], error) {
	// 参数校验
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 1000 {
		pageSize = 1000
	}

	opt := ApplyOptions(opts)
	return r.findPageWithSnapshot(ctx, opt, page, pageSize, func(db *gorm.DB) *gorm.DB {
		if model != nil {
			return db.Where(model)
		}
		return db
	})
}

func (r *RepositoryImpl[T]) findPageWithSnapshot(ctx context.Context, opt *QueryOption, page, pageSize int, apply func(*gorm.DB) *gorm.DB) (*PageResult[T], error) {
	db := r.withContext(ctx)

	var result *PageResult[T]
	err := db.Transaction(func(tx *gorm.DB) error {
		ctxWithTx := context.WithValue(ctx, ctxTxKey{}, tx)
		query := r.buildQuery(ctxWithTx, opt)
		if apply != nil {
			query = apply(query)
		}
		var err error
		result, err = r.findPageWithDB(query, page, pageSize)
		return err
	}, &sql.TxOptions{
		Isolation: sql.LevelRepeatableRead,
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

func (r *RepositoryImpl[T]) findPageWithDB(db *gorm.DB, page, pageSize int) (*PageResult[T], error) {
	// 统计总数
	var total int64
	if err := db.Model(r.newModelPtr()).Count(&total).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to count records", err)
	}

	// 计算分页参数
	offset := (page - 1) * pageSize

	// 查询数据
	var list []T
	if err := db.Offset(offset).Limit(pageSize).Find(&list).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to find records", err)
	}

	// 计算总页数
	pages := int64(0)
	if pageSize > 0 {
		pages = int64(math.Ceil(float64(total) / float64(pageSize)))
	}

	return &PageResult[T]{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
		Pages:    pages,
	}, nil
}
