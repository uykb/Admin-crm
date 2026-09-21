package api

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"apeadmin-gin/internal/core"
	"apeadmin-gin/internal/pkg/response"
	"github.com/gin-gonic/gin"
)

// 构建信息（可用 ldflags 注入：-X 'apeadmin-gin/internal/api.buildVersion=xxx'）
var (
	buildVersion = ""
	buildCommit  = ""
	buildTime    = ""
)

// SystemHandler 系统版本与更新 Handler
type SystemHandler struct{}

// Version 系统版本信息（GET /system/version）
func (h *SystemHandler) Version(c *gin.Context) {
	cfg := core.GetConfig()
	version := ""
	if cfg != nil {
		version = cfg.App.Version
	}
	if buildVersion != "" {
		version = buildVersion
	}

	// 启动时间 = 进程启动时间（与 dashboard 共用）
	startedAt := processStartTime.Format(time.RFC3339)

	c.JSON(http.StatusOK, response.Success(map[string]interface{}{
		"name":          "ApeAdmin-Gin",
		"version":       version,
		"build_commit":  buildCommit,
		"build_time":    buildTime,
		"go_version":    runtime.Version(),
		"go_os_arch":    runtime.GOOS + "/" + runtime.GOARCH,
		"started_at":    startedAt,
		"update_supported": true,
	}))
}

// Update 上传系统更新包（POST /system/update）
// 约定：上传 zip 包，包含替换用可执行文件（服务重启后生效）。
// 当前版本实现为"预检 + 保存到指定目录"，不自动替换运行中二进制，
// 避免覆盖正在运行的可执行文件导致平台差异问题。
func (h *SystemHandler) Update(c *gin.Context) {
	cfg := core.GetConfig()
	if cfg == nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "配置未初始化"))
		return
	}

	// 1. 接收 multipart 文件
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少文件字段 file"))
		return
	}

	// 2. 扩展名校验
	ext := strings.ToLower(filepath.Ext(file.Filename))
	if ext != ".zip" && ext != ".exe" && ext != ".bin" {
		c.JSON(http.StatusBadRequest, response.Error(400, "仅支持 .zip / .exe / .bin 更新包"))
		return
	}

	// 3. 大小校验（与 file.max_size_mb 一致，默认 100MB）
	maxSize := int64(100 * 1024 * 1024)
	if cfg.File.MaxSizeMB > 0 {
		maxSize = int64(cfg.File.MaxSizeMB) * 1024 * 1024
	}
	if file.Size > maxSize {
		c.JSON(http.StatusBadRequest, response.Error(400, "更新包超出大小限制"))
		return
	}

	// 4. 保存到更新包目录（uploads/updates，与品牌图/插件目录同级）
	updateDir := filepath.Join(cfg.File.StorageDir, "updates")
	if err := os.MkdirAll(updateDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "创建更新目录失败"))
		return
	}
	// 保留原文件名（sanitize 防穿越）
	dst := filepath.Join(updateDir, fmt.Sprintf("%d_%s", time.Now().UnixNano(), sanitizeFileName(file.Filename)))
	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存更新包失败"))
		return
	}

	log.Printf("[system] 更新包已上传: %s (%d bytes)", file.Filename, file.Size)
	c.JSON(http.StatusOK, response.Success(gin.H{
		"message":   "更新包已保存，重启服务后生效",
		"filename":  file.Filename,
		"size":      file.Size,
		"saved_as":  filepath.Base(dst),
		"restart_required": true,
	}))
}

// GetServerIP 查询后端服务器公网出口 IP（GET /api/v1/system/ip 或 GET /api/v1/ip）
func (h *SystemHandler) GetServerIP(c *gin.Context) {
	client := &http.Client{Timeout: 5 * time.Second}
	var egressIP string

	resp, err := client.Get("https://api.ipify.org?format=json")
	if err == nil {
		defer resp.Body.Close()
		var res struct {
			IP string `json:"ip"`
		}
		if json.NewDecoder(resp.Body).Decode(&res) == nil {
			egressIP = res.IP
		}
	}
	if egressIP == "" {
		resp2, err2 := client.Get("https://icanhazip.com")
		if err2 == nil {
			defer resp2.Body.Close()
			b, _ := io.ReadAll(resp2.Body)
			egressIP = strings.TrimSpace(string(b))
		}
	}

	c.JSON(http.StatusOK, response.Success(gin.H{
		"egress_ip": egressIP,
		"client_ip": c.ClientIP(),
		"host":      c.Request.Host,
		"note":      "如果第三方平台（如海康开放平台/金蝶云星空）要求配置服务器出口 IP 白名单，请将 egress_ip 填入对方控制台白名单中。",
	}))
}