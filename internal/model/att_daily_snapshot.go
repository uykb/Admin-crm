package model

import (
	"time"

	"gorm.io/datatypes"
)

// AttDailySnapshot 每日打卡考勤快照
// 废弃传统的打卡流水表，每个员工每天只保留一条记录，使用 JSONB 存储原始打卡时间戳。
type AttDailySnapshot struct {
	ID            uint           `gorm:"primaryKey" json:"id"`
	UserID        string         `gorm:"type:varchar(64);index:idx_user_date,unique" json:"user_id"` // 员工ID/工号
	AttDate       time.Time      `gorm:"type:date;index:idx_user_date,unique" json:"att_date"`       // 业务考勤日
	FirstPunch    *time.Time     `gorm:"type:timestamp" json:"first_punch"`                          // 首卡时间
	LastPunch     *time.Time     `gorm:"type:timestamp" json:"last_punch"`                           // 末卡时间
	PunchCount    int            `gorm:"type:int;default:0" json:"punch_count"`                      // 当天打卡总次数
	MatchedShift  string         `gorm:"type:varchar(32)" json:"matched_shift"`                      // 匹配班次: day, night, unknown
	MatchedStatus string         `gorm:"type:varchar(32)" json:"matched_status"`                     // 考勤状态: normal, late, early, absent
	RawPunchTimes datatypes.JSON `gorm:"type:jsonb" json:"raw_punch_times"`                          // 原始打卡时间戳数组 (JSONB)
	IsArchived    bool           `gorm:"type:boolean;default:false;index" json:"is_archived"`        // 是否封板
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

func (AttDailySnapshot) TableName() string {
	return "att_daily_snapshot"
}
