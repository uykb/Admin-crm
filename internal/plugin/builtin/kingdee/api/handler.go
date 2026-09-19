package api

import (
	"net/http"
	"strconv"

	kdservice "apeadmin-gin/internal/plugin/builtin/kingdee/service"
	"apeadmin-gin/internal/pkg/response"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type KingdeeHandler struct {
	service *kdservice.KingdeeService
}

func NewKingdeeHandler(db *gorm.DB) *KingdeeHandler {
	return &KingdeeHandler{
		service: kdservice.NewKingdeeService(db),
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
	}
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
