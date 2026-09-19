package mcp

import (
	"fmt"

	"apeadmin-gin/internal/mcp"
	"apeadmin-gin/internal/plugin/builtin/tailscale/service"

	"gorm.io/gorm"
)

// RegisterTools 注册 AI Agent Tailscale 插件 MCP 工具
func RegisterTools(mgr *mcp.Manager, db *gorm.DB) {
	if mgr == nil {
		return
	}

	svc := service.NewTailscaleService(db)

	// 1. 查询组网节点工具 tailscale_list_devices
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_list_devices",
		Description: "获取当前 Tailnet 内所有组网机器设备、Tailscale IP 地址、操作系统与在线/离线状态",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "关键字搜索（匹配设备名、主机名、IP或所有者），留空返回全量节点",
				},
			},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:list"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			keyword, _ := args["keyword"].(string)
			devices, err := svc.ListCachedDevices(keyword)
			if err != nil {
				return nil, fmt.Errorf("查询 Tailscale 设备节点失败: %w", err)
			}
			return devices, nil
		},
	})

	// 2. 生成接入密钥工具 tailscale_create_auth_key
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_create_auth_key",
		Description: "快速生成用于新机器加入 Tailnet 组网的 Auth Key（支持复用、临时节点、自动核准与标签）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"purpose": map[string]interface{}{
					"type":        "string",
					"description": "密钥用途描述（如: 接入阿里云内网节点/测试服务器连通）",
				},
				"reusable": map[string]interface{}{
					"type":        "boolean",
					"description": "是否允许多台机器复用该 Key（默认 false）",
				},
				"ephemeral": map[string]interface{}{
					"type":        "boolean",
					"description": "是否为临时节点（关机自动从 Tailnet 注销解绑，默认 false）",
				},
				"preauthorized": map[string]interface{}{
					"type":        "boolean",
					"description": "接入时是否自动核准通过（默认 true）",
				},
				"expiry_days": map[string]interface{}{
					"type":        "integer",
					"description": "密钥有效天数（默认 30 天）",
				},
			},
			"required": []string{"purpose"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:key:create"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			purpose, _ := args["purpose"].(string)
			reusable, _ := args["reusable"].(bool)
			ephemeral, _ := args["ephemeral"].(bool)
			preauth, ok := args["preauthorized"].(bool)
			if !ok {
				preauth = true
			}
			expiryDays := 30
			if days, ok := args["expiry_days"].(float64); ok && days > 0 {
				expiryDays = int(days)
			} else if daysInt, ok := args["expiry_days"].(int); ok && daysInt > 0 {
				expiryDays = daysInt
			}

			res, err := svc.CreateAuthKey(reusable, ephemeral, preauth, nil, expiryDays, purpose, "AI_Agent")
			if err != nil {
				return nil, fmt.Errorf("生成 Auth Key 失败: %w", err)
			}

			return map[string]interface{}{
				"id":            res.ID,
				"auth_key":      res.Key,
				"expires":       res.Expires,
				"command":       "tailscale up --authkey=" + res.Key,
				"user_friendly": fmt.Sprintf("生成的 Auth Key 为: %s\n您可以在新设备上运行以下命令直接加入组网:\ntailscale up --authkey=%s", res.Key, res.Key),
			}, nil
		},
	})

	// 3. 注销解绑设备工具 tailscale_delete_device
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_delete_device",
		Description: "从 Tailnet 组网中下线并移除指定的设备节点",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "需要下线解绑的 Tailscale Device ID（可通过 tailscale_list_devices 获取）",
				},
			},
			"required": []string{"device_id"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			deviceID, _ := args["device_id"].(string)
			if deviceID == "" {
				return nil, fmt.Errorf("必须提供 device_id")
			}

			if err := svc.DeleteDevice(deviceID); err != nil {
				return nil, fmt.Errorf("注销设备失败: %w", err)
			}
			return fmt.Sprintf("设备 %s 已成功从 Tailnet 解绑注销", deviceID), nil
		},
	})

	// 4. 审批子网路由工具 tailscale_approve_subnet_routes
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_approve_subnet_routes",
		Description: "核准批准指定节点广播的内网子网路由 (如 192.168.1.0/24) 或出口节点 (Exit Node)",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "广播子网路由的 Tailscale Device ID",
				},
				"routes": map[string]interface{}{
					"type":        "array",
					"description": "需要核准批准的路由 CIDR 列表（如 [\"192.168.1.0/24\", \"0.0.0.0/0\"]）",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"device_id", "routes"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			deviceID, _ := args["device_id"].(string)
			rawRoutes, _ := args["routes"].([]interface{})
			if deviceID == "" || len(rawRoutes) == 0 {
				return nil, fmt.Errorf("必须提供 device_id 与 routes")
			}

			var routes []string
			for _, r := range rawRoutes {
				if s, ok := r.(string); ok {
					routes = append(routes, s)
				}
			}

			res, err := svc.ApproveSubnetRoutes(deviceID, routes)
			if err != nil {
				return nil, fmt.Errorf("核准子网路由失败: %w", err)
			}
			return res, nil
		},
	})
}
