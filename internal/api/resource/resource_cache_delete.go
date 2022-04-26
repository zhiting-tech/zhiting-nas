package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"path/filepath"
)

type DeleteCacheReq struct {
	Path string `uri:"path"`
}

func DeleteCache(c *gin.Context) {
	var (
		req DeleteCacheReq
		err error
	)
	fb := filebrowser.GetFB()
	defer func() {
		response.HandleResponse(c, err, nil)
	}()

	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	if err = req.validateRequest(); err != nil {
		return
	}
	absPath, _ := filepath.Abs(req.Path)
	cachePath, err := utils.GetNewPath(absPath)
	if err != nil {
		return
	}
	fmt.Println("cachePath:", cachePath)
	if err = fb.RemoveAll(cachePath); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
	}
}
func (req *DeleteCacheReq) validateRequest() (err error) {
	if req.Path == "" {
		return errors.Wrap(err, errors.BadRequest)
	}
	return err
}
