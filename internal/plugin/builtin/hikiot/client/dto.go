package client

// BaseResponse 海康开放平台通用响应结构
type BaseResponse struct {
	Code string      `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

// OrgDTO 海康组织节点结构
type OrgDTO struct {
	OrgIndexCode       string `json:"orgIndexCode"`
	OrgName            string `json:"orgName"`
	ParentOrgIndexCode string `json:"parentOrgIndexCode"`
}

// PersonDTO 海康人员数据结构
type PersonDTO struct {
	PersonID     string `json:"personId"`
	PersonName   string `json:"personName"`
	JobNo        string `json:"jobNo"`
	PhoneNo      string `json:"phoneNo"`
	OrgIndexCode string `json:"orgIndexCode"`
	OrgName      string `json:"orgName"`
}

// DoorDTO 海康门禁点结构
type DoorDTO struct {
	DoorIndexCode string `json:"doorIndexCode"`
	DoorName      string `json:"doorName"`
	ChannelNo     int    `json:"channelNo"`
	Status        int    `json:"status"` // 0: 离线, 1: 在线
}

// DoorControlParam 控门参数
type DoorControlParam struct {
	DoorIndexCode string `json:"doorIndexCode"`
	Command       int    `json:"command"` // 0: 关门, 1: 开门, 2: 常开, 3: 常关
}

// AttendanceRecordDTO 考勤打卡记录
type AttendanceRecordDTO struct {
	PersonID   string `json:"personId"`
	PersonName string `json:"personName"`
	JobNo      string `json:"jobNo"`
	ClockTime  string `json:"clockTime"`
	DoorName   string `json:"doorName"`
	VerifyMode int    `json:"verifyMode"` // 验证模式：1 人脸, 2 刷卡, 3 密码等
}

// PageResponse 分页查询数据结构
type PageResponse struct {
	Total    int         `json:"total"`
	PageNo   int         `json:"pageNo"`
	PageSize int         `json:"pageSize"`
	List     interface{} `json:"list"`
}
