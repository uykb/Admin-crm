package model

import (
	"time"

	"gorm.io/gorm"
)

// 常量定义：审批实例状态
const (
	InstanceStatusPending      = "PENDING"
	InstanceStatusPendingAudit = "PENDING_AUDIT"
	InstanceStatusApproved     = "APPROVED"
	InstanceStatusRejected     = "REJECTED"
	InstanceStatusCanceled     = "CANCELED"
	InstanceStatusFailed       = "FAILED"
	InstanceStatusAuditFailed  = "AUDIT_FAILED"
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
	ID          uint      `gorm:"primaryKey" json:"id"`
	FormID      string    `gorm:"size:64;index" json:"form_id"` // 单据/表单ID（如 BD_MATERIAL, SAL_SALEORDER）
	QueryFilter string    `gorm:"type:text" json:"query_filter"`
	ResultCount int       `json:"result_count"`
	ExecTimeMs  int64     `json:"exec_time_ms"`
	CreatedBy   string    `gorm:"size:64" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (KdQueryLog) TableName() string {
	return "kd_query_log"
}

// KdFlow 单据审批流程配置表
type KdFlow struct {
	ID                 uint           `gorm:"primaryKey" json:"id"`
	KingdeeFormId      string         `gorm:"size:64;not null;index" json:"kingdee_form_id"` // 如 SAL_SaleOrder, PUR_PurchaseOrder
	FlowName           string         `gorm:"size:128;not null" json:"flow_name"`           // 流程名称，如 销售订单审批
	FlowCode           string         `gorm:"size:64;index" json:"flow_code"`               // 流程编码
	Channel            string         `gorm:"size:32;default:'feishu'" json:"channel"`      // 审批通道: feishu
	FeishuApprovalCode string         `gorm:"size:128" json:"feishu_approval_code"`        // 飞书审批定义 Code
	BillNoField        string         `gorm:"size:64;default:'FBillNo'" json:"bill_no_field"`
	StatusField        string         `gorm:"size:64;default:'FDocumentStatus'" json:"status_field"`
	PollFilterStatus   string         `gorm:"size:16;default:'B'" json:"poll_filter_status"` // 待审批状态，默认 B (审核中)
	PollMaxBills       int            `gorm:"default:50" json:"poll_max_bills"`
	TitleTemplate      string         `gorm:"size:255" json:"title_template"` // 审批标题模板，如 "销售订单 - {FBillNo}"
	QueryFields        string         `gorm:"type:text" json:"query_fields"`  // 查询所需字段
	Enabled            bool           `gorm:"default:true" json:"enabled"`
	CreatedAt          time.Time      `json:"created_at"`
	UpdatedAt          time.Time      `json:"updated_at"`
	DeletedAt          gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KdFlow) TableName() string {
	return "kd_flow"
}

// KdFlowField 单据字段与飞书表单控件映射表
type KdFlowField struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	FlowID       uint      `gorm:"not null;index" json:"flow_id"`
	WidgetID     string    `gorm:"size:64" json:"widget_id"`     // 飞书控件 custom_id
	FieldAlias   string    `gorm:"size:64" json:"field_alias"`   // 飞书控件中文名称
	KingdeeField string    `gorm:"size:128" json:"kingdee_field"` // 金蝶对应字段 (支持逗号分隔复合字段)
	IsEntry      string    `gorm:"size:2;default:'0'" json:"is_entry"` // 0-表头，1-表身明细
	SortNo       int       `gorm:"default:0" json:"sort_no"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (KdFlowField) TableName() string {
	return "kd_flow_field"
}

// KdInstance 审批实例记录表
type KdInstance struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	FlowID        uint           `gorm:"not null;index" json:"flow_id"`
	BillFid       string         `gorm:"size:64;not null;index" json:"bill_fid"` // 金蝶单据 FID
	BillNo        string         `gorm:"size:64;index" json:"bill_no"`
	BillStatus    string         `gorm:"size:16" json:"bill_status"`
	InstanceID    string         `gorm:"size:128;index" json:"instance_id"` // 飞书审批 instance_code
	ApproveStatus string         `gorm:"size:32;default:'PENDING';index" json:"approve_status"`
	AuditResult   string         `gorm:"size:16" json:"audit_result"`
	ErrorMsg      string         `gorm:"type:text" json:"error_msg"`
	RetryCount    int            `gorm:"default:0" json:"retry_count"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}

func (KdInstance) TableName() string {
	return "kd_instance"
}

// KdUserMap 金蝶用户与飞书用户/审批人映射表
type KdUserMap struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	KdUserId       string    `gorm:"size:64;uniqueIndex;not null" json:"kd_user_id"` // 金蝶 FCreatorId 或用户ID
	KdUserName     string    `gorm:"size:64" json:"kd_user_name"`
	FeishuOpenId   string    `gorm:"size:128" json:"feishu_open_id"`
	ApproverOpenId string    `gorm:"size:128" json:"approver_open_id"`
	DeptName       string    `gorm:"size:64" json:"dept_name"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (KdUserMap) TableName() string {
	return "kd_user_map"
}

// AllModels 插件所有需要迁移的 GORM 模型
func AllModels() []interface{} {
	return []interface{}{
		&KdConfig{},
		&KdQueryLog{},
		&KdFlow{},
		&KdFlowField{},
		&KdInstance{},
		&KdUserMap{},
	}
}
