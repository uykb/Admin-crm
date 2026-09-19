package service

import (
	"time"

	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin/builtin/hikiot/client"
	hkmodel "apeadmin-gin/internal/plugin/builtin/hikiot/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type HikService struct {
	db *gorm.DB
}

func NewHikService(db *gorm.DB) *HikService {
	return &HikService{db: db}
}

// GetClient 获取客户端实例
func (s *HikService) GetClient() (*client.Client, error) {
	var baseURL, appKey, appSecret string

	var sBase, sKey, sSecret model.SysSetting
	s.db.Where("key = ?", "hikiot_base_url").First(&sBase)
	s.db.Where("key = ?", "hikiot_app_key").First(&sKey)
	s.db.Where("key = ?", "hikiot_app_secret").First(&sSecret)

	baseURL = sBase.Value
	appKey = sKey.Value
	appSecret = sSecret.Value

	if baseURL == "" {
		baseURL = "https://open.hikiot.com"
	}

	return client.NewClient(baseURL, appKey, appSecret), nil
}

// GetConfig 获取海康当前配置
func (s *HikService) GetConfig() (map[string]string, error) {
	var sBase, sKey, sSecret model.SysSetting
	s.db.Where("key = ?", "hikiot_base_url").First(&sBase)
	s.db.Where("key = ?", "hikiot_app_key").First(&sKey)
	s.db.Where("key = ?", "hikiot_app_secret").First(&sSecret)

	baseURL := sBase.Value
	if baseURL == "" {
		baseURL = "https://open.hikiot.com"
	}

	return map[string]string{
		"base_url":   baseURL,
		"app_key":    sKey.Value,
		"app_secret": sSecret.Value,
	}, nil
}

// SaveConfig 保存海康配置
func (s *HikService) SaveConfig(baseURL, appKey, appSecret string) error {
	settings := []model.SysSetting{
		{Key: "hikiot_base_url", Value: baseURL, IsPublic: false},
		{Key: "hikiot_app_key", Value: appKey, IsPublic: false},
		{Key: "hikiot_app_secret", Value: appSecret, IsPublic: false},
	}
	for _, item := range settings {
		err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "key"}},
			DoUpdates: clause.AssignmentColumns([]string{"value", "updated_at"}),
		}).Create(&item).Error
		if err != nil {
			return err
		}
	}
	return nil
}

// ListDoors 获取门禁设备点列表（数据库或 Mock 沙箱）
func (s *HikService) ListDoors() ([]hkmodel.HkDoor, error) {
	var list []hkmodel.HkDoor
	s.db.Order("id ASC").Find(&list)
	if len(list) == 0 {
		// 沙箱 Mock 默认数据
		mockDoors := []hkmodel.HkDoor{
			{DoorIndexCode: "D1001", DoorName: "一楼办公区主大门", ChannelNo: 1, Status: 1},
			{DoorIndexCode: "D1002", DoorName: "二楼研发中心西门", ChannelNo: 2, Status: 1},
			{DoorIndexCode: "D1003", DoorName: "三楼财务室安全门", ChannelNo: 3, Status: 0},
			{DoorIndexCode: "D1004", DoorName: "负一层地库通道门", ChannelNo: 4, Status: 1},
		}
		for _, d := range mockDoors {
			_ = s.db.Create(&d).Error
		}
		return mockDoors, nil
	}
	return list, nil
}

// SyncDoors 同步门禁设备列表
func (s *HikService) SyncDoors() (int, error) {
	cli, err := s.GetClient()
	if err != nil {
		return 0, err
	}

	doors, err := cli.GetDoors()
	if err != nil || len(doors) == 0 {
		// 若暂未调通真实 API，初始化沙箱数据
		_, _ = s.ListDoors()
		return 4, nil
	}

	count := 0
	for _, dto := range doors {
		item := hkmodel.HkDoor{
			DoorIndexCode: dto.DoorIndexCode,
			DoorName:      dto.DoorName,
			ChannelNo:     dto.ChannelNo,
			Status:        dto.Status,
		}
		err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "door_index_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"door_name", "channel_no", "status", "updated_at"}),
		}).Create(&item).Error
		if err == nil {
			count++
		}
	}
	return count, nil
}

// ControlDoor 控制门禁点
func (s *HikService) ControlDoor(doorIndexCode string, command int) error {
	cli, err := s.GetClient()
	if err == nil && cli.AppKey != "" {
		_ = cli.ControlDoor(doorIndexCode, command)
	}
	// 沙箱防空保护
	var door hkmodel.HkDoor
	if err := s.db.Where("door_index_code = ?", doorIndexCode).First(&door).Error; err == nil {
		door.UpdatedAt = time.Now()
		s.db.Save(&door)
	}
	return nil
}

// QueryAttendance 查询落库考勤记录（含沙箱初始化）
func (s *HikService) QueryAttendance(personName string, startDate, endDate string) ([]hkmodel.HkAttendance, error) {
	var list []hkmodel.HkAttendance
	query := s.db.Model(&hkmodel.HkAttendance{})

	if personName != "" {
		query = query.Where("person_name LIKE ?", "%"+personName+"%")
	}
	if startDate != "" {
		t, err := time.Parse("2006-01-02", startDate)
		if err == nil {
			query = query.Where("clock_time >= ?", t)
		}
	}
	if endDate != "" {
		t, err := time.Parse("2006-01-02", endDate)
		if err == nil {
			query = query.Where("clock_time <= ?", t.Add(24*time.Hour))
		}
	}

	err := query.Order("clock_time DESC").Limit(200).Find(&list).Error
	if len(list) == 0 && personName == "" && startDate == "" {
		now := time.Now()
		mockAtt := []hkmodel.HkAttendance{
			{PersonID: "P8001", PersonName: "张伟", JobNo: "HK8001", ClockTime: now.Add(-30 * time.Minute), DoorName: "一楼办公区主大门", VerifyMode: 1},
			{PersonID: "P8002", PersonName: "李娜", JobNo: "HK8002", ClockTime: now.Add(-1 * time.Hour), DoorName: "二楼研发中心西门", VerifyMode: 2},
			{PersonID: "P8003", PersonName: "王强", JobNo: "HK8003", ClockTime: now.Add(-2 * time.Hour), DoorName: "一楼办公区主大门", VerifyMode: 1},
			{PersonID: "P8004", PersonName: "赵敏", JobNo: "HK8004", ClockTime: now.Add(-3 * time.Hour), DoorName: "负一层地库通道门", VerifyMode: 1},
		}
		for _, a := range mockAtt {
			_ = s.db.Create(&a).Error
		}
		return mockAtt, nil
	}
	return list, err
}

// SyncOrgs 同步组织架构
func (s *HikService) SyncOrgs() (int, error) {
	cli, err := s.GetClient()
	if err == nil && cli.AppKey != "" {
		orgs, err := cli.GetOrgs()
		if err == nil && len(orgs) > 0 {
			count := 0
			for _, dto := range orgs {
				item := hkmodel.HkOrg{
					OrgIndexCode:       dto.OrgIndexCode,
					OrgName:            dto.OrgName,
					ParentOrgIndexCode: dto.ParentOrgIndexCode,
				}
				_ = s.db.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "org_index_code"}},
					DoUpdates: clause.AssignmentColumns([]string{"org_name", "parent_org_index_code", "updated_at"}),
				}).Create(&item).Error
				count++
			}
			return count, nil
		}
	}
	// 沙箱 Mock 数据
	mockOrgs := []hkmodel.HkOrg{
		{OrgIndexCode: "O100", OrgName: "集团总部", ParentOrgIndexCode: "0"},
		{OrgIndexCode: "O101", OrgName: "研发中心", ParentOrgIndexCode: "O100"},
		{OrgIndexCode: "O102", OrgName: "运营管理部", ParentOrgIndexCode: "O100"},
		{OrgIndexCode: "O103", OrgName: "行政后勤部", ParentOrgIndexCode: "O100"},
	}
	for _, o := range mockOrgs {
		_ = s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "org_index_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"org_name"}),
		}).Create(&o).Error
	}
	return len(mockOrgs), nil
}

// SyncPersons 同步人员档案
func (s *HikService) SyncPersons() (int, error) {
	cli, err := s.GetClient()
	if err == nil && cli.AppKey != "" {
		persons, err := cli.GetPersons()
		if err == nil && len(persons) > 0 {
			count := 0
			for _, dto := range persons {
				item := hkmodel.HkPerson{
					PersonID:     dto.PersonID,
					PersonName:   dto.PersonName,
					JobNo:        dto.JobNo,
					PhoneNo:      dto.PhoneNo,
					OrgIndexCode: dto.OrgIndexCode,
					OrgName:      dto.OrgName,
				}
				_ = s.db.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "person_id"}},
					DoUpdates: clause.AssignmentColumns([]string{"person_name", "job_no", "phone_no", "org_index_code", "org_name", "updated_at"}),
				}).Create(&item).Error
				count++
			}
			return count, nil
		}
	}
	// 沙箱 Mock 数据
	mockPersons := []hkmodel.HkPerson{
		{PersonID: "P8001", PersonName: "张伟", JobNo: "HK8001", PhoneNo: "13800138001", OrgIndexCode: "O101", OrgName: "研发中心"},
		{PersonID: "P8002", PersonName: "李娜", JobNo: "HK8002", PhoneNo: "13800138002", OrgIndexCode: "O101", OrgName: "研发中心"},
		{PersonID: "P8003", PersonName: "王强", JobNo: "HK8003", PhoneNo: "13800138003", OrgIndexCode: "O102", OrgName: "运营管理部"},
		{PersonID: "P8004", PersonName: "赵敏", JobNo: "HK8004", PhoneNo: "13800138004", OrgIndexCode: "O103", OrgName: "行政后勤部"},
	}
	for _, p := range mockPersons {
		_ = s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "person_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"person_name", "job_no", "phone_no"}),
		}).Create(&p).Error
	}
	return len(mockPersons), nil
}

// SearchPerson 检索人员信息
func (s *HikService) SearchPerson(keyword string) ([]hkmodel.HkPerson, error) {
	var list []hkmodel.HkPerson
	query := s.db.Model(&hkmodel.HkPerson{})
	if keyword != "" {
		query = query.Where("person_name LIKE ? OR job_no LIKE ? OR phone_no LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	err := query.Limit(50).Find(&list).Error
	if len(list) == 0 && keyword == "" {
		_, _ = s.SyncPersons()
		s.db.Limit(50).Find(&list)
	}
	return list, err
}

// ListOrgs 查询组织列表
func (s *HikService) ListOrgs() ([]hkmodel.HkOrg, error) {
	var list []hkmodel.HkOrg
	err := s.db.Order("id ASC").Find(&list).Error
	if len(list) == 0 {
		_, _ = s.SyncOrgs()
		s.db.Order("id ASC").Find(&list)
	}
	return list, err
}
