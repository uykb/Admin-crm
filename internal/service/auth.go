package service

import (
	"sort"
	"sync"
	"time"

	"apeadmin-gin/internal/dal"
	"apeadmin-gin/internal/model"
	"apeadmin-gin/internal/schema"
)

// ─── 权限缓存（M8：避免每次请求都查库）───
// 缓存用户权限集合，TTL 30s。角色/菜单变更后最迟 30s 生效，
// 换取高并发下的 DB 压力下降（权限校验是每请求必经路径）。
var (
	permCacheMu sync.Mutex
	permCache   = map[uint]permCacheEntry{}
)

type permCacheEntry struct {
	perms     map[string]bool
	expiresAt time.Time
}

const permCacheTTL = 30 * time.Second

func getCachedPerms(userID uint) (map[string]bool, bool) {
	permCacheMu.Lock()
	defer permCacheMu.Unlock()
	e, ok := permCache[userID]
	if !ok || time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.perms, true
}

func setCachedPerms(userID uint, perms map[string]bool) {
	permCacheMu.Lock()
	defer permCacheMu.Unlock()
	permCache[userID] = permCacheEntry{perms: perms, expiresAt: time.Now().Add(permCacheTTL)}
}

// InvalidateUserPerms 使某用户权限缓存失效（角色/菜单变更时调用）
func InvalidateUserPerms(userID uint) {
	permCacheMu.Lock()
	defer permCacheMu.Unlock()
	delete(permCache, userID)
}

// GetUserPermissions 获取用户权限集合（带 TTL 缓存）
func GetUserPermissions(userID uint) map[string]bool {
	if cached, ok := getCachedPerms(userID); ok {
		return cached
	}

	perms := make(map[string]bool)
	user, err := dal.GetUserWithRoles(userID)
	if err != nil {
		return perms
	}
	for _, role := range user.Roles {
		if role.Status != 1 {
			continue
		}
		menus, err := dal.GetMenusByRoleID(role.ID)
		if err != nil {
			continue
		}
		for _, menu := range menus {
			if menu.Status == 1 && menu.Permission != "" {
				perms[menu.Permission] = true
			}
		}
	}
	setCachedPerms(userID, perms)
	return perms
}

// GetUserInfo 获取用户信息（权限码 + 菜单树）
func GetUserInfo(userID uint) (*schema.UserInfo, error) {
	user, err := dal.GetUserByID(userID)
	if err != nil {
		return nil, err
	}

	var permissions []string
	var menuNodes []schema.MenuNode

	if user.IsSuperAdmin {
		permissions = []string{"*"}
		menus, err := dal.GetAllMenus()
		if err != nil {
			return nil, err
		}
		menuNodes = buildMenuTree(menus, 0)
	} else {
		perms := GetUserPermissions(userID)
		for p := range perms {
			permissions = append(permissions, p)
		}
		user, err := dal.GetUserWithRoles(userID)
		if err != nil {
			return nil, err
		}
		var allMenus []model.SysMenu
		for _, role := range user.Roles {
			if role.Status != 1 {
				continue
			}
			menus, err := dal.GetMenusByRoleID(role.ID)
			if err != nil {
				continue
			}
			allMenus = append(allMenus, menus...)
		}
		// 去重
		menuMap := make(map[uint]model.SysMenu)
		for _, m := range allMenus {
			if m.Status == 1 {
				menuMap[m.ID] = m
			}
		}
		var deduped []model.SysMenu
		for _, m := range menuMap {
			deduped = append(deduped, m)
		}
		// 排序：父级按 Sort 升序，保证菜单树顺序稳定（M7）
		sort.SliceStable(deduped, func(i, j int) bool {
			if deduped[i].ParentID != deduped[j].ParentID {
				return deduped[i].ParentID < deduped[j].ParentID
			}
			return deduped[i].Sort < deduped[j].Sort
		})
		menuNodes = buildMenuTree(deduped, 0)
	}

	roles, _ := dal.GetUserWithRoles(userID)
	roleCodes := make([]string, 0, len(roles.Roles))
	for _, r := range roles.Roles {
		roleCodes = append(roleCodes, r.Code)
	}

	return &schema.UserInfo{
		ID:          user.ID,
		Username:    user.Username,
		Nickname:    user.Nickname,
		Avatar:      user.Avatar,
		Permissions: permissions,
		Menus:       menuNodes,
		Roles:       roleCodes,
	}, nil
}

// buildMenuTree 构建菜单树
func buildMenuTree(menus []model.SysMenu, parentID uint) []schema.MenuNode {
	var nodes []schema.MenuNode
	for _, m := range menus {
		if m.ParentID == parentID {
			node := schema.MenuNode{
				ID:         m.ID,
				Name:       m.Name,
				ParentID:   m.ParentID,
				Type:       m.Type,
				Path:       m.Path,
				Component:  m.Component,
				Permission: m.Permission,
				Icon:       m.Icon,
				Sort:       m.Sort,
				Visible:    m.Visible,
				Status:     m.Status,
				Children:   buildMenuTree(menus, m.ID),
			}
			nodes = append(nodes, node)
		}
	}
	return nodes
}
