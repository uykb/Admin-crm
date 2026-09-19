package api

import (
	"net/http"

	"apeadmin-gin/internal/pkg/response"
	"apeadmin-gin/internal/plugin/builtin/tailscale/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type TailscaleHandler struct {
	svc *service.TailscaleService
}

func NewTailscaleHandler(db *gorm.DB) *TailscaleHandler {
	return &TailscaleHandler{
		svc: service.NewTailscaleService(db),
	}
}

// RenderUI 渲染嵌入式 Tailscale 控制台 UI
func (h *TailscaleHandler) RenderUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, TailscaleUIHTML)
}

// GetConfig 获取当前配置
func (h *TailscaleHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取配置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(cfg))
}

// SaveConfig 保存配置
func (h *TailscaleHandler) SaveConfig(c *gin.Context) {
	var req service.ConfigDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式无效: "+err.Error()))
		return
	}

	if err := h.svc.SaveConfig(req.Tailnet, req.APIKey, req.ClientID, req.ClientSecret); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存配置失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success("Tailscale 配置保存成功"))
}

// ListDevices 列出设备
func (h *TailscaleHandler) ListDevices(c *gin.Context) {
	keyword := c.Query("keyword")
	devices, err := h.svc.ListCachedDevices(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取设备失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(devices))
}

// SyncDevices 强制全量同步
func (h *TailscaleHandler) SyncDevices(c *gin.Context) {
	count, err := h.svc.SyncDevices()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "同步失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"synced_count": count}))
}

// DeleteDevice 下线节点
func (h *TailscaleHandler) DeleteDevice(c *gin.Context) {
	deviceID := c.Param("id")
	if deviceID == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少设备 ID"))
		return
	}

	if err := h.svc.DeleteDevice(deviceID); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "下线设备失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("设备已注销解绑"))
}

// ApproveRoutes 审批设备路由
func (h *TailscaleHandler) ApproveRoutes(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		Routes []string `json:"routes" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式错误: "+err.Error()))
		return
	}

	res, err := h.svc.ApproveSubnetRoutes(deviceID, req.Routes)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "审批路由失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(res))
}

// CreateAuthKey 生成新 Auth Key
func (h *TailscaleHandler) CreateAuthKey(c *gin.Context) {
	var req struct {
		Reusable      bool     `json:"reusable"`
		Ephemeral     bool     `json:"ephemeral"`
		Preauthorized bool     `json:"preauthorized"`
		Tags          []string `json:"tags"`
		ExpiryDays    int      `json:"expiry_days"`
		Purpose       string   `json:"purpose"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式错误: "+err.Error()))
		return
	}

	createdBy := c.GetString("username")
	if createdBy == "" {
		createdBy = "admin"
	}

	res, err := h.svc.CreateAuthKey(req.Reusable, req.Ephemeral, req.Preauthorized, req.Tags, req.ExpiryDays, req.Purpose, createdBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "生成 Auth Key 失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(res))
}

// ListKeyLogs 获取 Auth Key 日志
func (h *TailscaleHandler) ListKeyLogs(c *gin.Context) {
	logs, err := h.svc.ListKeyLogs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取密钥日志失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(logs))
}

// RevokeAuthKey 撤销 Key
func (h *TailscaleHandler) RevokeAuthKey(c *gin.Context) {
	keyID := c.Param("id")
	if keyID == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "缺少密钥 ID"))
		return
	}

	if err := h.svc.RevokeAuthKey(keyID); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "撤销密钥失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("密钥已强行撤销失效"))
}

// GetACL 查看 ACL
func (h *TailscaleHandler) GetACL(c *gin.Context) {
	aclStr, err := h.svc.GetACL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取 ACL 失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(aclStr))
}

// SetupRoutes 挂载路由
func SetupRoutes(group *gin.RouterGroup, handler *TailscaleHandler) {
	api := group.Group("/tailscale")
	{
		api.GET("/ui", handler.RenderUI)
		api.GET("/config", handler.GetConfig)
		api.POST("/config", handler.SaveConfig)
		api.GET("/devices", handler.ListDevices)
		api.POST("/sync/devices", handler.SyncDevices)
		api.DELETE("/devices/:id", handler.DeleteDevice)
		api.POST("/devices/:id/routes", handler.ApproveRoutes)
		api.POST("/keys", handler.CreateAuthKey)
		api.GET("/keys/logs", handler.ListKeyLogs)
		api.DELETE("/keys/:id", handler.RevokeAuthKey)
		api.GET("/acl", handler.GetACL)
	}
}
