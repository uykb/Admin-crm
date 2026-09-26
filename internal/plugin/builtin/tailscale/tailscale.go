package tailscale

import (
	"encoding/json"
	"log"

	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/core/netproxy"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin"
	tsapi "apeadmin-gin/internal/plugin/builtin/tailscale/api"
	tsmcp "apeadmin-gin/internal/plugin/builtin/tailscale/mcp"
	tsmodel "apeadmin-gin/internal/plugin/builtin/tailscale/model"
	tsproxy "apeadmin-gin/internal/plugin/builtin/tailscale/proxy"
	tsservice "apeadmin-gin/internal/plugin/builtin/tailscale/service"

	"gorm.io/gorm"
)

type TailscalePlugin struct{}

func (p *TailscalePlugin) Name() string {
	return "tailscale"
}

func (p *TailscalePlugin) DisplayName() string {
	return "Tailscale 虚拟组网与设备管控"
}

func (p *TailscalePlugin) Description() string {
	return "Tailscale 虚拟私有网络 (Tailnet) 节点设备、Auth Key 密钥与子网路由集成插件"
}

func (p *TailscalePlugin) Version() string {
	return "1.0.0"
}

func (p *TailscalePlugin) Author() string {
	return "ApeAdmin AI Team"
}

func (p *TailscalePlugin) Dependencies() []string {
	return nil
}

func (p *TailscalePlugin) OnLoad() error {
	db := core.GetDB()
	if db != nil {
		existing, err := dal.GetPluginByName(p.Name())
		if err == nil && existing != nil {
			if existing.Version != p.Version() || existing.DisplayName != p.DisplayName() {
				existing.DisplayName = p.DisplayName()
				existing.Description = p.Description()
				existing.Version = p.Version()
				existing.Author = p.Author()
				existing.Enabled = true
				_ = dal.UpdatePlugin(existing)
			}
		} else {
			record := &model.SysPlugin{
				Name:        p.Name(),
				DisplayName: p.DisplayName(),
				Description: p.Description(),
				Version:     p.Version(),
				Author:      p.Author(),
				Enabled:     true,
				ModulePath:  "(builtin)",
			}
			if err := dal.CreatePlugin(record); err != nil {
				log.Printf("[Plugin:tailscale] 登记 sys_plugin 失败: %v", err)
			}
		}
		p.ensureMenu(db)

		// 注册内网透明 HTTP 代理解析器
		resolver := tsproxy.NewTailscaleResolver(db)
		netproxy.RegisterResolver(resolver)
		log.Println("[Plugin:tailscale] 内网透明 HTTP 代理解析器已成功注册至系统底座")
	}
	log.Println("[Plugin:tailscale] 插件加载完成并在数据库登记")
	return nil
}

// ensureMenu 幂等注入 Tailscale 控制台菜单及按钮权限
func (p *TailscalePlugin) ensureMenu(db *gorm.DB) {
	if db == nil {
		return
	}
	var count int64
	db.Model(&model.SysMenu{}).Where("permission = ?", "tailscale:device:list").Count(&count)
	if count > 0 {
		return
	}

	// 1. Tailscale 组网 顶级目录 (M)
	dir := model.SysMenu{
		Name: "Tailscale 组网", ParentID: 0, Type: "M", Path: "/tailscale",
		Icon: "Connection", Sort: 6, Visible: 1, Status: 1,
	}
	if err := db.Create(&dir).Error; err != nil {
		log.Printf("[Plugin:tailscale] 创建顶级菜单失败: %v", err)
		return
	}

	// 2. 节点与密钥控制台 子菜单 (C)
	child := model.SysMenu{
		Name: "节点与密钥控制台", ParentID: dir.ID, Type: "C", Path: "ui",
		Component: "tailscale/ui", Permission: "tailscale:device:list",
		Icon: "Cpu", Sort: 1, Visible: 1, Status: 1,
	}
	if err := db.Create(&child).Error; err != nil {
		log.Printf("[Plugin:tailscale] 创建子菜单失败: %v", err)
		return
	}

	// 3. 按钮权限 (F)
	btns := []model.SysMenu{
		{Name: "设备查看", ParentID: child.ID, Type: "F", Permission: "tailscale:device:list", Sort: 1, Status: 1},
		{Name: "设备管控", ParentID: child.ID, Type: "F", Permission: "tailscale:device:control", Sort: 2, Status: 1},
		{Name: "生成AuthKey", ParentID: child.ID, Type: "F", Permission: "tailscale:key:create", Sort: 3, Status: 1},
		{Name: "配置管理", ParentID: child.ID, Type: "F", Permission: "tailscale:config:edit", Sort: 4, Status: 1},
		{Name: "ACL策略管理", ParentID: child.ID, Type: "F", Permission: "tailscale:acl:edit", Sort: 5, Status: 1},
	}
	for _, b := range btns {
		_ = db.Create(&b).Error
	}

	// 4. 超管角色绑定
	var adminRole model.SysRole
	if err := db.Where("code = ?", "admin").First(&adminRole).Error; err == nil {
		var menus []model.SysMenu
		db.Model(&adminRole).Association("Menus").Find(&menus)
		menus = append(menus, dir, child)
		for _, b := range btns {
			menus = append(menus, b)
		}
		_ = db.Model(&adminRole).Association("Menus").Replace(&menus)
	}

	log.Println("[Plugin:tailscale] 菜单已成功注入系统菜单树")
}

func (p *TailscalePlugin) Install() error {
	return nil
}

func (p *TailscalePlugin) Register(pr *plugin.PluginRouter) error {
	// 1. 自动迁移 Tailscale 数据库模型
	if pr.DB != nil {
		if err := pr.DB.AutoMigrate(tsmodel.AllModels()...); err != nil {
			log.Printf("[Plugin:tailscale] 自动迁移表结构失败: %v", err)
		}
	}

	// 2. 挂载 HTTP 路由 (包含公开 Webhook 端点与鉴权管理接口)
	handler := tsapi.NewTailscaleHandler(pr.DB)
	tsapi.SetupRoutes(pr.Authed, pr.Public, handler)

	// 3. 注册 AI Agent MCP 工具
	if pr.MCP != nil {
		tsmcp.RegisterTools(pr.MCP, pr.DB)
	}

	log.Println("[Plugin:tailscale] HTTP 路由与 MCP 工具已就绪")
	return nil
}

func (p *TailscalePlugin) Unregister() error {
	netproxy.UnregisterResolver("tailscale")
	log.Println("[Plugin:tailscale] 内网透明 HTTP 代理解析器已从系统底座注销")
	return nil
}

func (p *TailscalePlugin) Uninstall() error {
	return nil
}

func (p *TailscalePlugin) OnUnload() {
	netproxy.UnregisterResolver("tailscale")
}

type tailscaleConfigExport struct {
	Tailnet       string `json:"tailnet"`
	APIKey        string `json:"api_key"`
	ClientID      string `json:"client_id"`
	ClientSecret  string `json:"client_secret"`
	WebhookSecret string `json:"webhook_secret"`
	ProxyURL      string `json:"proxy_url"`
}

func (p *TailscalePlugin) GetConfigJSON() (string, error) {
	db := core.GetDB()
	svc := tsservice.NewTailscaleService(db)
	cfg, err := svc.GetConfig()
	if err != nil {
		return "", err
	}
	exp := tailscaleConfigExport{
		Tailnet:       cfg.Tailnet,
		APIKey:        cfg.APIKey,
		ClientID:      cfg.ClientID,
		ClientSecret:  cfg.ClientSecret,
		WebhookSecret: cfg.WebhookSecret,
		ProxyURL:      cfg.ProxyURL,
	}
	b, err := json.MarshalIndent(exp, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *TailscalePlugin) OnConfigUpdate(configJSON string) error {
	var m map[string]string
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return err
	}
	tailnet := m["tailnet"]
	if tailnet == "" {
		tailnet = m["tailscale_tailnet"]
	}
	apiKey := m["api_key"]
	if apiKey == "" {
		apiKey = m["tailscale_api_key"]
	}
	clientID := m["client_id"]
	if clientID == "" {
		clientID = m["tailscale_client_id"]
	}
	clientSecret := m["client_secret"]
	if clientSecret == "" {
		clientSecret = m["tailscale_client_secret"]
	}
	webhookSecret := m["webhook_secret"]
	if webhookSecret == "" {
		webhookSecret = m["tailscale_webhook_secret"]
	}
	proxyURL := m["proxy_url"]
	if proxyURL == "" {
		proxyURL = m["tailscale_proxy_url"]
	}

	db := core.GetDB()
	svc := tsservice.NewTailscaleService(db)
	if err := svc.SaveConfig(tailnet, apiKey, clientID, clientSecret, webhookSecret, proxyURL); err != nil {
		return err
	}
	if db != nil && proxyURL != "" {
		// 刷新代理缓存
		resolver := tsproxy.NewTailscaleResolver(db)
		resolver.RefreshConfig()
	}
	return nil
}

func (p *TailscalePlugin) TestConnection() error {
	db := core.GetDB()
	svc := tsservice.NewTailscaleService(db)
	_, err := svc.SyncDevices()
	return err
}

func init() {
	plugin.Register(&TailscalePlugin{})
}
