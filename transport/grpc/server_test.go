package grpc

import (
	"context"
	"errors"
	"testing"

	"github.com/aisgo/ais-go-pkg/logger"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func TestRecoveryInterceptor(t *testing.T) {
	log := logger.NewNop()
	interceptor := recoveryInterceptor(log)

	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		panic("boom")
	})
	if err == nil {
		t.Fatalf("expected error")
	}
	st, ok := status.FromError(err)
	if !ok {
		t.Fatalf("expected status error")
	}
	if st.Code() != codes.Internal {
		t.Fatalf("unexpected code: %v", st.Code())
	}
}

func TestLoggingInterceptor(t *testing.T) {
	log := logger.NewNop()
	interceptor := loggingInterceptor(log)

	expectedErr := errors.New("fail")
	_, err := interceptor(context.Background(), nil, &grpc.UnaryServerInfo{FullMethod: "/test.Service/Method"}, func(ctx context.Context, req interface{}) (interface{}, error) {
		return nil, expectedErr
	})
	if !errors.Is(err, expectedErr) {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestNewListenerMonolith(t *testing.T) {
	inProc := NewInProcListener()
	listener, err := NewListener(ListenerProviderParams{
		Config: Config{Mode: "monolith"},
		Logger: logger.NewNop(),
	}, inProc)
	if err != nil {
		t.Fatalf("new listener: %v", err)
	}
	if listener != inProc.Listener {
		t.Fatalf("expected in-proc listener")
	}
}

func TestNewListenerTCP(t *testing.T) {
	inProc := NewInProcListener()
	listener, err := NewListener(ListenerProviderParams{
		Config: Config{Mode: "microservice", Port: 0},
		Logger: logger.NewNop(),
	}, inProc)
	if err != nil {
		t.Fatalf("new listener: %v", err)
	}
	defer listener.Close()
}
