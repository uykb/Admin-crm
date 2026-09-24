package hikiot

import (
	"encoding/json"
	"log"

	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin"
	"apeadmin-gin/internal/attendance"
	hikapi "apeadmin-gin/internal/plugin/builtin/hikiot/api"
	hikmcp "apeadmin-gin/internal/plugin/builtin/hikiot/mcp"
	hikmodel "apeadmin-gin/internal/plugin/builtin/hikiot/model"
	hikservice "apeadmin-gin/internal/plugin/builtin/hikiot/service"

	"github.com/robfig/cron/v3"
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
	db.Model(&model.SysMenu{}).Where("name = ?", "门禁设备管控").Count(&count)
	if count > 0 {
		// 强制更新已有菜单的 Component 字段
		db.Model(&model.SysMenu{}).Where("name = ?", "门禁设备管控").Update("component", "hikiot/ui_doors")
		db.Model(&model.SysMenu{}).Where("name = ?", "打卡考勤记录").Update("component", "hikiot/ui_records")
		db.Model(&model.SysMenu{}).Where("name = ?", "排班考勤汇总").Update("component", "hikiot/ui_matrix")
		return // 已经是最新的菜单结构，跳过
	}

	// 0. 清理旧版统一控制台菜单
	db.Where("name = ?", "门禁考勤控制台").Delete(&model.SysMenu{})

	// 1. 获取或创建 海康互联 顶级目录 (M)
	var dir model.SysMenu
	err := db.Where("name = ? AND parent_id = 0", "海康互联").First(&dir).Error
	if err != nil {
		dir = model.SysMenu{
			Name: "海康互联", ParentID: 0, Type: "M", Path: "/hikiot",
			Icon: "VideoCamera", Sort: 5, Visible: 1, Status: 1,
		}
		db.Create(&dir)
	}

	// 2. 门禁设备管控
	child1 := model.SysMenu{
		Name: "门禁设备管控", ParentID: dir.ID, Type: "C", Path: "doors",
		Component: "hikiot/ui_doors", Permission: "hikiot:door:list",
		Icon: "Lock", Sort: 1, Visible: 1, Status: 1,
	}
	db.Create(&child1)

	// 3. 打卡考勤记录
	child2 := model.SysMenu{
		Name: "打卡考勤记录", ParentID: dir.ID, Type: "C", Path: "records",
		Component: "hikiot/ui_records", Permission: "hikiot:attendance:list",
		Icon: "Clock", Sort: 2, Visible: 1, Status: 1,
	}
	db.Create(&child2)

	// 4. 排班考勤汇总
	child3 := model.SysMenu{
		Name: "排班考勤汇总", ParentID: dir.ID, Type: "C", Path: "matrix",
		Component: "hikiot/ui_matrix", Permission: "hikiot:matrix:list",
		Icon: "DataBoard", Sort: 3, Visible: 1, Status: 1,
	}
	db.Create(&child3)

	// 5. 按钮权限 (F)
	btns := []model.SysMenu{
		{Name: "控门操作", ParentID: child1.ID, Type: "F", Permission: "hikiot:door:control", Sort: 1, Status: 1},
		{Name: "配置管理", ParentID: child1.ID, Type: "F", Permission: "hikiot:config:edit", Sort: 2, Status: 1},
	}
	for _, b := range btns {
		db.Create(&b)
	}

	// 6. 超管角色绑定
	var adminRole model.SysRole
	if err := db.Where("code = ?", "admin").First(&adminRole).Error; err == nil {
		var menus []model.SysMenu
		db.Model(&adminRole).Association("Menus").Find(&menus)
		menus = append(menus, child1, child2, child3)
		for _, b := range btns {
			menus = append(menus, b)
		}
		// 如果 dir 也是新的，追加进去
		if err != nil {
			menus = append(menus, dir)
		}
		_ = db.Model(&adminRole).Association("Menus").Replace(&menus)
	}

	log.Println("[Plugin:hikiot] 菜单已成功升级并注入系统菜单树")
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
		pr.Public.GET("/hikiot/test-persons", handler.TestPersons)
		// 暴露海康 Event 推送接收端（公有路由）
		pr.Public.POST("/hikiot/event/callback", attendance.HandleHikiotEvent(pr.DB))
	}

	// 3. 注册定时任务：每天上午 09:00 执行一次错峰查漏补缺
	c := cron.New()
	_, err := c.AddFunc("0 9 * * *", func() {
		// 每次执行时都获取最新的 client
		svc := hikservice.NewHikService(pr.DB)
		cli, err := svc.GetClient()
		if err == nil {
			_ = attendance.DailyReconciliation(pr.DB, cli)
		} else {
			log.Printf("[Plugin:hikiot] 获取 API Client 失败，无法执行 DailyReconciliation: %v", err)
		}
	})
	if err != nil {
		log.Printf("[Plugin:hikiot] 注册考勤 Cron 任务失败: %v", err)
	} else {
		c.Start()
		log.Println("[Plugin:hikiot] 已启动考勤定时任务，每天 09:00 执行核算")
	}

	// 4. 注册 AI Agent MCP 工具
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
	userToken := m["user_token"]
	if userToken == "" {
		userToken = m["hikiot_user_token"]
	}

	db := core.GetDB()
	svc := hikservice.NewHikService(db)
	return svc.SaveConfig(baseURL, appKey, appSecret, userToken)
}

func (p *HikPlugin) TestConnection() error {
	db := core.GetDB()
	svc := hikservice.NewHikService(db)
	return svc.TestConnection()
}

func init() {
	plugin.Register(&HikPlugin{})
}
