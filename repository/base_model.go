package repository

import (
	"time"

	"github.com/aisgo/ais-go-pkg/utils/id-generator/ulid"
	ulidv2 "github.com/oklog/ulid/v2"
	"gorm.io/gorm"
	"gorm.io/plugin/soft_delete"
)

/* ========================================================================
 * Base Model - 基础模型
 * ========================================================================
 * 职责: 定义所有模型的公共字段和方法
 * 使用: 所有 GORM 模型都应嵌入此结构体
 * 字段: 与其他微服务 BaseEntity 保持一致
 * ======================================================================== */

// BaseModel 所有模型的基类
// 包含通用字段：ID、创建时间、更新时间、软删除标记
type BaseModel struct {
	ID         ulidv2.ULID           `json:"id" gorm:"type:char(26);primaryKey;comment:主键ID(ULID)"`
	CreateTime time.Time             `json:"create_time" gorm:"column:create_time;autoCreateTime;comment:创建时间"`
	UpdateTime time.Time             `json:"update_time" gorm:"column:update_time;autoUpdateTime;comment:更新时间"`
	Deleted    soft_delete.DeletedAt `json:"-" gorm:"column:deleted;default:0;softDelete:flag;comment:软删除标记(1=已删除)"`
}

// BeforeCreate GORM 钩子：在创建记录前自动生成 ULID
// ULID 特性: 时间排序、URL 安全、大小写不敏感、128 位唯一性
func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if ulid.IsZero(m.ID) {
		m.ID = ulid.Generate()
	}
	return nil
}

// TableName 返回默认表名（可被子类覆盖）
func (BaseModel) TableName() string {
	return "" // 使用 GORM 默认表名
}
