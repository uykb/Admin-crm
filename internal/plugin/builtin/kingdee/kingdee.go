package kingdee

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin"
	kdapi "apeadmin-gin/internal/plugin/builtin/kingdee/api"
	kdmcp "apeadmin-gin/internal/plugin/builtin/kingdee/mcp"
	kdmodel "apeadmin-gin/internal/plugin/builtin/kingdee/model"
	kdservice "apeadmin-gin/internal/plugin/builtin/kingdee/service"

	"gorm.io/gorm"
)

type KingdeePlugin struct{}

func (p *KingdeePlugin) Name() string {
	return "kingdee"
}

func (p *KingdeePlugin) DisplayName() string {
	return "金蝶云星空 ERP 内网集成与飞书审批"
}

func (p *KingdeePlugin) Description() string {
	return "金蝶云星空 K3 Cloud ERP 内网账套对接与飞书移动集成审批插件，支持单据同步、飞书动态表单审批、状态反写与 AI MCP 运维"
}

func (p *KingdeePlugin) Version() string {
	return "1.1.0"
}

func (p *KingdeePlugin) Author() string {
	return "ApeAdmin Enterprise Team"
}

func (p *KingdeePlugin) Dependencies() []string {
	return nil
}

func (p *KingdeePlugin) OnLoad() error {
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
				log.Printf("[Plugin:kingdee] 登记 sys_plugin 失败: %v", err)
			}
		}
		p.ensureMenu(db)
	}
	log.Println("[Plugin:kingdee] 金蝶云星空 ERP & 飞书审批插件加载完成并在数据库登记")
	return nil
}

// ensureMenu 幂等注入金蝶控制台菜单与按钮权限
func (p *KingdeePlugin) ensureMenu(db *gorm.DB) {
	if db == nil {
		return
	}
	var count int64
	db.Model(&model.SysMenu{}).Where("permission = ?", "kingdee:data:query").Count(&count)
	if count > 0 {
		return
	}

	// 1. 顶级目录: 金蝶云星空 ERP
	dir := model.SysMenu{
		Name: "金蝶云星空 ERP", ParentID: 0, Type: "M", Path: "/kingdee",
		Icon: "Box", Sort: 7, Visible: 1, Status: 1,
	}
	if err := db.Create(&dir).Error; err != nil {
		log.Printf("[Plugin:kingdee] 创建顶级菜单失败: %v", err)
		return
	}

	// 2. 子菜单: ERP 业务控制台
	child := model.SysMenu{
		Name: "ERP 业务控制台", ParentID: dir.ID, Type: "C", Path: "ui",
		Component: "kingdee/ui", Permission: "kingdee:data:query",
		Icon: "Goods", Sort: 1, Visible: 1, Status: 1,
	}
	if err := db.Create(&child).Error; err != nil {
		log.Printf("[Plugin:kingdee] 创建子菜单失败: %v", err)
		return
	}

	// 3. 按钮权限
	btns := []model.SysMenu{
		{Name: "单据数据查询", ParentID: child.ID, Type: "F", Permission: "kingdee:data:query", Sort: 1, Status: 1},
		{Name: "账套配置编辑", ParentID: child.ID, Type: "F", Permission: "kingdee:config:edit", Sort: 2, Status: 1},
	}
	for _, b := range btns {
		_ = db.Create(&b).Error
	}

	// 4. 绑定给超级管理员角色
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

	log.Println("[Plugin:kingdee] 金蝶控制台菜单已成功注入系统菜单树")
}

func (p *KingdeePlugin) Install() error {
	return nil
}

func (p *KingdeePlugin) Register(pr *plugin.PluginRouter) error {
	// 1. 自动迁移数据表
	if pr.DB != nil {
		if err := pr.DB.AutoMigrate(kdmodel.AllModels()...); err != nil {
			log.Printf("[Plugin:kingdee] 自动迁移表结构失败: %v", err)
		}
		// 启动后台异步任务 (轮询待审批单据与自动回写审核)
		p.startBackgroundTasks(pr.DB)
	}

	// 2. 挂载 HTTP 路由
	if pr.Authed != nil {
		handler := kdapi.NewKingdeeHandler(pr.DB)
		kdapi.SetupRoutes(pr.Authed, handler)
	}

	// 3. 注册 AI Agent MCP 工具
	if pr.MCP != nil {
		kdmcp.RegisterTools(pr.MCP, pr.DB)
	}

	log.Println("[Plugin:kingdee] HTTP 路由、飞书审批后台任务与 MCP 工具已就绪")
	return nil
}

func (p *KingdeePlugin) startBackgroundTasks(db *gorm.DB) {
	if db == nil {
		return
	}
	feishuSvc := kdservice.NewFeishuApprovalService(db)
	kdSvc := kdservice.NewKingdeeService(db)

	// 1. Audit Task: 每 1 分钟扫描 PENDING_AUDIT / AUDIT_FAILED 实例并回写金蝶
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			var insts []kdmodel.KdInstance
			db.Where("approve_status IN ? AND retry_count < 5", []string{kdmodel.InstanceStatusPendingAudit, kdmodel.InstanceStatusAuditFailed}).Find(&insts)
			for _, inst := range insts {
				_ = feishuSvc.AuditInstance(&inst)
			}
		}
	}()

	// 2. Poll Task: 每 3 分钟扫描金蝶待审批单据 (状态=B) 并自动发起飞书审批
	go func() {
		ticker := time.NewTicker(3 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			var flows []kdmodel.KdFlow
			db.Where("enabled = ? AND channel = 'feishu'", true).Find(&flows)
			for _, flow := range flows {
				filterStr := fmt.Sprintf("%s = '%s'", flow.StatusField, flow.PollFilterStatus)
				limit := flow.PollMaxBills
				if limit <= 0 {
					limit = 50
				}
				rows, err := kdSvc.ExecuteBillQuery(flow.KingdeeFormId, "FID,"+flow.BillNoField+","+flow.StatusField+",FCreatorId", filterStr, limit)
				if err != nil || len(rows) == 0 {
					continue
				}
				for _, row := range rows {
					if len(row) >= 3 {
						billMap := map[string]interface{}{
							"FID":            row[0],
							flow.BillNoField: row[1],
							flow.StatusField: row[2],
						}
						if len(row) >= 4 {
							billMap["FCreatorId"] = row[3]
						}
						_, _, _ = feishuSvc.StartFeishuApproval(&flow, billMap)
					}
				}
			}
		}
	}()
}

func (p *KingdeePlugin) Unregister() error {
	return nil
}

func (p *KingdeePlugin) Uninstall() error {
	return nil
}

func (p *KingdeePlugin) OnUnload() {}

func (p *KingdeePlugin) GetConfigJSON() (string, error) {
	db := core.GetDB()
	svc := kdservice.NewKingdeeService(db)
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

func (p *KingdeePlugin) OnConfigUpdate(configJSON string) error {
	var cfg kdservice.ConfigDTO
	if err := json.Unmarshal([]byte(configJSON), &cfg); err != nil {
		return err
	}
	db := core.GetDB()
	svc := kdservice.NewKingdeeService(db)
	return svc.SaveConfig(&cfg)
}

func (p *KingdeePlugin) TestConnection() error {
	db := core.GetDB()
	svc := kdservice.NewKingdeeService(db)
	return svc.TestConnection()
}

func init() {
	plugin.Register(&KingdeePlugin{})
}
