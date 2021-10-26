package utils

import (
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"gorm.io/gorm"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// GetFolderSecret 获取文件夹的密钥
func GetFolderSecret(filePath, password string) (string, error) {
	folderId, err := GetFolderIdFromPath(filePath)
	if err != nil {
		err = errors.Wrap(err, status.FolderPathFailErr)
		return "", err
	}
	folderInfo, err := entity.GetFolderInfo(folderId)
	if err != nil {
		err = errors.Wrap(err, status.FolderPathFailErr)
		return "", err
	}
	// 如果不需要加密，返回空字符
	if folderInfo.IsEncrypt == 0 {
		return "", nil
	}

	secret, err := utils.DecryptString(password, folderInfo.Cipher)
	if err != nil {
		err = errors.Wrap(err, status.FolderPwdFailErr)
		return "", err
	}

	return secret, nil
}

// UpdateFolderPath 如果修改目录，修改folder数据表的目录信息（包含子目录）
func UpdateFolderPath(tx *gorm.DB, oldPath, newPath string) error {
	folders, err := entity.GetFoldersLikeAbsPath(oldPath)
	newPathS := strings.Split(newPath, "/")
	if err != nil {
		return err
	}
	updateMap := make(map[string]interface{})
	if len(newPathS) >= 2 {
		// 更新pool字段
		updateMap["pool_name"] = newPathS[1]
	}
	if len(newPathS) >= 3 {
		// 更新partition 字段
		updateMap["partition_name"] = newPathS[2]
	}
	// 循环每一条目录文件,TODO 优化，循环更新，存在问题
	for _, folder := range folders {
		if folder.AbsPath != oldPath {
			// 如果是下一级目录，则更新newPath
			nPath := filepath.Join(newPath, strings.TrimPrefix(folder.AbsPath, oldPath))

			// 更新目录
			updateMap["abs_path"] = nPath
			// 如果名称和abs的名称对得上，才需要修改
			if folder.Name == path.Base(folder.AbsPath) {
				updateMap["name"] = path.Base(nPath)
			}

			if err := entity.UpdateFolderInfo(tx, folder.ID, updateMap); err != nil {
				return err
			}
			continue
		}

		updateMap["abs_path"] = newPath
		if folder.Name == path.Base(folder.AbsPath) {
			//  如果名称和abs的名称对得上，才需要修改
			updateMap["name"] = path.Base(newPath)
		}

		if err := entity.UpdateFolderInfo(tx, folder.ID, updateMap); err != nil {
			return err
		}
	}

	return nil
}

// GetFolderSize 获取文件夹大小
func GetFolderSize(path string) (int64, error) {
	var size int64
	path = filepath.Join(config.AppSetting.UploadSavePath, path)
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return err
	})
	return size, err
}