package model

import (
	"time"

	"gorm.io/gorm"
)

// KdConfig 金蝶云星空插件配置表
type KdConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Key       string         `gorm:"size:64;uniqueIndex;not null" json:"key"`
	Value     string         `gorm:"type:text" json:"value"`
	IsPublic  bool           `gorm:"default:false" json:"is_public"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KdConfig) TableName() string {
	return "kd_config"
}

// KdQueryLog 金蝶数据查询审计日志
type KdQueryLog struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	FormID     string    `gorm:"size:64;index" json:"form_id"` // 单据/表单ID（如 BD_MATERIAL, SAL_SALEORDER）
	QueryFilter string    `gorm:"type:text" json:"query_filter"`
	ResultCount int       `json:"result_count"`
	ExecTimeMs int64     `json:"exec_time_ms"`
	CreatedBy  string    `gorm:"size:64" json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
}

func (KdQueryLog) TableName() string {
	return "kd_query_log"
}

// AllModels 插件所有需要迁移的 GORM 模型
func AllModels() []interface{} {
	return []interface{}{
		&KdConfig{},
		&KdQueryLog{},
	}
}
