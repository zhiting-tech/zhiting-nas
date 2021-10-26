package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	utils2 "gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"io"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
)

// chunk 分片上传
func (req *UploadFileReq) chunk(c *gin.Context) (resp UploadFileResp, err error) {
	user := session.Get(c)

	fs := filebrowser.GetFB()
	// hash作为分块文件的目录名称
	cachePath := req.getCachePath(user.UserID)

	if err = fs.Mkdir(cachePath); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// 判断该分块文件是否存在
	if req.isFileExist(filepath.Join(cachePath, req.chunkNumber)) {
		return
	}

	// 创建临时文件
	tmpFile, err := ioutil.TempFile(filepath.Join(fs.GetRoot(), cachePath), "temp-")
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	defer tmpFile.Close()

	// 复制文件内容
	_, err = io.Copy(tmpFile, req.uploadFile)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	err = nil
	fileName := strings.TrimPrefix(tmpFile.Name(), fs.GetRoot())

	// 以chunkNumber 重命名临时文件
	err = fs.Rename(fileName, filepath.Join(cachePath, req.chunkNumber))
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	resp.Chunks, err = GetChunksInfos(cachePath)

	return
}

// merge 合并分片
func (req *UploadFileReq) merge(newPath string, c *gin.Context) (resp UploadFileResp, err error) {
	user := session.Get(c)

	cachePath := req.getCachePath(user.UserID)
	fs := filebrowser.GetFB()
	file, err := fs.Open(cachePath)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	fileInfos, err := file.Readdir(-1)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	totalChunks, _ := strconv.Atoi(req.TotalChunks)
	if len(fileInfos) != totalChunks {
		err = errors.New(status.ChunkFileNotExistErr)
		return
	}

	// 创建临时文件
	tempFile, err := ioutil.TempFile(filepath.Join(fs.GetRoot(), cachePath), "temp-")
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	for i := range fileInfos {
		var chunkFile filebrowser.File
		chunkFile, err = fs.Open(filepath.Join(cachePath, strconv.Itoa(i+1)))
		if err != nil {
			return resp, err
		}

		var b []byte
		b, err = ioutil.ReadAll(chunkFile)
		if err != nil {
			return resp, err
		}

		tempFile.Write(b)
		chunkFile.Close()
	}
	tempFile.Close()

	// hash 校验
	rootPath := strings.TrimPrefix(tempFile.Name(), fs.GetRoot())
	if err = req.checkFileHash(rootPath); err != nil {
		return
	}

	dir := filepath.Dir(newPath)
	if err = fs.Mkdir(dir); err != nil {
		return
	}

	// 移动并重命名文件
	fileName := strings.TrimPrefix(tempFile.Name(), fs.GetRoot())

	// 获取目录的密钥且校验密码，如果密钥为空，则不需要加密，
	secret, err := utils.GetFolderSecret(req.path, c.GetHeader("pwd"))
	if err != nil {
		return
	}
	// 获取文件命名
	newPath = req.getNewName(newPath)

	// 获取获取到的密钥对文件进行加密
	if secret != "" {
		err = fs.CopyFileToTarget(fileName, newPath + types.FolderEncryptExt) // 如果需要的话，需要加上.env文件
		if err != nil {
			_ = fs.Remove(newPath + types.FolderEncryptExt)
			_ = fs.Remove(fileName)
			config.Logger.Errorf("merge chunk CopyFileToTarget %v", err)
			err = errors.New(errors.InternalServerErr)
			return
		}
		_, err = utils2.EncryptFile(secret, newPath + types.FolderEncryptExt, newPath)
		if err != nil {
			_ = fs.Remove(fileName)
			_ = fs.Remove(newPath + types.FolderEncryptExt)
			return
		}
		// 把源文件删除
		_ = fs.Remove(newPath + types.FolderEncryptExt)
	} else {
		err = fs.CopyFileToTarget(fileName, newPath)
		if err != nil {
			_ = fs.Remove(fileName)
			_ = fs.Remove(newPath)
			config.Logger.Errorf("merge chunk CopyFileToTarget %v", err)
			err = errors.New(errors.InternalServerErr)
			return
		}
	}

	// 成功后删除原来的文件
	_ = fs.RemoveAll(fileName)

	// 创建folder数据
	if err = req.createFolder(newPath, types.FolderTypeFile, user.UserID); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	resp, err = req.wrapResp(newPath, fs)
	if err == nil {
		// 如果合并成功，把分片的文件夹删除
		_ = fs.RemoveAll(cachePath)
	}

	return
}

func (req *UploadFileReq) checkMerge() (err error) {
	if req.TotalChunks == "" || req.Hash == "" {
		err = errors.New(status.ResourceHashInputNil)
		return
	}
	return
}