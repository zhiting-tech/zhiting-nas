package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	utils2 "gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type FileDownloadReq struct {
	Path string `uri:"path"`
}

type FileDownloadResp struct {
}

func FileDownload(c *gin.Context) {
	var (
		err  error
		req  FileDownloadReq
		resp FileDownloadResp
		secret string
		downloadPath string
	)

	defer func() {
		if secret != "" {
			// 如果存在解密文件，把解密文件删除
			_ = filebrowser.GetFB().Remove(downloadPath)
		}

		if err != nil {
			response.HandleResponse(c, err, &resp)
		}
	}()
	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 判断是否有可写权限
	write, _ := c.Get("write")
	if write.(int) == 0 {
		err = errors.Wrap(err, status.ResourceNotWriteAuthErr)
		return
	}
	pwd := c.GetHeader("pwd")
	secret, err = utils.GetFolderSecret(req.Path, pwd)
	if err != nil {
		return
	}

	downloadPath, err = utils.GetNewPath(req.Path)
	if err != nil {
		return
	}
	fb := filebrowser.GetFB()
	fileName := filepath.Base(downloadPath)

	// 如果有密钥密码，先解密，再提供下载
	if secret != "" {
		// 新增ext，防止冲突
		ext := strconv.FormatInt(time.Now().UnixNano(), 10)
		downloadPath, err = utils2.DecryptFile(pwd, downloadPath, fmt.Sprint(downloadPath, ".", ext))
		if err != nil {
			return
		}
	}

	// 没有加密文件夹，直接打开原文件
	open, err := fb.Open(downloadPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = errors.Wrap(err, status.ResourceNotExistErr)
		} else {
			err = errors.Wrap(err, errors.InternalServerErr)
		}
		return
	}

	fileInfo, err := fb.Stat(downloadPath)
	if err != nil {
		return
	}

	http.ServeContent(c.Writer, c.Request, fileName, fileInfo.ModTime(), open)

	return
}
