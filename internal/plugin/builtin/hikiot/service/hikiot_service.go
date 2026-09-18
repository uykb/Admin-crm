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

// SyncOrgs 同步组织机构
func (s *HikService) SyncOrgs() (int, error) {
	cli, err := s.GetClient()
	if err != nil {
		return 0, err
	}

	orgs, err := cli.GetOrgs()
	if err != nil {
		return 0, err
	}

	count := 0
	for _, dto := range orgs {
		item := hkmodel.HkOrg{
			OrgIndexCode:       dto.OrgIndexCode,
			OrgName:            dto.OrgName,
			ParentOrgIndexCode: dto.ParentOrgIndexCode,
		}
		err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "org_index_code"}},
			DoUpdates: clause.AssignmentColumns([]string{"org_name", "parent_org_index_code", "updated_at"}),
		}).Create(&item).Error
		if err == nil {
			count++
		}
	}
	return count, nil
}

// SyncPersons 同步人员档案
func (s *HikService) SyncPersons() (int, error) {
	cli, err := s.GetClient()
	if err != nil {
		return 0, err
	}

	persons, err := cli.GetPersons()
	if err != nil {
		return 0, err
	}

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
		err := s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "person_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"person_name", "job_no", "phone_no", "org_index_code", "org_name", "updated_at"}),
		}).Create(&item).Error
		if err == nil {
			count++
		}
	}
	return count, nil
}

// SyncDoors 同步门禁设备列表
func (s *HikService) SyncDoors() (int, error) {
	cli, err := s.GetClient()
	if err != nil {
		return 0, err
	}

	doors, err := cli.GetDoors()
	if err != nil {
		return 0, err
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
	if err != nil {
		return err
	}
	return cli.ControlDoor(doorIndexCode, command)
}

// QueryAttendance 查询落库考勤记录
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
	return list, err
}

// SearchPerson 检索人员信息
func (s *HikService) SearchPerson(keyword string) ([]hkmodel.HkPerson, error) {
	var list []hkmodel.HkPerson
	err := s.db.Where("person_name LIKE ? OR job_no LIKE ? OR phone_no LIKE ?",
		"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%").Limit(50).Find(&list).Error
	return list, err
}
