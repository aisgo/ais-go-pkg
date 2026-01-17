package repository

import (
    "context"
    "testing"

    ulidv2 "github.com/oklog/ulid/v2"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type tenantTestModel struct {
    ID       string      `gorm:"column:id;type:char(26);primaryKey"`
    TenantID ulidv2.ULID `gorm:"column:tenant_id;type:char(26);not null"`
    DeptID   *ulidv2.ULID `gorm:"column:dept_id;type:char(26)"`
    Name     string      `gorm:"column:name"`
}

func openTenantTestDB(t *testing.T) *gorm.DB {
    t.Helper()
    db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
    if err != nil {
        t.Fatalf("open db: %v", err)
    }
    if err := db.AutoMigrate(&tenantTestModel{}); err != nil {
        t.Fatalf("migrate: %v", err)
    }
    return db
}

func TestTenantFindByIDScope(t *testing.T) {
    db := openTenantTestDB(t)
    repo := NewRepository[tenantTestModel](db)

    tenantA := ulidv2.Make()
    tenantB := ulidv2.Make()

    a := &tenantTestModel{ID: ulidv2.Make().String(), Name: "a", TenantID: tenantA}
    b := &tenantTestModel{ID: ulidv2.Make().String(), Name: "b", TenantID: tenantB}

    if err := repo.Create(WithTenantContext(context.Background(), TenantContext{TenantID: tenantA}), a); err != nil {
        t.Fatalf("create a: %v", err)
    }
    if err := repo.Create(WithTenantContext(context.Background(), TenantContext{TenantID: tenantB}), b); err != nil {
        t.Fatalf("create b: %v", err)
    }

    ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA})
    if _, err := repo.FindByID(ctxA, b.ID); err == nil {
        t.Fatalf("expected not found for cross-tenant id")
    }

    if _, err := repo.FindByID(ctxA, a.ID); err != nil {
        t.Fatalf("expected find by id: %v", err)
    }
}
