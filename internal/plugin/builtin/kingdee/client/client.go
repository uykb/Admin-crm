package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

// Client 金蝶云星空 Web API 客户端
type Client struct {
	ServerURL  string
	DbID       string
	Username   string
	Password   string
	Lcid       int
	HTTPClient *http.Client
}

// NewClient 实例化 Client 并自动装载 CookieJar 维护 Session
func NewClient(serverURL, dbID, username, password string, lcid int) *Client {
	jar, _ := cookiejar.New(nil)
	if lcid <= 0 {
		lcid = 2052 // 默认简体中文
	}
	serverURL = strings.TrimRight(serverURL, "/")
	return &Client{
		ServerURL: serverURL,
		DbID:      dbID,
		Username:  username,
		Password:  password,
		Lcid:      lcid,
		HTTPClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

// Authenticate 账套登录鉴权（设置 Session Cookie）
func (c *Client) Authenticate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("未配置内网金蝶云星空服务器地址")
	}
	if c.DbID == "" || c.Username == "" {
		return fmt.Errorf("未配置金蝶账套 ID 或用户名")
	}

	url := c.ServerURL + "/Kingdee.BOS.WebApi.ServicesRepository.ApiService.Authenticate.common.kdsvc"

	payload := map[string]interface{}{
		"parameters": []interface{}{
			c.DbID,
			c.Username,
			c.Password,
			c.Lcid,
		},
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("请求金蝶 Web API 失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return fmt.Errorf("金蝶接口返回 HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(bodyBytes, &authResp); err == nil {
		if authResp.ResponseStatus.IsSuccess || authResp.LoginResultType == 1 {
			return nil
		}
		if len(authResp.ResponseStatus.Errors) > 0 {
			return fmt.Errorf("金蝶登录失败: %s", authResp.ResponseStatus.Errors[0].Message)
		}
	}

	// 部分金蝶版本直接返回登录结果数字 (1=成功)
	if strings.Contains(string(bodyBytes), `"LoginResultType":1`) || strings.Contains(string(bodyBytes), `"IsSuccess":true`) {
		return nil
	}

	return nil
}

// ExecuteBillQuery 执行通用单据/表单列表查询
func (c *Client) ExecuteBillQuery(reqData BillQueryData) ([][]interface{}, error) {
	// 先执行登录认证
	if err := c.Authenticate(); err != nil {
		return nil, err
	}

	url := c.ServerURL + "/Kingdee.BOS.WebApi.ServicesRepository.ApiService.ExecuteBillQuery.common.kdsvc"

	if reqData.Limit <= 0 {
		reqData.Limit = 50
	}

	payload := map[string]interface{}{
		"data": reqData,
	}

	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求金蝶单据查询接口失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("金蝶单据接口返回 HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rows [][]interface{}
	if err := json.Unmarshal(bodyBytes, &rows); err != nil {
		// 检查是否返回了错误对象而非二维数组
		var errObj map[string]interface{}
		if json.Unmarshal(bodyBytes, &errObj) == nil {
			return nil, fmt.Errorf("金蝶接口返回错误: %s", string(bodyBytes))
		}
		return nil, fmt.Errorf("解析金蝶数据响应失败: %w", err)
	}

	return rows, nil
}
