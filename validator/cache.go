package validator

import (
	"reflect"
	"sync"
)

/* ========================================================================
 * Type Cache - 类型信息缓存
 * ========================================================================
 * 职责: 缓存结构体类型信息，减少反射开销
 * ======================================================================== */

// fieldInfo 字段信息
type fieldInfo struct {
	name        string // 字段名
	validateTag string // validate 标签值
	errorMsgTag string // error_msg 标签值
	isStruct    bool   // 是否为结构体
	isPtr       bool   // 是否为指针类型
}

// typeCache 类型缓存
type typeCache struct {
	mu    sync.RWMutex
	cache map[reflect.Type][]fieldInfo
}

// newTypeCache 创建类型缓存
func newTypeCache() *typeCache {
	return &typeCache{
		cache: make(map[reflect.Type][]fieldInfo),
	}
}

// get 获取类型字段信息（带缓存）
func (tc *typeCache) get(t reflect.Type) ([]fieldInfo, bool) {
	tc.mu.RLock()
	defer tc.mu.RUnlock()
	info, exists := tc.cache[t]
	return info, exists
}

// set 设置类型字段信息
func (tc *typeCache) set(t reflect.Type, info []fieldInfo) {
	tc.mu.Lock()
	defer tc.mu.Unlock()
	tc.cache[t] = info
}

// getFieldsInfo 获取类型的字段信息（带缓存）
func (tc *typeCache) getFieldsInfo(t reflect.Type) []fieldInfo {
	// 检查缓存 (Read Lock)
	if info, exists := tc.get(t); exists {
		return info
	}

	// 缓存未命中，获取写锁 (Write Lock)
	tc.mu.Lock()
	defer tc.mu.Unlock()

	// 双重检查 (Double Checked Locking)
	// 防止多个 goroutine 同时发现缓存缺失并排队进入此处，导致重复解析
	if info, exists := tc.cache[t]; exists {
		return info
	}

	// 解析字段信息
	var fields []fieldInfo
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if field.PkgPath != "" {
			// 跳过未导出字段：反射读取 Interface() 会 panic
			continue
		}
		fieldType := field.Type
		isPtr := fieldType.Kind() == reflect.Ptr

		// 处理指针类型，获取底层类型
		if isPtr {
			fieldType = fieldType.Elem()
		}

		info := fieldInfo{
			name:        field.Name,
			validateTag: field.Tag.Get("validate"),
			errorMsgTag: field.Tag.Get(tagCustom),
			isStruct:    fieldType.Kind() == reflect.Struct,
			isPtr:       isPtr,
		}
		fields = append(fields, info)
	}

	// 存入缓存
	tc.cache[t] = fields
	return fields
}
