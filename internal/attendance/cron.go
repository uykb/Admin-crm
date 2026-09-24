package attendance

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/plugin/builtin/hikiot/client"

	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DailyReconciliation 每天 09:00 执行的查漏补缺 Cron Job（Pull 兜底模式）
// 目的：拉取前一天的物理数据（昨天 08:00 到 今天 08:00），防止 Event 漏推，并执行终态计算。
func DailyReconciliation(db *gorm.DB, cli *client.Client) error {
	now := time.Now()
	// 确保这是在 09:00 或之后运行的。需要核算的业务考勤日是昨天。
	// 这里严格取昨天的日期作为目标业务考勤日。
	targetAttDate := now.AddDate(0, 0, -1).Truncate(24 * time.Hour)

	// 物理拉取时间范围：昨天 08:00 到 今天 08:00
	sTime := time.Date(targetAttDate.Year(), targetAttDate.Month(), targetAttDate.Day(), 8, 0, 0, 0, targetAttDate.Location())
	eTime := sTime.AddDate(0, 0, 1)

	log.Printf("开始执行 %s 的考勤查漏补缺, API 拉取时间范围: %s 到 %s", targetAttDate.Format("2006-01-02"), sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"))

	// 1. 调用海康 API 拉取数据（已优化：利用内存 Token 和 pageSize=1000）
	records, err := cli.GetAttendanceRecords(sTime.Format("2006-01-02 15:04:05"), eTime.Format("2006-01-02 15:04:05"))
	if err != nil {
		log.Printf("Cron 查漏补缺：海康 API 拉取失败: %v", err)
		return fmt.Errorf("api pull failed: %w", err)
	}

	// 2. 在内存中按 PersonID 聚合所有打卡记录
	personPunches := make(map[string][]time.Time)
	for _, r := range records {
		pID := r.PersonNo
		if pID == "" {
			pID = r.PersonID
		}
		if pID == "" {
			continue
		}
		
		t, errT := time.Parse("2006-01-02 15:04:05", r.ClockTime)
		if errT != nil {
			t, _ = time.Parse("2006-01-02 15:04", r.ClockTime)
		}
		if t.IsZero() {
			continue
		}
		
		// 校验该物理打卡时间是否确实属于 targetAttDate 这一业务天
		if GetBusinessDate(t).Equal(targetAttDate) {
			personPunches[pID] = append(personPunches[pID], t)
		}
	}

	// 3. 遍历聚合后的数据，与本地快照比对，执行 Upsert
	err = db.Transaction(func(tx *gorm.DB) error {
		for pID, times := range personPunches {
			var snapshot model.AttDailySnapshot
			
			// 查现有记录
			res := tx.Where("user_id = ? AND att_date = ?", pID, targetAttDate).First(&snapshot)
			
			var existingTimes []time.Time
			if res.Error == nil {
				if snapshot.IsArchived {
					continue // 已封板，跳过
				}
				_ = json.Unmarshal(snapshot.RawPunchTimes, &existingTimes)
			}
			
			// 合并 API 数据和本地数据（以防推送数据里有，API里没有的极端情况，尽量保留最全的记录）
			mergedTimes := append(existingTimes, times...)
			mergedTimes = uniqueAndSortTimes(mergedTimes)
			
			// 只有当打卡次数有变化时（说明有漏推或缺失），或者是新记录时，才进行 Upsert
			if len(mergedTimes) != len(existingTimes) {
				matchRes := MatchShiftRules(targetAttDate, mergedTimes)
				
				firstPunch := mergedTimes[0]
				lastPunch := mergedTimes[len(mergedTimes)-1]
				rawBytes, _ := json.Marshal(mergedTimes)
				
				newSnapshot := model.AttDailySnapshot{
					UserID:        pID,
					AttDate:       targetAttDate,
					FirstPunch:    &firstPunch,
					LastPunch:     &lastPunch,
					PunchCount:    len(mergedTimes),
					MatchedShift:  matchRes.Shift,
					MatchedStatus: matchRes.Status,
					RawPunchTimes: datatypes.JSON(rawBytes),
				}
				
				if err := tx.Clauses(clause.OnConflict{
					Columns:   []clause.Column{{Name: "user_id"}, {Name: "att_date"}},
					DoUpdates: clause.AssignmentColumns([]string{
						"first_punch", "last_punch", "punch_count", 
						"matched_shift", "matched_status", "raw_punch_times", "updated_at",
					}),
				}).Create(&newSnapshot).Error; err != nil {
					log.Printf("更新员工 %s 考勤快照失败: %v", pID, err)
					return err
				}
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	log.Printf("Cron 查漏补缺执行完成，共核算 %d 名员工的考勤", len(personPunches))
	return nil
}
