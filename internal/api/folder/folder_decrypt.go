package folder

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
)

type DecryptFolderReq struct {
	Path     string `uri:"path"`
	Password string `json:"password"`
}

// DecryptFolder 解密文件夹
func DecryptFolder(c *gin.Context) {
	var (
		err  error
		req  DecryptFolderReq
		resp string
	)

	defer func() {
		response.HandleResponse(c, err, &resp)
	}()

	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	_, err = utils.GetFolderSecret(req.Path, req.Password)

	return
}