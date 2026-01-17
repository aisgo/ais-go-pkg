package repository

import (
    "context"

    "github.com/aisgo/ais-go-pkg/errors"
    "gorm.io/gorm"
)

const (
    tenantColumn = "tenant_id"
    deptColumn   = "dept_id"
)

func (r *RepositoryImpl[T]) applyTenantScope(ctx context.Context, db *gorm.DB) *gorm.DB {
    tc, ok := TenantFromContext(ctx)
    if !ok {
        db.AddError(errors.ErrUnauthenticated)
        return db
    }

    if err := r.ensureTenantFields(); err != nil {
        db.AddError(err)
        return db
    }

    db = db.Where(tenantColumn+" = ?", tc.TenantID)
    if !tc.IsAdmin && tc.DeptID != nil {
        db = db.Where(deptColumn+" = ?", *tc.DeptID)
    }

    return db
}

func (r *RepositoryImpl[T]) ensureTenantFields() error {
    schema, err := r.getSchema()
    if err != nil {
        return err
    }
    if _, ok := schema.FieldsByDBName[tenantColumn]; !ok {
        return errors.ErrInvalidArgument
    }
    return nil
}
