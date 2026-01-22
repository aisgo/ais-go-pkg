package middleware

import (
	"errors"
	"net/http"
	"testing"
	"time"
)

func TestAuthHeaderSignerAndVerifier(t *testing.T) {
	now := time.Unix(1700000000, 0)
	signer := NewAuthHeaderSigner(&AuthHeaderSignerConfig{
		Enabled: true,
		Secret:  "secret",
		Issuer:  "gateway",
		NowFunc: func() time.Time { return now },
	})
	user := &UserInfo{
		UserID:   "u1",
		TenantID: "t1",
		Roles:    []string{"admin"},
	}
	headers, err := signer.BuildHeaders(user)
	if err != nil {
		t.Fatalf("BuildHeaders error: %v", err)
	}
	if headers.Signature == "" {
		t.Fatalf("signature should not be empty")
	}

	httpHeader := http.Header{}
	WriteAuthHeaders(httpHeader, headers)
	values, err := ParseAuthHeaderValuesFromHeader(httpHeader)
	if err != nil {
		t.Fatalf("ParseAuthHeaderValuesFromHeader error: %v", err)
	}

	verifier := NewAuthHeaderVerifier(&AuthHeaderVerifierConfig{
		Enabled:        true,
		Secret:         "secret",
		AllowedIssuers: []string{"gateway"},
		NowFunc:        func() time.Time { return now.Add(10 * time.Second) },
	}, nil)
	ctx, err := verifier.Verify(values)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if ctx.User == nil || ctx.User.UserID != "u1" {
		t.Fatalf("unexpected user info: %+v", ctx.User)
	}
}

func TestAuthHeaderVerifierInvalidSignature(t *testing.T) {
	now := time.Unix(1700000000, 0)
	signer := NewAuthHeaderSigner(&AuthHeaderSignerConfig{
		Enabled: true,
		Secret:  "secret",
		Issuer:  "gateway",
		NowFunc: func() time.Time { return now },
	})
	headers, err := signer.BuildHeaders(&UserInfo{UserID: "u1"})
	if err != nil {
		t.Fatalf("BuildHeaders error: %v", err)
	}

	httpHeader := http.Header{}
	WriteAuthHeaders(httpHeader, headers)
	values, err := ParseAuthHeaderValuesFromHeader(httpHeader)
	if err != nil {
		t.Fatalf("ParseAuthHeaderValuesFromHeader error: %v", err)
	}

	verifier := NewAuthHeaderVerifier(&AuthHeaderVerifierConfig{
		Enabled:        true,
		Secret:         "wrong",
		AllowedIssuers: []string{"gateway"},
		NowFunc:        func() time.Time { return now },
	}, nil)
	if _, err := verifier.Verify(values); !errors.Is(err, ErrAuthHeaderInvalidSign) {
		t.Fatalf("expected invalid signature error, got: %v", err)
	}
}

func TestAuthHeaderVerifierExpired(t *testing.T) {
	now := time.Unix(1700000000, 0)
	signer := NewAuthHeaderSigner(&AuthHeaderSignerConfig{
		Enabled: true,
		Secret:  "secret",
		Issuer:  "gateway",
		NowFunc: func() time.Time { return now },
	})
	headers, err := signer.BuildHeaders(&UserInfo{UserID: "u1"})
	if err != nil {
		t.Fatalf("BuildHeaders error: %v", err)
	}

	httpHeader := http.Header{}
	WriteAuthHeaders(httpHeader, headers)
	values, err := ParseAuthHeaderValuesFromHeader(httpHeader)
	if err != nil {
		t.Fatalf("ParseAuthHeaderValuesFromHeader error: %v", err)
	}

	verifier := NewAuthHeaderVerifier(&AuthHeaderVerifierConfig{
		Enabled:        true,
		Secret:         "secret",
		AllowedIssuers: []string{"gateway"},
		MaxAge:         10 * time.Second,
		NowFunc:        func() time.Time { return now.Add(11 * time.Second) },
	}, nil)
	if _, err := verifier.Verify(values); !errors.Is(err, ErrAuthHeaderExpired) {
		t.Fatalf("expected expired error, got: %v", err)
	}
}

func TestAuthHeaderVerifierAllowEmptyUser(t *testing.T) {
	now := time.Unix(1700000000, 0)
	signer := NewAuthHeaderSigner(&AuthHeaderSignerConfig{
		Enabled: true,
		Secret:  "secret",
		Issuer:  "internal-service",
		NowFunc: func() time.Time { return now },
	})
	headers, err := signer.BuildHeaders(nil)
	if err != nil {
		t.Fatalf("BuildHeaders error: %v", err)
	}

	httpHeader := http.Header{}
	WriteAuthHeaders(httpHeader, headers)
	values, err := ParseAuthHeaderValuesFromHeader(httpHeader)
	if err != nil {
		t.Fatalf("ParseAuthHeaderValuesFromHeader error: %v", err)
	}

	verifier := NewAuthHeaderVerifier(&AuthHeaderVerifierConfig{
		Enabled:        true,
		Secret:         "secret",
		AllowedIssuers: []string{"internal-service"},
		AllowEmptyUser: true,
		NowFunc:        func() time.Time { return now },
	}, nil)
	ctx, err := verifier.Verify(values)
	if err != nil {
		t.Fatalf("Verify error: %v", err)
	}
	if ctx.User != nil {
		t.Fatalf("expected empty user, got: %+v", ctx.User)
	}
}
