package client

import "time"

// OAuthTokenResponse OAuth2 换取 Token 响应
type OAuthTokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
	Scope       string `json:"scope"`
}

// Device Tailscale 设备节点数据
type Device struct {
	ID                string   `json:"id"`
	Name              string   `json:"name"`
	Hostname          string   `json:"hostname"`
	User              string   `json:"user"`
	Addresses         []string `json:"addresses"`
	OS                string   `json:"os"`
	ClientVersion     string   `json:"clientVersion"`
	UpdateAvailable   bool     `json:"updateAvailable"`
	LastSeen          time.Time`json:"lastSeen"`
	Online            bool     `json:"online"`
	KeyExpiryDisabled bool     `json:"keyExpiryDisabled"`
	Tags              []string `json:"tags"`
	NodeKey           string   `json:"nodeKey"`
}

// DevicesResponse 设备列表响应
type DevicesResponse struct {
	Devices []Device `json:"devices"`
}

// KeyCapabilities 能力定义
type KeyCapabilities struct {
	Devices struct {
		Create struct {
			Reusable      bool     `json:"reusable"`
			Ephemeral     bool     `json:"ephemeral"`
			Preauthorized bool     `json:"preauthorized"`
			Tags          []string `json:"tags"`
		} `json:"create"`
	} `json:"devices"`
}

// CreateKeyRequest 创建 Auth Key 请求
type CreateKeyRequest struct {
	Capabilities KeyCapabilities `json:"capabilities"`
	ExpirySeconds int             `json:"expirySeconds,omitempty"`
}

// CreateKeyResponse 创建 Auth Key 响应
type CreateKeyResponse struct {
	ID           string          `json:"id"`
	Key          string          `json:"key"`
	Created      time.Time       `json:"created"`
	Expires      time.Time       `json:"expires"`
	Capabilities KeyCapabilities `json:"capabilities"`
}

// KeyInfo 密钥信息
type KeyInfo struct {
	ID           string          `json:"id"`
	Created      time.Time       `json:"created"`
	Expires      time.Time       `json:"expires"`
	Capabilities KeyCapabilities `json:"capabilities"`
}

// KeysResponse 密钥列表响应
type KeysResponse struct {
	Keys []KeyInfo `json:"keys"`
}

// RoutesResponse 子网路由响应
type RoutesResponse struct {
	AdvertisedRoutes []string `json:"advertisedRoutes"`
	EnabledRoutes    []string `json:"enabledRoutes"`
}

// SetRoutesRequest 设置审批路由请求
type SetRoutesRequest struct {
	Routes []string `json:"routes"`
}

// ACLResponse Policy HuJSON ACL
type ACLResponse struct {
	ACLs        []map[string]interface{} `json:"acls,omitempty"`
	Groups      map[string][]string      `json:"groups,omitempty"`
	TagOwners   map[string][]string      `json:"tagOwners,omitempty"`
	Hosts       map[string]string        `json:"hosts,omitempty"`
	AutoApprovers map[string]interface{} `json:"autoApprovers,omitempty"`
	HUJSON      string                   `json:"hujson,omitempty"`
}
