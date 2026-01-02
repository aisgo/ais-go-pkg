package validator

import (
	"fmt"
	"reflect"
	"strings"
	"sync"

	"github.com/go-playground/validator/v10"
)

/* ========================================================================
 * Validator - 自定义验证器
 * ========================================================================
 * 职责: 提供带自定义错误消息的结构体验证
 * 特性:
 *   - 支持 error_msg 标签定义自定义错误消息
 *   - 支持嵌套结构体验证
 *   - 类型缓存优化性能
 * 使用示例:
 *     type UserRequest struct {
 *         Email    string `validate:"required,email" error_msg:"required:邮箱必填|email:邮箱格式错误"`
 *         Password string `validate:"required,min=8" error_msg:"required:密码必填|min:密码至少8位"`
 *     }
 *     v := validator.New()
 *     if err := v.Validate(&req); err != nil {
 *         // 处理验证错误
 *     }
 * ======================================================================== */

// Validator 自定义验证器
type Validator struct {
	validator     *validator.Validate
	typeCache     *typeCache
	errorMsgCache map[string]map[string]string // 错误消息缓存
	mu            sync.RWMutex
}

// New 创建新的验证器
func New() *Validator {
	return &Validator{
		validator:     validator.New(),
		typeCache:     newTypeCache(),
		errorMsgCache: make(map[string]map[string]string),
	}
}

// Validate 验证结构体
// 返回 ValidationError 类型，包含按字段分组的错误消息
func (v *Validator) Validate(s interface{}) error {
	if s == nil {
		return nil
	}

	validationErrors := &ValidationError{Errors: make(map[string][]string)}
	v.validateRecursive(s, "", validationErrors)

	if validationErrors.HasErrors() {
		return validationErrors
	}
	return nil
}

// validateRecursive 递归验证结构体
func (v *Validator) validateRecursive(s interface{}, prefix string, validationErrors *ValidationError) {
	value := reflect.ValueOf(s)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}

	if value.Kind() != reflect.Struct {
		return
	}

	// 使用缓存获取字段信息
	fields := v.typeCache.getFieldsInfo(value.Type())

	// 遍历结构体字段
	for _, fieldInfo := range fields {
		fieldValue := value.FieldByName(fieldInfo.name)
		fullFieldName := fieldInfo.name
		if prefix != "" {
			fullFieldName = fmt.Sprintf("%s.%s", prefix, fieldInfo.name)
		}

		// 递归处理嵌套结构体
		if fieldInfo.isStruct {
			// 处理指针类型的嵌套结构体
			if fieldInfo.isPtr {
				if fieldValue.IsNil() {
					continue // 跳过 nil 指针
				}
				fieldValue = fieldValue.Elem()
			}
			v.validateRecursive(fieldValue.Interface(), fullFieldName, validationErrors)
			continue
		}

		// 跳过没有验证标签的字段
		if fieldInfo.validateTag == "" {
			continue
		}

		// 验证当前字段
		err := v.validator.Var(fieldValue.Interface(), fieldInfo.validateTag)
		if err == nil {
			continue
		}

		// 处理验证错误
		validationErrs, ok := err.(validator.ValidationErrors)
		if !ok {
			// 如果不是 ValidationErrors 类型，使用原始错误消息
			validationErrors.Add(fullFieldName, err.Error())
			continue
		}

		// 处理每个验证错误
		for _, fieldErr := range validationErrs {
			errorTag := fieldErr.Tag()
			customMsg := v.getCachedErrorMessage(fieldInfo.errorMsgTag, errorTag)
			message := customMsg
			if customMsg == "" {
				message = fieldErr.Error()
			}
			validationErrors.Add(fullFieldName, message)
		}
	}
}

// getCachedErrorMessage 获取缓存的错误消息
func (v *Validator) getCachedErrorMessage(errorMsgTag, rule string) string {
	if errorMsgTag == "" {
		return ""
	}

	// 尝试从缓存读取
	v.mu.RLock()
	if ruleMap, exists := v.errorMsgCache[errorMsgTag]; exists {
		if msg, found := ruleMap[rule]; found {
			v.mu.RUnlock()
			return msg
		}
	}
	v.mu.RUnlock()

	// 缓存未命中，解析并缓存
	v.mu.Lock()
	defer v.mu.Unlock()

	// 双重检查
	if ruleMap, exists := v.errorMsgCache[errorMsgTag]; exists {
		if msg, found := ruleMap[rule]; found {
			return msg
		}
	}

	// 解析错误消息标签
	ruleMap := v.parseErrorMessageTag(errorMsgTag)
	v.errorMsgCache[errorMsgTag] = ruleMap
	return ruleMap[rule]
}

// parseErrorMessageTag 解析错误消息标签
// 格式: "required:邮箱必填|email:邮箱格式错误"
func (v *Validator) parseErrorMessageTag(errorMsgTag string) map[string]string {
	ruleMap := make(map[string]string)
	ruleMessages := strings.Split(errorMsgTag, ruleSeparator)
	for _, ruleMessage := range ruleMessages {
		parts := strings.SplitN(ruleMessage, keyValueSep, 2)
		if len(parts) == 2 {
			ruleMap[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
		}
	}
	return ruleMap
}
