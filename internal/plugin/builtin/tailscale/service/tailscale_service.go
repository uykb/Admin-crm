package service

import (
	"encoding/json"
	"fmt"

	tsclient "apeadmin-gin/internal/plugin/builtin/tailscale/client"
	tsmodel "apeadmin-gin/internal/plugin/builtin/tailscale/model"

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
	Tailnet      string `json:"tailnet"`
	APIKey       string `json:"api_key"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
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

	return &ConfigDTO{
		Tailnet:      cfgMap["tailscale_tailnet"],
		APIKey:       cfgMap["tailscale_api_key"],
		ClientID:     cfgMap["tailscale_client_id"],
		ClientSecret: cfgMap["tailscale_client_secret"],
	}, nil
}

// SaveConfig 保存连接配置
func (s *TailscaleService) SaveConfig(tailnet, apiKey, clientID, clientSecret string) error {
	if s.db == nil {
		return fmt.Errorf("数据库连接不可用")
	}

	items := map[string]string{
		"tailscale_tailnet":       tailnet,
		"tailscale_api_key":       apiKey,
		"tailscale_client_id":     clientID,
		"tailscale_client_secret": clientSecret,
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
			record.Online = dev.Online
			record.KeyExpiryDisabled = dev.KeyExpiryDisabled
			record.Tags = string(tagsJSON)
			record.SubnetRoutes = string(routesJSON)
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
				Online:            dev.Online,
				KeyExpiryDisabled: dev.KeyExpiryDisabled,
				Tags:              string(tagsJSON),
				SubnetRoutes:      string(routesJSON),
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
		query = query.Where("name LIKE ? OR hostname LIKE ? OR ips LIKE ? OR user LIKE ?",
			likePattern, likePattern, likePattern, likePattern)
	}

	if err := query.Find(&devices).Error; err != nil {
		return nil, err
	}

	// 如果数据库为空，自动触发一次同步
	if len(devices) == 0 && keyword == "" {
		_, _ = s.SyncDevices()
		s.db.Order("online desc, last_seen desc").Find(&devices)
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

// RevokeAuthKey 撤销 Auth Key
func (s *TailscaleService) RevokeAuthKey(keyID string) error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}
	return cli.RevokeAuthKey(keyID)
}

// GetACL 获取当前 Policy ACL 策略
func (s *TailscaleService) GetACL() (string, error) {
	cli, err := s.getClient()
	if err != nil {
		return "", err
	}
	return cli.GetACL()
}
