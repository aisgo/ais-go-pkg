package validator

import (
	"testing"
	"time"
)

func TestValidate_AllowsStructValueInput(t *testing.T) {
	t.Parallel()

	type Inner struct {
		Email string `validate:"required,email" error_msg:"required:email required|email:email invalid"`
	}
	type Req struct {
		Inner Inner
		When  time.Time
	}

	v := New()

	if err := v.Validate(Req{}); err == nil {
		t.Fatalf("expected validation error, got nil")
	}
}
