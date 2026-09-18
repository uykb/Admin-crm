// ═══════════════════════════════════════════════════════════════════════════
// Plugin 包示例：Dev Example 备忘录插件（L1 编译期内置插件）
// ═══════════════════════════════════════════════════════════════════════════
//
// 【本插件演示内容】
// 1. 完整 CRUD：GET/POST/PUT/DELETE /dev-example/notes
// 2. 权限控制：dev_example:notes:create/edit/delete 三个按钮权限
// 3. 建表：OnLoad 中 AutoMigrate 自己的表（插件自治，不依赖主程序 allModels）
// 4. 菜单注入：OnLoad 中幂等创建菜单（前端页面 frontend/src/views/dev_example/notes/index.vue）
//
// 【前端配套】
//   - 页面：frontend/src/views/dev_example/notes/index.vue
//   - API ：frontend/src/api/index.ts 中的 getDevExampleNotes 等
//   - 菜单 path=dev-example/notes，component=dev_example/notes
// ═══════════════════════════════════════════════════════════════════════════

package dev_example

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/middleware"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/pkg/response"
	"apeadmin-gin/internal/plugin"
)

// DevExamplePlugin 备忘录示例插件
type DevExamplePlugin struct{}

// ─── 元数据方法（6 个）───

// Name 插件唯一标识
func (p *DevExamplePlugin) Name() string { return "dev_example" }

// DisplayName 显示名称
func (p *DevExamplePlugin) DisplayName() string { return "插件开发示例" }

// Description 描述信息
func (p *DevExamplePlugin) Description() string {
	return "插件开发示例（备忘录 CRUD），演示 L1 插件如何建表、注册路由、接入权限与注入菜单"
}

// Version 版本号
func (p *DevExamplePlugin) Version() string { return "1.0.0" }

// Author 作者信息
func (p *DevExamplePlugin) Author() string { return "ApeAdmin Team" }

// Dependencies 依赖（无）
func (p *DevExamplePlugin) Dependencies() []string { return nil }

// ─── 生命周期方法（6 个）───

// OnLoad 加载时调用：AutoMigrate 建表 + 登记 sys_plugin + 注入菜单
func (p *DevExamplePlugin) OnLoad() error {
	db := core.GetDB()
	if db == nil {
		log.Printf("[dev_example] 数据库未初始化，跳过建表")
		return nil
	}

	// 1. 自建表（幂等，AutoMigrate 只会补列不会删列）
	if err := db.AutoMigrate(&model.DevExampleNote{}); err != nil {
		log.Printf("[dev_example] 建表失败: %v", err)
		return err
	}

	// 2. 登记 sys_plugin（让插件管理页能看到）
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
			log.Printf("[dev_example] 登记 sys_plugin 失败: %v（不影响插件运行）", err)
		}
	}

	// 3. 注入菜单（幂等）：顶级菜单 dev-example + 子菜单 notes
	p.ensureMenu(db)

	log.Printf("[dev_example] 插件已加载")
	return nil
}

// ensureMenu 幂等创建菜单：顶级目录 + 备忘录子菜单 + 按钮权限
func (p *DevExamplePlugin) ensureMenu(db *gorm.DB) {
	// 已存在则跳过
	var count int64
	db.Model(&model.SysMenu{}).Where("component = ?", "dev_example/notes").Count(&count)
	if count > 0 {
		return
	}

	// 顶级目录（M）
	dir := model.SysMenu{
		Name: "插件开发示例", ParentID: 0, Type: "M", Path: "/dev-example",
		Icon: "MagicStick", Sort: 99, Visible: 1, Status: 1,
	}
	if err := db.Create(&dir).Error; err != nil {
		log.Printf("[dev_example] 创建顶级菜单失败: %v", err)
		return
	}

	// 子菜单（C）
	child := model.SysMenu{
		Name: "备忘录", ParentID: dir.ID, Type: "C", Path: "notes",
		Component: "dev_example/notes", Permission: "dev_example:notes:list",
		Icon: "Document", Sort: 1, Visible: 1, Status: 1,
	}
	if err := db.Create(&child).Error; err != nil {
		log.Printf("[dev_example] 创建子菜单失败: %v", err)
		return
	}

	// 按钮权限（F）
	for _, b := range []model.SysMenu{
		{Name: "备忘录新增", ParentID: child.ID, Type: "F", Permission: "dev_example:notes:create", Sort: 1, Status: 1},
		{Name: "备忘录编辑", ParentID: child.ID, Type: "F", Permission: "dev_example:notes:edit", Sort: 2, Status: 1},
		{Name: "备忘录删除", ParentID: child.ID, Type: "F", Permission: "dev_example:notes:delete", Sort: 3, Status: 1},
	} {
		if err := db.Create(&b).Error; err != nil {
			log.Printf("[dev_example] 创建按钮权限失败: %v", err)
		}
	}

	// 超管角色补挂菜单（幂等）
	var adminRole model.SysRole
	if err := db.Where("code = ?", "admin").First(&adminRole).Error; err == nil {
		var menus []model.SysMenu
		db.Model(&adminRole).Association("Menus").Find(&menus)
		menus = append(menus, dir, child)
		_ = db.Model(&adminRole).Association("Menus").Replace(&menus)
	}

	log.Printf("[dev_example] 菜单已注入：插件开发 → 备忘录")
}

// Install 预留
func (p *DevExamplePlugin) Install() error { return nil }

// Register 注册 HTTP 路由
// 路径前缀自动加 /api/v1，最终为 /api/v1/dev-example/notes
func (p *DevExamplePlugin) Register(r *plugin.PluginRouter) error {
	g := r.Authed.Group("/dev-example")
	{
		// 列表（需登录，权限由中间件控制）
		g.GET("/notes", middleware.RequirePermission("dev_example:notes:list"), p.listNotes)
		// 创建
		g.POST("/notes", middleware.RequirePermission("dev_example:notes:create"), p.createNote)
		// 更新
		g.PUT("/notes/:id", middleware.RequirePermission("dev_example:notes:edit"), p.updateNote)
		// 删除
		g.DELETE("/notes/:id", middleware.RequirePermission("dev_example:notes:delete"), p.deleteNote)
	}
	log.Printf("[dev_example] 路由已注册: /api/v1/dev-example/notes")
	return nil
}

// Unregister 预留
func (p *DevExamplePlugin) Unregister() error { return nil }

// Uninstall 预留
func (p *DevExamplePlugin) Uninstall() error { return nil }

// OnUnload 预留
func (p *DevExamplePlugin) OnUnload() {}

// ─── 业务 Handler ───

// listNotes 分页列表（GET /dev-example/notes）
func (p *DevExamplePlugin) listNotes(c *gin.Context) {
	db := core.GetDB()
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	var total int64
	db.Model(&model.DevExampleNote{}).Count(&total)

	var notes []model.DevExampleNote
	err := db.Order("id DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&notes).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}

	items := make([]map[string]interface{}, 0, len(notes))
	for _, n := range notes {
		items = append(items, map[string]interface{}{
			"id": n.ID, "title": n.Title, "content": n.Content,
			"priority": n.Priority, "completed": n.Completed,
			"created_at": n.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(http.StatusOK, response.SuccessPage(items, total, page, pageSize))
}

// createNote 创建（POST /dev-example/notes）
func (p *DevExamplePlugin) createNote(c *gin.Context) {
	var req struct {
		Title    string `json:"title" binding:"required"`
		Content  string `json:"content"`
		Priority int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if req.Priority < 0 || req.Priority > 2 {
		req.Priority = 0
	}
	note := model.DevExampleNote{Title: req.Title, Content: req.Content, Priority: req.Priority}
	if err := core.GetDB().Create(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"id": note.ID}))
}

// updateNote 更新（PUT /dev-example/notes/:id）
func (p *DevExamplePlugin) updateNote(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	db := core.GetDB()
	var note model.DevExampleNote
	if err := db.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "备忘录不存在"))
		return
	}
	var req struct {
		Title     string `json:"title"`
		Content   string `json:"content"`
		Priority  int    `json:"priority"`
		Completed bool   `json:"completed"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	updates := map[string]interface{}{}
	if req.Title != "" {
		updates["title"] = req.Title
	}
	updates["content"] = req.Content
	if req.Priority >= 0 && req.Priority <= 2 {
		updates["priority"] = req.Priority
	}
	updates["completed"] = req.Completed
	if err := db.Model(&note).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

// deleteNote 删除（DELETE /dev-example/notes/:id）
func (p *DevExamplePlugin) deleteNote(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	db := core.GetDB()
	var note model.DevExampleNote
	if err := db.First(&note, id).Error; err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "备忘录不存在"))
		return
	}
	if err := db.Delete(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// ─── init() 自动注册 ───

// init 注册到全局注册表（bootstrap/app.go 已 blank import 此包）
func init() {
	plugin.Register(&DevExamplePlugin{})
}