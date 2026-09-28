package api

import (
	"net/http"
	"strconv"

	"apeadmin-gin/internal/pkg/response"
	kdmodel "apeadmin-gin/internal/plugin/builtin/kingdee/model"
	kdservice "apeadmin-gin/internal/plugin/builtin/kingdee/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KingdeeHandler struct {
	db        *gorm.DB
	service   *kdservice.KingdeeService
	feishuSvc *kdservice.FeishuApprovalService
}

func NewKingdeeHandler(db *gorm.DB) *KingdeeHandler {
	return &KingdeeHandler{
		db:        db,
		service:   kdservice.NewKingdeeService(db),
		feishuSvc: kdservice.NewFeishuApprovalService(db),
	}
}

func SetupRoutes(rg *gin.RouterGroup, h *KingdeeHandler) {
	group := rg.Group("/kingdee")
	{
		group.GET("/ui", h.RenderUI)
		group.GET("/config", h.GetConfig)
		group.POST("/config", h.SaveConfig)
		group.POST("/test", h.TestConnection)
		group.GET("/materials", h.QueryMaterials)
		group.GET("/customers", h.QueryCustomers)
		group.GET("/sales-orders", h.QuerySalesOrders)
		group.POST("/query", h.ExecuteQuery)

		// 飞书审批相关 API
		group.GET("/flows", h.ListFlows)
		group.POST("/flows", h.SaveFlow)
		group.GET("/instances", h.ListInstances)
		group.POST("/instances/:id/audit", h.AuditInstance)
		group.POST("/instances/:id/cancel", h.CancelInstance)
		group.GET("/usermaps", h.ListUserMaps)
		group.POST("/usermaps", h.SaveUserMap)
	}

	// 飞书 Webhook 接收点 (免鉴权)
	webhookH := NewWebhookHandler(h.db)
	rg.POST("/kingdee/feishu/webhook", webhookH.HandleFeishuCallback)
}

// RenderUI 渲染嵌入式 Element Plus 控制台 HTML
func (h *KingdeeHandler) RenderUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, KingdeeUIHTML)
}

// GetConfig 获取配置
func (h *KingdeeHandler) GetConfig(c *gin.Context) {
	cfg, err := h.service.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(cfg))
}

// SaveConfig 保存配置
func (h *KingdeeHandler) SaveConfig(c *gin.Context) {
	var req kdservice.ConfigDTO
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数解析失败"))
		return
	}
	if err := h.service.SaveConfig(&req); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.SuccessMsg("配置保存成功"))
}

// TestConnection 测试连通性
func (h *KingdeeHandler) TestConnection(c *gin.Context) {
	if err := h.service.TestConnection(); err != nil {
		c.JSON(http.StatusOK, response.Success(map[string]interface{}{
			"ok":    false,
			"error": err.Error(),
		}))
		return
	}
	c.JSON(http.StatusOK, response.Success(map[string]interface{}{
		"ok":      true,
		"message": "金蝶云星空 Web API 登录认证成功！内网通道畅通。",
	}))
}

// QueryMaterials 查询物料列表
func (h *KingdeeHandler) QueryMaterials(c *gin.Context) {
	keyword := c.Query("keyword")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	items, err := h.service.QueryMaterials(keyword, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(items))
}

// QueryCustomers 查询客户列表
func (h *KingdeeHandler) QueryCustomers(c *gin.Context) {
	keyword := c.Query("keyword")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	items, err := h.service.QueryCustomers(keyword, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(items))
}

// QuerySalesOrders 查询销售订单列表
func (h *KingdeeHandler) QuerySalesOrders(c *gin.Context) {
	keyword := c.Query("keyword")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	items, err := h.service.QuerySalesOrders(keyword, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(items))
}

// ExecuteQuery 执行通用表单查询
func (h *KingdeeHandler) ExecuteQuery(c *gin.Context) {
	var req struct {
		FormID       string `json:"form_id" binding:"required"`
		FieldKeys    string `json:"field_keys" binding:"required"`
		FilterString string `json:"filter_string"`
		Limit        int    `json:"limit"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}

	rows, err := h.service.ExecuteBillQuery(req.FormID, req.FieldKeys, req.FilterString, req.Limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(rows))
}

// ListFlows 流程列表
func (h *KingdeeHandler) ListFlows(c *gin.Context) {
	var flows []kdmodel.KdFlow
	if err := h.db.Find(&flows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(flows))
}

// SaveFlow 保存流程配置
func (h *KingdeeHandler) SaveFlow(c *gin.Context) {
	var flow kdmodel.KdFlow
	if err := c.ShouldBindJSON(&flow); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if flow.ID > 0 {
		h.db.Save(&flow)
	} else {
		h.db.Create(&flow)
	}
	c.JSON(http.StatusOK, response.Success(flow))
}

// ListInstances 审批实例列表
func (h *KingdeeHandler) ListInstances(c *gin.Context) {
	var insts []kdmodel.KdInstance
	if err := h.db.Order("id DESC").Limit(100).Find(&insts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(insts))
}

// AuditInstance 手动重试审核
func (h *KingdeeHandler) AuditInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	var inst kdmodel.KdInstance
	if err := h.db.First(&inst, id).Error; err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "实例不存在"))
		return
	}
	ok := h.feishuSvc.AuditInstance(&inst)
	c.JSON(http.StatusOK, response.Success(map[string]interface{}{"success": ok}))
}

// CancelInstance 撤销审批实例
func (h *KingdeeHandler) CancelInstance(c *gin.Context) {
	idStr := c.Param("id")
	id, _ := strconv.Atoi(idStr)
	var inst kdmodel.KdInstance
	if err := h.db.First(&inst, id).Error; err != nil {
		c.JSON(http.StatusNotFound, response.Error(404, "实例不存在"))
		return
	}
	_ = h.feishuSvc.CancelFeishuInstance(&inst)
	inst.ApproveStatus = kdmodel.InstanceStatusCanceled
	h.db.Save(&inst)
	c.JSON(http.StatusOK, response.SuccessMsg("实例已撤销"))
}

// ListUserMaps 用户映射表
func (h *KingdeeHandler) ListUserMaps(c *gin.Context) {
	var maps []kdmodel.KdUserMap
	if err := h.db.Find(&maps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(maps))
}

// SaveUserMap 保存用户映射
func (h *KingdeeHandler) SaveUserMap(c *gin.Context) {
	var uMap kdmodel.KdUserMap
	if err := c.ShouldBindJSON(&uMap); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	if uMap.ID > 0 {
		h.db.Save(&uMap)
	} else {
		h.db.Create(&uMap)
	}
	c.JSON(http.StatusOK, response.Success(uMap))
}
