# Tenant-Aware Repository Design

**Goal:** Enforce default multi-tenant isolation in repository layer and make updates safer by removing Save-all-fields semantics.

## Context & Constraints
- tenant_id / dept_id are ULID (oklog/ulid/v2), injected into context by upstream (Fiber Locals -> context.Context).
- All multi-tenant tables use column names `tenant_id` and `dept_id`.
- Models are not guaranteed to embed a shared base struct; repository must handle field presence via GORM schema.

## Core Design
1) **Tenant context**
- Add repository-level TenantContext (TenantID, DeptID, IsAdmin, Roles, PolicyVersion).
- Provide WithTenantContext(ctx, tc) and TenantFromContext(ctx).
- Repository uses only context.Context; no Fiber dependency.

2) **Default tenant scope (mandatory)**
- All query paths (Find/Count/Page/Aggregate) automatically apply tenant scope when context has tenant info:
  - Always filter by tenant_id
  - If !IsAdmin and dept_id != nil, also filter by dept_id
- If context has no tenant info, return ErrUnauthenticated (safer than returning all data).

3) **Write path safety**
- Create/CreateBatch/UpsertBatch auto-populate tenant_id/dept_id on models using GORM schema/field setters.
- If model lacks tenant fields or ctx has no tenant, return explicit error (ErrInvalidArgument or ErrUnauthenticated).

4) **Update semantics**
- Replace Update Save semantics with Updates(struct) + tenant scope + primary key check.
- Zero-value updates require explicit UpdateByID (map + allowedFields) or a new UpdateWithFields API.

5) **Delete/Update filtering**
- UpdateByID/Delete/HardDelete always include tenant scope in WHERE clause.

## Error Strategy
- Missing tenant in context -> errors.ErrUnauthenticated
- Missing tenant fields on model -> errors.ErrInvalidArgument
- Cross-tenant access -> errors.ErrPermissionDenied (via tenant scope rows-affected = 0)

## Compatibility
- This is a behavior change: repository now requires tenant context by default.
- For non-tenant models, either add tenant fields or provide explicit opt-out (if requested later).

## Testing Strategy
- Use sqlite in-memory for repository tests.
- Cover: auto-fill tenant fields, tenant-filtered reads, safe Update semantics, tenant-filtered Update/Delete.

