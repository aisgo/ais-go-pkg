package repository

import (
    "context"
    "reflect"

    "github.com/aisgo/ais-go-pkg/errors"
    "gorm.io/gorm"
    "gorm.io/gorm/schema"
)

const (
    tenantColumn = "tenant_id"
    deptColumn   = "dept_id"
)

func (r *RepositoryImpl[T]) applyTenantScope(ctx context.Context, db *gorm.DB) *gorm.DB {
	if r.isTenantIgnored(r.newModelPtr()) {
		return db
	}

    tc, ok := TenantFromContext(ctx)
    if !ok {
        db.AddError(errors.ErrUnauthenticated)
        return db
    }

    _, deptField, err := r.tenantFields()
    if err != nil {
        db.AddError(err)
        return db
    }

    db = db.Where(tenantColumn+" = ?", tc.TenantID)
    if !tc.IsAdmin && tc.DeptID != nil && deptField != nil {
        db = db.Where(deptColumn+" = ?", *tc.DeptID)
    }

    return db
}

func (r *RepositoryImpl[T]) tenantFields() (*schema.Field, *schema.Field, error) {
    schema, err := r.getSchema()
    if err != nil {
        return nil, nil, err
    }
    if _, ok := schema.FieldsByDBName[tenantColumn]; !ok {
        return nil, nil, errors.ErrInvalidArgument
    }
    tenantField := schema.FieldsByDBName[tenantColumn]
    deptField := schema.FieldsByDBName[deptColumn]
    return tenantField, deptField, nil
}

func (r *RepositoryImpl[T]) setTenantFields(ctx context.Context, model any) error {
	if r.isTenantIgnored(model) {
		return nil
	}

    tc, ok := TenantFromContext(ctx)
    if !ok {
        return errors.ErrUnauthenticated
    }

    tenantField, deptField, err := r.tenantFields()
    if err != nil {
        return err
    }

    rv := reflect.ValueOf(model)
    if err := tenantField.Set(ctx, rv, tc.TenantID); err != nil {
        return err
    }

    if tc.DeptID != nil && deptField != nil {
        if err := deptField.Set(ctx, rv, tc.DeptID); err != nil {
            return err
        }
    }

    return nil
}

func (r *RepositoryImpl[T]) isTenantIgnored(model any) bool {
	if model == nil {
		return false
	}

	if ignorable, ok := model.(TenantIgnorable); ok {
		return ignorable.TenantIgnored()
	}

	rv := reflect.ValueOf(model)
	if rv.Kind() == reflect.Ptr && !rv.IsNil() {
		if ignorable, ok := rv.Elem().Interface().(TenantIgnorable); ok {
			return ignorable.TenantIgnored()
		}
	}

	return false
}
