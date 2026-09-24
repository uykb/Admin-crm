package service

import (
	"fmt"
	"sort"
	"strings"
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
	var baseURL, appKey, appSecret, userToken string

	var sBase, sKey, sSecret, sUser model.SysSetting
	s.db.Where("key = ?", "hikiot_base_url").First(&sBase)
	s.db.Where("key = ?", "hikiot_app_key").First(&sKey)
	s.db.Where("key = ?", "hikiot_app_secret").First(&sSecret)
	s.db.Where("key = ?", "hikiot_user_token").First(&sUser)

	baseURL = sBase.Value
	appKey = sKey.Value
	appSecret = sSecret.Value
	userToken = sUser.Value

	if baseURL == "" {
		baseURL = "https://open-api.hikiot.com"
	}

	c := client.NewClient(baseURL, appKey, appSecret)
	if userToken != "" {
		c.SetUserAccessToken(userToken)
	}
	return c, nil
}

// GetConfig 获取海康当前配置
func (s *HikService) GetConfig() (map[string]string, error) {
	var sBase, sKey, sSecret, sUser model.SysSetting
	s.db.Where("key = ?", "hikiot_base_url").First(&sBase)
	s.db.Where("key = ?", "hikiot_app_key").First(&sKey)
	s.db.Where("key = ?", "hikiot_app_secret").First(&sSecret)
	s.db.Where("key = ?", "hikiot_user_token").First(&sUser)

	baseURL := sBase.Value
	if baseURL == "" {
		baseURL = "https://open-api.hikiot.com"
	}

	return map[string]string{
		"base_url":   baseURL,
		"app_key":    sKey.Value,
		"app_secret": sSecret.Value,
		"user_token": sUser.Value,
	}, nil
}

// SaveConfig 保存海康配置
func (s *HikService) SaveConfig(baseURL, appKey, appSecret, userToken string) error {
	settings := []model.SysSetting{
		{Key: "hikiot_base_url", Value: baseURL, IsPublic: false},
		{Key: "hikiot_app_key", Value: appKey, IsPublic: false},
		{Key: "hikiot_app_secret", Value: appSecret, IsPublic: false},
		{Key: "hikiot_user_token", Value: userToken, IsPublic: false},
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

// ListDoors 获取门禁设备点列表（纯真实数据）
func (s *HikService) ListDoors() ([]hkmodel.HkDoor, error) {

	var list []hkmodel.HkDoor
	s.db.Order("id ASC").Find(&list)

	// 如果数据库为空，尝试自动触发真实 API 同步
	cli, errCli := s.GetClient()
	if len(list) == 0 && errCli == nil && cli.AppKey != "" && cli.AppSecret != "" {
		_, errSync := s.SyncDoors()
		if errSync == nil {
			s.db.Order("id ASC").Find(&list)
		}
	}

	if list == nil {
		list = []hkmodel.HkDoor{}
	}
	return list, nil
}

// TestConnection 测试海康互联云端开放平台 API 连通性（纯云端模式）
func (s *HikService) TestConnection() error {
	cli, err := s.GetClient()
	if err != nil {
		return err
	}
	if cli.AppKey == "" || cli.AppSecret == "" {
		return fmt.Errorf("未配置海康 AppKey 或 AppSecret，请先在插件配置中填入海康互联凭据")
	}

	tokenData, errCloud := cli.ExchangeAppToken()
	if errCloud != nil {
		return fmt.Errorf("海康互联云 API 认证失败: %w", errCloud)
	}

	if tokenData == nil || tokenData.AppAccessToken == "" {
		return fmt.Errorf("海康互联云 API 认证异常: 未能取得有效 appAccessToken")
	}

	return nil
}

// SyncDoors 同步门禁设备列表（纯真实数据）
func (s *HikService) SyncDoors() (int, error) {
	// 清理历史 Mock 沙箱数据
	s.db.Exec("DELETE FROM hk_door WHERE door_index_code LIKE 'D100%'")

	cli, err := s.GetClient()
	if err != nil {
		return 0, err
	}
	if cli.AppKey == "" || cli.AppSecret == "" {
		return 0, fmt.Errorf("未配置海康 AppKey 或 AppSecret，请先在插件配置中填写海康凭据")
	}

	doors, err := cli.GetDoors()
	if err != nil {
		return 0, fmt.Errorf("海康 API 门禁同步失败: %w", err)
	}

	if len(doors) == 0 {
		return 0, fmt.Errorf("海康 API 连通正常，但未返回门禁点。请确认：1. 海康互联账号下已添加/关联门禁设备；2. 应用已开通门禁相关 API 权限")
	}

	count := 0
	for _, dto := range doors {
		code := dto.GetCode()
		name := dto.GetName()
		if code == "" {
			continue
		}
		chNo := dto.GetChannelNo()
		status := dto.GetStatus()
		item := hkmodel.HkDoor{
			DoorIndexCode: code,
			DoorName:      name,
			ChannelNo:     chNo,
			Status:        status,
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
	var door hkmodel.HkDoor
	if err := s.db.Where("door_index_code = ?", doorIndexCode).First(&door).Error; err == nil {
		door.UpdatedAt = time.Now()
		s.db.Save(&door)
	}
	return nil
}

// QueryAttendance 实时按条件查询考勤记录（优先按条件调 API 直拉，不写库；API 无响应时备用查本地表）
func (s *HikService) QueryAttendance(personName string, startDate, endDate string) ([]hkmodel.HkAttendance, error) {
	// 1. 优先尝试从海康开放平台 API 实时拉取
	cli, errCli := s.GetClient()
	if errCli == nil && cli.AppKey != "" && cli.AppSecret != "" {
		sTime := startDate
		eTime := endDate
		if sTime == "" {
			sTime = time.Now().AddDate(0, 0, -7).Format("2006-01-02 00:00:00")
		}
		if eTime == "" {
			eTime = time.Now().Format("2006-01-02 23:59:59")
		}
		records, errRec := cli.GetAttendanceRecords(sTime, eTime)
		if errRec == nil {
			var apiList []hkmodel.HkAttendance
			for _, r := range records {
				t, _ := time.Parse("2006-01-02 15:04:05", r.ClockTime)
				if t.IsZero() {
					t, _ = time.Parse("2006-01-02 15:04", r.ClockTime)
				}

				pID := r.PersonNo
				if pID == "" {
					pID = r.PersonID
				}
				jNo := r.JobNumber
				if jNo == "" {
					jNo = r.JobNo
				}
				devName := r.DeviceName
				if devName == "" {
					devName = r.Address
				}

				// 内存按姓名/工号条件过滤
				if personName != "" {
					if !strings.Contains(r.PersonName, personName) && !strings.Contains(jNo, personName) {
						continue
					}
				}

				item := hkmodel.HkAttendance{
					PersonID:   pID,
					PersonName: r.PersonName,
					JobNo:      jNo,
					ClockTime:  t,
					DoorName:   devName,
					VerifyMode: r.VerifyMode,
				}
				apiList = append(apiList, item)
			}
			// 按时间倒序
			sort.Slice(apiList, func(i, j int) bool {
				return apiList[i].ClockTime.After(apiList[j].ClockTime)
			})
			return apiList, nil
		}
	}

	// 2. 备用方案：API 未配置或拉取失败时，退回到数据库查询已有历史记录
	var list []hkmodel.HkAttendance
	query := s.db.Model(&hkmodel.HkAttendance{}).
		Where("person_id IN (SELECT person_id FROM hk_person WHERE org_index_code = ? OR org_index_code = '' OR org_index_code IS NULL)", "BM54141022")

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

	err := query.Order("clock_time DESC").Limit(2000).Find(&list).Error
	if list == nil {
		list = []hkmodel.HkAttendance{}
	}
	return list, err
}

// SyncOrgs 同步组织架构（纯真实数据）
func (s *HikService) SyncOrgs() (int, error) {
	// 清理历史 Mock 沙箱数据
	s.db.Exec("DELETE FROM hk_org WHERE org_index_code LIKE 'O10%'")

	cli, err := s.GetClient()
	if err == nil && cli.AppKey != "" {
		orgs, err := cli.GetOrgs()
		if err == nil {
			count := 0
			for _, dto := range orgs {
				code := dto.GetCode()
				name := dto.GetName()
				if code == "" {
					continue
				}
				item := hkmodel.HkOrg{
					OrgIndexCode:       code,
					OrgName:            name,
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
	return 0, nil
}

// SyncPersons 手动同步人员档案（纯真实数据，指定部门 BM54141022）
func (s *HikService) SyncPersons() (int, error) {
	// 清理历史 Mock 沙箱数据
	s.db.Exec("DELETE FROM hk_person WHERE person_id LIKE 'P800%'")

	cli, err := s.GetClient()
	if err == nil && cli.AppKey != "" {
		persons, err := cli.GetPersons()
		if err == nil {
			count := 0
			for _, dto := range persons {
				id := dto.GetID()
				name := dto.GetName()
				if id == "" {
					continue
				}
				orgCode := dto.GetOrgCode()
				if orgCode == "" {
					orgCode = "BM54141022"
				}
				item := hkmodel.HkPerson{
					PersonID:     id,
					PersonName:   name,
					JobNo:        dto.GetJobNo(),
					PhoneNo:      dto.GetPhone(),
					OrgIndexCode: orgCode,
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
	return 0, nil
}

// SearchPerson 检索人员信息（仅针对指定部门 BM54141022）
func (s *HikService) SearchPerson(keyword string) ([]hkmodel.HkPerson, error) {
	// 清理历史 Mock 沙箱数据
	s.db.Exec("DELETE FROM hk_person WHERE person_id LIKE 'P800%'")

	var list []hkmodel.HkPerson
	query := s.db.Model(&hkmodel.HkPerson{}).
		Where("org_index_code = ? OR org_index_code = '' OR org_index_code IS NULL", "BM54141022")

	if keyword != "" {
		query = query.Where("person_name LIKE ? OR job_no LIKE ? OR phone_no LIKE ?",
			"%"+keyword+"%", "%"+keyword+"%", "%"+keyword+"%")
	}
	err := query.Find(&list).Error
	if len(list) == 0 && keyword == "" {
		_, _ = s.SyncPersons()
		s.db.Where("org_index_code = ? OR org_index_code = '' OR org_index_code IS NULL", "BM54141022").Find(&list)
	}
	if list == nil {
		list = []hkmodel.HkPerson{}
	}
	return list, err
}

// ListOrgs 查询组织列表（纯真实数据）
func (s *HikService) ListOrgs() ([]hkmodel.HkOrg, error) {
	// 清理历史 Mock 沙箱数据
	s.db.Exec("DELETE FROM hk_org WHERE org_index_code LIKE 'O10%'")

	var list []hkmodel.HkOrg
	err := s.db.Order("id ASC").Find(&list).Error
	if len(list) == 0 {
		_, _ = s.SyncOrgs()
		s.db.Order("id ASC").Find(&list)
	}
	if list == nil {
		list = []hkmodel.HkOrg{}
	}
	return list, err
}
