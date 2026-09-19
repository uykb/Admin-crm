package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.tailscale.com"

type Client struct {
	baseURL      string
	tailnet      string
	apiKey       string
	clientID     string
	clientSecret string
	token        string
	httpClient   *http.Client
}

func NewClient(tailnet, apiKey, clientID, clientSecret string) *Client {
	return &Client{
		baseURL:      DefaultBaseURL,
		tailnet:      tailnet,
		apiKey:       apiKey,
		clientID:     clientID,
		clientSecret: clientSecret,
		httpClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// ensureToken 获取可用的 Authorization Header Token
func (c *Client) ensureToken() (string, error) {
	// 如果配置了直接的 API Key，优先使用 API Key
	if c.apiKey != "" {
		return c.apiKey, nil
	}
	// 如果配置了 Token 变量
	if c.token != "" {
		return c.token, nil
	}
	// 否则尝试使用 OAuth Client ID & Secret 换取 Token
	if c.clientID != "" && c.clientSecret != "" {
		token, err := c.fetchOAuthToken()
		if err != nil {
			return "", fmt.Errorf("OAuth 鉴权失败: %w", err)
		}
		c.token = token
		return token, nil
	}
	return "", fmt.Errorf("未配置有效的 Tailscale API Key 或 OAuth Client 凭证")
}

// fetchOAuthToken 请求 OAuth Token 接口
func (c *Client) fetchOAuthToken() (string, error) {
	apiURL := fmt.Sprintf("%s/api/v2/oauth/token", c.baseURL)
	data := url.Values{}
	data.Set("client_id", c.clientID)
	data.Set("client_secret", c.clientSecret)

	req, err := http.NewRequest(http.MethodPost, apiURL, strings.NewReader(data.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var res OAuthTokenResponse
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", err
	}
	return res.AccessToken, nil
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(method, path string, body interface{}) ([]byte, error) {
	token, err := c.ensureToken()
	if err != nil {
		return nil, err
	}

	apiURL := fmt.Sprintf("%s%s", c.baseURL, path)
	var bodyReader io.Reader
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyReader = bytes.NewReader(jsonBytes)
	}

	req, err := http.NewRequest(method, apiURL, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("Tailscale API 响应错误 [%d]: %s", resp.StatusCode, string(respBytes))
	}

	return respBytes, nil
}

// ListDevices 获取 Tailnet 所有设备节点
func (c *Client) ListDevices() ([]Device, error) {
	if c.tailnet == "" {
		return nil, fmt.Errorf("未配置 Tailnet 名称")
	}
	path := fmt.Sprintf("/api/v2/tailnet/%s/devices", url.PathEscape(c.tailnet))
	bytes, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res DevicesResponse
	if err := json.Unmarshal(bytes, &res); err != nil {
		return nil, fmt.Errorf("解析设备列表响应失败: %w", err)
	}
	return res.Devices, nil
}

// GetDevice 获取单个设备节点详情
func (c *Client) GetDevice(deviceID string) (*Device, error) {
	path := fmt.Sprintf("/api/v2/device/%s", url.PathEscape(deviceID))
	bytes, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var dev Device
	if err := json.Unmarshal(bytes, &dev); err != nil {
		return nil, fmt.Errorf("解析设备详情失败: %w", err)
	}
	return &dev, nil
}

// DeleteDevice 删除/下线指定设备节点
func (c *Client) DeleteDevice(deviceID string) error {
	path := fmt.Sprintf("/api/v2/device/%s", url.PathEscape(deviceID))
	_, err := c.doRequest(http.MethodDelete, path, nil)
	return err
}

// GetDeviceRoutes 获取设备广播与已核准路由
func (c *Client) GetDeviceRoutes(deviceID string) (*RoutesResponse, error) {
	path := fmt.Sprintf("/api/v2/device/%s/routes", url.PathEscape(deviceID))
	bytes, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res RoutesResponse
	if err := json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// SetDeviceRoutes 审批并生效设备路由 (Subnet Router / Exit Node)
func (c *Client) SetDeviceRoutes(deviceID string, routes []string) (*RoutesResponse, error) {
	path := fmt.Sprintf("/api/v2/device/%s/routes", url.PathEscape(deviceID))
	reqBody := SetRoutesRequest{Routes: routes}
	bytes, err := c.doRequest(http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	var res RoutesResponse
	if err := json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return &res, nil
}

// CreateAuthKey 生成新 Auth Key
func (c *Client) CreateAuthKey(reusable, ephemeral, preauth bool, tags []string, expiryDays int) (*CreateKeyResponse, error) {
	if c.tailnet == "" {
		return nil, fmt.Errorf("未配置 Tailnet 名称")
	}
	path := fmt.Sprintf("/api/v2/tailnet/%s/keys", url.PathEscape(c.tailnet))

	reqBody := CreateKeyRequest{}
	reqBody.Capabilities.Devices.Create.Reusable = reusable
	reqBody.Capabilities.Devices.Create.Ephemeral = ephemeral
	reqBody.Capabilities.Devices.Create.Preauthorized = preauth
	reqBody.Capabilities.Devices.Create.Tags = tags

	if expiryDays > 0 {
		reqBody.ExpirySeconds = expiryDays * 86400
	}

	bytes, err := c.doRequest(http.MethodPost, path, reqBody)
	if err != nil {
		return nil, err
	}

	var res CreateKeyResponse
	if err := json.Unmarshal(bytes, &res); err != nil {
		return nil, fmt.Errorf("解析密钥生成响应失败: %w", err)
	}
	return &res, nil
}

// ListAuthKeys 获取可用 Auth Keys 列表
func (c *Client) ListAuthKeys() ([]KeyInfo, error) {
	if c.tailnet == "" {
		return nil, fmt.Errorf("未配置 Tailnet 名称")
	}
	path := fmt.Sprintf("/api/v2/tailnet/%s/keys", url.PathEscape(c.tailnet))
	bytes, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}

	var res KeysResponse
	if err := json.Unmarshal(bytes, &res); err != nil {
		return nil, err
	}
	return res.Keys, nil
}

// RevokeAuthKey 撤销指定 Auth Key
func (c *Client) RevokeAuthKey(keyID string) error {
	if c.tailnet == "" {
		return fmt.Errorf("未配置 Tailnet 名称")
	}
	path := fmt.Sprintf("/api/v2/tailnet/%s/keys/%s", url.PathEscape(c.tailnet), url.PathEscape(keyID))
	_, err := c.doRequest(http.MethodDelete, path, nil)
	return err
}

// GetACL 获取 Policy ACL 安全策略
func (c *Client) GetACL() (string, error) {
	if c.tailnet == "" {
		return "", fmt.Errorf("未配置 Tailnet 名称")
	}
	path := fmt.Sprintf("/api/v2/tailnet/%s/acl", url.PathEscape(c.tailnet))
	bytes, err := c.doRequest(http.MethodGet, path, nil)
	if err != nil {
		return "", err
	}
	return string(bytes), nil
}
