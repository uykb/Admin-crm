package model

// SysLog 操作日志
type SysLog struct {
	BaseLogModel
	UserID     uint   `gorm:"index" json:"user_id"`
	Username   string `gorm:"size:50" json:"username"`
	Method     string `gorm:"size:10" json:"method"`
	Path       string `gorm:"size:255" json:"path"`
	Params     string `gorm:"type:text" json:"params"`
	StatusCode int    `json:"status_code"`
	DurationMs int64  `json:"duration_ms"`
	IP         string `gorm:"size:50" json:"ip"`
	UserAgent  string `gorm:"size:255" json:"user_agent"`
}

func (SysLog) TableName() string { return "sys_log" }

// SysSetting 系统设置 KV
type SysSetting struct {
	BaseModel
	Key      string `gorm:"uniqueIndex;size:100;not null" json:"key"`
	Value    string `gorm:"type:text" json:"value"`
	IsPublic bool   `gorm:"default:false" json:"is_public"`
}

func (SysSetting) TableName() string { return "sys_setting" }

// SysFile 文件记录
type SysFile struct {
	BaseModel
	Name       string `gorm:"size:255;not null" json:"name"`
	Path       string `gorm:"size:500;not null" json:"path"`
	Size       int64  `json:"size"`
	FolderID   *uint  `gorm:"index" json:"folder_id"`
	MimeType   string `gorm:"size:100" json:"mime_type"`
	UploaderID uint   `gorm:"index" json:"uploader_id"`
}

func (SysFile) TableName() string { return "sys_file" }

// SysFileFolder 文件夹
type SysFileFolder struct {
	BaseModel
	Name     string `gorm:"size:100;not null" json:"name"`
	ParentID *uint  `gorm:"index" json:"parent_id"`
}

func (SysFileFolder) TableName() string { return "sys_file_folder" }

// DevExampleNote 插件开发示例——备忘录（dev-example 插件演示用）
type DevExampleNote struct {
	BaseModel
	Title     string `gorm:"size:200;not null" json:"title"`
	Content   string `gorm:"type:text" json:"content"`
	Priority  int    `gorm:"default:0" json:"priority"` // 0=普通 1=重要 2=紧急
	Completed bool   `gorm:"default:false" json:"completed"`
}

func (DevExampleNote) TableName() string { return "dev_example_note" }
