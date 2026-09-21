package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client 海康开放平台 API 客户端
type Client struct {
	BaseURL         string
	AppKey          string
	AppSecret       string
	AppAccessToken  string
	UserAccessToken string
	HTTPClient      *http.Client
}

// NewClient 创建海康客户端实例
func NewClient(baseURL, appKey, appSecret string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" || baseURL == "https://open.hikiot.com" || baseURL == "http://open.hikiot.com" {
		baseURL = "https://open-api.hikiot.com"
	}
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	return &Client{
		BaseURL:   baseURL,
		AppKey:    strings.TrimSpace(appKey),
		AppSecret: strings.TrimSpace(appSecret),
		HTTPClient: &http.Client{
			Transport: tr,
			Timeout:   15 * time.Second,
		},
	}
}

// SetUserAccessToken 设置用户 token
func (c *Client) SetUserAccessToken(token string) {
	c.UserAccessToken = strings.TrimSpace(token)
}

// DoRequest 发送签名与 Token 授权请求
func (c *Client) DoRequest(method, path string, bodyData interface{}, result interface{}) error {
	if c.AppKey == "" || c.AppSecret == "" {
		return fmt.Errorf("海康互联未设置 AppKey 或 AppSecret，请先在插件设置中配置凭据")
	}

	// 自动换取 AppAccessToken（非 exchangeAppToken 接口）
	if c.AppAccessToken == "" && !strings.Contains(path, "exchangeAppToken") {
		_, _ = c.ExchangeAppToken()
	}

	urlStr := c.BaseURL + path
	urlObj, err := http.NewRequest(method, urlStr, nil) // just for parsing url
	if err != nil {
		return fmt.Errorf("URL 解析失败: %w", err)
	}

	var bodyReader io.Reader
	if bodyData != nil {
		if method == "GET" {
			if m, ok := bodyData.(map[string]interface{}); ok {
				q := urlObj.URL.Query()
				for k, v := range m {
					q.Add(k, fmt.Sprintf("%v", v))
				}
				urlObj.URL.RawQuery = q.Encode()
				urlStr = urlObj.URL.String()
			}
		} else {
			b, err := json.Marshal(bodyData)
			if err != nil {
				return fmt.Errorf("序列化请求参数失败: %w", err)
			}
			bodyReader = bytes.NewBuffer(b)
		}
	}

	req, err := http.NewRequest(method, urlStr, bodyReader)
	if err != nil {
		return fmt.Errorf("创建 HTTP 请求失败: %w", err)
	}

	accept := "application/json"
	contentType := "application/json"
	timestamp := fmt.Sprintf("%d", time.Now().UnixNano()/1e6)
	nonce := uuid.New().String()

	signedMap := map[string]string{
		"x-ca-key":       c.AppKey,
		"x-ca-timestamp": timestamp,
		"x-ca-nonce":     nonce,
	}

	headersStr, headerKeys := BuildSignedHeaders(signedMap)
	signature := CalculateSignature(c.AppSecret, method, accept, contentType, headersStr, path)

	req.Header.Set("Accept", accept)
	req.Header.Set("Content-Type", contentType)
	if c.AppAccessToken != "" {
		req.Header.Set("App-Access-Token", c.AppAccessToken)
		req.Header.Set("token", c.AppAccessToken)
		req.Header.Set("access_token", c.AppAccessToken)
		req.Header.Set("Authorization", "Bearer "+c.AppAccessToken)
	}
	if c.UserAccessToken != "" {
		req.Header.Set("User-Access-Token", c.UserAccessToken)
	}
	req.Header.Set("x-ca-key", c.AppKey)
	req.Header.Set("x-ca-timestamp", timestamp)
	req.Header.Set("x-ca-nonce", nonce)
	req.Header.Set("x-ca-signature-headers", headerKeys)
	req.Header.Set("x-ca-signature", signature)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("发送海康 API 请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取海康 API 响应失败: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		if resp.StatusCode == http.StatusMethodNotAllowed {
			return fmt.Errorf("海康 API 返回 HTTP 405 Method Not Allowed。请确认 `base_url` 为 `https://open-api.hikiot.com`")
		}
		return fmt.Errorf("海康 API 返回错误状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	// 校验海康 API 返回的业务响应 code
	var baseResp BaseResponse
	if json.Unmarshal(respBody, &baseResp) == nil {
		cStr := baseResp.Code.String()
		if cStr != "0" && cStr != "200" && cStr != "" {
			return fmt.Errorf("海康 API 返回错误代码 [%s]: %s", cStr, baseResp.Msg)
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析海康响应失败: %w (原始响应: %s)", err, string(respBody))
		}
	}

	return nil
}

// GetOrgs 查询组织节点（海康互联云端）
func (c *Client) GetOrgs() ([]OrgDTO, error) {
	var resp BaseResponse
	
	// 尝试 GET 和 POST
	err := c.DoRequest("GET", "/team/v1/depart/list", map[string]interface{}{
		"pageNo":   1,
		"pageSize": 500,
	}, &resp)
	if err != nil {
		err = c.DoRequest("POST", "/team/v1/depart/list", map[string]interface{}{
			"pageNo":   1,
			"pageSize": 500,
		}, &resp)
	}
	if err != nil {
		err = c.DoRequest("GET", "/artemis/api/resource/v1/org/orgList", map[string]interface{}{
			"pageNo":   1,
			"pageSize": 500,
		}, &resp)
	}
	if err != nil {
		err = c.DoRequest("POST", "/artemis/api/resource/v1/org/orgList", map[string]interface{}{
			"pageNo":   1,
			"pageSize": 500,
		}, &resp)
	}
	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Data)
	var list []OrgDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// GetPersons 查询人员档案（海康互联云端）
func (c *Client) GetPersons() ([]PersonDTO, error) {
	var resp BaseResponse
	err := c.DoRequest("GET", "/team/v1/person/list", map[string]interface{}{
		"pageNo":   1,
		"pageSize": 500,
	}, &resp)
	if err != nil {
		err = c.DoRequest("POST", "/team/v1/person/list", map[string]interface{}{
			"pageNo":   1,
			"pageSize": 500,
		}, &resp)
	}
	if err != nil {
		err = c.DoRequest("GET", "/artemis/api/resource/v2/person/personList", map[string]interface{}{
			"pageNo":   1,
			"pageSize": 500,
		}, &resp)
	}
	if err != nil {
		err = c.DoRequest("POST", "/artemis/api/resource/v2/person/personList", map[string]interface{}{
			"pageNo":   1,
			"pageSize": 500,
		}, &resp)
	}
	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Data)
	var list []PersonDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// GetDoors 查询门禁设备列表（海康互联云端）
func (c *Client) GetDoors() ([]DoorDTO, error) {
	var resp BaseResponse
	
	// 首先尝试门禁资源专用接口
	err := c.DoRequest("GET", "/device/acs/v1/doorList", nil, &resp)
	if err != nil {
		// 如果失败，尝试分页查询设备列表接口
		err = c.DoRequest("GET", "/device/v1/page", map[string]interface{}{
			"page": 1,
			"size": 500,
		}, &resp)
	}

	if err != nil || resp.Data == nil {
		return nil, fmt.Errorf("海康 API 未能返回有效设备数据: %v", err)
	}

	b, _ := json.Marshal(resp.Data)

	// 1. 尝试直接 unmarshal 为 []DoorDTO
	var list []DoorDTO
	if json.Unmarshal(b, &list) == nil && len(list) > 0 {
		return list, nil
	}

	// 2. 尝试 unmarshal 为包含 list/rows/channels 的分页结构
	var objResp struct {
		List       []DoorDTO `json:"list"`
		Rows       []DoorDTO `json:"rows"`
		Channels   []DoorDTO `json:"channels"`
		DeviceList []DoorDTO `json:"deviceList"`
		Data       []DoorDTO `json:"data"`
	}
	if json.Unmarshal(b, &objResp) == nil {
		if len(objResp.List) > 0 {
			return objResp.List, nil
		}
		if len(objResp.Rows) > 0 {
			return objResp.Rows, nil
		}
		if len(objResp.Channels) > 0 {
			return objResp.Channels, nil
		}
		if len(objResp.DeviceList) > 0 {
			return objResp.DeviceList, nil
		}
		if len(objResp.Data) > 0 {
			return objResp.Data, nil
		}
	}

	return list, nil
}

// ControlDoor 远程控门指令（海康互联云端）
func (c *Client) ControlDoor(doorIndexCode string, command int) error {
	var resp BaseResponse
	err := c.DoRequest("POST", "/device/direct/v1/doorControl/remoteControlDoor", map[string]interface{}{
		"deviceSerial": doorIndexCode,
		"doorNo":       1,
		"cmd":          command,
	}, &resp)
	if err != nil {
		err = c.DoRequest("POST", "/artemis/api/acs/v1/door/control", map[string]interface{}{
			"doorIndexCodes": []string{doorIndexCode},
			"controlType":    command,
		}, &resp)
	}
	return err
}

// GetAttendanceRecords 查询考勤刷卡记录（海康互联云端）
func (c *Client) GetAttendanceRecords(startTime, endTime string) ([]AttendanceRecordDTO, error) {
	var resp BaseResponse
	err := c.DoRequest("GET", "/attendance/v1/event/page", map[string]interface{}{
		"startTime": startTime,
		"endTime":   endTime,
		"pageNo":    1,
		"pageSize":  1000,
	}, &resp)
	if err != nil {
		err = c.DoRequest("POST", "/attendance/v1/event/page", map[string]interface{}{
			"startTime": startTime,
			"endTime":   endTime,
			"pageNo":    1,
			"pageSize":  1000,
		}, &resp)
	}
	if err != nil {
		err = c.DoRequest("POST", "/artemis/api/acs/v2/door/events", map[string]interface{}{
			"startTime": startTime,
			"endTime":   endTime,
			"pageNo":    1,
			"pageSize":  1000,
		}, &resp)
	}
	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Data)
	var list []AttendanceRecordDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// ExchangeAppToken 海康互联云端获取 App Token
func (c *Client) ExchangeAppToken() (*AppTokenData, error) {
	var resp struct {
		Code FlexibleCode `json:"code"`
		Msg  string       `json:"msg"`
		Data AppTokenData  `json:"data"`
	}

	payload := map[string]interface{}{
		"appKey":    c.AppKey,
		"appSecret": c.AppSecret,
	}

	err := c.DoRequest("POST", "/auth/exchangeAppToken", payload, &resp)
	if err != nil {
		errV1 := c.DoRequest("POST", "/v1/auth/exchangeAppToken", payload, &resp)
		if errV1 != nil {
			return nil, err
		}
	}

	cStr := resp.Code.String()
	if cStr != "0" && cStr != "200" && cStr != "" {
		return nil, fmt.Errorf("海康云端 API 认证响应 [%s]: %s", cStr, resp.Msg)
	}

	if resp.Data.AppAccessToken != "" {
		c.AppAccessToken = resp.Data.AppAccessToken
	}

	return &resp.Data, nil
}
