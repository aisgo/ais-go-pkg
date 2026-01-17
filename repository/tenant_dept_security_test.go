package repository

import (
	"context"
	"testing"

	ulidv2 "github.com/oklog/ulid/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type deptTestModel struct {
	ID       string       `gorm:"column:id;type:char(26);primaryKey"`
	TenantID ulidv2.ULID  `gorm:"column:tenant_id;type:char(26);not null"`
	DeptID   *ulidv2.ULID `gorm:"column:dept_id;type:char(26)"`
	Name     string       `gorm:"column:name"`
}

func openDeptTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&deptTestModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestNonAdminMustProvideDeptID 测试非管理员用户必须提供DeptID
func TestNonAdminMustProvideDeptID(t *testing.T) {
	db := openDeptTestDB(t)
	repo := NewRepository[deptTestModel](db)

	tenantID := ulidv2.Make()
	deptID := ulidv2.Make()

	// 使用管理员上下文创建一条记录
	adminCtx := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &deptID,
		IsAdmin:  true,
	})

	m := &deptTestModel{ID: ulidv2.Make().String(), Name: "test"}
	if err := repo.Create(adminCtx, m); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 非管理员用户缺少 DeptID，应该无法查询
	nonAdminCtxNoDept := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   nil, // 没有提供 DeptID
		IsAdmin:  false,
	})

	// 尝试查询应该失败
	_, err := repo.FindByID(nonAdminCtxNoDept, m.ID)
	if err == nil {
		t.Fatalf("expected error when non-admin user has no dept_id, but query succeeded")
	}

	// 尝试统计应该失败
	_, err = repo.Count(nonAdminCtxNoDept, "1=1")
	if err == nil {
		t.Fatalf("expected error when non-admin user has no dept_id for count")
	}

	// 尝试创建应该失败
	m2 := &deptTestModel{ID: ulidv2.Make().String(), Name: "test2"}
	err = repo.Create(nonAdminCtxNoDept, m2)
	if err == nil {
		t.Fatalf("expected error when non-admin user has no dept_id for create")
	}
}

// TestNonAdminWithDeptIDCanAccess 测试非管理员提供DeptID后可以正常访问
func TestNonAdminWithDeptIDCanAccess(t *testing.T) {
	db := openDeptTestDB(t)
	repo := NewRepository[deptTestModel](db)

	tenantID := ulidv2.Make()
	deptID := ulidv2.Make()

	// 非管理员用户提供了 DeptID
	nonAdminCtx := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &deptID,
		IsAdmin:  false,
	})

	// 应该可以正常创建
	m := &deptTestModel{ID: ulidv2.Make().String(), Name: "test"}
	if err := repo.Create(nonAdminCtx, m); err != nil {
		t.Fatalf("create should succeed with dept_id: %v", err)
	}

	// 应该可以正常查询
	found, err := repo.FindByID(nonAdminCtx, m.ID)
	if err != nil {
		t.Fatalf("find should succeed with dept_id: %v", err)
	}
	if found.ID != m.ID {
		t.Fatalf("found wrong record")
	}

	// 应该可以正常统计
	count, err := repo.Count(nonAdminCtx, "1=1")
	if err != nil {
		t.Fatalf("count should succeed with dept_id: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected count 1, got %d", count)
	}
}

// TestAdminDoesNotRequireDeptID 测试管理员不需要DeptID
func TestAdminDoesNotRequireDeptID(t *testing.T) {
	db := openDeptTestDB(t)
	repo := NewRepository[deptTestModel](db)

	tenantID := ulidv2.Make()
	dept1 := ulidv2.Make()
	dept2 := ulidv2.Make()

	// 在部门1创建记录
	ctx1 := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept1,
		IsAdmin:  false,
	})
	m1 := &deptTestModel{ID: ulidv2.Make().String(), Name: "dept1"}
	if err := repo.Create(ctx1, m1); err != nil {
		t.Fatalf("create in dept1: %v", err)
	}

	// 在部门2创建记录
	ctx2 := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept2,
		IsAdmin:  false,
	})
	m2 := &deptTestModel{ID: ulidv2.Make().String(), Name: "dept2"}
	if err := repo.Create(ctx2, m2); err != nil {
		t.Fatalf("create in dept2: %v", err)
	}

	// 管理员即使不提供 DeptID 也能访问所有数据
	adminCtxNoDept := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   nil, // 管理员不需要提供 DeptID
		IsAdmin:  true,
	})

	// 应该能看到所有部门的记录
	count, err := repo.Count(adminCtxNoDept, "1=1")
	if err != nil {
		t.Fatalf("admin count should succeed without dept_id: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected admin to see 2 records, got %d", count)
	}

	// 应该能访问任意部门的记录
	if _, err := repo.FindByID(adminCtxNoDept, m1.ID); err != nil {
		t.Fatalf("admin should access dept1 record: %v", err)
	}
	if _, err := repo.FindByID(adminCtxNoDept, m2.ID); err != nil {
		t.Fatalf("admin should access dept2 record: %v", err)
	}
}
