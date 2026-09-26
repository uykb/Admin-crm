package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/url"
	"strings"
	"time"

	tsclient "apeadmin-gin/internal/plugin/builtin/tailscale/client"
	tsmodel "apeadmin-gin/internal/plugin/builtin/tailscale/model"
	tsproxy "apeadmin-gin/internal/plugin/builtin/tailscale/proxy"

	"golang.org/x/net/proxy"
	"gorm.io/gorm"
)

type TailscaleService struct {
	db *gorm.DB
}

func NewTailscaleService(db *gorm.DB) *TailscaleService {
	return &TailscaleService{db: db}
}

// ConfigDTO 配置传输对象
type ConfigDTO struct {
	Tailnet       string `json:"tailnet"`
	APIKey        string `json:"api_key"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	WebhookSecret string `json:"webhook_secret"`
	ProxyURL      string `json:"proxy_url"`
	AuthKey       string `json:"auth_key"`
	NodeHostname  string `json:"node_hostname"`
	TsnetRunning  bool   `json:"tsnet_running"`
}

// GetConfig 读取当前连接配置
func (s *TailscaleService) GetConfig() (*ConfigDTO, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}

	var configs []tsmodel.TsConfig
	s.db.Find(&configs)

	cfgMap := make(map[string]string)
	for _, c := range configs {
		cfgMap[c.Key] = c.Value
	}

	hostname := cfgMap["tailscale_node_hostname"]
	if hostname == "" {
		hostname = "apeadmin-crm"
	}

	return &ConfigDTO{
		Tailnet:       cfgMap["tailscale_tailnet"],
		APIKey:        cfgMap["tailscale_api_key"],
		ClientID:      cfgMap["tailscale_client_id"],
		ClientSecret:  cfgMap["tailscale_client_secret"],
		WebhookSecret: cfgMap["tailscale_webhook_secret"],
		ProxyURL:      cfgMap["tailscale_proxy_url"],
		AuthKey:       cfgMap["tailscale_auth_key"],
		NodeHostname:  hostname,
		TsnetRunning:  tsproxy.GetTsnetManager().IsRunning(),
	}, nil
}

// SaveConfig 保存连接配置
func (s *TailscaleService) SaveConfig(tailnet, apiKey, clientID, clientSecret, webhookSecret, proxyURL, authKey, nodeHostname string) error {
	if s.db == nil {
		return fmt.Errorf("数据库连接不可用")
	}

	if nodeHostname == "" {
		nodeHostname = "apeadmin-crm"
	}

	items := map[string]string{
		"tailscale_tailnet":        tailnet,
		"tailscale_api_key":        apiKey,
		"tailscale_client_id":      clientID,
		"tailscale_client_secret":  clientSecret,
		"tailscale_webhook_secret": webhookSecret,
		"tailscale_proxy_url":      proxyURL,
		"tailscale_auth_key":       authKey,
		"tailscale_node_hostname":  nodeHostname,
	}

	for k, v := range items {
		var cfg tsmodel.TsConfig
		if err := s.db.Where("key = ?", k).First(&cfg).Error; err == nil {
			cfg.Value = v
			s.db.Save(&cfg)
		} else {
			s.db.Create(&tsmodel.TsConfig{
				Key:      k,
				Value:    v,
				IsPublic: false,
			})
		}
	}

	// 若配置了 AuthKey 或 ProxyURL 则自动激活通信引擎
	if authKey != "" || proxyURL != "" {
		_ = tsproxy.GetTsnetManager().Start(authKey, nodeHostname)
		if proxyURL != "" {
			tsproxy.GetTsnetManager().SetProxyURL(proxyURL)
		}
	}

	return nil
}

// getClient 实例化 Tailscale API 客户端
func (s *TailscaleService) getClient() (*tsclient.Client, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	if cfg.Tailnet == "" {
		return nil, fmt.Errorf("未配置 Tailnet 名称，请先前往 [ Tailscale 密钥配置 ] 进行设置")
	}
	if cfg.APIKey == "" && (cfg.ClientID == "" || cfg.ClientSecret == "") {
		return nil, fmt.Errorf("未配置 Tailscale API Key 或 OAuth 凭证")
	}

	return tsclient.NewClient(cfg.Tailnet, cfg.APIKey, cfg.ClientID, cfg.ClientSecret), nil
}

// SyncDevices 从 Tailscale 官方同步网络设备节点并更新镜像表
func (s *TailscaleService) SyncDevices() (int, error) {
	cli, err := s.getClient()
	if err != nil {
		return 0, err
	}

	devices, err := cli.ListDevices()
	if err != nil {
		return 0, fmt.Errorf("获取设备列表失败: %w", err)
	}

	for _, dev := range devices {
		ipsJSON, _ := json.Marshal(dev.Addresses)
		tagsJSON, _ := json.Marshal(dev.Tags)

		routes, _ := cli.GetDeviceRoutes(dev.ID)
		routesJSON, _ := json.Marshal(routes)

		isOnline := dev.Online || dev.Connected
		if !isOnline && !dev.LastSeen.IsZero() && time.Since(dev.LastSeen) < 10*time.Minute {
			isOnline = true
		}

		var record tsmodel.TsDeviceCache
		if err := s.db.Where("device_id = ?", dev.ID).First(&record).Error; err == nil {
			record.Name = dev.Name
			record.Hostname = dev.Hostname
			record.User = dev.User
			record.IPs = string(ipsJSON)
			record.OS = dev.OS
			record.ClientVersion = dev.ClientVersion
			record.UpdateAvailable = dev.UpdateAvailable
			record.LastSeen = dev.LastSeen
			record.Online = isOnline
			record.Authorized = dev.Authorized
			record.KeyExpiryDisabled = dev.KeyExpiryDisabled
			record.ExpiresAt = dev.Expires
			record.Tags = string(tagsJSON)
			record.SubnetRoutes = string(routesJSON)
			record.NodeKey = dev.NodeKey
			s.db.Save(&record)
		} else {
			s.db.Create(&tsmodel.TsDeviceCache{
				DeviceID:          dev.ID,
				Name:              dev.Name,
				Hostname:          dev.Hostname,
				User:              dev.User,
				IPs:               string(ipsJSON),
				OS:                dev.OS,
				ClientVersion:     dev.ClientVersion,
				UpdateAvailable:   dev.UpdateAvailable,
				LastSeen:          dev.LastSeen,
				Online:            isOnline,
				Authorized:        dev.Authorized,
				KeyExpiryDisabled: dev.KeyExpiryDisabled,
				ExpiresAt:         dev.Expires,
				Tags:              string(tagsJSON),
				SubnetRoutes:      string(routesJSON),
				NodeKey:           dev.NodeKey,
			})
		}
	}

	return len(devices), nil
}

// ListCachedDevices 获取本地缓存的设备列表
func (s *TailscaleService) ListCachedDevices(keyword string) ([]tsmodel.TsDeviceCache, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}

	var devices []tsmodel.TsDeviceCache
	query := s.db.Order("online desc, last_seen desc")
	if keyword != "" {
		likePattern := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR hostname LIKE ? OR ips LIKE ? OR user LIKE ? OR tags LIKE ?",
			likePattern, likePattern, likePattern, likePattern, likePattern)
	}

	if err := query.Find(&devices).Error; err != nil {
		return nil, err
	}

	// 如果数据库为空，自动触发一次同步
	if len(devices) == 0 && keyword == "" {
		_, _ = s.SyncDevices()
		s.db.Order("online desc, last_seen desc").Find(&devices)
	}

	for i := range devices {
		if !devices[i].Online && !devices[i].LastSeen.IsZero() && time.Since(devices[i].LastSeen) < 10*time.Minute {
			devices[i].Online = true
		}
	}

	return devices, nil
}

// DeleteDevice 解绑并移除 Tailnet 设备
func (s *TailscaleService) DeleteDevice(deviceID string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	if err := cli.DeleteDevice(deviceID); err != nil {
		return err
	}

	// 删除本地缓存镜像
	if s.db != nil {
		s.db.Where("device_id = ?", deviceID).Delete(&tsmodel.TsDeviceCache{})
	}
	return nil
}

// SetKeyExpiry 设置设备免密钥过期
func (s *TailscaleService) SetKeyExpiry(deviceID string, disabled bool) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	if err := cli.SetDeviceKeyExpiry(deviceID, disabled); err != nil {
		return err
	}

	if s.db != nil {
		s.db.Model(&tsmodel.TsDeviceCache{}).Where("device_id = ?", deviceID).Update("key_expiry_disabled", disabled)
	}
	return nil
}

// SetDeviceName 修改设备名称
func (s *TailscaleService) SetDeviceName(deviceID string, name string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	if err := cli.SetDeviceName(deviceID, name); err != nil {
		return err
	}

	if s.db != nil {
		s.db.Model(&tsmodel.TsDeviceCache{}).Where("device_id = ?", deviceID).Update("name", name)
	}
	return nil
}

// SetDeviceTags 修改设备标签
func (s *TailscaleService) SetDeviceTags(deviceID string, tags []string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	if err := cli.SetDeviceTags(deviceID, tags); err != nil {
		return err
	}

	if s.db != nil {
		tagsJSON, _ := json.Marshal(tags)
		s.db.Model(&tsmodel.TsDeviceCache{}).Where("device_id = ?", deviceID).Update("tags", string(tagsJSON))
	}
	return nil
}

// SetDeviceAuthorized 设备核准授权
func (s *TailscaleService) SetDeviceAuthorized(deviceID string, authorized bool) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	if err := cli.SetDeviceAuthorized(deviceID, authorized); err != nil {
		return err
	}

	if s.db != nil {
		s.db.Model(&tsmodel.TsDeviceCache{}).Where("device_id = ?", deviceID).Update("authorized", authorized)
	}
	return nil
}

// SetDeviceIP 设置静态 IP
func (s *TailscaleService) SetDeviceIP(deviceID string, ipv4, ipv6 string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	return cli.SetDeviceIP(deviceID, ipv4, ipv6)
}

// GetDeviceRoutes 获取设备路由详情
func (s *TailscaleService) GetDeviceRoutes(deviceID string) (*tsclient.RoutesResponse, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}
	return cli.GetDeviceRoutes(deviceID)
}

// ApproveSubnetRoutes 审批设备广播的子网路由与 Exit Node
func (s *TailscaleService) ApproveSubnetRoutes(deviceID string, routes []string) (*tsclient.RoutesResponse, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}

	res, err := cli.SetDeviceRoutes(deviceID, routes)
	if err != nil {
		return nil, err
	}

	// 更新本地缓存
	_, _ = s.SyncDevices()
	return res, nil
}

// CreateAuthKey 生成新 Auth Key
func (s *TailscaleService) CreateAuthKey(reusable, ephemeral, preauth bool, tags []string, expiryDays int, purpose, createdBy string) (*tsclient.CreateKeyResponse, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}

	res, err := cli.CreateAuthKey(reusable, ephemeral, preauth, tags, expiryDays)
	if err != nil {
		return nil, err
	}

	// 记录本地 Key 审计日志
	if s.db != nil {
		tagsJSON, _ := json.Marshal(tags)
		s.db.Create(&tsmodel.TsKeyLog{
			KeyID:         res.ID,
			AuthKey:       res.Key,
			Reusable:      reusable,
			Ephemeral:     ephemeral,
			Preauthorized: preauth,
			Tags:          string(tagsJSON),
			Purpose:       purpose,
			CreatedBy:     createdBy,
			ExpiresAt:     res.Expires,
		})
	}

	return res, nil
}

// ListKeyLogs 获取本地生成的 Auth Key 历史
func (s *TailscaleService) ListKeyLogs() ([]tsmodel.TsKeyLog, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}
	var logs []tsmodel.TsKeyLog
	err := s.db.Order("created_at desc").Find(&logs).Error
	return logs, err
}

// RevokeAuthKey 撤销 Auth Key 并从本地数据库同步清除记录
func (s *TailscaleService) RevokeAuthKey(keyID string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}

	// 1. 调用 Tailscale 官方 API 撤销
	if err := cli.RevokeAuthKey(keyID); err != nil {
		log.Printf("[Plugin:tailscale] 调用官方 API 撤销 Key %s 失败或已在云端过期: %v", keyID, err)
	}

	// 2. 从本地 ts_key_log 记录中彻底删除 (无论云端是否存在都清理本地)
	if s.db != nil {
		s.db.Unscoped().Where("key_id = ?", keyID).Delete(&tsmodel.TsKeyLog{})
	}
	return nil
}

// GetACL 获取当前 Policy ACL 策略
func (s *TailscaleService) GetACL() (string, error) {
	cli, err := s.getClient()
	if err != nil {
		return "", err
	}
	return cli.GetACL()
}

// UpdateACL 更新 Policy ACL 策略
func (s *TailscaleService) UpdateACL(hujson string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}
	return cli.UpdateACL(hujson)
}

// ValidateACL 校验 Policy ACL 策略格式
func (s *TailscaleService) ValidateACL(hujson string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}
	return cli.ValidateACL(hujson)
}

// DiagnoseDevice 对内网节点进行连通性与 IoT 协议端口诊断
func (s *TailscaleService) DiagnoseDevice(target string, port int, protocol string) (map[string]interface{}, error) {
	target = strings.TrimSpace(target)
	if target == "" {
		return nil, fmt.Errorf("目标地址不能为空")
	}
	if port <= 0 || port > 65535 {
		port = 80
	}

	address := fmt.Sprintf("%s:%d", target, port)
	start := time.Now()

	var conn net.Conn
	var err error
	channel := "本地直连 (Direct Network)"

	// 1. 优先尝试 tsnet 内存 WireGuard 原生拨号 (零外部代理依赖)
	tsnetMgr := tsproxy.GetTsnetManager()
	if tsnetMgr.IsRunning() {
		channel = "tsnet 嵌入式 WireGuard 节点直连"
		conn, err = tsnetMgr.DialTimeout("tcp", address, 3*time.Second)
	}

	// 2. 其次尝试配置的 SOCKS5 代理通道
	if conn == nil {
		cfg, _ := s.GetConfig()
		if cfg != nil && cfg.ProxyURL != "" {
			if u, parseErr := url.Parse(cfg.ProxyURL); parseErr == nil && strings.HasPrefix(strings.ToLower(u.Scheme), "socks5") {
				channel = fmt.Sprintf("SOCKS5 代理通道 (%s)", u.Host)
				dialer, dialerErr := proxy.FromURL(u, proxy.Direct)
				if dialerErr == nil {
					conn, err = dialer.Dial("tcp", address)
				}
			}
		}
	}

	// 3. 最后降级尝试宿主机物理网络直连
	if conn == nil && err == nil {
		conn, err = net.DialTimeout("tcp", address, 3*time.Second)
	}

	latency := time.Since(start).Milliseconds()

	result := map[string]interface{}{
		"target":      target,
		"port":        port,
		"protocol":    protocol,
		"address":     address,
		"latency_ms":  latency,
		"channel":     channel,
		"connected":   err == nil,
		"checked_at":  time.Now().Format("2006-01-02 15:04:05"),
		"preset_hint": getProtocolHint(port),
	}

	if err != nil {
		result["error"] = err.Error()
		result["status"] = "unreachable"
		if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "no route") {
			if !tsnetMgr.IsRunning() {
				result["tip"] = "提示：云端容器无虚拟网卡。您已进入方案二架构：请在 [连接与代理配置] 填入或生成一个 Auth Key，系统将自动激活嵌入式 tsnet 节点，彻底打通内网！"
			} else {
				result["tip"] = "提示：tsnet 节点已建立组网，但目标节点未响应对应端口。请检查目标设备服务是否开启，或防火墙/ACL是否放行。"
			}
		}
	} else {
		defer conn.Close()
		result["status"] = "reachable"
	}

	return result, nil
}

// getProtocolHint 获取常见物联与工控端口提示
func getProtocolHint(port int) string {
	switch port {
	case 502:
		return "Modbus-TCP 工业控制协议"
	case 554:
		return "RTSP / ONVIF 实时视频流协议"
	case 1883, 8883:
		return "MQTT 物联网消息总线"
	case 80, 8080:
		return "HTTP 局域网 Web 管理端"
	case 443, 8443:
		return "HTTPS 安全 Web 服务"
	case 22:
		return "SSH 远程终端维护"
	case 8000, 8001:
		return "海康威视 / 局域网服务管理端口"
	default:
		return "自定义应用服务端口"
	}
}

// ProcessWebhook 处理 Tailscale Webhook 事件
func (s *TailscaleService) ProcessWebhook(rawBody []byte, signatureHeader string) (*tsclient.WebhookEvent, error) {
	cfg, _ := s.GetConfig()
	if cfg != nil && cfg.WebhookSecret != "" {
		// 校验 Webhook 签名 (HMAC-SHA256)
		mac := hmac.New(sha256.New, []byte(cfg.WebhookSecret))
		mac.Write(rawBody)
		expectedSig := hex.EncodeToString(mac.Sum(nil))

		// Signature Header 可能包含 t=xxx,v1=xxx 或直接为 hex
		actualSig := signatureHeader
		if strings.Contains(signatureHeader, "v1=") {
			parts := strings.Split(signatureHeader, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if strings.HasPrefix(p, "v1=") {
					actualSig = strings.TrimPrefix(p, "v1=")
					break
				}
			}
		}

		if actualSig != "" && !hmac.Equal([]byte(expectedSig), []byte(actualSig)) {
			// 签名不匹配依然记录日志但返回错误
			if s.db != nil {
				s.db.Create(&tsmodel.TsWebhookLog{
					EventType: "auth_failed",
					Tailnet:   cfg.Tailnet,
					Message:   "Webhook HMAC 签名校验失败",
					RawBody:   string(rawBody),
				})
			}
			return nil, fmt.Errorf("Webhook 签名校验未通过")
		}
	}

	var event tsclient.WebhookEvent
	if err := json.Unmarshal(rawBody, &event); err != nil {
		return nil, fmt.Errorf("解析 Webhook JSON 载荷失败: %w", err)
	}

	// 记录 Webhook 审计日志
	if s.db != nil {
		s.db.Create(&tsmodel.TsWebhookLog{
			EventType: event.Type,
			Tailnet:   event.Tailnet,
			Message:   event.Message,
			RawBody:   string(rawBody),
		})
	}

	// 针对特定事件类型执行自愈与联动更新
	switch event.Type {
	case "nodeCreated", "nodeDeleted", "subnetRoutesChanged", "nodeNeedsApproval":
		go func() {
			_, _ = s.SyncDevices()
		}()
	}

	return &event, nil
}

// ListWebhookLogs 获取 Webhook 事件日志
func (s *TailscaleService) ListWebhookLogs(limit int) ([]tsmodel.TsWebhookLog, error) {
	if s.db == nil {
		return nil, fmt.Errorf("数据库连接不可用")
	}
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	var logs []tsmodel.TsWebhookLog
	err := s.db.Order("created_at desc").Limit(limit).Find(&logs).Error
	return logs, err
}
