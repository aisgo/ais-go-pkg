# Tenant-Aware Repository Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Enforce default multi-tenant isolation in repository and make Update safer by replacing Save semantics.

**Architecture:** Add a tenant context helper, apply tenant scope to all queries, auto-fill tenant fields on writes, and change Update to struct Updates + tenant filtering. Use sqlite in-memory tests to verify behavior.

**Tech Stack:** Go 1.25.5, GORM, oklog/ulid/v2, gorm.io/driver/sqlite (tests only)

---

### Task 1: Tenant context helpers

**Files:**
- Create: `repository/tenant.go`
- Test: `repository/tenant_context_test.go`

**Step 1: Write the failing test**

```go
package repository

import (
    "context"
    "testing"

    ulidv2 "github.com/oklog/ulid/v2"
)

func TestTenantContextRoundTrip(t *testing.T) {
    tc := TenantContext{
        TenantID: ulidv2.Make(),
        IsAdmin:  false,
    }

    ctx := WithTenantContext(context.Background(), tc)
    got, ok := TenantFromContext(ctx)

    if !ok {
        t.Fatalf("expected tenant context")
    }
    if got.TenantID != tc.TenantID {
        t.Fatalf("unexpected tenant id: %v", got.TenantID)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./repository -run TestTenantContextRoundTrip`
Expected: FAIL with "undefined: TenantContext" or "undefined: WithTenantContext"

**Step 3: Write minimal implementation**

```go
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

type tenantCtxKey struct{}

func WithTenantContext(ctx context.Context, tc TenantContext) context.Context {
    return context.WithValue(ctx, tenantCtxKey{}, tc)
}

func TenantFromContext(ctx context.Context) (TenantContext, bool) {
    v := ctx.Value(tenantCtxKey{})
    if v == nil {
        return TenantContext{}, false
    }
    tc, ok := v.(TenantContext)
    return tc, ok
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./repository -run TestTenantContextRoundTrip`
Expected: PASS

**Step 5: Commit**

```bash
git add repository/tenant.go repository/tenant_context_test.go
git commit -m "feat: add tenant context helpers"
```

---

### Task 2: Query tenant scope (read path)

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Modify: `repository/query.go`
- Create: `repository/tenant_scope.go`
- Test: `repository/tenant_repository_test.go`

**Step 1: Write the failing test**

```go
package repository

import (
    "context"
    "testing"

    ulidv2 "github.com/oklog/ulid/v2"
    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
)

type tenantTestModel struct {
    BaseModel
    TenantID ulidv2.ULID  `gorm:"column:tenant_id;type:char(26);not null"`
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

    a := &tenantTestModel{Name: "a", TenantID: tenantA}
    b := &tenantTestModel{Name: "b", TenantID: tenantB}

    if err := repo.Create(WithTenantContext(context.Background(), TenantContext{TenantID: tenantA}), a); err != nil {
        t.Fatalf("create a: %v", err)
    }
    if err := repo.Create(WithTenantContext(context.Background(), TenantContext{TenantID: tenantB}), b); err != nil {
        t.Fatalf("create b: %v", err)
    }

    ctxA := WithTenantContext(context.Background(), TenantContext{TenantID: tenantA})
    if _, err := repo.FindByID(ctxA, b.ID.String()); err == nil {
        t.Fatalf("expected not found for cross-tenant id")
    }

    if _, err := repo.FindByID(ctxA, a.ID.String()); err != nil {
        t.Fatalf("expected find by id: %v", err)
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./repository -run TestTenantFindByIDScope`
Expected: FAIL (missing tenant scope or missing sqlite driver)

**Step 3: Add sqlite test dependency**

```go
// go.mod
require (
    // ...
    gorm.io/driver/sqlite v1.6.0
)
```

Run: `go mod tidy`

**Step 4: Implement tenant scope and hook it into queries**

```go
// repository/tenant_scope.go
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
        return db.AddError(errors.ErrUnauthenticated)
    }

    if err := r.ensureTenantFields(); err != nil {
        return db.AddError(err)
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
    // dept_id is optional; only required for filtering when provided
    return nil
}
```

```go
// repository/query.go (inside buildQuery)
func (r *RepositoryImpl[T]) buildQuery(ctx context.Context, opts *QueryOption) *gorm.DB {
    db := r.withContext(ctx)
    db = r.applyTenantScope(ctx, db)
    // existing logic...
    return db
}
```

**Step 5: Run test to verify it passes**

Run: `go test ./repository -run TestTenantFindByIDScope`
Expected: PASS

**Step 6: Commit**

```bash
git add go.mod go.sum repository/query.go repository/tenant_scope.go repository/tenant_repository_test.go
git commit -m "feat: enforce tenant scope on queries"
```

---

### Task 3: Auto-fill tenant fields on Create/CreateBatch/UpsertBatch

**Files:**
- Modify: `repository/crud.go`
- Modify: `repository/tenant_scope.go`
- Test: `repository/tenant_repository_test.go`

**Step 1: Write failing tests**

```go
func TestTenantCreateAutoFill(t *testing.T) {
    db := openTenantTestDB(t)
    repo := NewRepository[tenantTestModel](db)

    tenant := ulidv2.Make()
    ctx := WithTenantContext(context.Background(), TenantContext{TenantID: tenant})

    m := &tenantTestModel{Name: "auto"}
    if err := repo.Create(ctx, m); err != nil {
        t.Fatalf("create: %v", err)
    }
    if m.TenantID != tenant {
        t.Fatalf("tenant not set")
    }
}

func TestTenantCreateMissingContext(t *testing.T) {
    db := openTenantTestDB(t)
    repo := NewRepository[tenantTestModel](db)

    m := &tenantTestModel{Name: "no-ctx"}
    if err := repo.Create(context.Background(), m); err == nil {
        t.Fatalf("expected error without tenant context")
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./repository -run TestTenantCreateAutoFill`
Expected: FAIL (tenant not set)

**Step 3: Implement auto-fill helpers**

```go
// repository/tenant_scope.go
func (r *RepositoryImpl[T]) setTenantFields(ctx context.Context, model any) error {
    tc, ok := TenantFromContext(ctx)
    if !ok {
        return errors.ErrUnauthenticated
    }

    schema, err := r.getSchema()
    if err != nil {
        return err
    }

    tenantField, ok := schema.FieldsByDBName[tenantColumn]
    if !ok {
        return errors.ErrInvalidArgument
    }

    rv := reflect.ValueOf(model)
    if err := tenantField.Set(context.Background(), rv, tc.TenantID); err != nil {
        return err
    }

    if tc.DeptID != nil {
        if deptField, ok := schema.FieldsByDBName[deptColumn]; ok {
            if err := deptField.Set(context.Background(), rv, tc.DeptID); err != nil {
                return err
            }
        }
    }

    return nil
}
```

```go
// repository/crud.go (Create/CreateBatch/UpsertBatch)
if err := r.setTenantFields(ctx, model); err != nil {
    return err
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./repository -run TestTenantCreateAutoFill`
Expected: PASS

**Step 5: Commit**

```bash
git add repository/crud.go repository/tenant_scope.go repository/tenant_repository_test.go
git commit -m "feat: auto-fill tenant fields on create"
```

---

### Task 4: Safe Update semantics + tenant filtering on Update/Delete

**Files:**
- Modify: `repository/crud.go`
- Modify: `repository/interfaces.go`
- Modify: `repository/tenant_scope.go`
- Test: `repository/tenant_repository_test.go`

**Step 1: Write failing tests**

```go
func TestUpdateIgnoresZeroValues(t *testing.T) {
    db := openTenantTestDB(t)
    repo := NewRepository[tenantTestModel](db)
    tenant := ulidv2.Make()
    ctx := WithTenantContext(context.Background(), TenantContext{TenantID: tenant})

    m := &tenantTestModel{Name: "before"}
    if err := repo.Create(ctx, m); err != nil {
        t.Fatalf("create: %v", err)
    }

    m.Name = ""
    if err := repo.Update(ctx, m); err != nil {
        t.Fatalf("update: %v", err)
    }

    got, err := repo.FindByID(ctx, m.ID.String())
    if err != nil {
        t.Fatalf("find: %v", err)
    }
    if got.Name != "before" {
        t.Fatalf("expected name preserved, got: %s", got.Name)
    }
}

func TestUpdateByIDRespectsTenant(t *testing.T) {
    db := openTenantTestDB(t)
    repo := NewRepository[tenantTestModel](db)

    tenantA := ulidv2.Make()
    tenantB := ulidv2.Make()

    m := &tenantTestModel{Name: "before"}
    if err := repo.Create(WithTenantContext(context.Background(), TenantContext{TenantID: tenantA}), m); err != nil {
        t.Fatalf("create: %v", err)
    }

    ctxB := WithTenantContext(context.Background(), TenantContext{TenantID: tenantB})
    if err := repo.UpdateByID(ctxB, m.ID.String(), map[string]any{"name": "after"}, "name"); err == nil {
        t.Fatalf("expected not found for cross-tenant update")
    }
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./repository -run TestUpdateIgnoresZeroValues`
Expected: FAIL (Save overwrites zero values or no tenant filter)

**Step 3: Implement safe Update + tenant filters**

```go
// repository/interfaces.go (comment update)
// Update updates by primary key using Updates semantics (zero values ignored).
```

```go
// repository/crud.go (Update)
func (r *RepositoryImpl[T]) Update(ctx context.Context, model *T) error {
    if model == nil {
        return errors.ErrInvalidArgument
    }

    db := r.applyTenantScope(ctx, r.withContext(ctx))
    result := db.Model(model).Updates(model)
    if result.Error != nil {
        return result.Error
    }
    if result.RowsAffected == 0 {
        return gorm.ErrRecordNotFound
    }
    return nil
}
```

```go
// repository/crud.go (UpdateByID/Delete/HardDelete)
result := r.applyTenantScope(ctx, r.withContext(ctx)).
    Model(model).
    Where("id = ?", id).
    Updates(filteredUpdates)
```

```go
// repository/crud.go (Delete/HardDelete)
result := r.applyTenantScope(ctx, r.withContext(ctx)).
    Where("id = ?", id).
    Delete(model)
```

**Step 4: Run tests to verify they pass**

Run: `go test ./repository -run TestUpdateIgnoresZeroValues`
Expected: PASS

**Step 5: Commit**

```bash
git add repository/crud.go repository/interfaces.go repository/tenant_repository_test.go
git commit -m "feat: safe update semantics with tenant filtering"
```

---

### Task 5: Prevent tenant field mutation + docs update

**Files:**
- Modify: `repository/crud.go`
- Modify: `README.md`
- Test: `repository/tenant_repository_test.go`

**Step 1: Write failing test**

```go
func TestUpdateByIDCannotMutateTenant(t *testing.T) {
    db := openTenantTestDB(t)
    repo := NewRepository[tenantTestModel](db)

    tenant := ulidv2.Make()
    ctx := WithTenantContext(context.Background(), TenantContext{TenantID: tenant})

    m := &tenantTestModel{Name: "before"}
    if err := repo.Create(ctx, m); err != nil {
        t.Fatalf("create: %v", err)
    }

    otherTenant := ulidv2.Make()
    if err := repo.UpdateByID(ctx, m.ID.String(), map[string]any{"tenant_id": otherTenant}, "tenant_id"); err == nil {
        t.Fatalf("expected update to reject tenant_id change")
    }
}
```

**Step 2: Run test to verify it fails**

Run: `go test ./repository -run TestUpdateByIDCannotMutateTenant`
Expected: FAIL

**Step 3: Implement filterUpdates safeguard**

```go
// repository/crud.go (filterUpdates)
if field.DBName == tenantColumn || field.DBName == deptColumn {
    continue
}
```

**Step 4: Update README (tenant requirement)**

```md
### Repository - Multi-Tenant

Repository is tenant-enforced by default. Inject TenantContext into ctx before calling repository methods.

```go
ctx := repository.WithTenantContext(ctx, repository.TenantContext{
    TenantID: tenantID,
    DeptID:   deptID,
    IsAdmin:  false,
})

err := repo.Create(ctx, user)
```
```

**Step 5: Run tests to verify they pass**

Run: `go test ./repository -run TestUpdateByIDCannotMutateTenant`
Expected: PASS

**Step 6: Commit**

```bash
git add repository/crud.go README.md repository/tenant_repository_test.go
git commit -m "feat: lock tenant fields and document tenant context"
```

---

### Task 6: Full test run

**Step 1: Run full test suite**

Run: `go test ./...`
Expected: PASS

**Step 2: Commit (if needed)**

Only if any fixes were applied after Task 5.

