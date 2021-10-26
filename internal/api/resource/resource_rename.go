package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gorm.io/gorm"
	"os"
	"path/filepath"
	"unicode/utf8"
)

type RenameReq struct {
	Path string `uri:"path"`
	Name string `json:"name"`
}

func RenameResource(c *gin.Context) {
	var (
		err error
		req RenameReq
	)

	defer func() {
		response.HandleResponse(c, err, nil)
	}()
	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 判断是否有可写权限
	write, _ := c.Get("write")
	if write.(int) == 0 {
		err = errors.Wrap(err, status.ResourceNotWriteAuthErr)
		return
	}

	req.Path, err = utils.GetNewPath(req.Path)
	if err != nil {
		return
	}

	// 参数校验
	newPath, err := req.validateRequest()
	if err != nil {
		return
	}

	fs := filebrowser.GetFB()
	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		if err = fs.Rename(req.Path, newPath); err != nil {
			if os.IsNotExist(err) {
				err = errors.Wrap(err, status.ResourceNotExistErr)
				return err
			} else if os.IsExist(err) {
				err = errors.New(status.NameAlreadyExistErr)
				return err
			} else {
				err = errors.New(errors.InternalServerErr)
				return err
			}
		}
		// 批量修改folder数据
		if err = utils.UpdateFolderPath(tx, req.Path, newPath); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return
	}
}

// validateRequest重命名格式校验
func (req *RenameReq) validateRequest() (newPath string, err error) {
	// 命名格式
	if utf8.RuneCountInString(req.Name) > 100 {
		err = errors.New(status.ResourceNameTooLongErr)
		return
	}
	// 获取文件的目录
	fs := filebrowser.GetFB()
	dir, _ := filepath.Split(req.Path)
	newPath = filepath.Join(dir, req.Name)
	// 判断文件是否存在
	open, err := fs.Open(newPath)
	if err == nil {
		// 如果err不为空，newPath存在，需要报错
		err = errors.New(status.NameAlreadyExistErr)
		open.Close()
		return
	}
	// 设置为空
	err = nil
	return
}
