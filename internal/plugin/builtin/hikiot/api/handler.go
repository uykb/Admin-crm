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

// SaveConfig 保存海康开放平台配置
func (h *HikHandler) SaveConfig(c *gin.Context) {
	var req struct {
		BaseURL   string `json:"base_url"`
		AppKey    string `json:"app_key" binding:"required"`
		AppSecret string `json:"app_secret" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(400, "参数无效: "+err.Error()))
		return
	}

	if err := h.svc.SaveConfig(req.BaseURL, req.AppKey, req.AppSecret); err != nil {
		c.JSON(http.StatusInternalServerError, response.Error(500, "保存海康配置失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, response.Success("海康配置保存成功"))
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

// SetupRoutes 挂载路由规则
func SetupRoutes(group *gin.RouterGroup, handler *HikHandler) {
	api := group.Group("/hikiot")
	{
		api.POST("/config", handler.SaveConfig)
		api.POST("/sync/orgs", handler.SyncOrgs)
		api.POST("/sync/persons", handler.SyncPersons)
		api.POST("/sync/doors", handler.SyncDoors)
		api.POST("/doors/control", handler.ControlDoor)
		api.GET("/attendance/records", handler.ListAttendance)
		api.GET("/persons/search", handler.SearchPerson)
	}
}
