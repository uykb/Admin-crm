package dal

import (
	"apeadmin-gin/internal/model"

	"gorm.io/gorm"
)

// DataScope 数据权限范围常量（与数据库/前端约定一致）
// 注意：数值大小不反映权限范围大小，取最大范围必须用 dataScopeRank 换算
const (
	DataScopeSelf          = 1 // 本人
	DataScopeDeptAndChildren = 2 // 本部门及以下
	DataScopeDeptOnly      = 3 // 本部门
	DataScopeAll           = 4 // 全部
)

// dataScopeRank 数据范围等级（越大范围越广，用于多角色取最大范围）
func dataScopeRank(scope int) int {
	switch scope {
	case DataScopeAll:
		return 4
	case DataScopeDeptAndChildren:
		return 3
	case DataScopeDeptOnly:
		return 2
	default:
		return 1 // Self 与未知值
	}
}

// DataScopeScope 返回 GORM Scope：按用户角色的 data_scope 过滤当前查询
// 用法：db.Scopes(dal.DataScopeScope(user, "creator_id", "dept_id")).Find(&items)
func DataScopeScope(u *model.SysUser, ownerColumn, deptColumn string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		if u.IsSuperAdmin {
			return db
		}
		scope := maxDataScope(u.Roles)
		switch scope {
		case DataScopeAll:
			return db
		case DataScopeDeptAndChildren:
			deptIDs := descendantDeptIDs(u.DeptID)
			if len(deptIDs) == 0 {
				return db.Where("1 = 0") // 无部门数据
			}
			return db.Where(deptColumn+" IN ?", deptIDs)
		case DataScopeDeptOnly:
			if u.DeptID == nil {
				return db.Where("1 = 0")
			}
			return db.Where(deptColumn+" = ?", *u.DeptID)
		default:
			return db.Where(ownerColumn+" = ?", u.ID)
		}
	}
}

// maxDataScope 多角色取最大范围（按语义等级比较，而非数值）
func maxDataScope(roles []model.SysRole) int {
	best := DataScopeSelf
	bestRank := dataScopeRank(DataScopeSelf)
	for _, r := range roles {
		if r.Status != 1 {
			continue
		}
		if rank := dataScopeRank(r.DataScope); rank > bestRank {
			bestRank = rank
			best = r.DataScope
		}
	}
	return best
}

// descendantDeptIDs 获取部门及其子部门 ID 集合（内存缓存可后续优化）
func descendantDeptIDs(deptID *uint) []uint {
	if deptID == nil {
		return nil
	}
	depts, err := GetAllDepts()
	if err != nil || len(depts) == 0 {
		return []uint{*deptID}
	}
	// BFS 收集子树
	result := []uint{*deptID}
	childrenMap := make(map[uint][]uint)
	for _, d := range depts {
		childrenMap[d.ParentID] = append(childrenMap[d.ParentID], d.ID)
	}
	queue := []uint{*deptID}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, childID := range childrenMap[curr] {
			result = append(result, childID)
			queue = append(queue, childID)
		}
	}
	return result
}
