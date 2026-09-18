package api

import (
	"fmt"
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

// FileHandler 文件管理 API
type FileHandler struct{}

// ─── 文件夹 ───

// FolderTree 文件夹树（GET /files/folders/tree）
func (h *FileHandler) FolderTree(c *gin.Context) {
	folders, err := dal.ListFolders()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	// 根节点（id=0，代表全部文件）
	nodes := buildFolderTree(folders, 0)
	root := map[string]interface{}{
		"id": 0, "name": "全部文件", "parent_id": nil, "children": nodes,
	}
	c.JSON(http.StatusOK, response.Success([]interface{}{root}))
}

func buildFolderTree(folders []model.SysFileFolder, parentID uint) []map[string]interface{} {
	var nodes []map[string]interface{}
	for _, f := range folders {
		pid := uint(0)
		if f.ParentID != nil {
			pid = *f.ParentID
		}
		if pid == parentID {
			node := map[string]interface{}{
				"id": f.ID, "name": f.Name, "parent_id": f.ParentID,
				"children": buildFolderTree(folders, f.ID),
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}

// CreateFolder 新建文件夹（POST /files/folders）
func (h *FileHandler) CreateFolder(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	f := model.SysFileFolder{Name: req.Name, ParentID: req.ParentID}
	if err := dal.CreateFolder(&f); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(f))
}

// RenameFolder 重命名文件夹（PUT /files/folders/:id）
func (h *FileHandler) RenameFolder(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	folder, err := dal.GetFolderByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "文件夹不存在"))
		return
	}
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	folder.Name = req.Name
	if err := dal.UpdateFolder(folder); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("更新成功"))
}

// MoveFolder 移动文件夹（POST /files/folders/:id/move）
func (h *FileHandler) MoveFolder(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		FolderID *uint `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := dal.MoveFolder(uint(id), req.FolderID); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "不能移动到自身或其子文件夹"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("移动成功"))
}

// DeleteFolder 删除文件夹（DELETE /files/folders/:id，级联删除）
func (h *FileHandler) DeleteFolder(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	files, err := dal.DeleteFolder(uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	// 删除物理文件（失败仅记日志）
	cfg := core.GetConfig()
	for _, f := range files {
		if cfg != nil && cfg.File.StorageDir != "" {
			abs := filepath.Join(cfg.File.StorageDir, f.Path)
			_ = os.Remove(abs)
		}
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// ── 文件 ──

// List 文件列表（GET /files）
func (h *FileHandler) List(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	keyword := c.Query("keyword")
	folderID := uint(0)
	if v := c.Query("folder_id"); v != "" && v != "0" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			folderID = uint(n)
		}
	}
	files, total, err := dal.ListFiles(&folderID, keyword, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败"))
		return
	}
	// 补充 URL
	items := make([]map[string]interface{}, 0, len(files))
	for _, f := range files {
		items = append(items, map[string]interface{}{
			"id": f.ID, "name": f.Name, "size": f.Size,
			"mime_type": f.MimeType, "folder_id": f.FolderID,
			"uploader_id": f.UploaderID,
			"created_at":  f.CreatedAt.Format("2006-01-02 15:04:05"),
		})
	}
	c.JSON(http.StatusOK, response.SuccessPage(items, total, page, pageSize))
}

// Upload 上传文件（POST /files/upload?folder_id=）
func (h *FileHandler) Upload(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil || cfg.File.StorageDir == "" {
		c.JSON(http.StatusInternalServerError, response.Error(500, "存储目录未配置"))
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少文件字段 file"))
		return
	}
	// 大小校验
	maxSize := int64(100 * 1024 * 1024) // 默认 100MB
	if cfg.File.MaxSizeMB > 0 {
		maxSize = int64(cfg.File.MaxSizeMB) * 1024 * 1024
	}
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, response.Error(400, "文件大小超出限制"))
		return
	}

	folderID := uint(0)
	if v := c.Query("folder_id"); v != "" && v != "0" {
		if n, err := strconv.ParseUint(v, 10, 32); err == nil {
			folderID = uint(n)
		}
	}

	// 存储路径：storage_dir / <yyyyMM>/<timestamp>_<sanitized_name>
	sub := time.Now().Format("200601")
	dir := filepath.Join(cfg.File.StorageDir, sub)
	if err := os.MkdirAll(dir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建目录失败"))
		return
	}
	name := sanitizeFileName(file.Filename)
	relPath := sub + "/" + strconv.FormatInt(time.Now().UnixNano(), 10) + "_" + name
	dst := filepath.Join(cfg.File.StorageDir, relPath)
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存文件失败"))
		return
	}

	// 写 DB
	userID := c.GetUint("user_id")
	rec := model.SysFile{
		Name:       name,
		Path:       relPath,
		Size:       file.Size,
		FolderID:   &folderID,
		MimeType:   file.Header.Get("Content-Type"),
		UploaderID: userID,
	}
	if folderID == 0 {
		rec.FolderID = nil
	}
	if err := dal.CreateFile(&rec); err != nil {
		_ = os.Remove(dst)
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存记录失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"id": rec.ID, "name": rec.Name}))
}

// Delete 删除文件（DELETE /files/:id）
func (h *FileHandler) Delete(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	rec, err := dal.GetFileByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "文件不存在"))
		return
	}
	if err := dal.DeleteFile(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	// 删除物理文件（失败仅记日志）
	cfg := core.GetConfig()
	if cfg != nil && cfg.File.StorageDir != "" {
		_ = os.Remove(filepath.Join(cfg.File.StorageDir, rec.Path))
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// Move 移动文件（POST /files/:id/move）
func (h *FileHandler) Move(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	var req struct {
		FolderID *uint `json:"folder_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if err := dal.MoveFile(uint(id), req.FolderID); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "移动失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("移动成功"))
}

// Download 下载/预览文件（GET /files/:id/download?preview=1）
func (h *FileHandler) Download(c *gin.Context) {
	id, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	rec, err := dal.GetFileByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "文件不存在"))
		return
	}
	cfg := core.GetConfig()
	if cfg == nil || cfg.File.StorageDir == "" {
		c.JSON(http.StatusInternalServerError, response.Error(500, "存储目录未配置"))
		return
	}
	abs := filepath.Join(cfg.File.StorageDir, rec.Path)
	if _, err := os.Stat(abs); err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "文件不存在或已被删除"))
		return
	}
	preview := c.Query("preview") == "1"
	if preview {
		c.File(abs)
		return
	}
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+urlEscapeFileName(rec.Name))
	c.File(abs)
}

// ── 素材存储（物理目录浏览）──

// AssetGroups 素材分组列表（GET /files/assets/groups）
func (h *FileHandler) AssetGroups(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil || cfg.File.StorageDir == "" {
		c.JSON(http.StatusInternalServerError, response.Error(500, "存储目录未配置"))
		return
	}
	groups := assetGroupDefs()
	result := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		abs := filepath.Join(cfg.File.StorageDir, g.key)
		count := 0
		_ = filepath.Walk(abs, func(path string, info os.FileInfo, err error) error {
			if err == nil && !info.IsDir() {
				count++
			}
			return nil
		})
		result = append(result, map[string]interface{}{
			"key": g.key, "name": g.name, "risk": g.risk,
			"note": g.note, "file_count": count,
		})
	}
	c.JSON(http.StatusOK, response.Success(result))
}

// AssetList 素材目录列表（GET /files/assets/list?group=&path=）
func (h *FileHandler) AssetList(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil || cfg.File.StorageDir == "" {
		c.JSON(http.StatusInternalServerError, response.Error(500, "存储目录未配置"))
		return
	}
	group := c.Query("group")
	rel := c.Query("path")
	def := assetGroupByKey(group)
	if def == nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "未知素材分组"))
		return
	}
	abs := filepath.Join(cfg.File.StorageDir, def.key)
	if rel != "" {
		// 防目录穿越
		if strings.Contains(rel, "..") || strings.HasPrefix(rel, "/") || strings.HasPrefix(rel, "\\") {
			c.JSON(http.StatusBadRequest, response.Error(400, "非法路径"))
			return
		}
		abs = filepath.Join(abs, filepath.FromSlash(rel))
	}
	entries, err := os.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "目录不存在"))
		return
	}
	dirs := make([]map[string]interface{}, 0)
	files := make([]map[string]interface{}, 0)
	for _, e := range entries {
		info, err := e.Info()
		if err != nil {
			continue
		}
		item := map[string]interface{}{
			"name": e.Name(), "path": e.Name(), "size": info.Size(),
			"modified_at": info.ModTime().Format(time.RFC3339),
		}
		if rel != "" {
			item["path"] = rel + "/" + e.Name()
		}
		if e.IsDir() {
			dirs = append(dirs, item)
		} else {
			files = append(files, item)
		}
	}
	c.JSON(http.StatusOK, response.Success(map[string]interface{}{
		"group": def.key, "path": rel, "dirs": dirs, "files": files,
	}))
}

// DeleteAsset 删除素材（DELETE /files/assets?group=&path=）
func (h *FileHandler) DeleteAsset(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil || cfg.File.StorageDir == "" {
		c.JSON(http.StatusInternalServerError, response.Error(500, "存储目录未配置"))
		return
	}
	group := c.Query("group")
	path := c.Query("path")
	def := assetGroupByKey(group)
	if def == nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "未知素材分组"))
		return
	}
	if path == "" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, response.Error(400, "非法路径"))
		return
	}
	abs := filepath.Join(cfg.File.StorageDir, def.key, filepath.FromSlash(path))
	info, err := os.Stat(abs)
	if err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "文件不存在"))
		return
	}
	if info.IsDir() {
		c.JSON(http.StatusBadRequest, response.Error(400, "仅支持删除文件"))
		return
	}
	if err := os.Remove(abs); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "删除失败"))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("删除成功"))
}

// DownloadAsset 素材下载/预览（GET /files/assets/download?group=&path=&preview=1）
func (h *FileHandler) AssetDownload(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil || cfg.File.StorageDir == "" {
		c.JSON(http.StatusInternalServerError, response.Error(500, "存储目录未配置"))
		return
	}
	group := c.Query("group")
	path := c.Query("path")
	def := assetGroupByKey(group)
	if def == nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "未知素材分组"))
		return
	}
	if path == "" || strings.Contains(path, "..") {
		c.JSON(http.StatusBadRequest, response.Error(400, "非法路径"))
		return
	}
	abs := filepath.Join(cfg.File.StorageDir, def.key, filepath.FromSlash(path))
	if _, err := os.Stat(abs); err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "文件不存在"))
		return
	}
	preview := c.Query("preview") == "1"
	if preview {
		c.File(abs)
		return
	}
	c.Header("Content-Disposition", "attachment; filename*=UTF-8''"+urlEscapeFileName(filepath.Base(path)))
	c.File(abs)
}

// ── 素材分组定义 ──

type assetGroupDef struct {
	key   string
	name  string
	risk  string
	note  string
}

func assetGroupDefs() []assetGroupDef {
	return []assetGroupDef{
		{key: "brand", name: "品牌图片", risk: "low", note: "登录页背景、Logo 等品牌图片"},
		{key: "plugins", name: "插件安装包", risk: "high", note: "业务安装包，删除后对应插件/版本将无法下载"},
		{key: "files", name: "文件管理存储", risk: "high", note: "文件管理模块上传的业务文件，删除后对应文件将无法下载"},
	}
}

func assetGroupByKey(key string) *assetGroupDef {
	for _, g := range assetGroupDefs() {
		if g.key == key {
			return &g
		}
	}
	return nil
}

// 小工具
func urlEscapeFileName(name string) string {
	return strings.ReplaceAll(urlQueryEscape(name), "+", "%20")
}

// sanitizeFileName 清理文件名：去除路径分隔符与危险字符，防止路径穿越
func sanitizeFileName(name string) string {
	name = filepath.Base(name)
	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.Map(func(r rune) rune {
		switch r {
		case '<', '>', ':', '"', '|', '?', '*', 0, '\n', '\r', '\t':
			return '_'
		}
		return r
	}, name)
	name = strings.TrimSpace(name)
	if name == "" || name == "." || name == ".." {
		return "unnamed"
	}
	return name
}

func urlQueryEscape(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_' || r == '.' || r == '~' {
			b.WriteRune(r)
			continue
		}
		b.WriteString(fmt.Sprintf("%%%02X", r))
	}
	return b.String()
}
