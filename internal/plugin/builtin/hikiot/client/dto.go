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
	OrgIndexCode       string `json:"orgIndexCode"`
	OrgName            string `json:"orgName"`
	ParentOrgIndexCode string `json:"parentOrgIndexCode"`
	DeptID             string `json:"deptId"`
	DeptName           string `json:"deptName"`
	ParentID           string `json:"parentId"`
}

func (o *OrgDTO) GetCode() string {
	if o.OrgIndexCode != "" {
		return o.OrgIndexCode
	}
	return o.DeptID
}

func (o *OrgDTO) GetName() string {
	if o.OrgName != "" {
		return o.OrgName
	}
	return o.DeptName
}

// PersonDTO 海康人员数据结构
type PersonDTO struct {
	PersonID     string `json:"personId"`
	PersonName   string `json:"personName"`
	JobNo        string `json:"jobNo"`
	PhoneNo      string `json:"phoneNo"`
	OrgIndexCode string `json:"orgIndexCode"`
	OrgName      string `json:"orgName"`
	Name         string `json:"name"`
	Phone        string `json:"phone"`
}

func (p *PersonDTO) GetID() string {
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

// DoorDTO 海康门禁点结构
type DoorDTO struct {
	DoorIndexCode string `json:"doorIndexCode"`
	DoorName      string `json:"doorName"`
	DeviceSerial  string `json:"deviceSerial"`
	DeviceName    string `json:"deviceName"`
	ChannelNo     int    `json:"channelNo"`
	DoorNo        int    `json:"doorNo"`
	Status        int    `json:"status"` // 0: 离线, 1: 在线
}

func (d *DoorDTO) GetCode() string {
	if d.DoorIndexCode != "" {
		return d.DoorIndexCode
	}
	if d.DeviceSerial != "" {
		if d.DoorNo > 0 {
			return fmt.Sprintf("%s_%d", d.DeviceSerial, d.DoorNo)
		}
		return d.DeviceSerial
	}
	return ""
}

func (d *DoorDTO) GetName() string {
	if d.DoorName != "" {
		return d.DoorName
	}
	if d.DeviceName != "" {
		return d.DeviceName
	}
	return "海康门禁点"
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
