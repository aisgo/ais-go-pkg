package errors

import (
	"errors"
	"fmt"
	"sync"

	"github.com/gofiber/fiber/v3"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

/* ========================================================================
 * AIS Error Package - 统一错误处理
 * ========================================================================
 * 职责: 定义业务错误码，提供错误包装和转换工具
 * 设计: 遵循 gRPC status codes 规范
 * ======================================================================== */

// ========================================================================
// 错误码定义
// ========================================================================

// ErrorCode 业务错误码
type ErrorCode int

const (
	// 通用错误 (1xxx)
	ErrCodeUnknown          ErrorCode = 1000 // 未知错误
	ErrCodeInvalidArgument  ErrorCode = 1001 // 参数无效
	ErrCodeNotFound         ErrorCode = 1002 // 资源不存在
	ErrCodeAlreadyExists    ErrorCode = 1003 // 资源已存在
	ErrCodePermissionDenied ErrorCode = 1004 // 权限不足
	ErrCodeUnauthenticated  ErrorCode = 1005 // 未认证
	ErrCodeInternal         ErrorCode = 1006 // 内部错误
	ErrCodeUnavailable      ErrorCode = 1007 // 服务不可用
	ErrCodeTimeout          ErrorCode = 1008 // 超时
	ErrCodeCanceled         ErrorCode = 1009 // 已取消
)

// ========================================================================
// 业务错误类型
// ========================================================================

// BizError 业务错误
type BizError struct {
	Code    ErrorCode // 业务错误码
	Message string    // 错误消息
	Cause   error     // 原始错误
}

// Error 实现 error 接口
func (e *BizError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

// Is 支持 errors.Is：按业务错误码匹配
func (e *BizError) Is(target error) bool {
	t, ok := target.(*BizError)
	if !ok {
		return false
	}
	return e.Code == t.Code
}

// Unwrap 支持 errors.Is 和 errors.As
func (e *BizError) Unwrap() error {
	return e.Cause
}

// ========================================================================
// 错误构造函数
// ========================================================================

// New 创建业务错误
func New(code ErrorCode, message string) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
	}
}

// Wrap 包装错误
func Wrap(code ErrorCode, message string, cause error) *BizError {
	return &BizError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Wrapf 格式化包装错误
func Wrapf(code ErrorCode, cause error, format string, args ...any) *BizError {
	return &BizError{
		Code:    code,
		Message: fmt.Sprintf(format, args...),
		Cause:   cause,
	}
}

// ========================================================================
// 预定义错误（便于 errors.Is 判断）
// ========================================================================

var (
	// 通用错误
	ErrInvalidArgument  = New(ErrCodeInvalidArgument, "invalid argument")
	ErrNotFound         = New(ErrCodeNotFound, "resource not found")
	ErrAlreadyExists    = New(ErrCodeAlreadyExists, "resource already exists")
	ErrPermissionDenied = New(ErrCodePermissionDenied, "permission denied")
	ErrUnauthenticated  = New(ErrCodeUnauthenticated, "unauthenticated")
	ErrInternal         = New(ErrCodeInternal, "internal error")
	ErrUnavailable      = New(ErrCodeUnavailable, "service unavailable")
	ErrTimeout          = New(ErrCodeTimeout, "timeout")
	ErrCanceled         = New(ErrCodeCanceled, "canceled")
)

// ========================================================================
// 错误判断辅助函数
// ========================================================================

// Is 判断错误是否为指定类型
func Is(err, target error) bool {
	return errors.Is(err, target)
}

// As 将错误转换为指定类型
func As(err error, target any) bool {
	return errors.As(err, target)
}

// Code 获取错误码
func Code(err error) ErrorCode {
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr.Code
	}
	return ErrCodeUnknown
}

// IsNotFound 判断是否为 NotFound 错误
func IsNotFound(err error) bool {
	return Code(err) == ErrCodeNotFound
}

// AsBizError 将错误转换为 BizError
// 返回值: (*BizError, bool) - 如果是 BizError 返回实例和 true，否则返回 nil 和 false
func AsBizError(err error) (*BizError, bool) {
	if err == nil {
		return nil, false
	}
	var bizErr *BizError
	if errors.As(err, &bizErr) {
		return bizErr, true
	}
	return nil, false
}

// ========================================================================
// gRPC 错误转换
// ========================================================================

// errorCodeToGRPCCode 错误码到 gRPC 状态码映射
var errorCodeToGRPCCode = map[ErrorCode]codes.Code{
	ErrCodeUnknown:          codes.Unknown,
	ErrCodeInvalidArgument:  codes.InvalidArgument,
	ErrCodeNotFound:         codes.NotFound,
	ErrCodeAlreadyExists:    codes.AlreadyExists,
	ErrCodePermissionDenied: codes.PermissionDenied,
	ErrCodeUnauthenticated:  codes.Unauthenticated,
	ErrCodeInternal:         codes.Internal,
	ErrCodeUnavailable:      codes.Unavailable,
	ErrCodeTimeout:          codes.DeadlineExceeded,
	ErrCodeCanceled:         codes.Canceled,
}

// ToGRPCError 将业务错误转换为 gRPC 错误
func ToGRPCError(err error) error {
	if err == nil {
		return nil
	}

	var bizErr *BizError
	if errors.As(err, &bizErr) {
		grpcCode, ok := errorCodeToGRPCCode[bizErr.Code]
		if !ok {
			grpcCode = codes.Unknown
		}
		return status.Error(grpcCode, bizErr.Message)
	}

	// 非业务错误，返回 Internal
	return status.Error(codes.Internal, err.Error())
}

// FromGRPCError 将 gRPC 错误转换为业务错误
func FromGRPCError(err error) *BizError {
	if err == nil {
		return nil
	}

	st, ok := status.FromError(err)
	if !ok {
		return Wrap(ErrCodeUnknown, "unknown error", err)
	}

	// gRPC 状态码到业务错误码映射
	var code ErrorCode
	switch st.Code() {
	case codes.InvalidArgument:
		code = ErrCodeInvalidArgument
	case codes.NotFound:
		code = ErrCodeNotFound
	case codes.AlreadyExists:
		code = ErrCodeAlreadyExists
	case codes.PermissionDenied:
		code = ErrCodePermissionDenied
	case codes.Unauthenticated:
		code = ErrCodeUnauthenticated
	case codes.Unavailable:
		code = ErrCodeUnavailable
	case codes.DeadlineExceeded:
		code = ErrCodeTimeout
	case codes.Canceled:
		code = ErrCodeCanceled
	default:
		code = ErrCodeInternal
	}

	return New(code, st.Message())
}

// ========================================================================
// HTTP 错误转换
// ========================================================================

// httpStatusCode 业务错误码到 HTTP 状态码映射
var httpStatusCode = map[ErrorCode]int{
	ErrCodeUnknown:          500,
	ErrCodeInvalidArgument:  400,
	ErrCodeNotFound:         404,
	ErrCodeAlreadyExists:    409,
	ErrCodePermissionDenied: 403,
	ErrCodeUnauthenticated:  401,
	ErrCodeInternal:         500,
	ErrCodeUnavailable:      503,
	ErrCodeTimeout:          504,
	ErrCodeCanceled:         499,
}

var (
	httpStatusMu         sync.RWMutex
	httpStatusOverrides  = make(map[ErrorCode]int)
	httpStatusResolverFn func(ErrorCode) (int, bool)
)

// RegisterHTTPStatus 注册业务错误码与 HTTP 状态码映射
func RegisterHTTPStatus(code ErrorCode, status int) {
	httpStatusMu.Lock()
	defer httpStatusMu.Unlock()
	httpStatusOverrides[code] = status
}

// SetHTTPStatusResolver 设置自定义的 HTTP 状态码解析器
// 解析器返回 (status, true) 表示命中，否则继续使用默认映射。
func SetHTTPStatusResolver(resolver func(ErrorCode) (int, bool)) {
	httpStatusMu.Lock()
	defer httpStatusMu.Unlock()
	httpStatusResolverFn = resolver
}

func resolveHTTPStatus(code ErrorCode) (int, bool) {
	httpStatusMu.RLock()
	if status, ok := httpStatusOverrides[code]; ok {
		httpStatusMu.RUnlock()
		return status, true
	}
	resolver := httpStatusResolverFn
	httpStatusMu.RUnlock()

	if resolver != nil {
		if status, ok := resolver(code); ok {
			return status, true
		}
	}
	return 0, false
}

// HTTPResponse HTTP 响应结构
type HTTPResponse struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
	Data any    `json:"data,omitempty"`
}

// ToHTTPResponse 将业务错误转换为 HTTP 响应
func ToHTTPResponse(err error) (int, fiber.Map) {
	if err == nil {
		return 200, fiber.Map{"code": 0, "msg": "success"}
	}

	var bizErr *BizError
	if errors.As(err, &bizErr) {
		statusCode, ok := resolveHTTPStatus(bizErr.Code)
		if !ok {
			statusCode, ok = httpStatusCode[bizErr.Code]
			if !ok {
				statusCode = 500
			}
		}
		return statusCode, fiber.Map{
			"code": int(bizErr.Code),
			"msg":  bizErr.Message,
		}
	}

	// 非业务错误
	return 500, fiber.Map{
		"code": 500,
		"msg":  "internal server error",
	}
}
