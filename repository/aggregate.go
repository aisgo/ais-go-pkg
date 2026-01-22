package repository

import (
	"context"
	"database/sql"
	"regexp"
	"strings"

	"github.com/aisgo/ais-go-pkg/errors"
)

/* ========================================================================
 * Aggregate Repository Implementation - 聚合查询实现
 * ========================================================================
 * 职责: 实现 AggregateRepository 接口
 * 安全: 对列名进行白名单验证，防止 SQL 注入
 * ======================================================================== */

// columnRegex 列名正则表达式（只允许字母、数字、下划线）
var columnRegex = regexp.MustCompile(`^[a-zA-Z_][a-zA-Z0-9_]*$`)

// validateColumn 验证列名是否安全
func validateColumn(column string) error {
	if column == "" {
		return errors.New(errors.ErrCodeInvalidArgument, "column cannot be empty")
	}
	if strings.Contains(column, ".") {
		return errors.New(errors.ErrCodeInvalidArgument, "column must not contain table qualifier")
	}
	if !columnRegex.MatchString(column) {
		return errors.New(errors.ErrCodeInvalidArgument, "invalid column name: "+column)
	}
	return nil
}

// Sum 求和
func (r *RepositoryImpl[T]) Sum(ctx context.Context, column string, query string, args ...any) (float64, error) {
	if err := validateColumn(column); err != nil {
		return 0, err
	}

	var result float64
	db := r.applyTenantScope(ctx, r.withContext(ctx))

	if query != "" {
		db = db.Where(query, args...)
	}

	// 使用 GORM 的 Raw 方法确保列名安全
	sql := "COALESCE(SUM(" + column + "), 0)"
	if err := db.Model(r.newModelPtr()).Select(sql).Scan(&result).Error; err != nil {
		return 0, errors.Wrap(errors.ErrCodeInternal, "failed to sum records", err)
	}

	return result, nil
}

// Avg 平均值
func (r *RepositoryImpl[T]) Avg(ctx context.Context, column string, query string, args ...any) (float64, error) {
	if err := validateColumn(column); err != nil {
		return 0, err
	}

	var result float64
	db := r.applyTenantScope(ctx, r.withContext(ctx))

	if query != "" {
		db = db.Where(query, args...)
	}

	sql := "COALESCE(AVG(" + column + "), 0)"
	if err := db.Model(r.newModelPtr()).Select(sql).Scan(&result).Error; err != nil {
		return 0, errors.Wrap(errors.ErrCodeInternal, "failed to average records", err)
	}

	return result, nil
}

// Max 最大值
// 返回值类型取决于数据库驱动的扫描结果（int64/float64/string/[]byte/time.Time 等）
// 无记录时返回 nil
func (r *RepositoryImpl[T]) Max(ctx context.Context, column string, query string, args ...any) (any, error) {
	if err := validateColumn(column); err != nil {
		return nil, err
	}

	var result any
	db := r.applyTenantScope(ctx, r.withContext(ctx))

	if query != "" {
		db = db.Where(query, args...)
	}

	sqlQuery := "MAX(" + column + ")"
	row := db.Model(r.newModelPtr()).Select(sqlQuery).Row()
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to get max value", err)
	}

	if result == nil {
		return nil, nil
	}
	return result, nil
}

// Min 最小值
// 返回值类型取决于数据库驱动的扫描结果（int64/float64/string/[]byte/time.Time 等）
// 无记录时返回 nil
func (r *RepositoryImpl[T]) Min(ctx context.Context, column string, query string, args ...any) (any, error) {
	if err := validateColumn(column); err != nil {
		return nil, err
	}

	var result any
	db := r.applyTenantScope(ctx, r.withContext(ctx))

	if query != "" {
		db = db.Where(query, args...)
	}

	sqlQuery := "MIN(" + column + ")"
	row := db.Model(r.newModelPtr()).Select(sqlQuery).Row()
	if err := row.Scan(&result); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to get min value", err)
	}

	if result == nil {
		return nil, nil
	}
	return result, nil
}

// CountByGroup 分组统计
// 用于类似 GROUP BY COUNT(*) 的查询
func (r *RepositoryImpl[T]) CountByGroup(ctx context.Context, groupColumn, query string, args ...any) (map[string]int64, error) {
	if err := validateColumn(groupColumn); err != nil {
		return nil, err
	}

	type Result struct {
		Group string `gorm:"column:group_column"`
		Count int64
	}

	var results []Result
	db := r.applyTenantScope(ctx, r.withContext(ctx))

	if query != "" {
		db = db.Where(query, args...)
	}

	// 安全的列名使用
	sql := groupColumn + " as group_column, COUNT(*) as count"
	if err := db.Model(r.newModelPtr()).
		Select(sql).
		Group(groupColumn).
		Scan(&results).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to count by group", err)
	}

	resultMap := make(map[string]int64)
	for _, r := range results {
		resultMap[r.Group] = r.Count
	}

	return resultMap, nil
}

// SumWithCondition 带条件的求和（推荐使用）
// 使用结构体作为查询条件，更安全
func (r *RepositoryImpl[T]) SumWithCondition(ctx context.Context, column string, where any, opts ...Option) (float64, error) {
	if err := validateColumn(column); err != nil {
		return 0, err
	}

	var result float64
	db := r.buildQuery(ctx, ApplyOptions(opts))

	if where != nil {
		db = db.Where(where)
	}

	sql := "COALESCE(SUM(" + column + "), 0)"
	if err := db.Model(r.newModelPtr()).Select(sql).Scan(&result).Error; err != nil {
		return 0, errors.Wrap(errors.ErrCodeInternal, "failed to sum records", err)
	}

	return result, nil
}

// AvgWithCondition 带条件的平均值（推荐使用）
func (r *RepositoryImpl[T]) AvgWithCondition(ctx context.Context, column string, where any, opts ...Option) (float64, error) {
	if err := validateColumn(column); err != nil {
		return 0, err
	}

	var result float64
	db := r.buildQuery(ctx, ApplyOptions(opts))

	if where != nil {
		db = db.Where(where)
	}

	sql := "COALESCE(AVG(" + column + "), 0)"
	if err := db.Model(r.newModelPtr()).Select(sql).Scan(&result).Error; err != nil {
		return 0, errors.Wrap(errors.ErrCodeInternal, "failed to average records", err)
	}

	return result, nil
}

// MaxWithCondition 带条件的最大值（推荐使用）
func (r *RepositoryImpl[T]) MaxWithCondition(ctx context.Context, column string, where any, opts ...Option) (any, error) {
	if err := validateColumn(column); err != nil {
		return nil, err
	}

	var result any
	db := r.buildQuery(ctx, ApplyOptions(opts))

	if where != nil {
		db = db.Where(where)
	}

	sql := "MAX(" + column + ")"
	if err := db.Model(r.newModelPtr()).Select(sql).Scan(&result).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to get max value", err)
	}

	return result, nil
}

// MinWithCondition 带条件的最小值（推荐使用）
func (r *RepositoryImpl[T]) MinWithCondition(ctx context.Context, column string, where any, opts ...Option) (any, error) {
	if err := validateColumn(column); err != nil {
		return nil, err
	}

	var result any
	db := r.buildQuery(ctx, ApplyOptions(opts))

	if where != nil {
		db = db.Where(where)
	}

	sql := "MIN(" + column + ")"
	if err := db.Model(r.newModelPtr()).Select(sql).Scan(&result).Error; err != nil {
		return nil, errors.Wrap(errors.ErrCodeInternal, "failed to get min value", err)
	}

	return result, nil
}

// IsSafeColumnName 检查列名是否安全（用于调用方验证）
func IsSafeColumnName(column string) bool {
	return columnRegex.MatchString(column)
}

// SanitizeColumnName 清理列名，移除不安全字符
func SanitizeColumnName(column string) string {
	// 移除所有非字母数字下划线字符
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			return r
		}
		return -1
	}, column)
}
