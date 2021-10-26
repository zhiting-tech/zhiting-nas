package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"os"
)

type DeleteResourceReq struct {
	Paths []string `json:"paths"`
}

func DeleteResource(c *gin.Context) {

	var (
		req DeleteResourceReq
		err error
	)
	defer func() {
		response.HandleResponse(c, err, nil)
	}()

	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	user := session.Get(c)

	err = req.validateRequest(user.UserID)
	if err != nil {
		return
	}

	if err = req.remove(); err != nil {
		return
	}

}

func (req *DeleteResourceReq) validateRequest(userID int) error {
	for i, path := range req.Paths {
		// 找不到对应的数据
		auth, err := utils.GetFilePathAuth(userID, path)
		if err != nil {
			return err
		}
		// 没有删除权限
		if auth.Deleted == 0 {
			return errors.New(status.ResourceNotDeleteAuthErr)
		}

		req.Paths[i], err = utils.GetNewPath(path)
		if err != nil {
			return err
		}
	}
	return nil
}

// removeFile 删除文件和共享记录
func (req *DeleteResourceReq) remove() (err error) {
	for _, path := range req.Paths {
		fs := filebrowser.GetFB()
		if err = removeFile(fs, path); err != nil {
			return
		}
	}

	// 删除文件folder信息，TODO 删除关联信息
	if err = entity.DelFolderByAbsPaths(entity.GetDB(), req.Paths); err != nil {
		return
	}

	return
}

// removeFile 删除的文件操作
func removeFile(fs *filebrowser.FileBrowser, path string) (err error) {
	var fileInfo os.FileInfo
	fileInfo, err = fs.Stat(path)
	if err != nil {
		return
	}

	if fileInfo.IsDir() {
		if err = fs.RemoveAll(path); err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
	} else {
		if err = fs.Remove(path); err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
	}
	return
}
