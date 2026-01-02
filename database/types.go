package database

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
)

/* ========================================================================
 * JSONB Type - PostgreSQL JSONB 映射（公共定义）
 * ========================================================================
 * 职责: 统一定义 JSONB 类型，供各模块共享使用
 * ======================================================================== */

// JSONB 自定义类型，用于 Gorm 映射 PostgreSQL JSONB
type JSONB map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSONB) Value() (driver.Value, error) {
	if j == nil {
		return "{}", nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONB)
		return nil
	}
	var data []byte
	switch v := value.(type) {
	case []byte:
		data = v
	case string:
		data = []byte(v)
	default:
		return errors.New("unsupported type for JSONB scan")
	}
	return json.Unmarshal(data, j)
}

// ToStringMap 将 JSONB 转换为 map[string]string（用于 Proto 响应）
func (j JSONB) ToStringMap() map[string]string {
	result := make(map[string]string)
	for k, v := range j {
		switch val := v.(type) {
		case string:
			result[k] = val
		case float64:
			result[k] = fmt.Sprintf("%v", val)
		case bool:
			if val {
				result[k] = "true"
			} else {
				result[k] = "false"
			}
		default:
			if bytes, err := json.Marshal(v); err == nil {
				result[k] = string(bytes)
			}
		}
	}
	return result
}

// ToDoubleMap 将 JSONB 转换为 map[string]float64（用于 Metrics）
func (j JSONB) ToDoubleMap() map[string]float64 {
	result := make(map[string]float64)
	for k, v := range j {
		switch val := v.(type) {
		case float64:
			result[k] = val
		case int:
			result[k] = float64(val)
		case int64:
			result[k] = float64(val)
		}
	}
	return result
}
