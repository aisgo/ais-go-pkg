package validator

import (
	"fmt"
	"strings"
)

/* ========================================================================
 * Validator Types - 验证器类型定义
 * ========================================================================
 * 职责: 定义验证错误类型
 * ======================================================================== */

const (
	// tagCustom 自定义错误消息标签名
	tagCustom = "error_msg"
	// ruleSeparator 规则分隔符，用于分隔多个规则
	ruleSeparator = "|"
	// keyValueSep 键值分隔符，用于分隔规则名和错误消息
	keyValueSep = ":"
)

// ValidationError 按字段分组的验证错误
// 使用示例:
//
//	type UserRequest struct {
//	    Email    string `validate:"required,email" error_msg:"required:邮箱必填|email:邮箱格式错误"`
//	    Password string `validate:"required,min=8" error_msg:"required:密码必填|min:密码至少8位"`
//	}
type ValidationError struct {
	Errors map[string][]string // 字段名 -> 错误消息列表
}

// Error 实现 error 接口
func (v ValidationError) Error() string {
	var sb strings.Builder
	for field, msgs := range v.Errors {
		sb.WriteString(fmt.Sprintf("%s: %s; ", field, strings.Join(msgs, ", ")))
	}
	return sb.String()
}

// HasErrors 检查是否有验证错误
func (v ValidationError) HasErrors() bool {
	return len(v.Errors) > 0
}

// Add 添加字段错误
func (v *ValidationError) Add(field, message string) {
	if v.Errors == nil {
		v.Errors = make(map[string][]string)
	}
	v.Errors[field] = append(v.Errors[field], message)
}

// Get 获取字段错误消息
func (v *ValidationError) Get(field string) []string {
	if v.Errors == nil {
		return nil
	}
	return v.Errors[field]
}
