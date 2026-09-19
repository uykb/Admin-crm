package service

import (
	"fmt"
	"strconv"
	"time"

	kdclient "apeadmin-gin/internal/plugin/builtin/kingdee/client"
	kdmodel "apeadmin-gin/internal/plugin/builtin/kingdee/model"

	"gorm.io/gorm"
)

type KingdeeService struct {
	db *gorm.DB
}

func NewKingdeeService(db *gorm.DB) *KingdeeService {
	return &KingdeeService{db: db}
}

type ConfigDTO struct {
	ServerURL string `json:"server_url"`
	DbID      string `json:"db_id"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Lcid      int    `json:"lcid"`
}

// GetConfig 获取配置信息
func (s *KingdeeService) GetConfig() (*ConfigDTO, error) {
	if s.db == nil {
		return &ConfigDTO{Lcid: 2052}, nil
	}

	var configs []kdmodel.KdConfig
	s.db.Find(&configs)

	cfgMap := make(map[string]string)
	for _, c := range configs {
		cfgMap[c.Key] = c.Value
	}

	lcid, _ := strconv.Atoi(cfgMap["kingdee_lcid"])
	if lcid <= 0 {
		lcid = 2052
	}

	return &ConfigDTO{
		ServerURL: cfgMap["kingdee_server_url"],
		DbID:      cfgMap["kingdee_db_id"],
		Username:  cfgMap["kingdee_username"],
		Password:  cfgMap["kingdee_password"],
		Lcid:      lcid,
	}, nil
}

// SaveConfig 保存配置
func (s *KingdeeService) SaveConfig(cfg *ConfigDTO) error {
	if s.db == nil {
		return fmt.Errorf("数据库连接不可用")
	}

	lcidStr := strconv.Itoa(cfg.Lcid)
	if cfg.Lcid <= 0 {
		lcidStr = "2052"
	}

	items := map[string]string{
		"kingdee_server_url": cfg.ServerURL,
		"kingdee_db_id":      cfg.DbID,
		"kingdee_username":   cfg.Username,
		"kingdee_password":   cfg.Password,
		"kingdee_lcid":       lcidStr,
	}

	for k, v := range items {
		var item kdmodel.KdConfig
		if err := s.db.Where("key = ?", k).First(&item).Error; err == nil {
			item.Value = v
			s.db.Save(&item)
		} else {
			s.db.Create(&kdmodel.KdConfig{Key: k, Value: v, IsPublic: false})
		}
	}

	return nil
}

func (s *KingdeeService) getClient() (*kdclient.Client, error) {
	cfg, err := s.GetConfig()
	if err != nil {
		return nil, err
	}
	if cfg.ServerURL == "" {
		return nil, fmt.Errorf("未配置内网金蝶云星空服务器地址，请先在配置中填写")
	}
	if cfg.DbID == "" || cfg.Username == "" {
		return nil, fmt.Errorf("未配置金蝶账套 ID 或用户名")
	}
	return kdclient.NewClient(cfg.ServerURL, cfg.DbID, cfg.Username, cfg.Password, cfg.Lcid), nil
}

// TestConnection 测试连通性
func (s *KingdeeService) TestConnection() error {
	cli, err := s.getClient()
	if err != nil {
		return err
	}
	return cli.Authenticate()
}

// QueryMaterials 物料/商品数据查询 (BD_MATERIAL)
func (s *KingdeeService) QueryMaterials(keyword string, limit int) ([]kdclient.MaterialItem, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}

	filterStr := "FForbidStatus = 'A'" // 仅未禁用
	if keyword != "" {
		filterStr += fmt.Sprintf(" AND (FName LIKE '%%%s%%' OR FNumber LIKE '%%%s%%' OR FSpecification LIKE '%%%s%%')", keyword, keyword, keyword)
	}

	if limit <= 0 {
		limit = 50
	}

	startTime := time.Now()
	rows, err := cli.ExecuteBillQuery(kdclient.BillQueryData{
		FormID:       "BD_MATERIAL",
		FieldKeys:    "FMaterialId,FNumber,FName,FSpecification",
		FilterString: filterStr,
		Limit:        limit,
	})
	s.logQuery("BD_MATERIAL", filterStr, len(rows), time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, err
	}

	items := make([]kdclient.MaterialItem, 0, len(rows))
	for _, row := range rows {
		if len(row) >= 4 {
			items = append(items, kdclient.MaterialItem{
				ID:            fmt.Sprintf("%v", row[0]),
				Number:        fmt.Sprintf("%v", row[1]),
				Name:          fmt.Sprintf("%v", row[2]),
				Specification: fmt.Sprintf("%v", row[3]),
			})
		}
	}

	return items, nil
}

// QueryCustomers 客户数据查询 (BD_Customer)
func (s *KingdeeService) QueryCustomers(keyword string, limit int) ([]kdclient.CustomerItem, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}

	filterStr := "FForbidStatus = 'A'"
	if keyword != "" {
		filterStr += fmt.Sprintf(" AND (FName LIKE '%%%s%%' OR FNumber LIKE '%%%s%%')", keyword, keyword)
	}

	if limit <= 0 {
		limit = 50
	}

	startTime := time.Now()
	rows, err := cli.ExecuteBillQuery(kdclient.BillQueryData{
		FormID:       "BD_Customer",
		FieldKeys:    "FCUSTID,FNumber,FName",
		FilterString: filterStr,
		Limit:        limit,
	})
	s.logQuery("BD_Customer", filterStr, len(rows), time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, err
	}

	items := make([]kdclient.CustomerItem, 0, len(rows))
	for _, row := range rows {
		if len(row) >= 3 {
			items = append(items, kdclient.CustomerItem{
				ID:     fmt.Sprintf("%v", row[0]),
				Number: fmt.Sprintf("%v", row[1]),
				Name:   fmt.Sprintf("%v", row[2]),
			})
		}
	}

	return items, nil
}

// QuerySalesOrders 销售订单查询 (SAL_SALEORDER)
func (s *KingdeeService) QuerySalesOrders(keyword string, limit int) ([]kdclient.SalesOrderItem, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}

	filterStr := ""
	if keyword != "" {
		filterStr = fmt.Sprintf("FBillNo LIKE '%%%s%%' OR FCustomerId.FName LIKE '%%%s%%'", keyword, keyword)
	}

	if limit <= 0 {
		limit = 50
	}

	startTime := time.Now()
	rows, err := cli.ExecuteBillQuery(kdclient.BillQueryData{
		FormID:       "SAL_SALEORDER",
		FieldKeys:    "FID,FBillNo,FDate,FCustomerId.FName,FDocumentStatus",
		FilterString: filterStr,
		Limit:        limit,
		OrderString:  "FDate DESC",
	})
	s.logQuery("SAL_SALEORDER", filterStr, len(rows), time.Since(startTime).Milliseconds())

	if err != nil {
		return nil, err
	}

	items := make([]kdclient.SalesOrderItem, 0, len(rows))
	for _, row := range rows {
		if len(row) >= 5 {
			items = append(items, kdclient.SalesOrderItem{
				ID:             fmt.Sprintf("%v", row[0]),
				BillNo:         fmt.Sprintf("%v", row[1]),
				Date:           fmt.Sprintf("%v", row[2]),
				CustomerName:   fmt.Sprintf("%v", row[3]),
				DocumentStatus: fmt.Sprintf("%v", row[4]),
			})
		}
	}

	return items, nil
}

// ExecuteBillQuery 通用单据列表查询
func (s *KingdeeService) ExecuteBillQuery(formID, fieldKeys, filterString string, limit int) ([][]interface{}, error) {
	cli, err := s.getClient()
	if err != nil {
		return nil, err
	}

	startTime := time.Now()
	rows, err := cli.ExecuteBillQuery(kdclient.BillQueryData{
		FormID:       formID,
		FieldKeys:    fieldKeys,
		FilterString: filterString,
		Limit:        limit,
	})
	s.logQuery(formID, filterString, len(rows), time.Since(startTime).Milliseconds())
	return rows, err
}

func (s *KingdeeService) logQuery(formID, filter string, count int, execTimeMs int64) {
	if s.db != nil {
		s.db.Create(&kdmodel.KdQueryLog{
			FormID:      formID,
			QueryFilter: filter,
			ResultCount: count,
			ExecTimeMs:  execTimeMs,
			CreatedBy:   "system",
			CreatedAt:   time.Now(),
		})
	}
}
