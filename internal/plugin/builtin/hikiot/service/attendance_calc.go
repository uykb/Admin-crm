package service

import (
	"apeadmin-gin/internal/plugin/builtin/hikiot/model"
	"fmt"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var DebugLogs []string

func logDebug(format string, args ...interface{}) {
	msg := fmt.Sprintf(format, args...)
	DebugLogs = append(DebugLogs, msg)
	if len(DebugLogs) > 100 {
		DebugLogs = DebugLogs[len(DebugLogs)-100:]
	}
	fmt.Println(msg)
}

// CalculateMonthlyAttendance 执行指定月份的智能考勤排班判定
func (s *HikService) CalculateMonthlyAttendance(monthStr string) error {
	return s.calculateAttendanceInternal(monthStr, false)
}

func (s *HikService) calculateAttendanceInternal(monthStr string, skipAPI bool) error {
	loc := time.FixedZone("CST", 8*3600)
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
	todayStr := time.Now().In(loc).Format("2006-01-02")

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

		// 找出 abnormalDates 中的最早和最晚日期，合并为单次或少量 API 请求
		var minDate, maxDate string
		for dateStr := range abnormalDates {
			if minDate == "" || dateStr < minDate {
				minDate = dateStr
			}
			if maxDate == "" || dateStr > maxDate {
				maxDate = dateStr
			}
		}

		if minDate != "" && maxDate != "" && !skipAPI {
			tMax, _ := time.Parse("2006-01-02", maxDate)
			eTime := tMax.AddDate(0, 0, 1).Format("2006-01-02")

			records, errRec := cli.GetAttendanceRecords(minDate, eTime)
			if errRec == nil && len(records) > 0 {
				for _, r := range records {
					t, _ := time.ParseInLocation("2006-01-02 15:04:05", r.ClockTime, loc)
					if t.IsZero() {
						t, _ = time.ParseInLocation("2006-01-02 15:04", r.ClockTime, loc)
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
	manualMap := make(map[string]bool)
	for _, r := range existingResults {
		if r.IsManual && r.ShiftType != "" && r.ShiftType != "异常" {
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
	// 补齐没有任何打卡记录的历史天数，自动置为"休息"
	// 说明：凡走到这里的 (person, date)，在主会话算法里没有产生任何归属记录，
	// 即真正无打卡（夜班工人次日 08:00 的下班卡已被归属到前一日的夜班会话，不会误计入本日）。
	todayStrLimit := time.Now().In(loc).Format("2006-01-02")
	existingMap := make(map[string]model.HkAttendanceResult)
	for _, er := range existingResults {
		existingMap[er.PersonID+"_"+er.Date] = er
	}
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
				// 已有有效非异常记录，不覆盖
				if er, found := existingMap[key]; found {
					validShift := er.ShiftType != "" && er.ShiftType != "异常" && er.ShiftType != "缺卡"
					if validShift {
						continue
					}
				}
				// 走到这里说明本日无任何归属打卡 → 休息
				resultMap[key] = model.HkAttendanceResult{
					PersonID:   p.PersonID,
					PersonName: p.PersonName,
					JobNo:      p.JobNo,
					Date:       dateStr,
					ShiftType:  "休息",
					IsManual:   false,
					Remark:     "智能判定无打卡",
				}
			}
		}
	}

	var resultsToSave []model.HkAttendanceResult
	for _, v := range resultMap {
		resultsToSave = append(resultsToSave, v)
	}

	logDebug("DEBUG calculateAttendanceInternal saving %d records. Ex: %v\n", len(resultsToSave), len(resultMap))

	if len(resultsToSave) > 0 {
		err = s.db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "person_id"}, {Name: "date"}},
			DoUpdates: clause.AssignmentColumns([]string{"shift_type", "first_clock", "last_clock", "remark", "updated_at"}),
		}).CreateInBatches(&resultsToSave, 100).Error
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

// CalculateSingleAttendance 执行指定人员指定日期的独立核算

// CalculateSingleAttendance 执行指定人员指定日期的独立核算
func (s *HikService) CalculateSingleAttendance(personID, dateStr string) error {
	// 1. 删除人工锁或旧记录
	err := s.db.Where("person_id = ? AND date = ?", personID, dateStr).Delete(&model.HkAttendanceResult{}).Error
	if err != nil {
		return err
	}

	// 2. 从海康接口拉取该日的原始打卡（拉取该日和次日）
	tDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		return err
	}
	eDate := tDate.AddDate(0, 0, 1).Format("2006-01-02")

	cli, errCli := s.GetClient()
	if errCli != nil {
		return errCli
	}
	records, errRec := cli.GetAttendanceRecords(dateStr, eDate)
	logDebug("DEBUG CalculateSingleAttendance records count: %d, err: %v\n", len(records), errRec)

	// 打印前3条记录的PersonID，帮助诊断
	for i, r := range records {
		if i >= 3 {
			break
		}
		logDebug("DEBUG sample record[%d]: PersonID=%q PersonNo=%q ClockTime=%q\n", i, r.PersonID, r.PersonNo, r.ClockTime)
	}

	loc := time.FixedZone("CST", 8*3600)
	if errRec == nil && len(records) > 0 {
		var toCreate []model.HkAttendance
		matched := 0
		for _, r := range records {
			// 宽松匹配：PersonID 或 PersonNo 包含目标ID
			pID := r.PersonID
			if pID == "" {
				pID = r.PersonNo
			}
			if pID != personID {
				continue
			}
			matched++
			// 解析时间，兼容有秒/无秒格式
			t, errParse := time.ParseInLocation("2006-01-02 15:04:05", r.ClockTime, loc)
			if errParse != nil {
				t, errParse = time.ParseInLocation("2006-01-02 15:04", r.ClockTime, loc)
			}
			if errParse != nil || t.IsZero() {
				logDebug("DEBUG parse failed ClockTime=%q\n", r.ClockTime)
				continue
			}
			toCreate = append(toCreate, model.HkAttendance{
				PersonID:   personID,
				PersonName: r.PersonName,
				ClockTime:  t,
			})
		}
		logDebug("DEBUG CalculateSingleAttendance matched=%d toCreate=%d\n", matched, len(toCreate))
		for _, v := range toCreate {
			s.db.Clauses(clause.OnConflict{DoNothing: true}).Create(&v)
		}
	}

	// 3. 执行本地的轻量级核算算法（跳过API请求）
	monthStr := dateStr[:7]
	return s.calculateAttendanceInternal(monthStr, true)
}
