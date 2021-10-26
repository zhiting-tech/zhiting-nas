package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	utils2 "gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"io"
	"path/filepath"
)

// uploadOneFile 单个文件上传
func (req *UploadFileReq) uploadOneFile(newPath string,  c *gin.Context) (resp UploadFileResp, err error) {
	// 判断上传的目录存不存在
	dir := filepath.Dir(newPath)
	fs := filebrowser.GetFB()
	if err = fs.Mkdir(dir); err != nil {
		return
	}

	// 如果保存路径已重复，需要重命名
	newPath = req.getNewName(newPath)

	// 获取目录的密钥且校验密码，如果密钥为空，则不需要加密，
	secret, err := utils.GetFolderSecret(req.path, c.GetHeader("pwd"))
	if err != nil {
		return
	}
	var destFile filebrowser.File
	if secret != "" {
		destFile, err = fs.Create(newPath + types.FolderEncryptExt) // 文件需要加密，拼接上后缀
	} else {
		destFile, err = fs.Create(newPath)
	}

	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	defer destFile.Close()

	// 复制内容
	_, err = io.Copy(destFile, req.uploadFile)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// 如果密钥不为空，需要进行加密
	if secret != "" {
		// 校验hash
		if err = req.checkFileHash(newPath + types.FolderEncryptExt); err != nil {
			_ = fs.Remove(newPath + types.FolderEncryptExt)
			return
		}
		_, err = utils2.EncryptFile(secret, newPath + types.FolderEncryptExt, newPath)
		if err != nil {
			_ = fs.Remove(newPath + types.FolderEncryptExt)
			return
		}
		// 把原文件删除
		_ = fs.Remove(newPath + types.FolderEncryptExt)
	} else {
		// 校验hash
		if err = req.checkFileHash(newPath); err != nil {
			return
		}
	}

	// 获取登陆用户
	user := session.Get(c)
	// 创建folder数据
	if err = req.createFolder(newPath, types.FolderTypeFile, user.UserID); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	resp, err = req.wrapResp(newPath, fs)

	return
}