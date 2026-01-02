package repository

import (
	"context"

	"gorm.io/gorm"
)

/* ========================================================================
 * Repository Interfaces - 仓储接口定义
 * ========================================================================
 * 职责: 定义通用仓储接口
 * 设计: 使用泛型提供类型安全的数据访问
 * ======================================================================== */

// QueryOption 查询选项
type QueryOption struct {
	// Preloads 预加载关联（如 "User", "User.Profile"）
	Preloads []string
	// Scopes 查询作用域（如软删除、租户过滤）
	Scopes []func(*gorm.DB) *gorm.DB
	// OrderBy 排序（如 "created_at DESC"）
	OrderBy string
	// Select 选择字段（如 "id, name, email"）
	Select []string
	// Joins 连接查询（如 "JOIN orders ON orders.user_id = users.id"）
	Joins []string
}

// Option 应用查询选项
type Option func(*QueryOption)

// WithPreloads 设置预加载
func WithPreloads(preloads ...string) Option {
	return func(o *QueryOption) {
		o.Preloads = preloads
	}
}

// WithScopes 设置查询作用域
func WithScopes(scopes ...func(*gorm.DB) *gorm.DB) Option {
	return func(o *QueryOption) {
		o.Scopes = scopes
	}
}

// WithOrderBy 设置排序
func WithOrderBy(orderBy string) Option {
	return func(o *QueryOption) {
		o.OrderBy = orderBy
	}
}

// WithSelect 设置选择字段
func WithSelect(selects ...string) Option {
	return func(o *QueryOption) {
		o.Select = selects
	}
}

// WithJoins 设置连接查询
func WithJoins(joins ...string) Option {
	return func(o *QueryOption) {
		o.Joins = joins
	}
}

// ApplyOptions 应用查询选项
func ApplyOptions(opts []Option) *QueryOption {
	o := &QueryOption{}
	for _, opt := range opts {
		opt(o)
	}
	return o
}

// PageResult 分页结果
type PageResult[T any] struct {
	List     []T   `json:"list" doc:"数据列表"`
	Total    int64 `json:"total" doc:"总记录数"`
	Page     int   `json:"page" doc:"当前页码"`
	PageSize int   `json:"page_size" doc:"每页大小"`
	Pages    int64 `json:"pages" doc:"总页数"`
}

// CRUDRepository CRUD 操作接口
type CRUDRepository[T any] interface {
	// Create 创建单条记录
	Create(ctx context.Context, model *T) error

	// CreateBatch 批量创建记录
	CreateBatch(ctx context.Context, models []*T, batchSize int) error

	// Update 更新记录（根据主键）
	Update(ctx context.Context, model *T) error

	// UpdateByID 根据 ID 更新指定字段
	UpdateByID(ctx context.Context, id int64, updates map[string]any) error

	// UpdateBatch 批量更新记录
	UpdateBatch(ctx context.Context, models []*T) error

	// Delete 软删除记录（设置 deleted_at）
	Delete(ctx context.Context, id int64) error

	// DeleteBatch 批量软删除记录
	DeleteBatch(ctx context.Context, ids []int64) error

	// HardDelete 硬删除记录（从数据库移除）
	HardDelete(ctx context.Context, id int64) error
}

// QueryRepository 查询操作接口
type QueryRepository[T any] interface {
	// FindByID 根据 ID 查找记录
	FindByID(ctx context.Context, id int64, opts ...Option) (*T, error)

	// FindByIDs 根据 ID 列表查找多条记录
	FindByIDs(ctx context.Context, ids []int64, opts ...Option) ([]*T, error)

	// FindOne 查找单条记录（使用自定义条件）
	FindOne(ctx context.Context, query string, args ...any) (*T, error)

	// FindOneWithOpts 查找单条记录（带选项）
	FindOneWithOpts(ctx context.Context, query string, opts []Option, args ...any) (*T, error)

	// FindByQuery 查找多条记录（使用自定义条件）
	FindByQuery(ctx context.Context, query string, args ...any) ([]*T, error)

	// FindByQueryWithOpts 查找多条记录（带选项）
	FindByQueryWithOpts(ctx context.Context, query string, opts []Option, args ...any) ([]*T, error)

	// Count 统计记录数
	Count(ctx context.Context, query string, args ...any) (int64, error)

	// Exists 检查记录是否存在
	Exists(ctx context.Context, query string, args ...any) (bool, error)
}

// PageRepository 分页查询接口
type PageRepository[T any] interface {
	// FindPage 分页查询
	FindPage(ctx context.Context, page, pageSize int, query string, args ...any) (*PageResult[T], error)

	// FindPageWithOpts 分页查询（带选项）
	FindPageWithOpts(ctx context.Context, page, pageSize int, query string, opts []Option, args ...any) (*PageResult[T], error)
}

// AggregateRepository 聚合查询接口
type AggregateRepository[T any] interface {
	// Sum 求和
	Sum(ctx context.Context, column string, query string, args ...any) (float64, error)

	// Avg 平均值
	Avg(ctx context.Context, column string, query string, args ...any) (float64, error)

	// Max 最大值
	Max(ctx context.Context, column string, query string, args ...any) (any, error)

	// Min 最小值
	Min(ctx context.Context, column string, query string, args ...any) (any, error)
}

// TransactionRepository 事务支持接口
type TransactionRepository[T any] interface {
	// Transaction 在事务中执行操作
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error

	// WithTx 创建事务版本的仓储
	WithTx(tx *gorm.DB) Repository[T]
}

// Repository 通用仓储接口
// 组合了所有子接口
type Repository[T any] interface {
	CRUDRepository[T]
	QueryRepository[T]
	PageRepository[T]
	AggregateRepository[T]
	TransactionRepository[T]

	// GetDB 获取底层 GORM DB 实例（用于复杂查询）
	GetDB() *gorm.DB
}
