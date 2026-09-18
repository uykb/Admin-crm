package dal

import (
	"apeadmin-gin/internal/model"

	"gorm.io/gorm"
)

// ─── 文件夹 ───

// GetFolderByID 根据 ID 查询文件夹
func GetFolderByID(id uint) (*model.SysFileFolder, error) {
	var f model.SysFileFolder
	err := gormDB.First(&f, id).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// ListFolders 查询全部文件夹
func ListFolders() ([]model.SysFileFolder, error) {
	var folders []model.SysFileFolder
	err := gormDB.Order("parent_id ASC, id ASC").Find(&folders).Error
	return folders, err
}

// CreateFolder 创建文件夹
func CreateFolder(f *model.SysFileFolder) error {
	return gormDB.Create(f).Error
}

// UpdateFolder 更新文件夹（名称 / 父级）
func UpdateFolder(f *model.SysFileFolder) error {
	return gormDB.Model(&model.SysFileFolder{}).Where("id = ?", f.ID).
		Updates(map[string]interface{}{"name": f.Name, "parent_id": f.ParentID}).Error
}

// DeleteFolder 删除文件夹（级联：子文件夹 + 文件记录一并删除，返回删除的文件记录）
func DeleteFolder(id uint) ([]model.SysFile, error) {
	// 收集该文件夹及其全部子孙文件夹 ID
	deptIDs := collectFolderIDs(id)

	// 删除文件夹
	if err := gormDB.Where("id IN ?", deptIDs).Delete(&model.SysFileFolder{}).Error; err != nil {
		return nil, err
	}

	// 删除文件记录并返回
	var files []model.SysFile
	if err := gormDB.Where("folder_id IN ?", deptIDs).Find(&files).Error; err != nil {
		return nil, err
	}
	if err := gormDB.Where("folder_id IN ?", deptIDs).Delete(&model.SysFile{}).Error; err != nil {
		return nil, err
	}
	return files, nil
}

// collectFolderIDs 收集文件夹及其子孙 ID
func collectFolderIDs(id uint) []uint {
	all, err := ListFolders()
	if err != nil {
		return []uint{id}
	}
	childrenMap := make(map[uint][]uint)
	for _, f := range all {
		if f.ParentID != nil {
			childrenMap[*f.ParentID] = append(childrenMap[*f.ParentID], f.ID)
		}
	}
	result := []uint{id}
	queue := []uint{id}
	for len(queue) > 0 {
		curr := queue[0]
		queue = queue[1:]
		for _, child := range childrenMap[curr] {
			result = append(result, child)
			queue = append(queue, child)
		}
	}
	return result
}

// MoveFolder 移动文件夹（含循环检测）
func MoveFolder(id uint, newParent *uint) error {
	if newParent != nil && *newParent == id {
		return gorm.ErrInvalidData
	}
	// 防止移动到自己的子孙节点（循环）
	if newParent != nil {
		sub := collectFolderIDs(id)
		for _, s := range sub {
			if *newParent == s {
				return gorm.ErrInvalidData
			}
		}
	}
	return gormDB.Model(&model.SysFileFolder{}).Where("id = ?", id).Update("parent_id", newParent).Error
}

// ─── 文件 ───

// ListFiles 分页查询文件（可按文件夹/关键字过滤）
func ListFiles(folderID *uint, keyword string, page, pageSize int) ([]model.SysFile, int64, error) {
	query := gormDB.Model(&model.SysFile{})
	if folderID != nil && *folderID > 0 {
		query = query.Where("folder_id = ?", *folderID)
	}
	if keyword != "" {
		like := "%" + keyword + "%"
		query = query.Where("name LIKE ? OR mime_type LIKE ?", like, like)
	}
	var total int64
	query.Count(&total)
	var files []model.SysFile
	err := query.Order("created_at DESC").Offset((page - 1) * pageSize).Limit(pageSize).Find(&files).Error
	return files, total, err
}

// GetFileByID 查询文件记录
func GetFileByID(id uint) (*model.SysFile, error) {
	var f model.SysFile
	err := gormDB.First(&f, id).Error
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// CreateFile 创建文件记录
func CreateFile(f *model.SysFile) error {
	return gormDB.Create(f).Error
}

// DeleteFile 删除文件记录
func DeleteFile(id uint) error {
	return gormDB.Delete(&model.SysFile{}, id).Error
}

// MoveFile 移动文件
func MoveFile(id uint, folderID *uint) error {
	return gormDB.Model(&model.SysFile{}).Where("id = ?", id).Update("folder_id", folderID).Error
}