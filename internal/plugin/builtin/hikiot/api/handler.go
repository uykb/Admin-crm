package api

import (
	"net/http"

	"apeadmin-gin/internal/pkg/response"
	"apeadmin-gin/internal/plugin/builtin/hikiot/service"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type HikHandler struct {
	svc *service.HikService
}

func NewHikHandler(db *gorm.DB) *HikHandler {
	return &HikHandler{
		svc: service.NewHikService(db),
	}
}

// RenderUI 渲染海康互联可视化管理面板
func (h *HikHandler) RenderUI(c *gin.Context) {
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.String(http.StatusOK, HikUIHTML)
}

// GetConfig 获取海康开放平台配置
func (h *HikHandler) GetConfig(c *gin.Context) {
	cfg, err := h.svc.GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "获取配置失败"))
		return
	}
	c.JSON(http.StatusOK, response.Success(cfg))
}

// SaveConfig 保存插件配置
func (h *HikHandler) SaveConfig(c *gin.Context) {
	var req struct {
		BaseURL   string `json:"base_url"`
		AppKey    string `json:"app_key"`
		AppSecret string `json:"app_secret"`
		UserToken string `json:"user_token"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误: "+err.Error()))
		return
	}

	err := h.svc.SaveConfig(req.BaseURL, req.AppKey, req.AppSecret, req.UserToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存配置失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("保存成功"))
}

// TestPersons 测试人员分页接口直接返回
func (h *HikHandler) TestPersons(c *gin.Context) {
	page := c.Query("page")
	if page == "" {
		page = "1"
	}
	size := c.Query("size")
	if size == "" {
		size = "50"
	}
	departNo := c.Query("departNo")

	cli, err := h.svc.GetClient()
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	params := map[string]interface{}{
		"page":          page,
		"size":          size,
		"hasLeafDepart": true,
	}
	if departNo != "" {
		params["departNo"] = departNo
	}

	var resp map[string]interface{}
	err = cli.DoRequest("GET", "/team/v1/person/page", params, &resp)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, resp)
}

// ListDoors 获取门禁设备列表
func (h *HikHandler) ListDoors(c *gin.Context) {
	list, err := h.svc.ListDoors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询门禁设备失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(list))
}

// SyncDoors 手动同步门禁列表
func (h *HikHandler) SyncDoors(c *gin.Context) {
	count, err := h.svc.SyncDoors()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "同步海康门禁点失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"synced_count": count}))
}

// ControlDoor 控制门禁点
func (h *HikHandler) ControlDoor(c *gin.Context) {
	var req struct {
		DoorIndexCode string `json:"door_index_code" binding:"required"`
		Command       int    `json:"command"` // 0: 关门, 1: 开门, 2: 常开, 3: 常关
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数格式错误"))
		return
	}

	if err := h.svc.ControlDoor(req.DoorIndexCode, req.Command); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "控门指令发送失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success("控门指令已下发"))
}

// ListAttendance 查询打卡记录
func (h *HikHandler) ListAttendance(c *gin.Context) {
	personName := c.Query("person_name")
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	list, err := h.svc.QueryAttendance(personName, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询打卡记录失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success(list))
}

// SearchPerson 检索人员
func (h *HikHandler) SearchPerson(c *gin.Context) {
	keyword := c.Query("keyword")
	list, err := h.svc.SearchPerson(keyword)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "人员检索失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(list))
}

// SyncOrgs 手动同步组织架构
func (h *HikHandler) SyncOrgs(c *gin.Context) {
	count, err := h.svc.SyncOrgs()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "同步海康组织架构失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"synced_count": count}))
}

// SyncPersons 手动同步人员档案
func (h *HikHandler) SyncPersons(c *gin.Context) {
	count, err := h.svc.SyncPersons()
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "同步海康人员失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(gin.H{"synced_count": count}))
}

// CalculateMatrix 一键智能排班计算
func (h *HikHandler) CalculateMatrix(c *gin.Context) {
	var req struct {
		Month string `json:"month" binding:"required"` // "2026-09"
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	err := h.svc.CalculateMonthlyAttendance(req.Month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "排班计算失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("计算完成"))
}

// GetMatrixData 获取排班矩阵数据
func (h *HikHandler) GetMatrixData(c *gin.Context) {
	month := c.Query("month")
	if month == "" {
		c.JSON(http.StatusBadRequest, response.Error(400, "必须指定 month"))
		return
	}
	data, err := h.svc.GetMonthlyAttendanceMatrix(month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "查询失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success(data))
}

// UpdateMatrixResult 手工覆盖排班结果
func (h *HikHandler) UpdateMatrixResult(c *gin.Context) {
	var req struct {
		PersonID  string `json:"person_id" binding:"required"`
		Date      string `json:"date" binding:"required"`
		ShiftType string `json:"shift_type" binding:"required"`
		Remark    string `json:"remark"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数错误"))
		return
	}
	err := h.svc.UpdateAttendanceResult(req.PersonID, req.Date, req.ShiftType, req.Remark)
	if err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "更新失败: "+err.Error()))
		return
	}
	c.JSON(http.StatusOK, response.Success("更新成功"))
}

// SetupRoutes 挂载路由规则
func SetupRoutes(group *gin.RouterGroup, handler *HikHandler) {
	api := group.Group("/hikiot")
	{
		api.GET("/ui", handler.RenderUI)
		api.GET("/config", handler.GetConfig)
		api.POST("/config", handler.SaveConfig)
		api.GET("/doors", handler.ListDoors)
		api.POST("/sync/doors", handler.SyncDoors)
		api.POST("/doors/control", handler.ControlDoor)
		api.GET("/attendance/records", handler.ListAttendance)
		api.GET("/persons/search", handler.SearchPerson)
		api.POST("/sync/orgs", handler.SyncOrgs)
		api.POST("/sync/persons", handler.SyncPersons)
		api.GET("/attendance/matrix", handler.GetMatrixData)
		api.POST("/attendance/calculate", handler.CalculateMatrix)
		api.PUT("/attendance/result", handler.UpdateMatrixResult)
	}
}
