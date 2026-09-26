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
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "赋予接入设备的标签列表（如 [\"tag:iot-door\", \"tag:gateway\"]）",
					"items": map[string]interface{}{
						"type": "string",
					},
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

			var tags []string
			if rawTags, ok := args["tags"].([]interface{}); ok {
				for _, t := range rawTags {
					if ts, ok := t.(string); ok {
						tags = append(tags, ts)
					}
				}
			}

			res, err := svc.CreateAuthKey(reusable, ephemeral, preauth, tags, expiryDays, purpose, "AI_Agent")
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

			return map[string]interface{}{
				"device_id":         deviceID,
				"enabled_routes":    res.EnabledRoutes,
				"advertised_routes": res.AdvertisedRoutes,
				"message":           "子网路由已成功核准生效",
			}, nil
		},
	})

	// 5. 设置免密钥过期工具 tailscale_set_key_expiry (P1 关键能力: 物联网设备永不掉线)
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_set_key_expiry",
		Description: "设置指定 Tailscale 节点设备的密钥是否免过期（针对无人值守物联设备推荐设置为免过期，防止 90/180 天秘钥过期掉线）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "Tailscale 设备 ID",
				},
				"disabled": map[string]interface{}{
					"type":        "boolean",
					"description": "true 为永久免过期，false 为恢复定期过期校验",
				},
			},
			"required": []string{"device_id", "disabled"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			deviceID, _ := args["device_id"].(string)
			disabled, ok := args["disabled"].(bool)
			if deviceID == "" || !ok {
				return nil, fmt.Errorf("参数 device_id 和 disabled 不能为空")
			}
			if err := svc.SetKeyExpiry(deviceID, disabled); err != nil {
				return nil, fmt.Errorf("设置密钥过期策略失败: %w", err)
			}
			statusText := "永久免过期"
			if !disabled {
				statusText = "开启定期过期校验"
			}
			return fmt.Sprintf("设备 %s 密钥过期策略已成功更新为: %s", deviceID, statusText), nil
		},
	})

	// 6. 设备重命名工具 tailscale_set_device_name (P1 MagicDNS 固定映射)
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_set_device_name",
		Description: "修改 Tailscale 设备节点的显示名称与 MagicDNS 域名",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "Tailscale 设备 ID",
				},
				"name": map[string]interface{}{
					"type":        "string",
					"description": "新的设备名称（如: factory-gateway-01 / door-controller-1f）",
				},
			},
			"required": []string{"device_id", "name"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			deviceID, _ := args["device_id"].(string)
			name, _ := args["name"].(string)
			if deviceID == "" || name == "" {
				return nil, fmt.Errorf("必须提供 device_id 与 name")
			}
			if err := svc.SetDeviceName(deviceID, name); err != nil {
				return nil, fmt.Errorf("重命名设备失败: %w", err)
			}
			return fmt.Sprintf("设备 %s 已重命名为: %s", deviceID, name), nil
		},
	})

	// 7. 设备标签分类工具 tailscale_set_device_tags (P3 零信任微隔离)
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_set_device_tags",
		Description: "为指定设备设置标签 Tags（用于 ACL 零信任微隔离策略，如 tag:iot-door, tag:sensor）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"device_id": map[string]interface{}{
					"type":        "string",
					"description": "Tailscale 设备 ID",
				},
				"tags": map[string]interface{}{
					"type":        "array",
					"description": "标签列表（如 [\"tag:iot-door\", \"tag:factory\"]）",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"device_id", "tags"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			deviceID, _ := args["device_id"].(string)
			rawTags, _ := args["tags"].([]interface{})
			if deviceID == "" {
				return nil, fmt.Errorf("必须提供 device_id")
			}
			var tags []string
			for _, t := range rawTags {
				if s, ok := t.(string); ok {
					tags = append(tags, s)
				}
			}
			if err := svc.SetDeviceTags(deviceID, tags); err != nil {
				return nil, fmt.Errorf("更新设备标签失败: %w", err)
			}
			return fmt.Sprintf("设备 %s 标签已成功更新为: %v", deviceID, tags), nil
		},
	})

	// 8. 节点连通性与 IoT 协议诊断工具 tailscale_diagnose_device (P2 物联诊断)
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_diagnose_device",
		Description: "对 Tailnet 组网内的 IP、域名或子网设备进行连通性与端口诊断（支持 Modbus:502, RTSP:554, MQTT:1883, HTTP:80, SSH:22）",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{
					"type":        "string",
					"description": "目标 Tailscale 100.x IP、MagicDNS 域名或子网局域网 IP (如 100.88.2.1 或 192.168.1.100)",
				},
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "目标服务端口（默认 80；物联常用: 502=Modbus, 554=RTSP, 1883=MQTT, 22=SSH）",
				},
				"protocol": map[string]interface{}{
					"type":        "string",
					"description": "协议类型提示（如 TCP, Modbus, RTSP, MQTT, HTTP）",
				},
			},
			"required": []string{"target"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:list"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			target, _ := args["target"].(string)
			port := 80
			if p, ok := args["port"].(float64); ok && p > 0 {
				port = int(p)
			} else if pInt, ok := args["port"].(int); ok && pInt > 0 {
				port = pInt
			}
			protocol, _ := args["protocol"].(string)

			res, err := svc.DiagnoseDevice(target, port, protocol)
			if err != nil {
				return nil, fmt.Errorf("诊断失败: %w", err)
			}
			return res, nil
		},
	})

	// 10. 节点远程 SSH 执行命令工具 tailscale_ssh_exec (方案二: tsnet 原生隧道执行)
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "tailscale_ssh_exec",
		Description: "经由 Tailscale 原生加密隧道向内网或边缘 Linux/工控节点执行 SSH 维护诊断指令",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"target": map[string]interface{}{
					"type":        "string",
					"description": "目标设备 Tailscale 100.x IP 或 MagicDNS 域名",
				},
				"port": map[string]interface{}{
					"type":        "integer",
					"description": "SSH 端口（默认 22）",
				},
				"user": map[string]interface{}{
					"type":        "string",
					"description": "SSH 用户名（默认 root）",
				},
				"password": map[string]interface{}{
					"type":        "string",
					"description": "SSH 登录密码（若节点开启 Tailscale SSH 可留空）",
				},
				"command": map[string]interface{}{
					"type":        "string",
					"description": "待执行的 Linux Shell 命令（如 uname -a, df -h, systemctl status xxx 等）",
				},
			},
			"required": []string{"target", "command"},
		},
		PluginName:          "tailscale",
		Category:            "network",
		RequiredPermissions: []string{"tailscale:device:control"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			target, _ := args["target"].(string)
			port := 22
			if p, ok := args["port"].(float64); ok && p > 0 {
				port = int(p)
			} else if pInt, ok := args["port"].(int); ok && pInt > 0 {
				port = pInt
			}
			user, _ := args["user"].(string)
			password, _ := args["password"].(string)
			command, _ := args["command"].(string)

			output, err := svc.ExecuteSSHCommand(target, port, user, password, "", command)
			if err != nil {
				return map[string]interface{}{
					"target": target,
					"status": "failed",
					"output": output,
					"error":  err.Error(),
				}, nil
			}

			return map[string]interface{}{
				"target": target,
				"status": "success",
				"output": output,
			}, nil
		},
	})
}
