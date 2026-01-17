package postgres

import (
	"strings"
	"testing"
)

func TestSanitizeDSN(t *testing.T) {
	dsn := "postgres://user:secret@localhost:5432/db?sslmode=disable"
	got := sanitizeDSN(dsn)
	if strings.Contains(got, "secret") {
		t.Fatalf("password leaked in sanitized DSN: %s", got)
	}
	if !strings.Contains(got, "***") && !strings.Contains(got, "%2A%2A%2A") {
		t.Fatalf("expected masked password, got: %s", got)
	}
}

func TestSanitizeDSNInvalid(t *testing.T) {
	dsn := "postgres://%zz"
	got := sanitizeDSN(dsn)
	if got != dsn {
		t.Fatalf("expected original DSN on parse error")
	}
}
