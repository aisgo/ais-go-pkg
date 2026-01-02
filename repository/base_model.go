package repository

import (
	"time"

	"github.com/aisgo/ais-go-pkg/utils/id-generator/snowflake"

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
	ID         int64                 `json:"id,string" gorm:"primaryKey;comment:主键ID"`
	CreateTime time.Time             `json:"create_time" gorm:"column:create_time;autoCreateTime;comment:创建时间"`
	UpdateTime time.Time             `json:"update_time" gorm:"column:update_time;autoUpdateTime;comment:更新时间"`
	Deleted    soft_delete.DeletedAt `json:"-" gorm:"column:deleted;default:0;softDelete:flag;comment:软删除标记(1=已删除)"`
}

// BeforeCreate GORM 钩子：在创建记录前自动生成雪花 ID
// 注意: 在多实例部署环境中，必须配置环境变量 SNOWFLAKE_NODE_ID
func (m *BaseModel) BeforeCreate(tx *gorm.DB) error {
	if m.ID == 0 {
		m.ID = snowflake.Generate()
	}
	return nil
}

// TableName 返回默认表名（可被子类覆盖）
func (BaseModel) TableName() string {
	return "" // 使用 GORM 默认表名
}
