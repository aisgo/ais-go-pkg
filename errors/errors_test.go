package errors

import (
	errorspkg "errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func resetHTTPOverrides() {
	httpStatusMu.Lock()
	defer httpStatusMu.Unlock()
	httpStatusOverrides = make(map[ErrorCode]int)
	httpStatusResolverFn = nil
}

func TestBizErrorIsAndUnwrap(t *testing.T) {
	cause := errorspkg.New("root")
	err := Wrap(ErrCodeNotFound, "missing", cause)

	if !Is(err, ErrNotFound) {
		t.Fatalf("expected Is to match ErrNotFound")
	}
	if !errorspkg.Is(err, cause) {
		t.Fatalf("expected errors.Is to match cause")
	}
}

func TestToGRPCError(t *testing.T) {
	err := New(ErrCodeInvalidArgument, "bad")
	grpcErr := ToGRPCError(err)
	st, ok := status.FromError(grpcErr)
	if !ok {
		t.Fatalf("expected grpc status")
	}
	if st.Code() != codes.InvalidArgument {
		t.Fatalf("unexpected grpc code: %v", st.Code())
	}
}

func TestFromGRPCError(t *testing.T) {
	grpcErr := status.Error(codes.NotFound, "missing")
	bizErr := FromGRPCError(grpcErr)
	if bizErr == nil {
		t.Fatalf("expected biz error")
	}
	if bizErr.Code != ErrCodeNotFound {
		t.Fatalf("unexpected code: %v", bizErr.Code)
	}
	if bizErr.Message != "missing" {
		t.Fatalf("unexpected message: %q", bizErr.Message)
	}
}

func TestToHTTPResponse(t *testing.T) {
	resetHTTPOverrides()
	defer resetHTTPOverrides()

	statusCode, body := ToHTTPResponse(nil)
	if statusCode != 200 {
		t.Fatalf("unexpected status for nil error: %d", statusCode)
	}
	if body["code"].(int) != 0 {
		t.Fatalf("unexpected code for nil error: %v", body["code"])
	}

	RegisterHTTPStatus(ErrCodeNotFound, 410)
	statusCode, _ = ToHTTPResponse(New(ErrCodeNotFound, "gone"))
	if statusCode != 410 {
		t.Fatalf("expected override status, got: %d", statusCode)
	}

	resetHTTPOverrides()
	SetHTTPStatusResolver(func(code ErrorCode) (int, bool) {
		if code == ErrCodePermissionDenied {
			return 451, true
		}
		return 0, false
	})
	statusCode, _ = ToHTTPResponse(New(ErrCodePermissionDenied, "deny"))
	if statusCode != 451 {
		t.Fatalf("expected resolver status, got: %d", statusCode)
	}
}
