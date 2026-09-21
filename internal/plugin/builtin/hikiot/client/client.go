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
	BaseURL   string
	AppKey    string
	AppSecret string
	HTTPClient *http.Client
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

// DoRequest 发送签名请求
func (c *Client) DoRequest(method, path string, bodyData interface{}, result interface{}) error {
	if c.AppKey == "" || c.AppSecret == "" {
		return fmt.Errorf("海康互联未设置 AppKey 或 AppSecret，请先在插件设置中配置凭据")
	}

	url := c.BaseURL + path
	var bodyReader io.Reader

	if bodyData != nil {
		b, err := json.Marshal(bodyData)
		if err != nil {
			return fmt.Errorf("序列化请求参数失败: %w", err)
		}
		bodyReader = bytes.NewBuffer(b)
	}

	req, err := http.NewRequest(method, url, bodyReader)
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
			return fmt.Errorf("海康 API 返回 HTTP 405 Method Not Allowed。主要原因：\n1. `base_url` 填写了 `http://` 触发了服务器重定向(301/302)到 `https://`，导致 POST 请求降级为 GET 请求。建议将 `base_url` 协议明确修改为 `https://`；\n2. `base_url` 端口或路径不正确（海康 Artemis 网关私有部署默认端口通常为 8443，如 `https://192.168.10.x:8443`）；\n3. 填写的 URL 域名/IP 指向了 Web 前端界面而非 Artemis API 开放网关端口。")
		}
		return fmt.Errorf("海康 API 返回错误状态码 %d: %s", resp.StatusCode, string(respBody))
	}

	// 校验海康 Artemis API 返回的业务响应 code
	var baseResp BaseResponse
	if json.Unmarshal(respBody, &baseResp) == nil {
		if baseResp.Code != "0" && baseResp.Code != "200" && baseResp.Code != "" {
			return fmt.Errorf("海康 API 返回错误代码 [%s]: %s", baseResp.Code, baseResp.Msg)
		}
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("解析海康响应失败: %w (原始响应: %s)", err, string(respBody))
		}
	}

	return nil
}

// GetOrgs 查询组织节点
func (c *Client) GetOrgs() ([]OrgDTO, error) {
	var resp BaseResponse
	err := c.DoRequest("POST", "/artemis/api/resource/v1/org/orgList", map[string]interface{}{
		"pageNo":   1,
		"pageSize": 500,
	}, &resp)
	if err != nil {
		return nil, err
	}

	// 转换为 OrgDTO 切片
	b, _ := json.Marshal(resp.Data)
	var list []OrgDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// GetPersons 查询人员档案
func (c *Client) GetPersons() ([]PersonDTO, error) {
	var resp BaseResponse
	err := c.DoRequest("POST", "/artemis/api/resource/v2/person/personList", map[string]interface{}{
		"pageNo":   1,
		"pageSize": 500,
	}, &resp)
	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Data)
	var list []PersonDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// GetDoors 查询门禁设备列表
func (c *Client) GetDoors() ([]DoorDTO, error) {
	var resp BaseResponse
	err := c.DoRequest("POST", "/artemis/api/resource/v1/door/doorList", map[string]interface{}{
		"pageNo":   1,
		"pageSize": 500,
	}, &resp)
	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Data)
	var list []DoorDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// ControlDoor 远程控门指令
func (c *Client) ControlDoor(doorIndexCode string, command int) error {
	var resp BaseResponse
	err := c.DoRequest("POST", "/artemis/api/acs/v1/door/control", map[string]interface{}{
		"doorIndexCodes": []string{doorIndexCode},
		"controlType":    command,
	}, &resp)
	return err
}

// GetAttendanceRecords 查询考勤刷卡记录
func (c *Client) GetAttendanceRecords(startTime, endTime string) ([]AttendanceRecordDTO, error) {
	var resp BaseResponse
	err := c.DoRequest("POST", "/artemis/api/acs/v2/door/events", map[string]interface{}{
		"startTime": startTime,
		"endTime":   endTime,
		"pageNo":    1,
		"pageSize":  1000,
	}, &resp)
	if err != nil {
		return nil, err
	}

	b, _ := json.Marshal(resp.Data)
	var list []AttendanceRecordDTO
	_ = json.Unmarshal(b, &list)
	return list, nil
}

// ExchangeAppToken 海康互联云端获取 App Token 测试鉴权
func (c *Client) ExchangeAppToken() error {
	var resp struct {
		Code string      `json:"code"`
		Msg  string      `json:"msg"`
		Data interface{} `json:"data"`
	}

	payload := map[string]interface{}{
		"appKey":    c.AppKey,
		"appSecret": c.AppSecret,
	}

	err := c.DoRequest("POST", "/auth/exchangeAppToken", payload, &resp)
	if err != nil {
		errV1 := c.DoRequest("POST", "/v1/auth/exchangeAppToken", payload, &resp)
		if errV1 != nil {
			return err
		}
	}

	if resp.Code != "0" && resp.Code != "200" && resp.Code != "" {
		return fmt.Errorf("海康云端 API 认证响应 [%s]: %s", resp.Code, resp.Msg)
	}

	return nil
}
