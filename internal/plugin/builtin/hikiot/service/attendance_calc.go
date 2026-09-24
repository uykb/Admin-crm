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
	loc := time.Local
	monthStart, err := time.ParseInLocation("2006-01", monthStr, loc)
	if err != nil {
		return fmt.Errorf("月份格式错误，应为 YYYY-MM")
	}

	// 1. 找出当月存在的异常或缺失日期（截至今天）
	var existingResults []model.HkAttendanceResult
	s.db.Where("date LIKE ?", monthStr+"%").Find(&existingResults)
	
	// 获取部门人员
	var allPersons []model.HkPerson
	s.db.Where("org_index_code = ?", "BM54141022").Find(&allPersons)

	normalMap := make(map[string]bool)
	for _, r := range existingResults {
		if r.ShiftType != "异常" && r.ShiftType != "缺卡" && r.ShiftType != "空白" && r.ShiftType != "" {
			normalMap[r.PersonID+"_"+r.Date] = true
		}
	}

	abnormalDates := make(map[string]bool)
	todayStr := time.Now().Format("2006-01-02")
	
	// 遍历当月每一天直到今天，检查是否有人考勤异常
	for d := 1; d <= 31; d++ {
		dateStr := fmt.Sprintf("%s-%02d", monthStr, d)
		if dateStr > todayStr {
			break
		}
		// 校验该日期是否有效
		if _, err := time.Parse("2006-01-02", dateStr); err != nil {
			continue
		}
		
		isAbnormal := false
		for _, p := range allPersons {
			if !normalMap[p.PersonID+"_"+dateStr] {
				isAbnormal = true
				break
			}
		}
		if isAbnormal {
			abnormalDates[dateStr] = true
		}
	}

	var rawRecords []model.HkAttendance
	cli, errCli := s.GetClient()
	
	if errCli == nil && cli.AppKey != "" && cli.AppSecret != "" {
		toCreateMap := make(map[string]model.HkAttendance)
		
		// 仅针对异常的日期去向海康 API 发起精准请求（大幅节省接口调用量）
		for dateStr := range abnormalDates {
			sTime := dateStr
			// 对于异常的当天，必须把下一天的数据也拉过来，防止“夜班”跨天导致的缺卡误判
			tDate, _ := time.Parse("2006-01-02", dateStr)
			eTime := tDate.AddDate(0, 0, 1).Format("2006-01-02")
			
			records, errRec := cli.GetAttendanceRecords(sTime, eTime)
			if errRec == nil && len(records) > 0 {
				for _, r := range records {
					t, _ := time.Parse("2006-01-02 15:04:05", r.ClockTime)
					if t.IsZero() {
						t, _ = time.Parse("2006-01-02 15:04", r.ClockTime)
					}
					if t.IsZero() {
						continue
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
					
					// 智能映射验证方式
					vMode := r.VerifyMode
					if r.WayOfClock != "" {
						if strings.Contains(r.WayOfClock, "脸") {
							vMode = 1
						} else if strings.Contains(r.WayOfClock, "卡") {
							vMode = 2
						} else if strings.Contains(r.WayOfClock, "指纹") {
							vMode = 3
						} else {
							vMode = 1
						}
					} else if vMode == 0 {
						vMode = 1
					}

					key := fmt.Sprintf("%s_%s", pID, t.Format("2006-01-02 15:04:05"))
					toCreateMap[key] = model.HkAttendance{
						PersonID:   pID,
						PersonName: r.PersonName,
						JobNo:      jNo,
						ClockTime:  t,
						DoorName:   devName,
						VerifyMode: vMode,
					}
				}
			}
		}
		
		for _, v := range toCreateMap {
			rawRecords = append(rawRecords, v)
		}
		// 按时间升序排序
		sort.Slice(rawRecords, func(i, j int) bool {
			return rawRecords[i].ClockTime.Before(rawRecords[j].ClockTime)
		})

		// 确保人员信息也是最新的
		_, _ = s.SyncPersons()
	}

	// 2. 备用方案：如果 API 无数据，退回到从本地数据库拉取历史全量考勤流水（兼容离线模式）
	if len(rawRecords) == 0 {
		queryStart := monthStart.AddDate(0, 0, -1)
		queryEnd := monthStart.AddDate(0, 1, 2)
		_ = s.db.Where("clock_time >= ? AND clock_time < ? AND person_id IN (SELECT person_id FROM hk_person WHERE org_index_code = ?)", queryStart, queryEnd, "BM54141022").
			Order("person_id ASC, clock_time ASC").
			Find(&rawRecords).Error
	}

	// 限制为 BM54141022 部门范围
	var bmPersons []model.HkPerson
	s.db.Where("org_index_code = ?", "BM54141022").Find(&bmPersons)
	bmMap := make(map[string]bool)
	for _, p := range bmPersons {
		bmMap[p.PersonID] = true
	}

	var records []model.HkAttendance
	for _, r := range rawRecords {
		if len(bmMap) == 0 || bmMap[r.PersonID] {
			records = append(records, r)
		}
	}

	// 查出已被人工修改过的记录
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
		if r.PersonName != "" {
			personNames[r.PersonID] = r.PersonName
		}
		if r.JobNo != "" {
			personJobs[r.PersonID] = r.JobNo
		}
	}

	// 仅获取 BM54141022 部门人员
	var allPersons []model.HkPerson
	s.db.Where("org_index_code = ?", "BM54141022").Find(&allPersons)
	for _, p := range allPersons {
		if _, ok := personNames[p.PersonID]; !ok {
			personNames[p.PersonID] = p.PersonName
			personJobs[p.PersonID] = p.JobNo
		}
	}

	resultMap := make(map[string]model.HkAttendanceResult)

	for pID, recs := range personRecords {
		// 会话分组 (Session Grouping)
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
				// 如果距离会话首卡超过 16 小时，认为进入了下一个班次会话
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
			dur := t2.Sub(t1)
			h1 := t1.Hour()

			// 计算归属日期
			var dateStr string
			if h1 >= 4 && h1 < 16 {
				// 白班候选：归属于当天
				dateStr = t1.Format("2006-01-02")
			} else {
				// 夜班候选
				if h1 < 4 {
					// 凌晨打卡 (如 01:00)，说明是昨晚开始的夜班
					dateStr = t1.AddDate(0, 0, -1).Format("2006-01-02")
				} else {
					// 16:00 - 23:59 打卡，归属于当天
					dateStr = t1.Format("2006-01-02")
				}
			}

			// 只处理属于所选月份的归属日期
			if len(dateStr) < 7 || dateStr[:7] != monthStr {
				continue
			}

			// 如果人工已修改，跳过
			if manualMap[fmt.Sprintf("%s_%s", pID, dateStr)] {
				continue
			}

			shiftType := "异常"
			remark := ""

			if t1.Equal(t2) {
				shiftType = "异常"
				remark = "仅单次打卡"
			} else if dur < 4*time.Hour {
				shiftType = "异常"
				remark = "打卡间隔<4小时"
			} else if h1 >= 4 && h1 < 16 {
				// 白班判断逻辑 (标准: 08:00 - 20:00，允许弹性)
				if t1.Format("15:04:05") <= "08:30:00" && t2.Format("15:04:05") >= "19:30:00" {
					shiftType = "白班"
				} else if dur >= 8*time.Hour {
					shiftType = "白班"
					if t1.Format("15:04:05") > "08:30:00" {
						remark = "迟到"
					} else if t2.Format("15:04:05") < "19:30:00" {
						remark = "早退"
					}
				} else {
					shiftType = "异常"
					remark = "工时不足"
				}
			} else {
				// 夜班判断逻辑 (标准: 20:00 - 次日08:00，允许弹性)
				if t1.Format("15:04:05") <= "20:30:00" && (t2.Format("15:04:05") >= "07:30:00" || t2.Day() != t1.Day()) && dur >= 8*time.Hour {
					shiftType = "夜班"
				} else if dur >= 8*time.Hour {
					shiftType = "夜班"
				} else {
					shiftType = "异常"
					remark = "工时不足"
				}
			}

			newRes := model.HkAttendanceResult{
				PersonID:   pID,
				PersonName: personNames[pID],
				JobNo:      personJobs[pID],
				Date:       dateStr,
				ShiftType:  shiftType,
				FirstClock: t1.Format("2006-01-02 15:04:05"),
				LastClock:  t2.Format("2006-01-02 15:04:05"),
				IsManual:   false,
				Remark:     remark,
			}

			key := fmt.Sprintf("%s_%s", pID, dateStr)
			if old, exists := resultMap[key]; exists {
				// 如果同一个 (PersonID, Date) 计算出了多个结果，内存去重防 Postgres 报错 (SQLSTATE 21000)
				if (newRes.ShiftType == "白班" || newRes.ShiftType == "夜班") && old.ShiftType == "异常" {
					resultMap[key] = newRes
				} else if old.ShiftType == "异常" && newRes.ShiftType == "异常" {
					if newRes.FirstClock < old.FirstClock {
						old.FirstClock = newRes.FirstClock
					}
					if newRes.LastClock > old.LastClock {
						old.LastClock = newRes.LastClock
					}
					resultMap[key] = old
				}
			} else {
				resultMap[key] = newRes
			}
		}
	}

	// 补齐没有任何打卡记录的历史天数，自动置为“休息”，防止下次继续当做异常请求API
	todayStrLimit := time.Now().Format("2006-01-02")
	for _, p := range allPersons {
		for d := 1; d <= 31; d++ {
			dateStr := fmt.Sprintf("%s-%02d", monthStr, d)
			if dateStr >= todayStrLimit {
				break
			}
			if _, err := time.Parse("2006-01-02", dateStr); err != nil {
				continue
			}
			key := fmt.Sprintf("%s_%s", p.PersonID, dateStr)
			if _, exists := resultMap[key]; !exists {
				// 检查原来是否已经有手工记录或正常记录
				hasNormal := false
				for _, er := range existingResults {
					if er.PersonID == p.PersonID && er.Date == dateStr && (er.IsManual || er.ShiftType == "请假" || er.ShiftType == "休息") {
						hasNormal = true
						break
					}
				}
				if !hasNormal {
					resultMap[key] = model.HkAttendanceResult{
						PersonID:   p.PersonID,
						PersonName: p.PersonName,
						JobNo:      p.JobNo,
						Date:       dateStr,
						ShiftType:  "休息", // 标记为休息，停止对该日期的 API 轮询
						IsManual:   false,
						Remark:     "智能判定无打卡",
					}
				}
			}
		}
	}

	var resultsToSave []model.HkAttendanceResult
	for _, v := range resultMap {
		resultsToSave = append(resultsToSave, v)
	}

	if len(resultsToSave) > 0 {
		err = s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "person_id"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"shift_type", "first_clock", "last_clock", "remark", "updated_at"}),
		}).Create(&resultsToSave).Error
	}

	return err
}

// MatrixRow 前端矩阵单行结构
type MatrixRow struct {
	PersonID   string                              `json:"person_id"`
	PersonName string                              `json:"person_name"`
	JobNo      string                              `json:"job_no"`
	Days       map[string]string                   `json:"days"`    // key: "01", value: "白班"
	Details    map[string]model.HkAttendanceResult `json:"details"` // 完整信息
}

// GetMonthlyAttendanceMatrix 获取月度矩阵视图数据（仅针对 BM54141022 部门）
func (s *HikService) GetMonthlyAttendanceMatrix(monthStr string) ([]MatrixRow, error) {
	var allPersons []model.HkPerson
	s.db.Where("org_index_code = ?", "BM54141022").Find(&allPersons)

	validPersonIDs := make(map[string]bool)
	rowMap := make(map[string]*MatrixRow)
	for _, p := range allPersons {
		validPersonIDs[p.PersonID] = true
		rowMap[p.PersonID] = &MatrixRow{
			PersonID:   p.PersonID,
			PersonName: p.PersonName,
			JobNo:      p.JobNo,
			Days:       make(map[string]string),
			Details:    make(map[string]model.HkAttendanceResult),
		}
	}

	var results []model.HkAttendanceResult
	err := s.db.Where("date LIKE ?", monthStr+"%").Order("date ASC").Find(&results).Error
	if err != nil {
		return nil, err
	}

	for _, r := range results {
		if !validPersonIDs[r.PersonID] {
			continue // 过滤非 BM54141022 部门人员
		}
		if len(r.Date) >= 10 {
			day := r.Date[len(r.Date)-2:]
			rowMap[r.PersonID].Days[day] = r.ShiftType
			rowMap[r.PersonID].Details[day] = r
		}
	}

	var out []MatrixRow
	for _, v := range rowMap {
		out = append(out, *v)
	}

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
