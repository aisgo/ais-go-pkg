package response

import (
	"net/http"

	"ais.local/ais-go-pkg/errors"

	"github.com/gofiber/fiber/v3"
)

/* ========================================================================
 * Response - 统一响应处理
 * ========================================================================
 * 职责: 提供统一的 HTTP 响应处理函数
 * 特性:
 *   - 标准 JSON 响应格式
 *   - 与 errors 包集成，自动识别 BizError
 *   - 支持分页响应
 *   - 快捷响应函数
 * ======================================================================== */

// newResp 创建响应对象
func newResp(code int, msg string, data interface{}) *Result {
	resp := &Result{
		Code: code,
		Msg:  msg,
	}

	// 确保 data 字段不为 nil
	// 注意：当 resp.data == []interface{}{} 时，序列化为 null
	if data == nil {
		resp.Data = &struct{}{}
	} else {
		resp.Data = data
	}

	return resp
}

// respJSONWithStatusCode 返回 JSON 响应
func respJSONWithStatusCode(c fiber.Ctx, code int, msg string, data ...interface{}) error {
	var firstData interface{}
	if len(data) > 0 {
		firstData = data[0]
	}

	// 确保 HTTP 状态码在有效范围内
	if code > http.StatusNetworkAuthenticationRequired || code < http.StatusContinue {
		code = http.StatusInternalServerError
	}

	resp := newResp(code, msg, firstData)
	return c.Status(code).JSON(resp)
}

/* ========================================================================
 * 成功响应
 * ======================================================================== */

// Ok 返回成功响应 (默认消息 "ok")
func Ok(c fiber.Ctx) error {
	return respJSONWithStatusCode(c, http.StatusOK, "ok")
}

// OkWithData 返回成功响应（带数据）
func OkWithData(c fiber.Ctx, data interface{}) error {
	return respJSONWithStatusCode(c, http.StatusOK, "ok", data)
}

// OkWithMsg 返回成功响应（自定义消息）
func OkWithMsg(c fiber.Ctx, msg string, data ...interface{}) error {
	return respJSONWithStatusCode(c, http.StatusOK, msg, data...)
}

// Success 返回成功响应（自定义消息和数据）
func Success(c fiber.Ctx, msg string, data interface{}) error {
	return respJSONWithStatusCode(c, http.StatusOK, msg, data)
}

/* ========================================================================
 * 错误响应
 * ======================================================================== */

// Error 返回错误响应
// 自动识别 BizError 类型，使用其 HTTP 状态码和错误消息
func Error(c fiber.Ctx, err error) error {
	if err == nil {
		return Ok(c)
	}

	// 检查是否为 BizError
	if bizErr, ok := errors.AsBizError(err); ok {
		statusCode, resp := errors.ToHTTPResponse(bizErr)
		return c.Status(statusCode).JSON(Result{
			Code: resp["code"].(int),
			Msg:  resp["msg"].(string),
			Data: &struct{}{},
		})
	}

	// 普通错误，返回 500
	return respJSONWithStatusCode(c, http.StatusInternalServerError, err.Error())
}

// ErrorWithCode 返回错误响应（指定 HTTP 状态码）
func ErrorWithCode(c fiber.Ctx, code int, err error) error {
	if err == nil {
		return c.Status(code).JSON(Result{
			Code: code,
			Msg:  "ok",
			Data: &struct{}{},
		})
	}

	// 检查是否为 BizError
	if bizErr, ok := errors.AsBizError(err); ok {
		// 优先使用 BizError 的 HTTP 状态码，但允许覆盖
		statusCode, _ := errors.ToHTTPResponse(bizErr)
		if code != http.StatusInternalServerError {
			statusCode = code
		}
		return c.Status(statusCode).JSON(Result{
			Code: int(bizErr.Code),
			Msg:  bizErr.Message,
			Data: &struct{}{},
		})
	}

	return respJSONWithStatusCode(c, code, err.Error())
}

// ErrorWithMsg 返回错误响应（自定义消息）
func ErrorWithMsg(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusInternalServerError, msg)
}

/* ========================================================================
 * 分页响应
 * ======================================================================== */

// PageData 返回分页数据
func PageData(c fiber.Ctx, list interface{}, total int64, page, pageSize int) error {
	pageResult := &PageResult{
		List:     list,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	}
	return OkWithData(c, pageResult)
}

/* ========================================================================
 * 快捷响应
 * ======================================================================== */

// BadRequest 返回 400 错误
func BadRequest(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusBadRequest, msg)
}

// Unauthorized 返回 401 错误
func Unauthorized(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusUnauthorized, msg)
}

// Forbidden 返回 403 错误
func Forbidden(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusForbidden, msg)
}

// NotFound 返回 404 错误
func NotFound(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusNotFound, msg)
}

// InternalError 返回 500 错误
func InternalError(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusInternalServerError, msg)
}

// ServiceUnavailable 返回 503 错误
func ServiceUnavailable(c fiber.Ctx, msg string) error {
	return respJSONWithStatusCode(c, http.StatusServiceUnavailable, msg)
}
