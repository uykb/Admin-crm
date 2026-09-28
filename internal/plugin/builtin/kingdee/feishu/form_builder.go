package feishu

import (
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	kdmodel "apeadmin-gin/internal/plugin/builtin/kingdee/model"
)

type FormBuilder struct{}

func NewFormBuilder() *FormBuilder {
	return &FormBuilder{}
}

// BuildForm 构建飞书审批实例的 Form JSON 字符串
func (fb *FormBuilder) BuildForm(
	headerFields []kdmodel.KdFlowField,
	entryFields []kdmodel.KdFlowField,
	bill map[string]interface{},
	entryRows []map[string]interface{},
	controls []map[string]interface{},
) (string, error) {
	formItems := make([]map[string]interface{}, 0, len(controls))

	for _, ctrl := range controls {
		ctrlType, _ := ctrl["type"].(string)
		ctrlID, _ := ctrl["id"].(string)

		if ctrlType == "fieldList" {
			item := fb.buildFieldList(ctrl, entryFields, entryRows)
			formItems = append(formItems, item)
		} else {
			field := fb.matchField(headerFields, ctrl)
			rawVal := fb.resolveRaw(bill, field)
			formItems = append(formItems, map[string]interface{}{
				"id":    ctrlID,
				"type":  ctrlType,
				"value": fb.formatValue(rawVal, ctrlType),
			})
		}
	}

	b, err := json.Marshal(formItems)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (fb *FormBuilder) buildFieldList(
	ctrl map[string]interface{},
	entryFields []kdmodel.KdFlowField,
	entryRows []map[string]interface{},
) map[string]interface{} {
	ctrlID, _ := ctrl["id"].(string)

	var children []map[string]interface{}
	if rawChild, ok := ctrl["children"].([]interface{}); ok {
		for _, c := range rawChild {
			if cm, ok := c.(map[string]interface{}); ok {
				children = append(children, cm)
			}
		}
	} else if rawVal, ok := ctrl["value"].([]interface{}); ok {
		for _, v := range rawVal {
			if vm, ok := v.(map[string]interface{}); ok {
				children = append(children, vm)
			}
		}
	}

	valueRows := make([][]map[string]interface{}, 0, len(entryRows))
	for _, row := range entryRows {
		rowItems := make([]map[string]interface{}, 0, len(children))
		for _, child := range children {
			childType, _ := child["type"].(string)
			childID, _ := child["id"].(string)
			field := fb.matchField(entryFields, child)
			rawVal := fb.resolveRaw(row, field)

			rowItems = append(rowItems, map[string]interface{}{
				"id":    childID,
				"type":  childType,
				"value": fb.formatValue(rawVal, childType),
			})
		}
		valueRows = append(valueRows, rowItems)
	}

	return map[string]interface{}{
		"id":    ctrlID,
		"type":  "fieldList",
		"value": valueRows,
	}
}

func (fb *FormBuilder) matchField(fields []kdmodel.KdFlowField, ctrl map[string]interface{}) *kdmodel.KdFlowField {
	customID, _ := ctrl["custom_id"].(string)
	name, _ := ctrl["name"].(string)

	for _, f := range fields {
		if f.WidgetID != "" && customID != "" && f.WidgetID == customID {
			return &f
		}
	}
	for _, f := range fields {
		if f.FieldAlias != "" && name != "" && f.FieldAlias == name {
			return &f
		}
	}
	return nil
}

func (fb *FormBuilder) resolveRaw(data map[string]interface{}, field *kdmodel.KdFlowField) interface{} {
	if field == nil || data == nil || field.KingdeeField == "" {
		return nil
	}

	kf := field.KingdeeField
	if strings.Contains(kf, ",") {
		parts := strings.Split(kf, ",")
		resMap := make(map[string]interface{})
		for _, part := range parts {
			key := strings.TrimSpace(part)
			if key != "" {
				resMap[key] = data[key]
			}
		}
		return resMap
	}

	return data[kf]
}

func (fb *FormBuilder) formatValue(raw interface{}, ctrlType string) interface{} {
	if ctrlType == "dateInterval" {
		return fb.formatDateInterval(raw)
	}

	if raw == nil || fmt.Sprintf("%v", raw) == "" {
		if ctrlType == "amount" || ctrlType == "number" {
			return 0
		}
		return ""
	}

	if ctrlType == "amount" || ctrlType == "number" {
		valStr := fmt.Sprintf("%v", raw)
		if f, err := strconv.ParseFloat(valStr, 64); err == nil {
			return f
		}
		return 0
	}

	return fmt.Sprintf("%v", raw)
}

func (fb *FormBuilder) formatDateInterval(raw interface{}) map[string]interface{} {
	startStr := ""
	endStr := ""

	if m, ok := raw.(map[string]interface{}); ok {
		startStr = fmt.Sprintf("%v", m["FStartDate"])
		endStr = fmt.Sprintf("%v", m["FEndDate"])
	} else if raw != nil {
		startStr = fmt.Sprintf("%v", raw)
		endStr = fmt.Sprintf("%v", raw)
	}

	if startStr != "" && !strings.Contains(startStr, "+") && !strings.Contains(startStr, "Z") {
		startStr += "+08:00"
	}
	if endStr != "" && !strings.Contains(endStr, "+") && !strings.Contains(endStr, "Z") {
		endStr += "+08:00"
	}

	interval := 0.0
	if startStr != "" && endStr != "" {
		// 计算时间差天数
		layout := "2006-01-02T15:04:05"
		sClean := strings.Replace(strings.Split(startStr, "+")[0], " ", "T", 1)
		eClean := strings.Replace(strings.Split(endStr, "+")[0], " ", "T", 1)
		if len(sClean) > 19 {
			sClean = sClean[:19]
		}
		if len(eClean) > 19 {
			eClean = eClean[:19]
		}

		t1, err1 := time.Parse(layout, sClean)
		t2, err2 := time.Parse(layout, eClean)
		if err1 == nil && err2 == nil {
			diffDays := t2.Sub(t1).Hours() / 24.0
			interval = math.Round(diffDays*10) / 10.0
		}
	}

	return map[string]interface{}{
		"start":    startStr,
		"end":      endStr,
		"interval": interval,
	}
}
