package attendance

import (
	"fmt"
	"log"
	"time"

	"apeadmin-gin/internal/model"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ArchiveMonth 历史考勤封板与清理机制
// 作用：将指定月份的考勤状态锁定为 archived，前端禁止修改，后端拒绝重算。
// 空间优化：封板后清空 raw_punch_times 字段（由包含全天碎数据的数组清空为 "[]"）。
func ArchiveMonth(db *gorm.DB, year, month int) error {
	// 构造查询的起止时间
	loc := time.Local
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, loc)
	endOfMonth := startOfMonth.AddDate(0, 1, 0)

	log.Printf("开始对 %04d-%02d 考勤月份进行封板清理...", year, month)

	// 使用事务保证一致性
	err := db.Transaction(func(tx *gorm.DB) error {
		// PostgreSQL 的 jsonb 设置为空数组可以直接用 '[]'::jsonb，这里用 datatypes.JSON("[]")
		emptyArrayJSON := datatypes.JSON("[]")

		// 批量更新，仅处理未归档的记录
		// UPDATE att_daily_snapshot SET is_archived = true, raw_punch_times = '[]'
		// WHERE att_date >= startOfMonth AND att_date < endOfMonth AND is_archived = false
		res := tx.Model(&model.AttDailySnapshot{}).
			Where("att_date >= ? AND att_date < ? AND is_archived = ?", startOfMonth, endOfMonth, false).
			Updates(map[string]interface{}{
				"is_archived":     true,
				"raw_punch_times": emptyArrayJSON,
			})

		if res.Error != nil {
			return res.Error
		}

		log.Printf("封板完成，共处理了 %d 条每日考勤快照记录", res.RowsAffected)
		return nil
	})

	if err != nil {
		log.Printf("月份封板失败: %v", err)
		return fmt.Errorf("archive month failed: %w", err)
	}

	return nil
}
