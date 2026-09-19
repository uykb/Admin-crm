package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"time"
)

// normalizeServerURL 自动规范化金蝶服务地址（确保以 /k3cloud 结尾）
func normalizeServerURL(rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return ""
	}
	if !strings.HasPrefix(rawURL, "http://") && !strings.HasPrefix(rawURL, "https://") {
		rawURL = "http://" + rawURL
	}
	u, err := url.Parse(rawURL)
	if err != nil {
		return strings.TrimRight(rawURL, "/")
	}

	path := strings.TrimRight(u.Path, "/")
	idx := strings.Index(strings.ToLower(path), "/k3cloud")
	if idx >= 0 {
		path = path[:idx+len("/k3cloud")]
	} else {
		path = path + "/k3cloud"
	}
	u.Path = path
	return strings.TrimRight(u.String(), "/")
}

// Client 金蝶云星空 Web API 客户端
type Client struct {
	ServerURL  string
	DbID       string
	Username   string
	Password   string
	AppID      string
	AppSecret  string
	Lcid       int
	HTTPClient *http.Client
}

// NewClient 实例化 Client 并自动装载 CookieJar 维护 Session
func NewClient(serverURL, dbID, username, password, appID, appSecret string, lcid int) *Client {
	jar, _ := cookiejar.New(nil)
	if lcid <= 0 {
		lcid = 2052 // 默认简体中文
	}
	serverURL = normalizeServerURL(serverURL)
	return &Client{
		ServerURL: serverURL,
		DbID:      strings.TrimSpace(dbID),
		Username:  strings.TrimSpace(username),
		Password:  strings.TrimSpace(password),
		AppID:     strings.TrimSpace(appID),
		AppSecret: strings.TrimSpace(appSecret),
		Lcid:      lcid,
		HTTPClient: &http.Client{
			Jar:     jar,
			Timeout: 30 * time.Second,
		},
	}
}

// tryAuthenticateUrl 使用指定 URL 与参数调用金蝶鉴权接口
func (c *Client) tryAuthenticateUrl(endpoint string, params []interface{}) error {
	url := c.ServerURL + endpoint
	payload := map[string]interface{}{
		"parameters": params,
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
	bodyBytes = bytes.TrimPrefix(bodyBytes, []byte("\xef\xbb\xbf"))
	bodyBytes = bytes.TrimSpace(bodyBytes)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var authResp AuthResponse
	if err := json.Unmarshal(bodyBytes, &authResp); err == nil {
		if authResp.ResponseStatus.IsSuccess || authResp.LoginResultType == 1 {
			return nil
		}
		if len(authResp.ResponseStatus.Errors) > 0 {
			return fmt.Errorf("%s", authResp.ResponseStatus.Errors[0].Message)
		}
	}

	// 部分金蝶版本直接返回登录结果数字 (1=成功)
	if strings.Contains(string(bodyBytes), `"LoginResultType":1`) || strings.Contains(string(bodyBytes), `"IsSuccess":true`) {
		return nil
	}

	return fmt.Errorf("%s", string(bodyBytes))
}

type authCandidate struct {
	Endpoint string
	Params   []interface{}
}

// Authenticate 账套登录鉴权（自动尝试标准 ValidateUser, LoginByAppSecret 与 ApiService 候选服务）
func (c *Client) Authenticate() error {
	if c.ServerURL == "" {
		return fmt.Errorf("未配置内网金蝶云星空服务地址")
	}
	if c.DbID == "" || c.Username == "" {
		return fmt.Errorf("未配置金蝶账套 ID 或登录用户")
	}

	var candidates []authCandidate

	// 1. 金蝶官方标准 ValidateUser 密码登录服务
	if c.Password != "" {
		candidates = append(candidates, authCandidate{
			Endpoint: "/Kingdee.BOS.WebApi.ServicesStub.AuthService.ValidateUser.common.kdsvc",
			Params:   []interface{}{c.DbID, c.Username, c.Password, c.Lcid},
		})
	}

	// 2. 金蝶官方标准 LoginByAppSecret 应用授权服务
	if c.AppID != "" && c.AppSecret != "" {
		candidates = append(candidates, authCandidate{
			Endpoint: "/Kingdee.BOS.WebApi.ServicesStub.AuthService.LoginByAppSecret.common.kdsvc",
			Params:   []interface{}{c.DbID, c.Username, c.AppID, c.AppSecret, c.Lcid},
		})
	}

	// 3. ApiService.Authenticate 兼容入口 (密码模式)
	if c.Password != "" {
		candidates = append(candidates, authCandidate{
			Endpoint: "/Kingdee.BOS.WebApi.ServicesRepository.ApiService.Authenticate.common.kdsvc",
			Params:   []interface{}{c.DbID, c.Username, c.Password, c.Lcid},
		})
	}

	// 4. ApiService.Authenticate 兼容入口 (AppID/AppSecret 模式)
	if c.AppID != "" && c.AppSecret != "" {
		candidates = append(candidates, authCandidate{
			Endpoint: "/Kingdee.BOS.WebApi.ServicesRepository.ApiService.Authenticate.common.kdsvc",
			Params:   []interface{}{c.DbID, c.Username, c.AppID, c.AppSecret, c.Lcid},
		})
	}

	if len(candidates) == 0 {
		return fmt.Errorf("未配置金蝶登录密码或应用密钥 (AppSecret)")
	}

	var lastErr error
	for _, cand := range candidates {
		if err := c.tryAuthenticateUrl(cand.Endpoint, cand.Params); err == nil {
			return nil
		} else {
			lastErr = err
		}
	}

	return fmt.Errorf("金蝶登录未成功，服务器响应: %w", lastErr)
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
	bodyBytes = bytes.TrimPrefix(bodyBytes, []byte("\xef\xbb\xbf"))
	bodyBytes = bytes.TrimSpace(bodyBytes)

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("金蝶单据接口返回 HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var rows [][]interface{}
	if err := json.Unmarshal(bodyBytes, &rows); err != nil {
		// 检查是否返回了错误对象 map
		var errObj map[string]interface{}
		if json.Unmarshal(bodyBytes, &errObj) == nil {
			if respStatus, ok := errObj["ResponseStatus"].(map[string]interface{}); ok {
				if errs, ok := respStatus["Errors"].([]interface{}); ok && len(errs) > 0 {
					if errMap, ok := errs[0].(map[string]interface{}); ok {
						if msg, ok := errMap["Message"].(string); ok {
							return nil, fmt.Errorf("金蝶接口返回错误: %s", msg)
						}
					}
				}
			}
			return nil, fmt.Errorf("金蝶接口返回错误: %s", string(bodyBytes))
		}

		// 检查是否返回了包裹在数组中的错误对象 [{"Result": ...}] 或 [{"ResponseStatus": ...}]
		var errArr []map[string]interface{}
		if json.Unmarshal(bodyBytes, &errArr) == nil && len(errArr) > 0 {
			first := errArr[0]
			if res, ok := first["Result"].(map[string]interface{}); ok {
				if respStatus, ok := res["ResponseStatus"].(map[string]interface{}); ok {
					if errs, ok := respStatus["Errors"].([]interface{}); ok && len(errs) > 0 {
						if errMap, ok := errs[0].(map[string]interface{}); ok {
							if msg, ok := errMap["Message"].(string); ok {
								return nil, fmt.Errorf("金蝶接口返回错误: %s", msg)
							}
						}
					}
				}
			}
			return nil, fmt.Errorf("金蝶接口返回错误: %s", string(bodyBytes))
		}

		return nil, fmt.Errorf("解析金蝶数据响应失败: %w, 服务器原始响应内容: %s", err, string(bodyBytes))
	}

	return rows, nil
}
