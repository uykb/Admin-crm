package service

import (
	"fmt"
	"log"
	"regexp"
	"strings"

	kdfeishu "apeadmin-gin/internal/plugin/builtin/kingdee/feishu"
	kdmodel "apeadmin-gin/internal/plugin/builtin/kingdee/model"

	"gorm.io/gorm"
)

type FeishuApprovalService struct {
	db          *gorm.DB
	kdService   *KingdeeService
	formBuilder *kdfeishu.FormBuilder
}

func NewFeishuApprovalService(db *gorm.DB) *FeishuApprovalService {
	return &FeishuApprovalService{
		db:          db,
		kdService:   NewKingdeeService(db),
		formBuilder: kdfeishu.NewFormBuilder(),
	}
}

// 固定审批人默认 OpenID 常量 (也可在 sys_config / kd_config 中覆盖)
const (
	FinanceManagerOpenID  = "ou_f50ced66d45672487cc0231e7cd3457a"
	PurchaseManagerOpenID = "ou_98e0fcfc47a036faa2c9fad36dcf23ee"
	SalesManagerOpenID    = "ou_66780683b2322e69353677e931ddf7bc"
)

var DeptSupervisorOpenIDs = map[string]string{
	"销售一部": "ou_edaebc0515e57894395c3e440c56727c",
	"销售二部": "ou_6d2abb6da8fc9daf097aedf9007afd91",
	"销售三部": "ou_601bfcc8e9533d0f622f88dbb2c394d7",
}

func (s *FeishuApprovalService) getFeishuClient() (*kdfeishu.Client, error) {
	cfg, err := s.kdService.GetConfig()
	if err != nil {
		return nil, err
	}
	if cfg.FeishuAppID == "" || cfg.FeishuAppSecret == "" {
		return nil, fmt.Errorf("未配置飞书 AppID 或 AppSecret，请先在配置中设置飞书开放平台应用信息")
	}
	return kdfeishu.NewClient(cfg.FeishuAppID, cfg.FeishuAppSecret, "https://open.feishu.cn"), nil
}

// StartFeishuApproval 发起飞书审批
func (s *FeishuApprovalService) StartFeishuApproval(flow *kdmodel.KdFlow, bill map[string]interface{}) (*kdmodel.KdInstance, bool, error) {
	fid := str(bill["FID"])
	if fid == "" {
		return nil, false, fmt.Errorf("单据缺少 FID，无法发起审批 | formId=%s", flow.KingdeeFormId)
	}

	// 1. 幂等预检
	var exist kdmodel.KdInstance
	err := s.db.Where("flow_id = ? AND bill_fid = ?", flow.ID, fid).First(&exist).Error
	if err == nil && !s.isRestartable(exist.ApproveStatus) {
		log.Printf("[KingdeeFeishu] 单据已存在活跃审批实例，跳过 | billNo=%s status=%s", exist.BillNo, exist.ApproveStatus)
		return &exist, false, nil
	}

	// 2. 创建飞书审批实例
	instanceCode, err := s.createFeishuInstance(flow, bill)
	if err != nil {
		return nil, false, err
	}

	billNo := str(bill[flow.BillNoField])
	billStatus := str(bill[flow.StatusField])

	if exist.ID > 0 {
		// 重新发起
		exist.BillNo = billNo
		exist.BillStatus = billStatus
		exist.InstanceID = instanceCode
		exist.ApproveStatus = kdmodel.InstanceStatusPending
		exist.ErrorMsg = ""
		exist.RetryCount = 0
		s.db.Save(&exist)
		log.Printf("[KingdeeFeishu] 审批重新发起成功 | billNo=%s instance_code=%s", billNo, instanceCode)
		return &exist, true, nil
	}

	inst := kdmodel.KdInstance{
		FlowID:        flow.ID,
		BillFid:       fid,
		BillNo:        billNo,
		BillStatus:    billStatus,
		InstanceID:    instanceCode,
		ApproveStatus: kdmodel.InstanceStatusPending,
		RetryCount:    0,
	}
	s.db.Create(&inst)
	log.Printf("[KingdeeFeishu] 审批发起成功 | billNo=%s instance_code=%s", billNo, instanceCode)
	return &inst, true, nil
}

func (s *FeishuApprovalService) isRestartable(status string) bool {
	return status == kdmodel.InstanceStatusCanceled ||
		status == kdmodel.InstanceStatusRejected ||
		status == kdmodel.InstanceStatusFailed
}

func (s *FeishuApprovalService) createFeishuInstance(flow *kdmodel.KdFlow, bill map[string]interface{}) (string, error) {
	if flow.FeishuApprovalCode == "" {
		return "", fmt.Errorf("流程配置 [%s] 未设置飞书审批 Code", flow.FlowName)
	}

	feishuCli, err := s.getFeishuClient()
	if err != nil {
		return "", err
	}

	// 自动订阅审批事件
	_ = feishuCli.SubscribeApprovalEvent(flow.FeishuApprovalCode)

	// 解析申请人与二级审批人
	applicantOpenID := s.resolveApplicant(bill, feishuCli)
	secondLevelOpenID := s.resolveSecondLevelApprover(flow, bill, feishuCli)

	// 查字段映射与表身明细
	var headerFields []kdmodel.KdFlowField
	var entryFields []kdmodel.KdFlowField
	s.db.Where("flow_id = ? AND is_entry = '0'", flow.ID).Order("sort_no ASC").Find(&headerFields)
	s.db.Where("flow_id = ? AND is_entry = '1'", flow.ID).Order("sort_no ASC").Find(&entryFields)

	entryRows := s.queryEntries(flow, bill, entryFields)

	// 获取飞书审批定义控件
	controls, err := feishuCli.GetApprovalDefinition(flow.FeishuApprovalCode)
	if err != nil {
		return "", fmt.Errorf("获取飞书审批定义失败: %w", err)
	}

	// 构建 Form JSON
	formJson, err := s.formBuilder.BuildForm(headerFields, entryFields, bill, entryRows, controls)
	if err != nil {
		return "", fmt.Errorf("构建飞书 Form JSON 失败: %w", err)
	}

	title := s.renderTitle(flow, bill)

	// 双节点设置: node_first(财务经理) ➔ node_second(业务主管/二级)
	nodeApprovers := []map[string]interface{}{
		{"key": "node_first", "value": []string{FinanceManagerOpenID}},
		{"key": "node_second", "value": []string{secondLevelOpenID}},
	}

	return feishuCli.CreateApprovalInstance(flow.FeishuApprovalCode, applicantOpenID, formJson, title, nodeApprovers)
}

func (s *FeishuApprovalService) resolveApplicant(bill map[string]interface{}, feishuCli *kdfeishu.Client) string {
	creatorID := str(bill["FCreatorId"])
	if creatorID != "" {
		var uMap kdmodel.KdUserMap
		if err := s.db.Where("kd_user_id = ?", creatorID).First(&uMap).Error; err == nil && uMap.FeishuOpenId != "" {
			return uMap.FeishuOpenId
		}
		if openID, err := feishuCli.GetOpenIDByUserID(creatorID); err == nil && openID != "" {
			return openID
		}
	}
	return s.getFallbackOpenID()
}

func (s *FeishuApprovalService) resolveSecondLevelApprover(flow *kdmodel.KdFlow, bill map[string]interface{}, feishuCli *kdfeishu.Client) string {
	switch flow.KingdeeFormId {
	case "PUR_PurchaseOrder", "PUR_PriceCategory":
		return PurchaseManagerOpenID
	case "BD_SAL_PriceList":
		return SalesManagerOpenID
	default:
		return s.resolveApprover(bill, feishuCli)
	}
}

func (s *FeishuApprovalService) resolveApprover(bill map[string]interface{}, feishuCli *kdfeishu.Client) string {
	creatorID := str(bill["FCreatorId"])
	if creatorID != "" {
		var uMap kdmodel.KdUserMap
		if err := s.db.Where("kd_user_id = ?", creatorID).First(&uMap).Error; err == nil && uMap.ApproverOpenId != "" {
			return uMap.ApproverOpenId
		}
	}

	// 部门主管匹配
	dept := s.resolveDeptFromBill(bill)
	if dept != "" {
		if supervisor, ok := DeptSupervisorOpenIDs[dept]; ok && supervisor != "" {
			return supervisor
		}
	}

	// 销售员邮箱兼容
	salesman := str(bill["FSalesManId"])
	if strings.Contains(salesman, "@") {
		if openID, err := feishuCli.GetOpenIDByEmail(salesman); err == nil && openID != "" {
			return openID
		}
	}

	return s.getFallbackOpenID()
}

func (s *FeishuApprovalService) resolveDeptFromBill(bill map[string]interface{}) string {
	creatorName := str(bill["FCreatorId.FName"])
	if creatorName != "" {
		var uMap kdmodel.KdUserMap
		if err := s.db.Where("kd_user_name = ?", creatorName).First(&uMap).Error; err == nil && uMap.DeptName != "" {
			return uMap.DeptName
		}
	}
	return ""
}

func (s *FeishuApprovalService) queryEntries(flow *kdmodel.KdFlow, bill map[string]interface{}, entryFields []kdmodel.KdFlowField) []map[string]interface{} {
	if len(entryFields) == 0 {
		return nil
	}

	fieldKeys := make([]string, 0, len(entryFields))
	for _, f := range entryFields {
		if f.KingdeeField != "" {
			fieldKeys = append(fieldKeys, f.KingdeeField)
		}
	}
	if len(fieldKeys) == 0 {
		return nil
	}

	billNo := str(bill[flow.BillNoField])
	if billNo == "" {
		return nil
	}

	filter := fmt.Sprintf("%s = '%s'", flow.BillNoField, billNo)
	rows, err := s.kdService.ExecuteBillQuery(flow.KingdeeFormId, strings.Join(fieldKeys, ","), filter, 200)
	if err != nil || len(rows) == 0 {
		return nil
	}

	entryMaps := make([]map[string]interface{}, 0, len(rows))
	for _, row := range rows {
		item := make(map[string]interface{})
		for i, k := range fieldKeys {
			if i < len(row) {
				item[k] = row[i]
			}
		}
		entryMaps = append(entryMaps, item)
	}
	return entryMaps
}

func (s *FeishuApprovalService) renderTitle(flow *kdmodel.KdFlow, bill map[string]interface{}) string {
	tmpl := flow.TitleTemplate
	if tmpl == "" {
		tmpl = fmt.Sprintf("%s - {%s}", flow.FlowName, flow.BillNoField)
	}

	re := regexp.MustCompile(`\{(\w+)\}`)
	return re.ReplaceAllStringFunc(tmpl, func(m string) string {
		key := strings.Trim(m, "{}")
		return str(bill[key])
	})
}

func (s *FeishuApprovalService) getFallbackOpenID() string {
	var cfg kdmodel.KdConfig
	if err := s.db.Where("key = ?", "default_approver_openid").First(&cfg).Error; err == nil && cfg.Value != "" {
		return cfg.Value
	}
	return FinanceManagerOpenID
}

// CancelFeishuInstance 撤销飞书审批实例
func (s *FeishuApprovalService) CancelFeishuInstance(inst *kdmodel.KdInstance) error {
	if inst == nil || inst.InstanceID == "" {
		return nil
	}
	feishuCli, err := s.getFeishuClient()
	if err != nil {
		return err
	}
	return feishuCli.CancelInstance(inst.InstanceID)
}

// HandleCallback 处理飞书 Webhook 事件回调 (APPROVED / REJECTED)
func (s *FeishuApprovalService) HandleCallback(instanceCode, status string) (string, error) {
	var inst kdmodel.KdInstance
	if err := s.db.Where("instance_id = ?", instanceCode).First(&inst).Error; err != nil {
		return "mapping_not_found", nil
	}

	if inst.ApproveStatus != kdmodel.InstanceStatusPending {
		return "already_processed", nil
	}

	if status == "APPROVED" {
		inst.ApproveStatus = kdmodel.InstanceStatusPendingAudit
		s.db.Save(&inst)
		log.Printf("[KingdeeFeishu] 飞书审批通过，待异步回写金蝶 | billNo=%s instance_code=%s", inst.BillNo, instanceCode)
		return "pending_audit", nil
	}

	if status == "REJECTED" {
		inst.ApproveStatus = kdmodel.InstanceStatusRejected
		s.db.Save(&inst)
		log.Printf("[KingdeeFeishu] 飞书审批拒绝 | billNo=%s instance_code=%s", inst.BillNo, instanceCode)
		return "rejected", nil
	}

	return "no_action_" + status, nil
}

// AuditInstance 回写金蝶 Audit (先预检 IsAudited)
func (s *FeishuApprovalService) AuditInstance(inst *kdmodel.KdInstance) bool {
	var flow kdmodel.KdFlow
	if err := s.db.First(&flow, inst.FlowID).Error; err != nil {
		inst.ErrorMsg = "单据配置不存在"
		s.db.Save(inst)
		return false
	}

	cli, err := s.kdService.getClient()
	if err != nil {
		inst.ErrorMsg = err.Error()
		s.db.Save(inst)
		return false
	}

	// 1. 预检是否已审核
	audited, _ := cli.IsAudited(flow.KingdeeFormId, inst.BillFid)
	if audited {
		inst.ApproveStatus = kdmodel.InstanceStatusApproved
		inst.AuditResult = "1"
		inst.ErrorMsg = ""
		inst.RetryCount = 0
		s.db.Save(inst)
		s.linkOrder(inst)
		return true
	}

	// 2. 执行 Audit 审核 API
	if err := cli.Audit(flow.KingdeeFormId, inst.BillFid); err == nil {
		inst.ApproveStatus = kdmodel.InstanceStatusApproved
		inst.AuditResult = "1"
		inst.ErrorMsg = ""
		inst.RetryCount = 0
		s.db.Save(inst)
		s.linkOrder(inst)
		log.Printf("[KingdeeFeishu] 金蝶回写审核成功 | billNo=%s fid=%s", inst.BillNo, inst.BillFid)
		return true
	} else {
		inst.ErrorMsg = err.Error()
		inst.RetryCount++
		if inst.RetryCount >= 5 {
			inst.ApproveStatus = kdmodel.InstanceStatusAuditFailed
		}
		s.db.Save(inst)
		log.Printf("[KingdeeFeishu] 金蝶回写审核失败 | billNo=%s err=%v", inst.BillNo, err)
		return false
	}
}

func (s *FeishuApprovalService) linkOrder(inst *kdmodel.KdInstance) {
	// 可选扩展：回写订单关联状态
	s.db.Exec("UPDATE crm_order SET status = '1' WHERE del_flag = 0 AND (order_no = ? OR kingdee_fid = ?)", inst.BillNo, inst.BillFid)
}

func str(v interface{}) string {
	if v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}
