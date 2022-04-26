package entity

import (
	"gorm.io/gorm"
)

// FolderAuth 目录相关的数据表
type FolderAuth struct {
	ID       int    `gorm:"primary_key"`
	Nickname string // 用户昵称
	Face     string // 头像
	Uid      int    // 用户ID
	FromUser string // 来源用户
	IsShare  int    // 对于uid用户是否为共享文件,（除uid用户个人的私人文件夹都为共享文件夹）
	FolderId int    `gorm:"index"` // 文件夹ID
	Read     int    // 是否可读
	Write    int    // 是否可写
	Deleted  int    // 是否可删除
}

func (folder FolderAuth) TableName() string {
	return "folder_auth"
}

// DelFolderAuth 删除权限
func DelFolderAuth(tx *gorm.DB, folderId int) error {
	if err := tx.Where("folder_id = ?", folderId).Delete(&FolderAuth{}).Error; err != nil {
		return err
	}
	return nil
}

func DelFolderAuthByUid(userIDs []int) error {
	if err := GetDB().Where("uid IN ?", userIDs).Delete(&FolderAuth{}).Error; err != nil {
		return err
	}
	return nil
}

// GetFolderAuthByFolderId 根据目录ID获取目录的权限信息
func GetFolderAuthByFolderId(folderId int) ([]*FolderAuth, error) {
	var list []*FolderAuth
	if err := GetDB().Where("folder_id = ?", folderId).Find(&list).Error; err != nil {
		return nil, err
	}

	return list, nil
}

// DelFolderAuthByUidAndFolderId  删除指定 uId&&folderId的值
func DelFolderAuthByUidAndFolderId(uId, folderId int) (err error) {
	err = GetDB().Where("uid = ? and folder_id = ?", uId, folderId).Delete(&FolderAuth{}).Error
	return
}

// BatchInsertAuth 批量插入权限
func BatchInsertAuth(tx *gorm.DB, auths []FolderAuth) error {
	return tx.Create(&auths).Error
}

// InsertAuth 插入权限
func InsertAuth(tx *gorm.DB, auth FolderAuth) error {
	return tx.Create(&auth).Error
}

// GetFolderAuthByUidAndFolderId 获取用户对某个文件的权限信息
func GetFolderAuthByUidAndFolderId(uid, folderId int) (*FolderAuth, error) {
	var folderAuth FolderAuth
	if err := GetDB().Where("uid = ? and folder_id = ?", uid, folderId).Find(&folderAuth).Error; err != nil {
		return nil, err
	}
	return &folderAuth, nil
}

// DelAllFolderAuthRecode 移除所有folder_auth表数据
func DelAllFolderAuthRecode() {
	GetDB().Exec("delete from folder_auth")
}
