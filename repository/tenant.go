package repository

import (
	"context"

	ulidv2 "github.com/oklog/ulid/v2"
)

// TenantContext carries tenant-scoped claims for repository enforcement.
type TenantContext struct {
	// TenantID 租户ID，必填字段
	// 所有租户模型的查询和操作都会自动过滤此租户ID
	TenantID ulidv2.ULID

	// DeptID 部门ID，可选字段
	// 非管理员用户必须提供，用于实现租户内的部门级数据隔离
	DeptID *ulidv2.ULID

	// IsAdmin 是否为租户管理员
	// 管理员可以跨部门访问租户内的所有数据
	IsAdmin bool

	// PolicyVersion 权限策略版本号（预留字段）
	// 用于缓存失效和权限变更检测，当前版本未使用
	PolicyVersion int64

	// Roles 用户角色列表（预留字段）
	// 未来可用于基于角色的访问控制(RBAC)，当前版本未使用
	Roles []string

	// UserID 当前操作用户ID
	// 用于审计日志和操作追踪
	UserID ulidv2.ULID
}

// TenantIgnorable marks models that should bypass tenant enforcement.
type TenantIgnorable interface {
	TenantIgnored() bool
}

type tenantCtxKey struct{}

// WithTenantContext injects TenantContext into context.Context.
func WithTenantContext(ctx context.Context, tc TenantContext) context.Context {
	return context.WithValue(ctx, tenantCtxKey{}, tc)
}

// TenantFromContext reads TenantContext from context.Context.
func TenantFromContext(ctx context.Context) (TenantContext, bool) {
	v := ctx.Value(tenantCtxKey{})
	if v == nil {
		return TenantContext{}, false
	}
	tc, ok := v.(TenantContext)
	return tc, ok
}
