package api

import (
	"io"
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

	if err := h.svc.SaveConfig(req.Tailnet, req.APIKey, req.ClientID, req.ClientSecret, req.WebhookSecret, req.ProxyURL); err != nil {
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

// SetKeyExpiry 设置免密钥过期 (P1)
func (h *TailscaleHandler) SetKeyExpiry(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		Disabled bool `json:"disabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式错误: "+err.Error()))
		return
	}

	if err := h.svc.SetKeyExpiry(deviceID, req.Disabled); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "设置密钥过期策略失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("密钥过期策略更新成功"))
}

// SetDeviceName 修改设备名称 (P1)
func (h *TailscaleHandler) SetDeviceName(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "必须提供有效的设备名称"))
		return
	}

	if err := h.svc.SetDeviceName(deviceID, req.Name); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "修改设备名称失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("设备名称已更新"))
}

// SetDeviceTags 修改设备标签 (P3)
func (h *TailscaleHandler) SetDeviceTags(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		Tags []string `json:"tags" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式错误: "+err.Error()))
		return
	}

	if err := h.svc.SetDeviceTags(deviceID, req.Tags); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新设备标签失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("设备标签已更新"))
}

// SetDeviceAuthorized 设备核准授权 (P1)
func (h *TailscaleHandler) SetDeviceAuthorized(c *gin.Context) {
	deviceID := c.Param("id")
	var req struct {
		Authorized bool `json:"authorized"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式错误: "+err.Error()))
		return
	}

	if err := h.svc.SetDeviceAuthorized(deviceID, req.Authorized); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "设置设备核准状态失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("设备核准状态已更新"))
}

// GetDeviceRoutes 获取设备广播与生效路由 (P2)
func (h *TailscaleHandler) GetDeviceRoutes(c *gin.Context) {
	deviceID := c.Param("id")
	routes, err := h.svc.GetDeviceRoutes(deviceID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取设备路由失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(routes))
}

// ApproveRoutes 审批设备路由 (P2)
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

// DiagnoseDevice 连通性与 IoT 协议诊断 (P2)
func (h *TailscaleHandler) DiagnoseDevice(c *gin.Context) {
	var req struct {
		Target   string `json:"target" binding:"required"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "必须提供目标地址 target: "+err.Error()))
		return
	}

	res, err := h.svc.DiagnoseDevice(req.Target, req.Port, req.Protocol)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "诊断执行失败: "+err.Error()))
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

// GetACL 查看 ACL (P3)
func (h *TailscaleHandler) GetACL(c *gin.Context) {
	aclStr, err := h.svc.GetACL()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取 ACL 失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(aclStr))
}

// UpdateACL 更新 ACL (P3)
func (h *TailscaleHandler) UpdateACL(c *gin.Context) {
	var req struct {
		HUJSON string `json:"hujson" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "必须提供有效的 ACL HuJSON 内容"))
		return
	}

	if err := h.svc.UpdateACL(req.HUJSON); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新 ACL 失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("ACL 策略已成功更新并生效"))
}

// ValidateACL 校验 ACL (P3)
func (h *TailscaleHandler) ValidateACL(c *gin.Context) {
	var req struct {
		HUJSON string `json:"hujson" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "必须提供有效的 ACL HuJSON 内容"))
		return
	}

	if err := h.svc.ValidateACL(req.HUJSON); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "ACL 语法校验不通过: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("ACL 语法校验通过"))
}

// ReceiveWebhook 接收 Tailscale Webhook 事件回调 (P1 公共端点)
func (h *TailscaleHandler) ReceiveWebhook(c *gin.Context) {
	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "读取请求体失败"))
		return
	}

	sigHeader := c.GetHeader("Tailscale-Webhook-Signature")
	event, err := h.svc.ProcessWebhook(bodyBytes, sigHeader)
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok", "event_type": event.Type})
}

// ListWebhookLogs 获取 Webhook 事件历史日志 (P1)
func (h *TailscaleHandler) ListWebhookLogs(c *gin.Context) {
	logs, err := h.svc.ListWebhookLogs(50)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取 Webhook 日志失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(logs))
}

// SetupRoutes 挂载路由
func SetupRoutes(authed *gin.RouterGroup, public *gin.RouterGroup, handler *TailscaleHandler) {
	if public != nil {
		public.POST("/tailscale/webhook", handler.ReceiveWebhook)
	}

	if authed != nil {
		api := authed.Group("/tailscale")
		{
			api.GET("/ui", handler.RenderUI)
			api.GET("/config", handler.GetConfig)
			api.POST("/config", handler.SaveConfig)
			api.GET("/devices", handler.ListDevices)
			api.POST("/sync/devices", handler.SyncDevices)
			api.DELETE("/devices/:id", handler.DeleteDevice)
			api.POST("/devices/:id/key-expiry", handler.SetKeyExpiry)
			api.POST("/devices/:id/name", handler.SetDeviceName)
			api.POST("/devices/:id/tags", handler.SetDeviceTags)
			api.POST("/devices/:id/authorized", handler.SetDeviceAuthorized)
			api.GET("/devices/:id/routes", handler.GetDeviceRoutes)
			api.POST("/devices/:id/routes", handler.ApproveRoutes)
			api.POST("/diagnose", handler.DiagnoseDevice)
			api.POST("/keys", handler.CreateAuthKey)
			api.GET("/keys/logs", handler.ListKeyLogs)
			api.DELETE("/keys/:id", handler.RevokeAuthKey)
			api.GET("/acl", handler.GetACL)
			api.POST("/acl", handler.UpdateACL)
			api.POST("/acl/validate", handler.ValidateACL)
			api.GET("/webhook/logs", handler.ListWebhookLogs)
		}
	}
}
