package mcp

import (
	"fmt"

	"apeadmin-gin/internal/mcp"
	"apeadmin-gin/internal/plugin/builtin/hikiot/service"

	"gorm.io/gorm"
)

// RegisterTools 注册海康 AI Agent MCP 工具集
func RegisterTools(mgr *mcp.Manager, db *gorm.DB) {
	if mgr == nil {
		return
	}

	svc := service.NewHikService(db)

	// 1. 控门工具 hikiot_door_control
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "hikiot_door_control",
		Description: "海康互联门禁点控制工具（如开门、关门、常开）。需声明 door_index_code 及指令 code。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"door_index_code": map[string]interface{}{
					"type":        "string",
					"description": "门禁编号 (doorIndexCode)",
				},
				"command": map[string]interface{}{
					"type":        "integer",
					"description": "操作指令 (0: 关门, 1: 开门, 2: 常开, 3: 常关)",
				},
			},
			"required": []string{"door_index_code", "command"},
		},
		PluginName:          "hikiot",
		Category:            "device",
		RequiredPermissions: []string{"hikiot:door:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			doorIndexCode, ok := args["door_index_code"].(string)
			if !ok || doorIndexCode == "" {
				return nil, fmt.Errorf("缺少必填参数 door_index_code")
			}
			cmdRaw, ok := args["command"]
			if !ok {
				return nil, fmt.Errorf("缺少必填参数 command")
			}

			var command int
			switch v := cmdRaw.(type) {
			case float64:
				command = int(v)
			case int:
				command = v
			}

			if err := svc.ControlDoor(doorIndexCode, command); err != nil {
				return nil, fmt.Errorf("控门指令下发失败: %w", err)
			}

			return map[string]interface{}{
				"status":  "success",
				"message": fmt.Sprintf("门禁点 [%s] 已成功执行指令 [%d]", doorIndexCode, command),
			}, nil
		},
	})

	// 2. 考勤打卡查询工具 hikiot_attendance_query
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "hikiot_attendance_query",
		Description: "查询海康互联考勤打卡刷卡记录，可按姓名或日期范围筛选。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"person_name": map[string]interface{}{
					"type":        "string",
					"description": "员工姓名 (模糊匹配，可选)",
				},
				"start_date": map[string]interface{}{
					"type":        "string",
					"description": "开始日期，格式 YYYY-MM-DD (可选)",
				},
				"end_date": map[string]interface{}{
					"type":        "string",
					"description": "结束日期，格式 YYYY-MM-DD (可选)",
				},
			},
		},
		PluginName:          "hikiot",
		Category:            "attendance",
		RequiredPermissions: []string{"hikiot:attendance:list"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			personName, _ := args["person_name"].(string)
			startDate, _ := args["start_date"].(string)
			endDate, _ := args["end_date"].(string)

			records, err := svc.QueryAttendance(personName, startDate, endDate)
			if err != nil {
				return nil, fmt.Errorf("查询考勤记录失败: %w", err)
			}

			return map[string]interface{}{
				"total":   len(records),
				"records": records,
			}, nil
		},
	})

	// 3. 人员档案查询工具 hikiot_person_search
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "hikiot_person_search",
		Description: "搜索海康组织架构下的员工档案信息（工号、手机号、部门）。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键字（如人员姓名、工号或电话号码）",
				},
			},
			"required": []string{"keyword"},
		},
		PluginName:          "hikiot",
		Category:            "personnel",
		RequiredPermissions: []string{},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			keyword, _ := args["keyword"].(string)
			list, err := svc.SearchPerson(keyword)
			if err != nil {
				return nil, fmt.Errorf("检索人员失败: %w", err)
			}

			return map[string]interface{}{
				"total":   len(list),
				"persons": list,
			}, nil
		},
	})
}
