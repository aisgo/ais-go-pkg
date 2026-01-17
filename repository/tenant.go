package repository

import (
    "context"

    ulidv2 "github.com/oklog/ulid/v2"
)

// TenantContext carries tenant-scoped claims for repository enforcement.
type TenantContext struct {
    TenantID      ulidv2.ULID
    DeptID        *ulidv2.ULID
    IsAdmin       bool
    PolicyVersion int64
    Roles         []string
    UserID        ulidv2.ULID
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
