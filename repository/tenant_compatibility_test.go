package repository

import (
	"context"
	"testing"

	ulidv2 "github.com/oklog/ulid/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// 测试非指针的dept_id字段
type nonPointerDeptModel struct {
	ID       string      `gorm:"column:id;type:char(26);primaryKey"`
	TenantID ulidv2.ULID `gorm:"column:tenant_id;type:char(26);not null"`
	DeptID   ulidv2.ULID `gorm:"column:dept_id;type:char(26);not null"` // 非指针
	Name     string      `gorm:"column:name"`
}

// 测试字符串类型的Max/Min
type stringMaxMinModel struct {
	ID       string      `gorm:"column:id;type:char(26);primaryKey"`
	TenantID ulidv2.ULID `gorm:"column:tenant_id;type:char(26);not null"`
	Code     string      `gorm:"column:code;type:varchar(50)"`
	Amount   float64     `gorm:"column:amount"`
}

func (stringMaxMinModel) TenantIgnored() bool {
	return true
}

func openNonPointerDeptDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&nonPointerDeptModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

func openStringMaxMinDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&stringMaxMinModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestNonPointerDeptIDField 测试非指针的dept_id字段
func TestNonPointerDeptIDField(t *testing.T) {
	db := openNonPointerDeptDB(t)
	repo := NewRepository[nonPointerDeptModel](db)

	tenantID := ulidv2.Make()
	deptID := ulidv2.Make()

	// 非管理员用户提供dept_id（指针）
	ctx := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &deptID, // 提供指针
		IsAdmin:  false,
	})

	// 创建记录，setTenantFields 应该正确处理非指针字段
	m := &nonPointerDeptModel{ID: ulidv2.Make().String(), Name: "test"}
	if err := repo.Create(ctx, m); err != nil {
		t.Fatalf("create with non-pointer dept_id field: %v", err)
	}

	// 验证dept_id被正确设置
	if m.DeptID != deptID {
		t.Fatalf("expected dept_id %v, got %v", deptID, m.DeptID)
	}

	// 能够查询到记录
	found, err := repo.FindByID(ctx, m.ID)
	if err != nil {
		t.Fatalf("find by id: %v", err)
	}
	if found.DeptID != deptID {
		t.Fatalf("expected dept_id %v, got %v", deptID, found.DeptID)
	}
}

// TestMaxMinWithStringColumn 测试Max/Min支持字符串列
func TestMaxMinWithStringColumn(t *testing.T) {
	db := openStringMaxMinDB(t)
	repo := NewRepository[stringMaxMinModel](db)

	// 创建一些记录
	for _, code := range []string{"A001", "B002", "C003"} {
		m := &stringMaxMinModel{
			ID:   ulidv2.Make().String(),
			Code: code,
		}
		if err := repo.Create(context.Background(), m); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	// 测试字符串列的Max
	maxCode, err := repo.Max(context.Background(), "code", "")
	if err != nil {
		t.Fatalf("max on string column: %v", err)
	}
	if maxCode == nil {
		t.Fatalf("expected max value, got nil")
	}
	// 字符串比较，最大值应该是 "C003"
	if maxStr, ok := maxCode.(string); !ok {
		t.Fatalf("expected string type, got %T", maxCode)
	} else if maxStr != "C003" {
		t.Fatalf("expected max C003, got %s", maxStr)
	}

	// 测试字符串列的Min
	minCode, err := repo.Min(context.Background(), "code", "")
	if err != nil {
		t.Fatalf("min on string column: %v", err)
	}
	if minCode == nil {
		t.Fatalf("expected min value, got nil")
	}
	// 字符串比较，最小值应该是 "A001"
	if minStr, ok := minCode.(string); !ok {
		t.Fatalf("expected string type, got %T", minCode)
	} else if minStr != "A001" {
		t.Fatalf("expected min A001, got %s", minStr)
	}
}

// TestMaxMinWithNumericColumn 测试Max/Min支持数值列
func TestMaxMinWithNumericColumn(t *testing.T) {
	db := openStringMaxMinDB(t)
	repo := NewRepository[stringMaxMinModel](db)

	// 创建一些记录
	for _, amount := range []float64{100.5, 200.8, 50.2} {
		m := &stringMaxMinModel{
			ID:     ulidv2.Make().String(),
			Amount: amount,
		}
		if err := repo.Create(context.Background(), m); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	// 测试数值列的Max
	maxAmount, err := repo.Max(context.Background(), "amount", "")
	if err != nil {
		t.Fatalf("max on numeric column: %v", err)
	}
	if maxAmount == nil {
		t.Fatalf("expected max value, got nil")
	}
	if maxFloat, ok := maxAmount.(float64); !ok {
		t.Fatalf("expected float64 type, got %T", maxAmount)
	} else if maxFloat != 200.8 {
		t.Fatalf("expected max 200.8, got %v", maxFloat)
	}
}

// TestGlobalTableWithAggregates 测试全局表（TenantIgnored）可以使用聚合方法
func TestGlobalTableWithAggregates(t *testing.T) {
	db := openStringMaxMinDB(t)
	repo := NewRepository[stringMaxMinModel](db)

	// 创建一些记录
	for i := 0; i < 5; i++ {
		m := &stringMaxMinModel{
			ID:     ulidv2.Make().String(),
			Amount: float64(i * 10),
		}
		if err := repo.Create(context.Background(), m); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	// 全局表应该可以直接使用聚合方法，无需TenantContext
	count, err := repo.Count(context.Background(), "1=1")
	if err != nil {
		t.Fatalf("count on global table: %v", err)
	}
	if count != 5 {
		t.Fatalf("expected count 5, got %d", count)
	}

	// Sum
	sum, err := repo.Sum(context.Background(), "amount", "")
	if err != nil {
		t.Fatalf("sum on global table: %v", err)
	}
	if sum != 100 { // 0+10+20+30+40 = 100
		t.Fatalf("expected sum 100, got %f", sum)
	}

	// Max
	max, err := repo.Max(context.Background(), "amount", "")
	if err != nil {
		t.Fatalf("max on global table: %v", err)
	}
	if max == nil {
		t.Fatalf("expected max value, got nil")
	}

	// Min
	min, err := repo.Min(context.Background(), "amount", "")
	if err != nil {
		t.Fatalf("min on global table: %v", err)
	}
	if min == nil {
		t.Fatalf("expected min value, got nil")
	}
}
