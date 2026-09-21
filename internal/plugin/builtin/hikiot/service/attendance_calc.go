package service

import (
	"apeadmin-gin/internal/plugin/builtin/hikiot/model"
	"fmt"
	"sort"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// CalculateMonthlyAttendance 执行指定月份的智能考勤排班判定
func (s *HikService) CalculateMonthlyAttendance(monthStr string) error {
	// monthStr format: "2006-01"
	loc := time.Local
	monthStart, err := time.ParseInLocation("2006-01", monthStr, loc)
	if err != nil {
		return fmt.Errorf("月份格式错误，应为 YYYY-MM")
	}
	// 查询窗口：当月第一天 00:00 到 下个月第二天 12:00（为了覆盖月末最后一天的夜班跨天）
	queryStart := monthStart.AddDate(0, 0, -1) // 往前多查一天，以防月初夜班跨天
	queryEnd := monthStart.AddDate(0, 1, 1).Add(12 * time.Hour)

	var records []model.HkAttendance
	err = s.db.Where("clock_time >= ? AND clock_time <= ?", queryStart, queryEnd).
		Order("person_id ASC, clock_time ASC").
		Find(&records).Error
	if err != nil {
		return err
	}

	// 先把已有的人工修改记录查出来，用于跳过
	var existingResults []model.HkAttendanceResult
	s.db.Where("date LIKE ?", monthStr+"%").Find(&existingResults)
	manualMap := make(map[string]bool)
	for _, r := range existingResults {
		if r.IsManual {
			key := fmt.Sprintf("%s_%s", r.PersonID, r.Date)
			manualMap[key] = true
		}
	}

	// 按人员分组
	personRecords := make(map[string][]model.HkAttendance)
	personNames := make(map[string]string)
	personJobs := make(map[string]string)
	for _, r := range records {
		personRecords[r.PersonID] = append(personRecords[r.PersonID], r)
		personNames[r.PersonID] = r.PersonName
		personJobs[r.PersonID] = r.JobNo
	}

	var resultsToSave []model.HkAttendanceResult

	for pID, recs := range personRecords {
		// 聚合 Session
		type Session struct {
			First time.Time
			Last  time.Time
		}
		var sessions []Session
		var cur *Session

		for _, r := range recs {
			if cur == nil {
				cur = &Session{First: r.ClockTime, Last: r.ClockTime}
			} else {
				if r.ClockTime.Sub(cur.First) > 16*time.Hour {
					sessions = append(sessions, *cur)
					cur = &Session{First: r.ClockTime, Last: r.ClockTime}
				} else {
					cur.Last = r.ClockTime
				}
			}
		}
		if cur != nil {
			sessions = append(sessions, *cur)
		}

		for _, sess := range sessions {
			t1 := sess.First
			t2 := sess.Last
			dateStr := t1.Format("2006-01-02")
			
			// 只处理属于所选月份的归属日期
			if dateStr[:7] != monthStr {
				continue
			}

			// 如果人工已修改，跳过
			if manualMap[fmt.Sprintf("%s_%s", pID, dateStr)] {
				continue
			}

			shiftType := "异常"
			if t1.Equal(t2) {
				shiftType = "异常" // 只有一次打卡
			} else {
				h1 := t1.Hour()
				if h1 >= 4 && h1 < 12 {
					// 可能是白班
					if t1.Format("15:04:05") <= "08:00:59" && t2.Format("15:04:05") >= "20:00:00" {
						shiftType = "白班"
					}
				} else if h1 >= 16 && h1 <= 23 {
					// 可能是夜班
					// 夜班要求 t2 是第二天，并且时间 >= 08:00
					if t1.Format("15:04:05") <= "20:00:59" {
						if t2.Day() != t1.Day() && t2.Format("15:04:05") >= "08:00:00" {
							shiftType = "夜班"
						} else if t2.Sub(t1) >= 11*time.Hour+50*time.Minute {
							// 跨月或跨年的天数比较特殊，直接用时长判断。11小时50分以上算作完成夜班
							shiftType = "夜班"
						}
					}
				}
			}

			resultsToSave = append(resultsToSave, model.HkAttendanceResult{
				PersonID:   pID,
				PersonName: personNames[pID],
				JobNo:      personJobs[pID],
				Date:       dateStr,
				ShiftType:  shiftType,
				FirstClock: t1.Format("2006-01-02 15:04:05"),
				LastClock:  t2.Format("2006-01-02 15:04:05"),
				IsManual:   false,
			})
		}
	}

	if len(resultsToSave) > 0 {
		// 批量 Upsert
		err = s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "person_id"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"shift_type", "first_clock", "last_clock", "updated_at"}),
		}).Create(&resultsToSave).Error
	}

	return err
}

// MatrixRow 前端矩阵单行结构
type MatrixRow struct {
	PersonID   string            `json:"person_id"`
	PersonName string            `json:"person_name"`
	JobNo      string            `json:"job_no"`
	Days       map[string]string `json:"days"`    // key: "01", value: "白班"
	Details    map[string]model.HkAttendanceResult `json:"details"` // 完整信息
}

// GetMonthlyAttendanceMatrix 获取月度矩阵视图数据
func (s *HikService) GetMonthlyAttendanceMatrix(monthStr string) ([]MatrixRow, error) {
	var results []model.HkAttendanceResult
	err := s.db.Where("date LIKE ?", monthStr+"%").Order("date ASC").Find(&results).Error
	if err != nil {
		return nil, err
	}

	// 我们也需要把那些没打卡的人（本部门的人）列出来，或者仅列出有打卡记录的人。
	// 为了完整性，先查所有员工
	var allPersons []model.HkPerson
	s.db.Find(&allPersons)
	
	rowMap := make(map[string]*MatrixRow)
	for _, p := range allPersons {
		rowMap[p.PersonID] = &MatrixRow{
			PersonID:   p.PersonID,
			PersonName: p.PersonName,
			JobNo:      p.JobNo,
			Days:       make(map[string]string),
			Details:    make(map[string]model.HkAttendanceResult),
		}
	}

	for _, r := range results {
		if _, ok := rowMap[r.PersonID]; !ok {
			rowMap[r.PersonID] = &MatrixRow{
				PersonID:   r.PersonID,
				PersonName: r.PersonName,
				JobNo:      r.JobNo,
				Days:       make(map[string]string),
				Details:    make(map[string]model.HkAttendanceResult),
			}
		}
		day := r.Date[len(r.Date)-2:] // "2026-09-01" -> "01"
		rowMap[r.PersonID].Days[day] = r.ShiftType
		rowMap[r.PersonID].Details[day] = r
	}

	var out []MatrixRow
	for _, v := range rowMap {
		out = append(out, *v)
	}

	// 按姓名/工号排序
	sort.Slice(out, func(i, j int) bool {
		return out[i].PersonID < out[j].PersonID
	})

	return out, nil
}

// UpdateAttendanceResult 手工调整
func (s *HikService) UpdateAttendanceResult(personID, date, shiftType, remark string) error {
	var res model.HkAttendanceResult
	err := s.db.Where("person_id = ? AND date = ?", personID, date).First(&res).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// 新增
			var p model.HkPerson
			s.db.Where("person_id = ?", personID).First(&p)
			res = model.HkAttendanceResult{
				PersonID:   personID,
				PersonName: p.PersonName,
				JobNo:      p.JobNo,
				Date:       date,
				ShiftType:  shiftType,
				Remark:     remark,
				IsManual:   true,
			}
			return s.db.Create(&res).Error
		}
		return err
	}
	res.ShiftType = shiftType
	res.Remark = remark
	res.IsManual = true
	return s.db.Save(&res).Error
}
