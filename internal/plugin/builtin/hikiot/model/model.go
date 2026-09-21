package model

import "time"

// HkOrg 海康组织结构实体
type HkOrg struct {
	ID                 uint      `gorm:"primaryKey" json:"id"`
	OrgIndexCode       string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"org_index_code"`
	OrgName            string    `gorm:"type:varchar(128);not null" json:"org_name"`
	ParentOrgIndexCode string    `gorm:"type:varchar(64)" json:"parent_org_index_code"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

func (HkOrg) TableName() string {
	return "hk_org"
}

// HkPerson 海康人员档案实体
type HkPerson struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	PersonID     string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"person_id"`
	PersonName   string    `gorm:"type:varchar(64);not null" json:"person_name"`
	JobNo        string    `gorm:"type:varchar(64)" json:"job_no"`
	PhoneNo      string    `gorm:"type:varchar(32)" json:"phone_no"`
	OrgIndexCode string    `gorm:"type:varchar(64)" json:"org_index_code"`
	OrgName      string    `gorm:"type:varchar(128)" json:"org_name"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func (HkPerson) TableName() string {
	return "hk_person"
}

// HkDoor 海康门禁设备点实体
type HkDoor struct {
	ID            uint      `gorm:"primaryKey" json:"id"`
	DoorIndexCode string    `gorm:"type:varchar(64);uniqueIndex;not null" json:"door_index_code"`
	DoorName      string    `gorm:"type:varchar(128);not null" json:"door_name"`
	ChannelNo     int       `gorm:"default:1" json:"channel_no"`
	Status        int       `gorm:"default:1" json:"status"` // 0: 离线, 1: 在线
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (HkDoor) TableName() string {
	return "hk_door"
}

// HkAttendance 海康打卡考勤记录实体
type HkAttendance struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PersonID   string    `gorm:"type:varchar(64);index;not null" json:"person_id"`
	PersonName string    `gorm:"type:varchar(64);not null" json:"person_name"`
	JobNo      string    `gorm:"type:varchar(64)" json:"job_no"`
	ClockTime  time.Time `gorm:"index;not null" json:"clock_time"`
	DoorName   string    `gorm:"type:varchar(128)" json:"door_name"`
	VerifyMode int       `gorm:"default:1" json:"verify_mode"`
	CreatedAt  time.Time `json:"created_at"`
}

func (HkAttendance) TableName() string {
	return "hk_attendance"
}

// HkAttendanceResult 月度考勤智能判定结果
type HkAttendanceResult struct {
	ID         uint      `gorm:"primaryKey" json:"id"`
	PersonID   string    `gorm:"type:varchar(64);uniqueIndex:idx_person_date;not null" json:"person_id"`
	PersonName string    `gorm:"type:varchar(64);not null" json:"person_name"`
	JobNo      string    `gorm:"type:varchar(64)" json:"job_no"`
	Date       string    `gorm:"type:varchar(32);uniqueIndex:idx_person_date;not null" json:"date"`        // YYYY-MM-DD (归属日期)
	ShiftType  string    `gorm:"type:varchar(32);not null" json:"shift_type"`        // 白班, 夜班, 异常, 空白, 请假, 休息
	FirstClock string    `gorm:"type:varchar(32)" json:"first_clock"`                // 首次打卡时间 2006-01-02 15:04:05
	LastClock  string    `gorm:"type:varchar(32)" json:"last_clock"`                 // 末次打卡时间 2006-01-02 15:04:05
	IsManual   bool      `gorm:"default:false" json:"is_manual"`                     // 是否人工修改过
	Remark     string    `gorm:"type:varchar(255)" json:"remark"`                    // 备注
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

func (HkAttendanceResult) TableName() string {
	return "hk_attendance_result"
}

// AllModels 返回该插件需要 AutoMigrate 的模型列表
func AllModels() []interface{} {
	return []interface{}{
		&HkOrg{},
		&HkPerson{},
		&HkDoor{},
		&HkAttendance{},
		&HkAttendanceResult{},
	}
}
