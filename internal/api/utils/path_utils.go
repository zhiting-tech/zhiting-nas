package utils

import (
	"crypto/sha256"
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"io"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// GetAbsFolderIdFromPath  根据/s/:id/xxx路径获取出文件夹folder的ID
func GetAbsFolderIdFromPath(path string)(fileId int,err error) {
	// 获取根目录Id
	path = filepath.Join(path)
	rex := regexp.MustCompile(`^/s/[1-9][0-9]*`)
	idStr := strings.TrimPrefix(rex.FindString(path), "/s/")
	folderId, err := strconv.Atoi(idStr)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// 获取根目录详情
	folderInfo, err := entity.GetFolderInfo(folderId)
	if err != nil {
		err = errors.Wrap(err,errors.NotFound)
		return 
	}

	// 匹配出完整的路径
	lastStr := strings.TrimPrefix(path, rex.FindString(path))
	combinationPath := filepath.Join(folderInfo.AbsPath, lastStr)
	// 查询出文件夹详情
	absFolder, err := entity.GetFolderInfoByAbsPath(combinationPath)
	if err != nil {
		err = errors.Wrap(err,errors.NotFound)
		return
	}

	fileId = absFolder.ID
	return
}

// GetFolderIdFromPath 根据/s/:id/xxx路径提取出根目录的folder的ID
func GetFolderIdFromPath(path string) (folderId int, err error) {
	path = filepath.Join(path)
	rex := regexp.MustCompile(`^/s/[1-9][0-9]*`)
	idStr := strings.TrimPrefix(rex.FindString(path), "/s/")
	folderId, err = strconv.Atoi(idStr)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	return
}

// GetNewPath 获取实际路径，把/s/替换掉
func GetNewPath(path string) (newPath string, err error) {
	path = filepath.Join(path)
	if path == "" || path == "/" {
		return
	}
	// 判断路径正确性，必须以/s/开头
	if !CheckPath(path) {
		err = errors.New(status.ResourcePathIllegalErr)
		return
	}
	// 提取folder_id
	folderId, err := GetFolderIdFromPath(path)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// 获取folder详情
	info, err := entity.GetFolderInfo(folderId)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// 原路径去掉/s/:id的信息
	rex := regexp.MustCompile(`/s/[1-9][0-9]*`)
	folderPath := strings.TrimPrefix(path, rex.FindString(path))

	// 拼接上路径的实际信息
	newPath = filepath.Join(info.AbsPath, folderPath)
	return
}

// CheckPath 检查路径是否合法，统一由/s/:id, id为folder_id的ID
func CheckPath(path string) bool {
	rex := regexp.MustCompile(`^/s/[1-9][0-9]*`)
	return rex.MatchString(path)
}

// SHA256File 文件sha256
func SHA256File(path string) string {
	fs := filebrowser.GetFB()
	file, err := fs.Open(path)
	if err != nil {
		return ""
	}
	defer file.Close()

	h := sha256.New()
	_, err = io.Copy(h, file)
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// GetFilePathAuth 获取登陆用户对应目录的权限
func GetFilePathAuth(userID int, path string) (*entity.FolderRow, error) {
	path, _ = filepath.Abs(path)
	if path == "" || path == "/" {
		return nil, nil
	}
	folderId, err := GetFolderIdFromPath(path)
	if err != nil {
		err = errors.New(status.ResourcePathIllegalErr)
		return nil, err
	}

	return entity.GetRelateFolderInfo(folderId, userID)
}
