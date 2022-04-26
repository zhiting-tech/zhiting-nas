package resource

import (
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gorm.io/gorm"
	"path/filepath"
	"strconv"
)

// createDir 创建目录
func (req *UploadFileReq) createDir(newPath string, uid int) (resp UploadFileResp, err error) {
	fs := filebrowser.GetFB()

	// 判断文件夹是否已存在
	_, err = fs.Stat(newPath)
	if err == nil {
		// isAutoRename为true时 pc端上传文件夹先创建文件夹
		isAutoRename, _ := strconv.ParseBool(req.IsAutoRename)
		if !isAutoRename {
			err = errors.New(status.NameAlreadyExistErr)
			return
		}
		newPath, _ = filepath.Abs(newPath)
		newPath = req.getNewName(newPath)
	}
	err = nil

	// 创建目录
	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		// 把目录信息记录到folder表
		if err = req.createFolder(newPath, types.FolderTypeDir, uid); err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return err
		}
		if err = fs.Mkdir(newPath); err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return err
		}
		return nil
	}); err != nil {
		return
	}

	resp, err = req.wrapResp(newPath, fs)
	return
}
