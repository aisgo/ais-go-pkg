package grpc

import (
	"context"
	"fmt"
	"net"
	"runtime/debug"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/backoff"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/status"
	"google.golang.org/grpc/test/bufconn"

	"ais.local/ais-go-pkg/logger"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

/* ========================================================================
 * gRPC Server - 模块间通信
 * ========================================================================
 * 职责: 提供 gRPC 服务，支持 TCP 和 BufConn 模式
 * 技术: google.golang.org/grpc
 * ======================================================================== */

const bufSize = 1024 * 1024

type Config struct {
	Port int    `yaml:"port"`
	Mode string `yaml:"mode"` // monolith or microservice
}

type ListenerProviderParams struct {
	fx.In
	Config Config
	Logger *logger.Logger
}

// InProcListener 是一个全局的 bufconn 监听器，仅在 Monolith 模式下使用
type InProcListener struct {
	*bufconn.Listener
}

func NewInProcListener() *InProcListener {
	return &InProcListener{Listener: bufconn.Listen(bufSize)}
}

// NewListener 创建 gRPC 监听器 (TCP 或 BufConn)
func NewListener(p ListenerProviderParams, inProc *InProcListener) (net.Listener, error) {
	if p.Config.Mode == "monolith" {
		p.Logger.Info("Using In-Memory gRPC Listener (BufConn)")
		return inProc.Listener, nil
	}

	p.Logger.Info("Using TCP gRPC Listener", zap.Int("port", p.Config.Port))
	return net.Listen("tcp", fmt.Sprintf(":%d", p.Config.Port))
}

type ServerParams struct {
	fx.In
	Lc       fx.Lifecycle
	Listener net.Listener
	Logger   *logger.Logger
}

// recoveryInterceptor 创建 panic 恢复拦截器
func recoveryInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		defer func() {
			if r := recover(); r != nil {
				log.Error("gRPC panic recovered",
					zap.Any("panic", r),
					zap.String("method", info.FullMethod),
					zap.String("stack", string(debug.Stack())),
				)
				err = status.Errorf(codes.Internal, "internal server error")
			}
		}()
		return handler(ctx, req)
	}
}

// loggingInterceptor 创建日志拦截器
func loggingInterceptor(log *logger.Logger) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp interface{}, err error) {
		start := time.Now()
		resp, err = handler(ctx, req)
		duration := time.Since(start)

		if err != nil {
			log.Warn("gRPC request failed",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
				zap.Error(err),
			)
		} else if duration > 500*time.Millisecond {
			// 记录慢请求
			log.Warn("gRPC slow request",
				zap.String("method", info.FullMethod),
				zap.Duration("duration", duration),
			)
		}

		return resp, err
	}
}

// NewServer 创建 gRPC Server 并管理生命周期
func NewServer(p ServerParams) *grpc.Server {
	// 配置拦截器: Recovery, Logging
	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			recoveryInterceptor(p.Logger), // Panic 恢复
			loggingInterceptor(p.Logger),  // 日志记录
		),
		// Keepalive 配置，防止空闲连接堆积
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     5 * time.Minute,  // 空闲连接最大时间
			MaxConnectionAge:      30 * time.Minute, // 连接最大生命周期
			MaxConnectionAgeGrace: 10 * time.Second, // 优雅关闭等待时间
			Time:                  30 * time.Second, // 发送 ping 的间隔
			Timeout:               10 * time.Second, // ping 超时时间
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second, // 客户端 ping 最小间隔
			PermitWithoutStream: true,             // 允许没有活跃 stream 时 ping
		}),
		// 限制最大消息大小（防止 OOM）
		grpc.MaxRecvMsgSize(16 * 1024 * 1024), // 16MB
		grpc.MaxSendMsgSize(16 * 1024 * 1024), // 16MB
	}
	s := grpc.NewServer(opts...)

	p.Lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			// 创建 channel 用于传递启动错误
			errChan := make(chan error, 1)

			go func() {
				p.Logger.Info("Starting gRPC Server")
				if err := s.Serve(p.Listener); err != nil {
					p.Logger.Error("gRPC Server failed", zap.Error(err))
					errChan <- err
				}
			}()

			// 等待一小段时间，确保服务器成功启动
			select {
			case err := <-errChan:
				// 服务器立即启动失败
				return err
			case <-time.After(100 * time.Millisecond):
				// 服务器似乎成功启动
				return nil
			case <-ctx.Done():
				// 上下文被取消
				return ctx.Err()
			}
		},
		OnStop: func(ctx context.Context) error {
			p.Logger.Info("Stopping gRPC Server")
			s.GracefulStop()
			return nil
		},
	})
	return s
}

// ClientFactory 用于创建 gRPC 客户端
type ClientFactory func(target string) (*grpc.ClientConn, error)

// NewClientFactory 返回一个创建 ClientConn 的函数
// 如果是 Monolith 模式，自动使用 BufConn Dialer
func NewClientFactory(cfg Config, inProc *InProcListener) ClientFactory {
	return func(target string) (*grpc.ClientConn, error) {
		opts := []grpc.DialOption{
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			// 添加默认超时配置
			grpc.WithDefaultCallOptions(
				grpc.MaxCallRecvMsgSize(16*1024*1024), // 16MB
				grpc.MaxCallSendMsgSize(16*1024*1024), // 16MB
			),
			// 添加连接超时配置
			grpc.WithConnectParams(grpc.ConnectParams{
				Backoff: backoff.Config{
					MaxDelay:  30 * time.Second,
					BaseDelay: 1 * time.Second,
				},
				MinConnectTimeout: 10 * time.Second,
			}),
		}

		if cfg.Mode == "monolith" {
			// 在 Monolith 模式下，忽略 target IP，直接连接 InProcListener
			opts = append(opts, grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
				return inProc.Dial()
			}))
			// 使用 passthrough resolver，避免默认 dns resolver 导致 "produced zero addresses"
			target = "passthrough:///bufconn"
		}

		return grpc.NewClient(target, opts...)
	}
}
