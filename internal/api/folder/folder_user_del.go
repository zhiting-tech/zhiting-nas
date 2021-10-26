package folder

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
)

type DelByIdsReq struct {
	UserIDs []int `json:"user_ids"`
}

type DelByIdsResp struct {
}

func DeleteFolderByIds(c *gin.Context) {
	var (
		req  DelByIdsReq
		resp DelByIdsResp
		err  error
		fs   = filebrowser.GetFB()
	)

	defer func() {
		response.HandleResponse(c, err, &resp)
	}()

	// 用户退出/被移除家庭或企业是否自动删除私人文件夹和其它文件
	if config.AppSetting.IsAutoDel == 0 {
		return
	}

	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 查找私人文件
	folderInfos, err := entity.GetPrivateFolders(req.UserIDs)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	// 移除私人文件
	for _, folderInfo := range folderInfos {
		err = removeFolderAndRecode(fs, folderInfo.AbsPath)
		if err != nil {
			return
		}
	}

	// 查找和移除用户初始化生成的个人文件
	for _, v := range req.UserIDs {
		folderRow, err := entity.GetRelateFolderInfoByUid(types.FolderSelfDirUid, v)
		if err != nil {
			return
		}
		err = removeFolderAndRecode(fs, folderRow.AbsPath)
		if err != nil {
			return
		}
	}

	// 删除属于用户uid的所有权限Auth
	if err = entity.DelFolderAuthByUid(req.UserIDs); err != nil {
		err = errors.Wrap(err, status.FolderRemoveErr)
		return
	}
}

// removeFolderAndRecode 移除文件和文件记录
func removeFolderAndRecode(fs *filebrowser.FileBrowser, absPath string) (err error) {
	// 磁盘删除
	if err = fs.RemoveAll(absPath); err != nil {
		err = errors.Wrap(err, status.FolderRemoveErr)
		return
	}
	// 删除私人文件
	if err = entity.DelFolder(entity.GetDB(), absPath); err != nil {
		err = errors.Wrap(err, status.FolderRemoveErr)
		return err
	}
	return
}
