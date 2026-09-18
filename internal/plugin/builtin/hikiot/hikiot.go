package hikiot

import (
	"log"

	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin"
	hikapi "apeadmin-gin/internal/plugin/builtin/hikiot/api"
	hikmcp "apeadmin-gin/internal/plugin/builtin/hikiot/mcp"
	hikmodel "apeadmin-gin/internal/plugin/builtin/hikiot/model"
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
	}
	log.Println("[Plugin:hikiot] 插件加载完成并在数据库登记")
	return nil
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

func init() {
	plugin.Register(&HikPlugin{})
}
