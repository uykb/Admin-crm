package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/pkg/response"
)

// isImageMagic 通过文件头魔数判断是否为常见图片格式
func isImageMagic(b []byte) bool {
	if len(b) < 12 {
		return false
	}
	// PNG: 89 50 4E 47 0D 0A 1A 0A
	if b[0] == 0x89 && b[1] == 0x50 && b[2] == 0x4E && b[3] == 0x47 {
		return true
	}
	// JPEG: FF D8 FF
	if b[0] == 0xFF && b[1] == 0xD8 && b[2] == 0xFF {
		return true
	}
	// GIF: 47 49 46 38 ('GIF8')
	if b[0] == 0x47 && b[1] == 0x49 && b[2] == 0x46 && b[3] == 0x38 {
		return true
	}
	// WebP: 'RIFF' .... 'WEBP'（第 8-11 字节）
	if b[0] == 0x52 && b[1] == 0x49 && b[2] == 0x46 && b[3] == 0x46 &&
		b[8] == 0x57 && b[9] == 0x45 && b[10] == 0x42 && b[11] == 0x50 {
		return true
	}
	// ICO: 00 00 01 00
	if b[0] == 0x00 && b[1] == 0x00 && b[2] == 0x01 && b[3] == 0x00 {
		return true
	}
	return false
}

type RoleHandler struct{}

func (h *RoleHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	roles, total, err := dal.ListRoles(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessPage(roles, total, page, pageSize))
}

func (h *RoleHandler) ListAll(c *gin.Context) {
	roles, err := dal.ListAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(roles))
}

func (h *RoleHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	role, err := dal.GetRoleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "角色不存在"))
		return
	}
	c.JSON(http.StatusOK, response.Success(role))
}

func (h *RoleHandler) Create(c *gin.Context) {
	var req struct {
		Name      string `json:"name" binding:"required"`
		Code      string `json:"code" binding:"required"`
		DataScope int    `json:"data_scope"`
		Sort      int    `json:"sort"`
		Remark    string `json:"remark"`
		MenuIDs   []uint `json:"menu_ids"`
		Status    int    `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	role := model.SysRole{
		Name:      req.Name,
		Code:      req.Code,
		DataScope: req.DataScope,
		Sort:      req.Sort,
		Remark:    req.Remark,
		Status:    req.Status,
	}
	if role.Status == 0 { role.Status = 1 }
	if role.DataScope == 0 { role.DataScope = 1 }
	if err := dal.CreateRole(&role); err != nil {
		c.JSON(http.StatusConflict, response.Error(409, "角色编码已存在"))
		return
	}
	if len(req.MenuIDs) > 0 {
		dal.AssignRoleMenus(role.ID, req.MenuIDs)
	}
	c.JSON(http.StatusOK, response.Success(role))
}

func (h *RoleHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	role, err := dal.GetRoleByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "角色不存在"))
		return
	}
	var req struct {
		Name      string `json:"name"`
		DataScope int    `json:"data_scope"`
		Sort      int    `json:"sort"`
		Remark    string `json:"remark"`
		MenuIDs   []uint `json:"menu_ids"`
		Status    *int   `json:"status"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if req.Name != "" { role.Name = req.Name }
	if req.DataScope != 0 { role.DataScope = req.DataScope }
	role.Sort = req.Sort
	role.Remark = req.Remark
	if req.Status != nil { role.Status = *req.Status }
	if err := dal.UpdateRole(role); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	if req.MenuIDs != nil {
		dal.AssignRoleMenus(role.ID, req.MenuIDs)
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

func (h *RoleHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := dal.DeleteRole(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// ─── 菜单 ───

type MenuHandler struct{}

func (h *MenuHandler) Tree(c *gin.Context) {
	menus, err := dal.GetMenuTree()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(menus))
}

func (h *MenuHandler) Create(c *gin.Context) {
	var menu model.SysMenu
	if err := c.ShouldBindJSON(&menu); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if menu.Status == 0 { menu.Status = 1 }
	if menu.Visible == 0 { menu.Visible = 1 }
	if err := dal.CreateMenu(&menu); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(menu))
}

func (h *MenuHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var m model.SysMenu
	if err := c.ShouldBindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	m.ID = uint(id)
	if err := dal.UpdateMenu(&m); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

func (h *MenuHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := dal.DeleteMenu(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// ─── 部门 ───

type DeptHandler struct{}

func (h *DeptHandler) Tree(c *gin.Context) {
	depts, err := dal.GetAllDepts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(depts))
}

func (h *DeptHandler) Create(c *gin.Context) {
	var dept model.SysDept
	if err := c.ShouldBindJSON(&dept); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if dept.Status == 0 { dept.Status = 1 }
	if err := dal.CreateDept(&dept); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(dept))
}

func (h *DeptHandler) Update(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var dept model.SysDept
	if err := c.ShouldBindJSON(&dept); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	dept.ID = uint(id)
	if err := dal.UpdateDept(&dept); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

func (h *DeptHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := dal.DeleteDept(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// ─── 日志 ───

type LogHandler struct{}

func (h *LogHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	logs, total, err := dal.ListLogs(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessPage(logs, total, page, pageSize))
}

func (h *LogHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	log, err := dal.GetLogByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "日志不存在"))
		return
	}
	c.JSON(http.StatusOK, response.Success(log))
}

func (h *LogHandler) Clear(c *gin.Context) {
	if err := dal.ClearLogs(); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "清空失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("清空成功"))
}

func (h *LogHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	if err := dal.DeleteLog(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// ─── 系统设置 ───

type SettingHandler struct{}

func (h *SettingHandler) List(c *gin.Context) {
	settings, err := dal.ListSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(settings))
}

func (h *SettingHandler) BatchUpdate(c *gin.Context) {
	var req map[string]string
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := dal.BatchUpdateSettings(req); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

func (h *SettingHandler) Update(c *gin.Context) {
	key := c.Param("key")
	var req struct {
		Value string `json:"value" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := dal.UpdateSetting(key, req.Value); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

// BrandImage 品牌图片上传（Logo / 登录页背景），返回可访问的 URL
func (h *SettingHandler) BrandImage(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "配置未初始化"))
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少文件字段 file"))
		return
	}

	// 扩展名校验（仅图片；禁止 svg 防止同域存储型 XSS）
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowed := map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, ".ico": true}
	if !allowed[ext] {
		c.JSON(http.StatusBadRequest, response.Error(400, "仅支持图片格式（jpg/png/gif/webp/ico）"))
		return
	}

	// 大小校验（10MB）
	if file.Size > 10*1024*1024 {
		c.JSON(http.StatusBadRequest, response.Error(400, "图片大小不能超过 10MB"))
		return
	}

	// 魔数校验：防止伪造扩展名的非图片文件（PNG/JPEG/GIF/WebP/ICO）
	opened, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "读取文件失败"))
		return
	}
	defer opened.Close()
	head := make([]byte, 12)
	n, _ := io.ReadFull(opened, head)
	if n < 12 || !isImageMagic(head[:n]) {
		c.JSON(http.StatusBadRequest, response.Error(400, "文件内容不是有效图片"))
		return
	}

	// 保存到 uploads/brand/ 目录
	dir := filepath.Join(cfg.File.StorageDir, "brand")
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建目录失败"))
		return
	}
	filename := fmt.Sprintf("%d%s", time.Now().UnixNano(), ext)
	dst := filepath.Join(dir, filename)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存文件失败"))
		return
	}

	// 返回可访问 URL（通过静态托管 /uploads 前缀映射到 storage_dir）
	url := fmt.Sprintf("/uploads/brand/%s", filename)
	c.JSON(http.StatusOK, response.Success(gin.H{"url": url}))
}
