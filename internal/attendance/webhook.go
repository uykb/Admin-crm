package attendance

import (
	"encoding/json"
	"net/http"
	"sort"
	"time"

	"apeadmin-gin/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// HandleHikiotEvent 接收并处理海康打卡事件订阅推送
// 这是 Push(实时) 模式的入口，实时更新当天的考勤快照。
func HandleHikiotEvent(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var rawPayload map[string]interface{}
		if err := c.ShouldBindJSON(&rawPayload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"msg": "invalid payload"})
			return
		}

		// 提取真正包含事件数据的对象（兼容 eventInfo, data 或平铺格式）
		var dataMap map[string]interface{}
		if ei, ok := rawPayload["eventInfo"].(map[string]interface{}); ok {
			dataMap = ei
		} else if dt, ok := rawPayload["data"].(map[string]interface{}); ok {
			dataMap = dt
		} else {
			dataMap = rawPayload
		}

		// 动态提取 personID
		var personID string
		for _, key := range []string{"personNo", "employeeNo", "personId"} {
			if val, ok := dataMap[key].(string); ok && val != "" {
				personID = val
				break
			}
		}

		// 动态提取打卡时间
		var clockTimeStr string
		for _, key := range []string{"gmtCreate", "actionTime", "clockTime", "happenTime", "time"} {
			if ct, ok := dataMap[key].(string); ok && ct != "" {
				clockTimeStr = ct
				break
			}
		}

		// 忽略无关或无效事件
		if personID == "" || clockTimeStr == "" {
			c.JSON(http.StatusOK, gin.H{"msg": "ignore non-punch event"})
			return
		}

		// 处理带 T 的时间格式或者普通格式
		var punchTime time.Time
		var err error
		if len(clockTimeStr) > 10 && clockTimeStr[10] == 'T' {
			punchTime, err = time.Parse("2006-01-02T15:04:05Z07:00", clockTimeStr) // ISO8601
		} else {
			punchTime, err = time.Parse("2006-01-02 15:04:05", clockTimeStr)
		}
		
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"msg": "invalid clock_time format"})
			return
		}

		// 使用核心时间逻辑：计算业务考勤日
		attDate := GetBusinessDate(punchTime)

		// 开启事务执行 Upsert
		err = db.Transaction(func(tx *gorm.DB) error {
			var snapshot model.AttDailySnapshot
			
			// 尝试查出现有记录
			result := tx.Where("user_id = ? AND att_date = ?", personID, attDate).First(&snapshot)
			
			var rawTimes []time.Time
			if result.Error == nil {
				// 记录已存在，检查是否已归档封板
				if snapshot.IsArchived {
					return nil // 封板后禁止修改
				}
				_ = json.Unmarshal(snapshot.RawPunchTimes, &rawTimes)
			}
			
			// 追加新时间
			rawTimes = append(rawTimes, punchTime)
			// 去重排序（避免设备网络延迟导致的重复推送）
			rawTimes = uniqueAndSortTimes(rawTimes)
			
			// 触发规则引擎，重新推断班次和状态
			matchRes := MatchShiftRules(attDate, rawTimes)
			
			firstPunch := rawTimes[0]
			lastPunch := rawTimes[len(rawTimes)-1]
			rawBytes, _ := json.Marshal(rawTimes)
			
			// 构建 Upsert 数据
			newSnapshot := model.AttDailySnapshot{
				UserID:        personID,
				AttDate:       attDate,
				FirstPunch:    &firstPunch,
				LastPunch:     &lastPunch,
				PunchCount:    len(rawTimes),
				MatchedShift:  matchRes.Shift,
				MatchedStatus: matchRes.Status,
				RawPunchTimes: datatypes.JSON(rawBytes),
			}
			
			// 利用 PostgreSQL/MySQL 原生支持的 Upsert 进行原子级覆盖存储
			return tx.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "user_id"}, {Name: "att_date"}}, // 以 user_id + att_date 为唯一键
				DoUpdates: clause.AssignmentColumns([]string{
					"first_punch", "last_punch", "punch_count", 
					"matched_shift", "matched_status", "raw_punch_times", "updated_at",
				}),
			}).Create(&newSnapshot).Error
		})

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"msg": "internal error", "error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{"msg": "success"})
	}
}

// uniqueAndSortTimes 对时间数组去重并排序
func uniqueAndSortTimes(times []time.Time) []time.Time {
	if len(times) <= 1 {
		return times
	}
	sort.Slice(times, func(i, j int) bool {
		return times[i].Before(times[j])
	})
	
	var res []time.Time
	res = append(res, times[0])
	for i := 1; i < len(times); i++ {
		// 忽略完全一样的时间
		if !times[i].Equal(times[i-1]) {
			res = append(res, times[i])
		}
	}
	return res
}
