package feishu

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type DefinitionCache struct {
	Controls  []map[string]interface{}
	Timestamp time.Time
}

type Client struct {
	AppID           string
	AppSecret       string
	Domain          string
	httpClient      *http.Client
	token           string
	tokenExpireTime time.Time
	tokenMutex      sync.Mutex
	defCache        map[string]*DefinitionCache
	defMutex        sync.Mutex
}

func NewClient(appID, appSecret, domain string) *Client {
	if domain == "" {
		domain = "https://open.feishu.cn"
	}
	domain = strings.TrimRight(domain, "/")
	return &Client{
		AppID:     strings.TrimSpace(appID),
		AppSecret: strings.TrimSpace(appSecret),
		Domain:    domain,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		defCache: make(map[string]*DefinitionCache),
	}
}

// GetTenantAccessToken 获取飞书 tenant_access_token (自动带内存缓存)
func (c *Client) GetTenantAccessToken() (string, error) {
	c.tokenMutex.Lock()
	defer c.tokenMutex.Unlock()

	if c.token != "" && time.Now().Before(c.tokenExpireTime) {
		return c.token, nil
	}

	url := c.Domain + "/open-apis/auth/v3/tenant_access_token/internal"
	payload := map[string]string{
		"app_id":     c.AppID,
		"app_secret": c.AppSecret,
	}

	jsonBytes, _ := json.Marshal(payload)
	resp, err := c.httpClient.Post(url, "application/json; charset=utf-8", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", fmt.Errorf("请求飞书 Token 接口失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", fmt.Errorf("解析飞书 Token 响应失败: %w", err)
	}

	code, _ := res["code"].(float64)
	if code != 0 {
		msg, _ := res["msg"].(string)
		return "", fmt.Errorf("获取飞书 Token 失败 code=%.0f, msg=%s", code, msg)
	}

	token, ok := res["tenant_access_token"].(string)
	if !ok || token == "" {
		return "", fmt.Errorf("飞书 Token 返回为空")
	}

	expire, _ := res["expire"].(float64)
	if expire <= 0 {
		expire = 7200
	}

	c.token = token
	// 预留 5 分钟提前刷新
	c.tokenExpireTime = time.Now().Add(time.Duration(expire-300) * time.Second)
	return c.token, nil
}

// GetApprovalDefinition 获取审批表单控件定义（自动缓存 5 分钟）
func (c *Client) GetApprovalDefinition(approvalCode string) ([]map[string]interface{}, error) {
	c.defMutex.Lock()
	if cached, ok := c.defCache[approvalCode]; ok && time.Since(cached.Timestamp) < 5*time.Minute {
		c.defMutex.Unlock()
		return cached.Controls, nil
	}
	c.defMutex.Unlock()

	token, err := c.GetTenantAccessToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/open-apis/approval/v4/approvals/%s?locale=zh-CN", c.Domain, approvalCode)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求飞书审批定义接口失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var res map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return nil, fmt.Errorf("解析飞书审批定义失败: %w", err)
	}

	data, _ := res["data"].(map[string]interface{})
	formStr, _ := data["form"].(string)
	if formStr == "" {
		return nil, fmt.Errorf("审批定义 %s 的 form 控件列表为空", approvalCode)
	}

	var controls []map[string]interface{}
	if err := json.Unmarshal([]byte(formStr), &controls); err != nil {
		return nil, fmt.Errorf("解析 form 控件列表失败: %w", err)
	}

	c.defMutex.Lock()
	c.defCache[approvalCode] = &DefinitionCache{
		Controls:  controls,
		Timestamp: time.Now(),
	}
	c.defMutex.Unlock()

	return controls, nil
}

// CreateApprovalInstance 创建飞书审批实例
func (c *Client) CreateApprovalInstance(approvalCode, openID, formStr, title string, nodeApprovers []map[string]interface{}) (string, error) {
	token, err := c.GetTenantAccessToken()
	if err != nil {
		return "", err
	}

	url := c.Domain + "/open-apis/approval/v4/instances"
	body := map[string]interface{}{
		"approval_code": approvalCode,
		"open_id":       openID,
		"form":          formStr,
	}
	if title != "" {
		body["title"] = title
	}
	if len(nodeApprovers) > 0 {
		body["node_approver_open_id_list"] = nodeApprovers
	}

	jsonBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("创建飞书审批实例请求失败: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	var res map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", fmt.Errorf("解析创建审批实例响应失败: %w", err)
	}

	code, _ := res["code"].(float64)
	if code != 0 {
		msg, _ := res["msg"].(string)
		return "", fmt.Errorf("创建飞书审批实例失败 code=%.0f, msg=%s", code, msg)
	}

	data, _ := res["data"].(map[string]interface{})
	instanceCode, _ := data["instance_code"].(string)
	if instanceCode == "" {
		return "", fmt.Errorf("飞书接口未返回 instance_code")
	}

	return instanceCode, nil
}

// CancelInstance 撤销飞书审批实例
func (c *Client) CancelInstance(instanceCode string) error {
	token, err := c.GetTenantAccessToken()
	if err != nil {
		return err
	}

	// 1. 先查询实例详情获取 approval_code 和发起人 open_id
	url := fmt.Sprintf("%s/open-apis/approval/v4/instances/%s?user_id_type=open_id", c.Domain, instanceCode)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	var detail map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &detail)
	data, _ := detail["data"].(map[string]interface{})
	approvalCode, _ := data["approval_code"].(string)
	initiatorOpenId, _ := data["open_id"].(string)

	if approvalCode == "" || initiatorOpenId == "" {
		return fmt.Errorf("取消审批失败：无法获取实例详情中的 approval_code 或 open_id")
	}

	// 2. 调用取消接口
	cancelUrl := c.Domain + "/open-apis/approval/v4/instances/cancel?user_id_type=open_id"
	cancelBody := map[string]interface{}{
		"instance_code": instanceCode,
		"approval_code": approvalCode,
		"user_id":       initiatorOpenId,
	}

	jsonBytes, _ := json.Marshal(cancelBody)
	reqCancel, _ := http.NewRequest("POST", cancelUrl, bytes.NewBuffer(jsonBytes))
	reqCancel.Header.Set("Authorization", "Bearer "+token)
	reqCancel.Header.Set("Content-Type", "application/json; charset=utf-8")

	respCancel, err := c.httpClient.Do(reqCancel)
	if err != nil {
		return err
	}
	respCancel.Body.Close()
	return nil
}

// SubscribeApprovalEvent 订阅具体审批定义的事件
func (c *Client) SubscribeApprovalEvent(approvalCode string) error {
	token, err := c.GetTenantAccessToken()
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/open-apis/approval/v4/approvals/%s/subscribe", c.Domain, approvalCode)
	req, _ := http.NewRequest("POST", url, bytes.NewBufferString("{}"))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	_ = json.Unmarshal(bodyBytes, &res)

	code, _ := res["code"].(float64)
	if code == 0 || code == 1390007 { // 1390007 为重复订阅幂等成功码
		log.Printf("[FeishuClient] 审批定义 %s 订阅成功", approvalCode)
		return nil
	}
	return fmt.Errorf("订阅审批事件失败: %s", string(bodyBytes))
}

// GetOpenIDByUserID 通过 user_id (如金蝶制单人ID) 查询飞书 OpenID
func (c *Client) GetOpenIDByUserID(userID string) (string, error) {
	return c.getOpenIDByUserType(userID, "user_id")
}

// GetOpenIDByUnionID 通过 union_id 查询飞书 OpenID
func (c *Client) GetOpenIDByUnionID(unionID string) (string, error) {
	return c.getOpenIDByUserType(unionID, "union_id")
}

func (c *Client) getOpenIDByUserType(id, userType string) (string, error) {
	token, err := c.GetTenantAccessToken()
	if err != nil {
		return "", err
	}

	url := fmt.Sprintf("%s/open-apis/contact/v3/users/%s?user_id_type=%s", c.Domain, id, userType)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", err
	}

	data, _ := res["data"].(map[string]interface{})
	user, _ := data["user"].(map[string]interface{})
	openID, _ := user["open_id"].(string)
	return openID, nil
}

// GetOpenIDByEmail 通过邮箱查询飞书 OpenID
func (c *Client) GetOpenIDByEmail(email string) (string, error) {
	token, err := c.GetTenantAccessToken()
	if err != nil {
		return "", err
	}

	url := c.Domain + "/open-apis/contact/v3/users/batch_get_id"
	body := map[string]interface{}{
		"emails": []string{email},
	}

	jsonBytes, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)
	var res map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &res); err != nil {
		return "", err
	}

	data, _ := res["data"].(map[string]interface{})
	userList, _ := data["user_list"].([]interface{})
	if len(userList) > 0 {
		first, _ := userList[0].(map[string]interface{})
		openID, _ := first["open_id"].(string)
		if openID == "" {
			openID, _ = first["user_id"].(string)
		}
		return openID, nil
	}
	return "", fmt.Errorf("邮箱未找到匹配的飞书用户")
}
