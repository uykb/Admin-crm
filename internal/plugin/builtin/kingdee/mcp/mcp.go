package mcp

import (
	"fmt"

	"apeadmin-gin/internal/mcp"
	kdservice "apeadmin-gin/internal/plugin/builtin/kingdee/service"

	"gorm.io/gorm"
)

// RegisterTools 向系统 MCP 管理器注册金蝶云星空 AI 工具
func RegisterTools(mgr *mcp.Manager, db *gorm.DB) {
	if mgr == nil {
		return
	}

	svc := kdservice.NewKingdeeService(db)

	// 1. kingdee_query_materials
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "kingdee_query_materials",
		Description: "查询内网金蝶云星空 ERP 系统中的物料/商品数据 (BD_MATERIAL)。支持按关键词搜索物料编码、名称或规格。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词（如物料名称、编码、规格）",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回结果条数限制（默认 20）",
				},
			},
		},
		PluginName:          "kingdee",
		Category:            "erp",
		RequiredPermissions: []string{"kingdee:data:query"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			keyword, _ := args["keyword"].(string)
			limit, _ := args["limit"].(float64)
			limitInt := int(limit)
			if limitInt <= 0 {
				limitInt = 20
			}

			items, err := svc.QueryMaterials(keyword, limitInt)
			if err != nil {
				return nil, fmt.Errorf("金蝶物料查询失败: %w", err)
			}
			return items, nil
		},
	})

	// 2. kingdee_query_customers
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "kingdee_query_customers",
		Description: "查询内网金蝶云星空 ERP 中的客户数据 (BD_Customer)。支持按客户名称或编码筛选。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键词（如客户名称或编码）",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回限制条数（默认 20）",
				},
			},
		},
		PluginName:          "kingdee",
		Category:            "erp",
		RequiredPermissions: []string{"kingdee:data:query"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			keyword, _ := args["keyword"].(string)
			limit, _ := args["limit"].(float64)
			limitInt := int(limit)
			if limitInt <= 0 {
				limitInt = 20
			}

			items, err := svc.QueryCustomers(keyword, limitInt)
			if err != nil {
				return nil, fmt.Errorf("金蝶客户查询失败: %w", err)
			}
			return items, nil
		},
	})

	// 3. kingdee_query_sales_orders
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "kingdee_query_sales_orders",
		Description: "查询内网金蝶云星空 ERP 中的销售订单 (SAL_SALEORDER)。支持按订单编号或客户名称搜索。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"keyword": map[string]interface{}{
					"type":        "string",
					"description": "订单号或客户名称",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回数量（默认 20）",
				},
			},
		},
		PluginName:          "kingdee",
		Category:            "erp",
		RequiredPermissions: []string{"kingdee:data:query"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			keyword, _ := args["keyword"].(string)
			limit, _ := args["limit"].(float64)
			limitInt := int(limit)
			if limitInt <= 0 {
				limitInt = 20
			}

			items, err := svc.QuerySalesOrders(keyword, limitInt)
			if err != nil {
				return nil, fmt.Errorf("金蝶销售订单查询失败: %w", err)
			}
			return items, nil
		},
	})

	// 4. kingdee_execute_bill_query
	mgr.RegisterTool(&mcp.ToolEntry{
		Name:        "kingdee_execute_bill_query",
		Description: "通用金蝶云星空单据列表查询接口 (ExecuteBillQuery)。可以指定 FormId、FieldKeys 与 FilterString 灵活动态查询任何金蝶单据。",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"form_id": map[string]interface{}{
					"type":        "string",
					"description": "金蝶单据 FormId（如 BD_MATERIAL, SAL_SALEORDER, AR_RECEIVEBILL）",
				},
				"field_keys": map[string]interface{}{
					"type":        "string",
					"description": "查询的字段列表（逗号分隔，如 FMaterialId,FNumber,FName）",
				},
				"filter_string": map[string]interface{}{
					"type":        "string",
					"description": "SQL 过滤条件（如 FForbidStatus = 'A' AND FName LIKE '%测试%'）",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "返回结果上限（默认 20）",
				},
			},
			"required": []interface{}{"form_id", "field_keys"},
		},
		PluginName:          "kingdee",
		Category:            "erp",
		RequiredPermissions: []string{"kingdee:data:query"},
		Handler: func(args map[string]interface{}) (interface{}, error) {
			formID, _ := args["form_id"].(string)
			fieldKeys, _ := args["field_keys"].(string)
			filterString, _ := args["filter_string"].(string)
			limit, _ := args["limit"].(float64)
			limitInt := int(limit)
			if limitInt <= 0 {
				limitInt = 20
			}

			if formID == "" || fieldKeys == "" {
				return nil, fmt.Errorf("form_id 与 field_keys 为必填参数")
			}

			rows, err := svc.ExecuteBillQuery(formID, fieldKeys, filterString, limitInt)
			if err != nil {
				return nil, fmt.Errorf("执行金蝶通用查询失败: %w", err)
			}
			return rows, nil
		},
	})
}
