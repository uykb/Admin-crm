package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// FlexibleCode 兼容 JSON 数字 (0, 200) 与 字符串 ("0", "200")
type FlexibleCode string

func (fc *FlexibleCode) UnmarshalJSON(data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	switch val := v.(type) {
	case string:
		*fc = FlexibleCode(val)
	case float64:
		*fc = FlexibleCode(fmt.Sprintf("%d", int64(val)))
	default:
		*fc = FlexibleCode(fmt.Sprintf("%v", val))
	}
	return nil
}

func (fc FlexibleCode) String() string {
	return strings.TrimSpace(string(fc))
}

// BaseResponse 海康开放平台通用响应结构
type BaseResponse struct {
	Code FlexibleCode `json:"code"`
	Msg  string       `json:"msg"`
	Data interface{}  `json:"data"`
}

// AppTokenData 海康互联 AppToken 数据
type AppTokenData struct {
	AppKey          string `json:"appKey"`
	AppAccessToken string `json:"appAccessToken"`
	ExpiresIn       int    `json:"expiresIn"`
	RefreshAppToken string `json:"refreshAppToken"`
}

// OrgDTO 海康组织节点结构
type OrgDTO struct {
	OrgIndexCode       string      `json:"orgIndexCode"`
	OrgName            string      `json:"orgName"`
	ParentOrgIndexCode string      `json:"parentOrgIndexCode"`
	DeptID             string      `json:"deptId"`
	DeptName           string      `json:"deptName"`
	DepartNo           string      `json:"departNo"`
	DepartName         string      `json:"departName"`
	ParentID           interface{} `json:"parentId"`
	IsLeaf             bool        `json:"isLeaf"`
}

func (o *OrgDTO) GetCode() string {
	if o.DepartNo != "" {
		return o.DepartNo
	}
	if o.OrgIndexCode != "" {
		return o.OrgIndexCode
	}
	return o.DeptID
}

func (o *OrgDTO) GetName() string {
	if o.DepartName != "" {
		return o.DepartName
	}
	if o.OrgName != "" {
		return o.OrgName
	}
	return o.DeptName
}

func (o *OrgDTO) GetParentCode() string {
	if o.ParentOrgIndexCode != "" {
		return o.ParentOrgIndexCode
	}
	if o.ParentID != nil {
		return fmt.Sprintf("%v", o.ParentID)
	}
	return ""
}

// PersonDTO 海康人员数据结构
type PersonDTO struct {
	PersonID     string `json:"personId"`
	PersonNo     string `json:"personNo"`
	PersonName   string `json:"personName"`
	JobNo        string `json:"jobNo"`
	JobNumber    string `json:"jobNumber"`
	PhoneNo      string `json:"phoneNo"`
	OrgIndexCode string `json:"orgIndexCode"`
	OrgName      string `json:"orgName"`
	DepartNo     string `json:"departNo"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
}

func (p *PersonDTO) GetID() string {
	if p.PersonNo != "" {
		return p.PersonNo
	}
	if p.PersonID != "" {
		return p.PersonID
	}
	return p.JobNo
}

func (p *PersonDTO) GetName() string {
	if p.PersonName != "" {
		return p.PersonName
	}
	return p.Name
}

func (p *PersonDTO) GetPhone() string {
	if p.PhoneNo != "" {
		return p.PhoneNo
	}
	return p.Phone
}

func (p *PersonDTO) GetJobNo() string {
	if p.JobNumber != "" {
		return p.JobNumber
	}
	return p.JobNo
}

func (p *PersonDTO) GetOrgCode() string {
	if p.DepartNo != "" {
		return p.DepartNo
	}
	return p.OrgIndexCode
}

// DoorDTO 海康门禁点结构（兼容设备通道/资源点所有变体字段）
type DoorDTO struct {
	DoorIndexCode string      `json:"doorIndexCode"`
	DoorName      string      `json:"doorName"`
	DeviceSerial  string      `json:"deviceSerial"`
	DeviceName    string      `json:"deviceName"`
	ResourceName  string      `json:"resourceName"`
	ChannelName   string      `json:"channelName"`
	SerialNo      string      `json:"serialNo"`
	Model         string      `json:"model"`
	DeviceModel   string      `json:"deviceModel"`
	ChannelNo     interface{} `json:"channelNo"`
	DoorNo        interface{} `json:"doorNo"`
	ResourceNo    interface{} `json:"resourceNo"`
	Status        interface{} `json:"status"` // 0: 离线, 1: 在线
	OnlineStatus  interface{} `json:"onlineStatus"`
	IsOnline      interface{} `json:"isOnline"`
	ResourceType  string      `json:"resourceType"`
	Type          string      `json:"type"`
}

func (d *DoorDTO) GetCode() string {
	if d.DoorIndexCode != "" {
		return d.DoorIndexCode
	}
	sn := d.DeviceSerial
	if sn == "" {
		sn = d.SerialNo
	}
	chNo := d.GetChannelNo()
	if sn != "" {
		if chNo > 0 {
			return fmt.Sprintf("%s-%d", sn, chNo)
		}
		return sn
	}
	return ""
}

func (d *DoorDTO) GetName() string {
	if d.ResourceName != "" {
		return d.ResourceName
	}
	if d.ChannelName != "" {
		return d.ChannelName
	}
	if d.DoorName != "" {
		return d.DoorName
	}
	if d.DeviceName != "" {
		return d.DeviceName
	}
	return "海康门禁点"
}

func (d *DoorDTO) GetChannelNo() int {
	if v, ok := d.ChannelNo.(float64); ok {
		return int(v)
	}
	if v, ok := d.DoorNo.(float64); ok {
		return int(v)
	}
	if v, ok := d.ResourceNo.(float64); ok {
		return int(v)
	}
	return 1
}

func (d *DoorDTO) GetStatus() int {
	if v, ok := d.Status.(float64); ok {
		return int(v)
	}
	if v, ok := d.OnlineStatus.(float64); ok {
		return int(v)
	}
	if v, ok := d.IsOnline.(bool); ok {
		if v {
			return 1
		}
		return 0
	}
	return 1
}

// DoorControlParam 控门参数
type DoorControlParam struct {
	DoorIndexCode string `json:"doorIndexCode"`
	Command       int    `json:"command"` // 0: 关门, 1: 开门, 2: 常开, 3: 常关
}

// AttendanceRecordDTO 考勤打卡记录
type AttendanceRecordDTO struct {
	PersonNo        string `json:"personNo"`
	PersonID        string `json:"personId"`
	PersonName      string `json:"personName"`
	JobNumber       string `json:"jobNumber"`
	JobNo           string `json:"jobNo"`
	DepartmentName  string `json:"departmentName"`
	DepartmentNo    string `json:"departmentNo"`
	AttendanceDate  string `json:"attendanceDate"`
	Workday         string `json:"workday"`
	ClockTime       string `json:"clockTime"`
	DeviceSerial    string `json:"deviceSerial"`
	DeviceName      string `json:"deviceName"`
	Address         string `json:"address"`
	VerifyMode      int    `json:"verifyMode"`
	WayOfClock      string `json:"wayOfClock"`
}

// PageResponse 分页查询数据结构
type PageResponse struct {
	Total    int         `json:"total"`
	PageNo   int         `json:"pageNo"`
	PageSize int         `json:"pageSize"`
	List     interface{} `json:"list"`
}
