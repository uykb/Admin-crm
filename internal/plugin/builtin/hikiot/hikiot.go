package hikiot

import (
	"encoding/json"
	"log"

	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin"
	hikapi "apeadmin-gin/internal/plugin/builtin/hikiot/api"
	hikmcp "apeadmin-gin/internal/plugin/builtin/hikiot/mcp"
	hikmodel "apeadmin-gin/internal/plugin/builtin/hikiot/model"
	hikservice "apeadmin-gin/internal/plugin/builtin/hikiot/service"

	"gorm.io/gorm"
)

type HikPlugin struct{}

func (p *HikPlugin) Name() string {
	return "hikiot"
}

func (p *HikPlugin) DisplayName() string {
	return "海康互联 (Hik-Connect) 门禁考勤"
}

func (p *HikPlugin) Description() string {
	return "海康威视开放平台门禁点控制、考勤打卡数据与人员组织结构对接插件"
}

func (p *HikPlugin) Version() string {
	return "1.0.0"
}

func (p *HikPlugin) Author() string {
	return "ApeAdmin AI Team"
}

func (p *HikPlugin) Dependencies() []string {
	return nil
}

func (p *HikPlugin) OnLoad() error {
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
				log.Printf("[Plugin:hikiot] 登记 sys_plugin 失败: %v", err)
			}
		}
		p.ensureMenu(db)
	}
	log.Println("[Plugin:hikiot] 插件加载完成并在数据库登记")
	return nil
}

// ensureMenu 幂等注入海康互联控制台菜单及按钮权限
func (p *HikPlugin) ensureMenu(db *gorm.DB) {
	if db == nil {
		return
	}
	var count int64
	db.Model(&model.SysMenu{}).Where("permission = ?", "hikiot:door:list").Count(&count)
	if count > 0 {
		return
	}

	// 1. 海康互联 顶级目录 (M)
	dir := model.SysMenu{
		Name: "海康互联", ParentID: 0, Type: "M", Path: "/hikiot",
		Icon: "VideoCamera", Sort: 5, Visible: 1, Status: 1,
	}
	if err := db.Create(&dir).Error; err != nil {
		log.Printf("[Plugin:hikiot] 创建顶级菜单失败: %v", err)
		return
	}

	// 2. 管控中心 子菜单 (C)
	child := model.SysMenu{
		Name: "门禁考勤控制台", ParentID: dir.ID, Type: "C", Path: "ui",
		Component: "hikiot/ui", Permission: "hikiot:door:list",
		Icon: "Key", Sort: 1, Visible: 1, Status: 1,
	}
	if err := db.Create(&child).Error; err != nil {
		log.Printf("[Plugin:hikiot] 创建子菜单失败: %v", err)
		return
	}

	// 3. 按钮权限 (F)
	btns := []model.SysMenu{
		{Name: "控门操作", ParentID: child.ID, Type: "F", Permission: "hikiot:door:control", Sort: 1, Status: 1},
		{Name: "考勤查看", ParentID: child.ID, Type: "F", Permission: "hikiot:attendance:list", Sort: 2, Status: 1},
		{Name: "配置管理", ParentID: child.ID, Type: "F", Permission: "hikiot:config:edit", Sort: 3, Status: 1},
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

	log.Println("[Plugin:hikiot] 菜单已成功注入系统菜单树")
}

func (p *HikPlugin) Install() error {
	return nil
}

func (p *HikPlugin) Register(pr *plugin.PluginRouter) error {
	// 1. 自动迁移海康数据库模型
	if pr.DB != nil {
		if err := pr.DB.AutoMigrate(hikmodel.AllModels()...); err != nil {
			log.Printf("[Plugin:hikiot] 自动迁移表结构失败: %v", err)
		}
	}

	// 2. 挂载 HTTP 路由
	if pr.Authed != nil {
		handler := hikapi.NewHikHandler(pr.DB)
		hikapi.SetupRoutes(pr.Authed, handler)
	}
	if pr.Public != nil {
		handler := hikapi.NewHikHandler(pr.DB)
		pr.Public.GET("/hikiot/debug", handler.DebugHikiot)
	}

	// 3. 注册 AI Agent MCP 工具
	if pr.MCP != nil {
		hikmcp.RegisterTools(pr.MCP, pr.DB)
	}

	log.Println("[Plugin:hikiot] HTTP 路由与 MCP 工具已就绪")
	return nil
}

func (p *HikPlugin) Unregister() error {
	return nil
}

func (p *HikPlugin) Uninstall() error {
	return nil
}

func (p *HikPlugin) OnUnload() {
	log.Println("[Plugin:hikiot] 插件已卸载")
}

func (p *HikPlugin) GetConfigJSON() (string, error) {
	db := core.GetDB()
	svc := hikservice.NewHikService(db)
	cfg, err := svc.GetConfig()
	if err != nil {
		return "", err
	}
	b, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (p *HikPlugin) OnConfigUpdate(configJSON string) error {
	var m map[string]string
	if err := json.Unmarshal([]byte(configJSON), &m); err != nil {
		return err
	}
	baseURL := m["base_url"]
	if baseURL == "" {
		baseURL = m["hikiot_base_url"]
	}
	appKey := m["app_key"]
	if appKey == "" {
		appKey = m["hikiot_app_key"]
	}
	appSecret := m["app_secret"]
	if appSecret == "" {
		appSecret = m["hikiot_app_secret"]
	}

	db := core.GetDB()
	svc := hikservice.NewHikService(db)
	return svc.SaveConfig(baseURL, appKey, appSecret)
}

func (p *HikPlugin) TestConnection() error {
	db := core.GetDB()
	svc := hikservice.NewHikService(db)
	return svc.TestConnection()
}

func init() {
	plugin.Register(&HikPlugin{})
}
