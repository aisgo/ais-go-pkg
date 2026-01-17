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

	// 应用租户隔离
	db = db.Where(tenantColumn+" = ?", tc.TenantID)

	// 如果模型有部门字段，非管理员必须提供 DeptID
	if !tc.IsAdmin && deptField != nil {
		if tc.DeptID == nil {
			db.AddError(errors.New(errors.ErrCodeUnauthenticated, "non-admin user must provide dept_id"))
			return db
		}
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

	// 如果模型有部门字段
	if deptField != nil {
		// 非管理员必须提供 DeptID
		if !tc.IsAdmin && tc.DeptID == nil {
			return errors.New(errors.ErrCodeUnauthenticated, "non-admin user must provide dept_id")
		}
		// 如果提供了 DeptID，则设置
		if tc.DeptID != nil {
			// 检查字段类型是否为指针
			fieldType := deptField.FieldType
			if fieldType.Kind() == reflect.Ptr {
				// 字段是指针类型，直接设置 *ULID
				if err := deptField.Set(ctx, rv, tc.DeptID); err != nil {
					return err
				}
			} else {
				// 字段是值类型，设置 ULID 值（解引用）
				if err := deptField.Set(ctx, rv, *tc.DeptID); err != nil {
					return err
				}
			}
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
