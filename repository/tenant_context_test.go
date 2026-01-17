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
