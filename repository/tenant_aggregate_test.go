package repository

import (
	"context"
	"testing"

	ulidv2 "github.com/oklog/ulid/v2"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type tenantAggregateTestModel struct {
	ID       string       `gorm:"column:id;type:char(26);primaryKey"`
	TenantID ulidv2.ULID  `gorm:"column:tenant_id;type:char(26);not null"`
	DeptID   *ulidv2.ULID `gorm:"column:dept_id;type:char(26)"`
	Amount   float64      `gorm:"column:amount"`
	Status   string       `gorm:"column:status"`
}

func openAggregateTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.AutoMigrate(&tenantAggregateTestModel{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}
	return db
}

// TestTenantCountRespectsTenant 测试Count方法的租户隔离
func TestTenantCountRespectsTenant(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantA := ulidv2.Make()
	tenantB := ulidv2.Make()

	ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA, IsAdmin: true})
	ctxB := WithTenantContext(context.Background(), TenantContext{TenantID: tenantB, IsAdmin: true})

	// 租户A创建2条记录
	for i := 0; i < 2; i++ {
		m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 100}
		if err := repo.Create(ctxA, m); err != nil {
			t.Fatalf("create for tenant A: %v", err)
		}
	}

	// 租户B创建3条记录
	for i := 0; i < 3; i++ {
		m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 200}
		if err := repo.Create(ctxB, m); err != nil {
			t.Fatalf("create for tenant B: %v", err)
		}
	}

	// 租户A只能统计到2条
	countA, err := repo.Count(ctxA, "1=1")
	if err != nil {
		t.Fatalf("count for tenant A: %v", err)
	}
	if countA != 2 {
		t.Fatalf("expected 2 records for tenant A, got %d", countA)
	}

	// 租户B只能统计到3条
	countB, err := repo.Count(ctxB, "1=1")
	if err != nil {
		t.Fatalf("count for tenant B: %v", err)
	}
	if countB != 3 {
		t.Fatalf("expected 3 records for tenant B, got %d", countB)
	}
}

// TestTenantExistsRespectsTenant 测试Exists方法的租户隔离
func TestTenantExistsRespectsTenant(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantA := ulidv2.Make()
	tenantB := ulidv2.Make()

	ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA, IsAdmin: true})
	ctxB := WithTenantContext(context.Background(), TenantContext{TenantID: tenantB, IsAdmin: true})

	// 租户A创建状态为"active"的记录
	m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Status: "active"}
	if err := repo.Create(ctxA, m); err != nil {
		t.Fatalf("create: %v", err)
	}

	// 租户A可以看到自己的记录
	exists, err := repo.Exists(ctxA, "status = ?", "active")
	if err != nil {
		t.Fatalf("exists for tenant A: %v", err)
	}
	if !exists {
		t.Fatalf("expected record to exist for tenant A")
	}

	// 租户B看不到租户A的记录
	exists, err = repo.Exists(ctxB, "status = ?", "active")
	if err != nil {
		t.Fatalf("exists for tenant B: %v", err)
	}
	if exists {
		t.Fatalf("expected record not to exist for tenant B")
	}
}

// TestTenantSumRespectsTenant 测试Sum方法的租户隔离
func TestTenantSumRespectsTenant(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantA := ulidv2.Make()
	tenantB := ulidv2.Make()

	ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA, IsAdmin: true})
	ctxB := WithTenantContext(context.Background(), TenantContext{TenantID: tenantB, IsAdmin: true})

	// 租户A创建2条记录，总金额200
	for i := 0; i < 2; i++ {
		m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 100}
		if err := repo.Create(ctxA, m); err != nil {
			t.Fatalf("create for tenant A: %v", err)
		}
	}

	// 租户B创建3条记录，总金额600
	for i := 0; i < 3; i++ {
		m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 200}
		if err := repo.Create(ctxB, m); err != nil {
			t.Fatalf("create for tenant B: %v", err)
		}
	}

	// 租户A只能求和自己的数据
	sumA, err := repo.Sum(ctxA, "amount", "")
	if err != nil {
		t.Fatalf("sum for tenant A: %v", err)
	}
	if sumA != 200 {
		t.Fatalf("expected sum 200 for tenant A, got %f", sumA)
	}

	// 租户B只能求和自己的数据
	sumB, err := repo.Sum(ctxB, "amount", "")
	if err != nil {
		t.Fatalf("sum for tenant B: %v", err)
	}
	if sumB != 600 {
		t.Fatalf("expected sum 600 for tenant B, got %f", sumB)
	}
}

// TestTenantAvgRespectsTenant 测试Avg方法的租户隔离
func TestTenantAvgRespectsTenant(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantA := ulidv2.Make()
	tenantB := ulidv2.Make()

	ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA, IsAdmin: true})
	ctxB := WithTenantContext(context.Background(), TenantContext{TenantID: tenantB, IsAdmin: true})

	// 租户A创建金额为100的记录
	m1 := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 100}
	if err := repo.Create(ctxA, m1); err != nil {
		t.Fatalf("create for tenant A: %v", err)
	}

	// 租户B创建金额为200的记录
	m2 := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 200}
	if err := repo.Create(ctxB, m2); err != nil {
		t.Fatalf("create for tenant B: %v", err)
	}

	// 租户A的平均值应该是100
	avgA, err := repo.Avg(ctxA, "amount", "")
	if err != nil {
		t.Fatalf("avg for tenant A: %v", err)
	}
	if avgA != 100 {
		t.Fatalf("expected avg 100 for tenant A, got %f", avgA)
	}

	// 租户B的平均值应该是200
	avgB, err := repo.Avg(ctxB, "amount", "")
	if err != nil {
		t.Fatalf("avg for tenant B: %v", err)
	}
	if avgB != 200 {
		t.Fatalf("expected avg 200 for tenant B, got %f", avgB)
	}
}

// TestTenantMaxMinRespectsTenant 测试Max/Min方法的租户隔离
func TestTenantMaxMinRespectsTenant(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantA := ulidv2.Make()
	tenantB := ulidv2.Make()

	ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA, IsAdmin: true})
	ctxB := WithTenantContext(context.Background(), TenantContext{TenantID: tenantB, IsAdmin: true})

	// 租户A创建金额50-150的记录
	for _, amt := range []float64{50, 100, 150} {
		m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: amt}
		if err := repo.Create(ctxA, m); err != nil {
			t.Fatalf("create for tenant A: %v", err)
		}
	}

	// 租户B创建金额200-400的记录
	for _, amt := range []float64{200, 300, 400} {
		m := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: amt}
		if err := repo.Create(ctxB, m); err != nil {
			t.Fatalf("create for tenant B: %v", err)
		}
	}

	// 租户A的最大值应该是150
	maxA, err := repo.Max(ctxA, "amount", "")
	if err != nil {
		t.Fatalf("max for tenant A: %v", err)
	}
	if maxA == nil {
		t.Fatalf("expected max value for tenant A, got nil")
	}
	maxAFloat, ok := maxA.(float64)
	if !ok {
		t.Fatalf("expected float64 for max, got %T", maxA)
	}
	if maxAFloat != 150 {
		t.Fatalf("expected max 150 for tenant A, got %v", maxAFloat)
	}

	// 租户A的最小值应该是50
	minA, err := repo.Min(ctxA, "amount", "")
	if err != nil {
		t.Fatalf("min for tenant A: %v", err)
	}
	if minA == nil {
		t.Fatalf("expected min value for tenant A, got nil")
	}
	minAFloat, ok := minA.(float64)
	if !ok {
		t.Fatalf("expected float64 for min, got %T", minA)
	}
	if minAFloat != 50 {
		t.Fatalf("expected min 50 for tenant A, got %v", minAFloat)
	}

	// 租户B的最大值应该是400
	maxB, err := repo.Max(ctxB, "amount", "")
	if err != nil {
		t.Fatalf("max for tenant B: %v", err)
	}
	if maxB == nil {
		t.Fatalf("expected max value for tenant B, got nil")
	}
	maxBFloat, ok := maxB.(float64)
	if !ok {
		t.Fatalf("expected float64 for max, got %T", maxB)
	}
	if maxBFloat != 400 {
		t.Fatalf("expected max 400 for tenant B, got %v", maxBFloat)
	}

	// 租户B的最小值应该是200
	minB, err := repo.Min(ctxB, "amount", "")
	if err != nil {
		t.Fatalf("min for tenant B: %v", err)
	}
	if minB == nil {
		t.Fatalf("expected min value for tenant B, got nil")
	}
	minBFloat, ok := minB.(float64)
	if !ok {
		t.Fatalf("expected float64 for min, got %T", minB)
	}
	if minBFloat != 200 {
		t.Fatalf("expected min 200 for tenant B, got %v", minBFloat)
	}
}

// TestDeptLevelIsolation 测试部门级隔离
func TestDeptLevelIsolation(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantID := ulidv2.Make()
	dept1 := ulidv2.Make()
	dept2 := ulidv2.Make()

	// 部门1的普通用户上下文
	ctxDept1 := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept1,
		IsAdmin:  false,
	})

	// 部门2的普通用户上下文
	ctxDept2 := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept2,
		IsAdmin:  false,
	})

	// 在部门1创建记录
	m1 := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 100}
	if err := repo.Create(ctxDept1, m1); err != nil {
		t.Fatalf("create for dept1: %v", err)
	}

	// 在部门2创建记录
	m2 := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 200}
	if err := repo.Create(ctxDept2, m2); err != nil {
		t.Fatalf("create for dept2: %v", err)
	}

	// 部门1只能看到自己的记录
	count1, err := repo.Count(ctxDept1, "1=1")
	if err != nil {
		t.Fatalf("count for dept1: %v", err)
	}
	if count1 != 1 {
		t.Fatalf("expected 1 record for dept1, got %d", count1)
	}

	// 部门2只能看到自己的记录
	count2, err := repo.Count(ctxDept2, "1=1")
	if err != nil {
		t.Fatalf("count for dept2: %v", err)
	}
	if count2 != 1 {
		t.Fatalf("expected 1 record for dept2, got %d", count2)
	}

	// 部门1不能访问部门2的记录
	if _, err := repo.FindByID(ctxDept1, m2.ID); err == nil {
		t.Fatalf("expected not found for cross-dept access")
	}
}

// TestAdminCanAccessAllDepts 测试管理员可以跨部门访问
func TestAdminCanAccessAllDepts(t *testing.T) {
	db := openAggregateTestDB(t)
	repo := NewRepository[tenantAggregateTestModel](db)

	tenantID := ulidv2.Make()
	dept1 := ulidv2.Make()
	dept2 := ulidv2.Make()

	// 部门1的普通用户上下文
	ctxDept1 := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept1,
		IsAdmin:  false,
	})

	// 部门2的普通用户上下文
	ctxDept2 := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept2,
		IsAdmin:  false,
	})

	// 管理员上下文
	ctxAdmin := WithTenantContext(context.Background(), TenantContext{
		TenantID: tenantID,
		DeptID:   &dept1, // 管理员也有部门，但不受限制
		IsAdmin:  true,
	})

	// 在部门1创建记录
	m1 := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 100}
	if err := repo.Create(ctxDept1, m1); err != nil {
		t.Fatalf("create for dept1: %v", err)
	}

	// 在部门2创建记录
	m2 := &tenantAggregateTestModel{ID: ulidv2.Make().String(), Amount: 200}
	if err := repo.Create(ctxDept2, m2); err != nil {
		t.Fatalf("create for dept2: %v", err)
	}

	// 管理员可以看到所有部门的记录
	countAdmin, err := repo.Count(ctxAdmin, "1=1")
	if err != nil {
		t.Fatalf("count for admin: %v", err)
	}
	if countAdmin != 2 {
		t.Fatalf("expected 2 records for admin, got %d", countAdmin)
	}

	// 管理员可以访问任意部门的记录
	if _, err := repo.FindByID(ctxAdmin, m1.ID); err != nil {
		t.Fatalf("admin should access dept1 record: %v", err)
	}
	if _, err := repo.FindByID(ctxAdmin, m2.ID); err != nil {
		t.Fatalf("admin should access dept2 record: %v", err)
	}

	// 管理员的求和应该包含所有部门
	sumAdmin, err := repo.Sum(ctxAdmin, "amount", "")
	if err != nil {
		t.Fatalf("sum for admin: %v", err)
	}
	if sumAdmin != 300 {
		t.Fatalf("expected sum 300 for admin, got %f", sumAdmin)
	}
}
