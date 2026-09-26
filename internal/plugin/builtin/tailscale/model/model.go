package model

import (
	"time"

	"gorm.io/gorm"
)

// TsConfig Tailscale 插件私有配置项
type TsConfig struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	Key       string         `gorm:"size:100;uniqueIndex;not null" json:"key"`
	Value     string         `gorm:"type:text" json:"value"`
	IsPublic  bool           `gorm:"default:false" json:"is_public"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TsConfig) TableName() string {
	return "ts_config"
}

// TsDeviceCache 本地缓存 Tailnet 设备镜像
type TsDeviceCache struct {
	ID                uint           `gorm:"primaryKey" json:"id"`
	DeviceID          string         `gorm:"size:100;uniqueIndex;not null" json:"device_id"`
	Name              string         `gorm:"size:200" json:"name"`
	Hostname          string         `gorm:"size:200" json:"hostname"`
	User              string         `gorm:"size:100" json:"user"`
	IPs               string         `gorm:"type:text" json:"ips"` // JSON 数组存 100.x 与 IPv6
	OS                string         `gorm:"size:50" json:"os"`
	ClientVersion     string         `gorm:"size:50" json:"client_version"`
	UpdateAvailable   bool           `json:"update_available"`
	LastSeen          time.Time      `json:"last_seen"`
	Online            bool           `json:"online"`
	Authorized        bool           `json:"authorized"`
	KeyExpiryDisabled bool           `json:"key_expiry_disabled"`
	ExpiresAt         time.Time      `json:"expires_at"`
	Tags              string         `gorm:"size:500" json:"tags"`
	SubnetRoutes      string         `gorm:"type:text" json:"subnet_routes"`
	NodeKey           string         `gorm:"size:100" json:"node_key"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TsDeviceCache) TableName() string {
	return "ts_device_cache"
}

// TsWebhookLog Webhook 实时遥测事件审计日志
type TsWebhookLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	EventType string         `gorm:"size:100;index" json:"event_type"`
	Tailnet   string         `gorm:"size:100" json:"tailnet"`
	Message   string         `gorm:"type:text" json:"message"`
	RawBody   string         `gorm:"type:text" json:"raw_body"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TsWebhookLog) TableName() string {
	return "ts_webhook_log"
}

// TsKeyLog 本地 Auth Key 生成历史审计日志
type TsKeyLog struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	KeyID         string         `gorm:"size:100" json:"key_id"`
	AuthKey       string         `gorm:"size:200" json:"auth_key"`
	Reusable      bool           `json:"reusable"`
	Ephemeral     bool           `json:"ephemeral"`
	Preauthorized bool           `json:"preauthorized"`
	Tags          string         `gorm:"size:200" json:"tags"`
	Purpose       string         `gorm:"size:200" json:"purpose"`
	CreatedBy     string         `gorm:"size:100" json:"created_by"`
	ExpiresAt     time.Time      `json:"expires_at"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (TsKeyLog) TableName() string {
	return "ts_key_log"
}

// AllModels 返回 Tailscale 插件的所有数据模型
func AllModels() []interface{} {
	return []interface{}{
		&TsConfig{},
		&TsDeviceCache{},
		&TsKeyLog{},
		&TsWebhookLog{},
	}
}
