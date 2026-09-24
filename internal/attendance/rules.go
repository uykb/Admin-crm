package attendance

import (
	"time"
)

// MatchResult 班次与状态匹配结果
type MatchResult struct {
	Shift  string // day, night, unknown
	Status string // normal, late, early, absent, late_and_early
}

// GetBusinessDate 获取业务考勤日
// 核心时间逻辑：日切点设定为 08:00。
// 08:00 - 23:59 归属为当天。
// 00:00 - 07:59 归属为前一天（解决跨夜班问题）。
func GetBusinessDate(punchTime time.Time) time.Time {
	hour := punchTime.Hour()
	if hour < 8 {
		// 0点到8点前的打卡，算作前一天的考勤
		return punchTime.AddDate(0, 0, -1).Truncate(24 * time.Hour)
	}
	// 8点及以后的打卡，算作当天的考勤
	return punchTime.Truncate(24 * time.Hour)
}

// MatchShiftRules 规则引擎：根据当天所有打卡时间推断班次与考勤状态
// 规则：
// - 白班：首卡 <= 08:00 且 末卡 >= 20:00
// - 夜班：首卡 <= 20:00 且 末卡 >= 次日 08:00 (跨夜)
func MatchShiftRules(attDate time.Time, rawPunchTimes []time.Time) MatchResult {
	if len(rawPunchTimes) == 0 {
		return MatchResult{Shift: "unknown", Status: "absent"}
	}

	firstPunch := rawPunchTimes[0]
	lastPunch := rawPunchTimes[len(rawPunchTimes)-1]

	// 构造基准时间点
	// 白班标准：当天 08:00 到 20:00
	dayStart := time.Date(attDate.Year(), attDate.Month(), attDate.Day(), 8, 0, 0, 0, attDate.Location())
	dayEnd := time.Date(attDate.Year(), attDate.Month(), attDate.Day(), 20, 0, 0, 0, attDate.Location())

	// 夜班标准：当天 20:00 到 次日 08:00
	nightStart := dayEnd
	nightEnd := dayStart.AddDate(0, 0, 1)

	// 简单实现以 12:00 为界限区分到底是上白班还是上夜班
	// 如果首卡在白天（12:00 前），推断为白班
	if firstPunch.Hour() < 12 {
		status := "normal"
		if firstPunch.After(dayStart) {
			status = "late" // 迟于08:00打首卡，算迟到
		}
		if lastPunch.Before(dayEnd) {
			if status == "late" {
				status = "late_and_early" // 迟到且早退
			} else {
				status = "early" // 早退
			}
		}
		return MatchResult{Shift: "day", Status: status}
	}

	// 如果首卡在下午或晚上（12:00 之后），推断为夜班
	status := "normal"
	if firstPunch.After(nightStart) {
		status = "late" // 迟于20:00打首卡，算迟到
	}
	if lastPunch.Before(nightEnd) {
		if status == "late" {
			status = "late_and_early"
		} else {
			status = "early"
		}
	}
	return MatchResult{Shift: "night", Status: status}
}
