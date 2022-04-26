package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	utils2 "gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// chunk 分片上传
func (req *UploadFileReq) chunk(newPath string) (resp UploadFileResp, err error) {

	fs := filebrowser.GetFB()

	i, err := strconv.Atoi(req.chunkNumber)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// chunk是否已上传
	isExist, err := GetChunkMap().CheckChunk(req.Hash, i)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		resp.Chunks = GetChunkMap().GetChunks(req.Hash)
		return
	}

	if isExist {
		return
	}

	// 合并文件
	open, err := os.OpenFile(filepath.Join(fs.GetRoot(), newPath+req.Hash), os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	req.mergeFile(open, i)
	resp.Chunks = GetChunkMap().GetChunks(req.Hash)
	return
}

// merge 合并分片
func (req *UploadFileReq) merge(newPath string, c *gin.Context) (resp UploadFileResp, err error) {
	user := session.Get(c)

	fs := filebrowser.GetFB()
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	// hash 校验
	tempFilePath := filepath.Join(fs.GetRoot(), newPath+req.Hash)
	rootPath := strings.TrimPrefix(tempFilePath, fs.GetRoot())
	if err = req.checkFileHash(rootPath); err != nil {
		return
	}

	// 移动并重命名文件
	fileName := strings.TrimPrefix(tempFilePath, fs.GetRoot())
	// 获取目录的密钥且校验密码，如果密钥为空，则不需要加密，
	secret, err := utils.GetFolderSecret(req.path, c.GetHeader("pwd"))
	if err != nil {
		return
	}

	var isEncrypt = false // 是否加密  false为未加密
	// 获取文件命名
	newPath = req.getNewName(newPath)
	fs.Rename(fileName, newPath)
	// 获取获取到的密钥对文件进行加密
	if secret != "" {
		isEncrypt = true
		fs.Rename(fileName, newPath+types.FolderEncryptExt)
		_, err = utils2.EncryptFile(secret, newPath+types.FolderEncryptExt, newPath)
		if err != nil {
			_ = fs.Remove(newPath + types.FolderEncryptExt)
			return
		}
	}
	// 获取文件类型
	pathExt := utils.GetPathExt(newPath)
	// 转化为小写
	pathExt = strings.ToLower(pathExt)
	v, ok := FileTypeMap[pathExt]
	if ok && (v == types.FolderPhoto || v == types.FolderVideo) {
		if err = generationThumbnail(isEncrypt, path.Join(config.AppSetting.UploadSavePath, newPath), req.Hash, v); err != nil {
			fmt.Println("thumbnail failed:", err)
		}
	}

	// 成功后删除原来的文件
	// 创建folder数据
	if err = req.createFolder(newPath, types.FolderTypeFile, user.UserID); err != nil {
		fmt.Println("createFolder failed:", err)
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	resp, err = req.wrapResp(newPath, fs)
	if err == nil {
		// 如果合并成功，把分片的文件夹删除
		//_ = fs.RemoveAll(cachePath)
		GetChunkMap().DelChunk(req.Hash)
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

func (req *UploadFileReq) mergeFile(tempFile *os.File, i int) {
	var (
		err error
		b   []byte
	)
	b, err = ioutil.ReadAll(req.uploadFile)
	if err != nil {
		fmt.Println("mergeFile ReadAll err:", err)
		return
	}

	_, err = tempFile.WriteAt(b, types.ChunkSize*int64(i-1))

	if err != nil {
		return
	}
	chunkSize, err := strconv.Atoi(req.chunkSize)
	if err != nil {
		return
	}

	GetChunkMap().SetChunk(req.Hash, Chunk{
		ID:   i,
		Size: int64(chunkSize),
	})

}
