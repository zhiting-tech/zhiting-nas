package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"mime/multipart"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const (
	actionUpload      = "upload"
	actionChunk       = "chunk"
	actionMerge       = "merge"
	// FILE_CHUNK_SIZE = 2 * 1024 * 1024 // 文件分块大小2M
)

type UploadFileReq struct {
	path         string
	Action       string
	totalSize    string
	chunkNumber  string
	chunkSize    string
	TotalChunks  string
	Hash         string
	uploadFile   multipart.File
	header       *multipart.FileHeader
	IsAutoRename string // 上传文件夹时，先创建文件夹,根据该字段判断是否要给文件夹重命名
}

type UploadFileResp struct {
	Resource Info    `json:"resource"`
	Chunks   []Chunk `json:"chunks"`
}

type Chunk struct {
	ID   int   `json:"id"`
	Size int64 `json:"size"`
}

func FileUpload(c *gin.Context) {

	var (
		req  UploadFileReq
		resp UploadFileResp
		err  error
	)

	defer func() {
		if len(resp.Chunks) == 0 {
			resp.Chunks = make([]Chunk, 0)
		}
		response.HandleResponse(c, err, &resp)
	}()
	_ = c.Request.ParseMultipartForm(32 << 20)

	newPath, err := req.validateRequest(c)
	if err != nil {
		return
	}
	// 获取登陆用户
	user := session.Get(c)

	// 如果是目录则创建目录
	if req.IsDir(newPath) {
		resp, err = req.createDir(newPath, user.UserID)
		return
	}

	// 否则上传文件
	resp, err = req.upload(newPath, c)
}

// upload 上传文件
func (req *UploadFileReq) upload(newPath string, c *gin.Context) (resp UploadFileResp, err error) {
	switch req.Action {
	case actionUpload:
		resp, err = req.uploadOneFile(newPath, c)
	case actionChunk:
		resp, err = req.chunk(c)
	case actionMerge:
		resp, err = req.merge(newPath, c)
	}
	return
}

// getReq 获取请求参数
func (req *UploadFileReq) getReq(c *gin.Context) (err error) {
	req.path = c.Param("path")
	req.Action = c.Request.FormValue("action")
	req.Hash = c.Request.FormValue("hash")
	req.chunkNumber = c.Request.FormValue("chunk_number")
	req.chunkSize = c.Request.FormValue("chunk_size")
	req.TotalChunks = c.Request.FormValue("total_chunks")
	req.totalSize = c.Request.FormValue("total_size")
	req.IsAutoRename = c.Request.FormValue("is_auto_rename")

	return
}

// validateRequest 校验参数
func (req *UploadFileReq) validateRequest(c *gin.Context) (newPath string, err error) {
	if err = req.getReq(c); err != nil {
		return
	}
	// 判断是否有可写权限
	write, _ := c.Get("write")
	if write.(int) == 0 {
		err = errors.Wrap(err, status.ResourceNotWriteAuthErr)
		return
	}
	// 获取上传文件的大小
	uploadSize, err := strconv.Atoi(req.totalSize)
	if err != nil {
		err = errors.Wrap(err, status.ParamsIllegalErr)
	}

	// TODO folder有可能不是第一层目录
	folderId, _ := c.Get("folder_id")
	// 分区文件上传限制处理
	err = req.uploadLimit(folderId.(int), uploadSize)
	if err != nil {
		return
	}

	// 获取newPath
	newPath, err = req.handlePath()
	if err != nil {
		return
	}

	// 获取目录的密钥且校验密码，如果密钥为空，则不需要加密，
	_, err = utils.GetFolderSecret(req.path, c.GetHeader("pwd"))
	if err != nil {
		return
	}

	if req.IsDir(newPath) {
		return
	}
	switch req.Action {
	case actionChunk, actionUpload:
		if err = req.checkUploadFile(c); err != nil {
			return
		}
	case actionMerge:
		if err = req.checkMerge(); err != nil {
			return
		}
	default:
		err = errors.Newf(status.ParamsIllegalErr, "action")
	}

	return
}

// checkUploadFile 文件上传的校验
func (req *UploadFileReq) checkUploadFile(c *gin.Context) (err error) {
	if err = req.checkMerge(); err != nil {
		return
	}

	if req.chunkSize == "" || req.chunkNumber == "" {
		err = errors.New(errors.BadRequest)
		return
	}

	req.uploadFile, req.header, err = c.Request.FormFile("uploadfile")
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	defer req.uploadFile.Close()

	if req.header == nil || req.uploadFile == nil {
		err = errors.New(status.ResourceNotChosenErr)
		return
	}

	return
}

// IsDir 判断是否以/结尾
func (req *UploadFileReq) IsDir(newPath string) bool {
	rex := regexp.MustCompile(`/$`)
	return rex.MatchString(newPath)
}

// handlerPath 处理路径
func (req *UploadFileReq) handlePath() (newPath string, err error) {
	absPath, _ := filepath.Abs(req.path)
	if req.IsDir(req.path) {
		absPath = fmt.Sprintf("%s/", absPath)
	}
	newPath, err = utils.GetNewPath(absPath)
	if err != nil {
		return
	}
	// 创建目录时处理路径
	if req.Action == "" && utils.CheckPath(absPath) {
		newPath = fmt.Sprintf("%s/", newPath)
	}

	return
}

// splitPath 根据文件后缀切分路径
func (req *UploadFileReq) splitPathByExt(newPath string) (path, ext string) {
	ext = filepath.Ext(newPath)
	if ext == "" {
		return newPath, ext
	}

	trimPath := strings.TrimSuffix(newPath, ext)
	return trimPath, ext
}

// isFileExist 判断文件是否存在
func (req *UploadFileReq) isFileExist(newPath string) bool {
	fs := filebrowser.GetFB()
	_, err := fs.Stat(newPath)
	return err == nil
}

// rename 如果文件重复，会自动凭接上后缀
func (req *UploadFileReq) getNewName(path string) string {
	//  不存在则直接返回
	if !req.isFileExist(path) {
		return path
	}

	// 把文件名和后缀拆开
	prefix, fileExt := req.splitPathByExt(path)
	for i := 1; ; i++ {
		newPath := fmt.Sprintf("%s(%d)%s", prefix, i, fileExt)
		if !req.isFileExist(newPath) {
			return newPath
		}
	}
}

// checkFileHash 校验文件哈希，上传前后是否一致
func (req *UploadFileReq) checkFileHash(newPath string) (err error) {
	// 校验hash
	hash := utils.SHA256File(newPath)
	field := strings.Split(req.Hash, "-")
	if field[0] != hash {
		err = errors.New(status.ParamFileHashError)
		// 合并文件hash不匹配删除文件
		filebrowser.GetFB().RemoveAll(newPath)
		return
	}
	return
}

// getCachePath 获取文件存储在cache目录下的路径
func (req *UploadFileReq) getCachePath(userID int) string {
	return fmt.Sprintf("/cache/%d/%s", userID, req.Hash)
}

// createFolder 创建createFolder数据
func (req *UploadFileReq) createFolder(newPath string, folderType int, uid int) error {
	if _, err := entity.CreateFolder(entity.GetDB(), &entity.FolderInfo{
		Name:      path.Base(newPath),              // 名称
		Type:      folderType,                      // 目录
		AbsPath:   strings.TrimRight(newPath, "/"), // 完整路径
		Uid:       uid,                             // 创建人
		CreatedAt: time.Now().Unix(),               // 创建时间
	}); err != nil {
		return err
	}
	return nil
}

// wrapResp 包装返回数据
func (req *UploadFileReq) wrapResp(newPath string, fs *filebrowser.FileBrowser) (resp UploadFileResp, err error) {
	fileInfo, err := fs.Stat(newPath)
	if err != nil {
		return
	}

	// 去掉旧路径最后一个元素，拼接上新路径的文件名称，避免文件重命名问题
	firstPath := strings.Split(req.path, "/")
	firstPath = firstPath[:len(firstPath)-1]
	firstPathStr := strings.Join(firstPath, "/")

	resp.Resource = Info{
		Name:    fileInfo.Name(),
		Size:    fileInfo.Size(),
		ModTime: fileInfo.ModTime().Unix(),
		Path:    filepath.Join(firstPathStr, filepath.Base(newPath)),
	}

	if !fileInfo.IsDir() {
		resp.Resource.Type = GetAllFile
	}

	return
}

// uploadLimit 上传限制处理
func (req *UploadFileReq) uploadLimit(folderId, uploadSize int) (err error) {
	// 通过 folder_id 查找pool_name partition_name
	folderInfo, err := entity.GetFolderInfo(folderId)
	if err != nil {
		return
	}
	pathSlice := strings.Split(folderInfo.AbsPath, "/")
	if len(pathSlice) < 3 {
		return errors.New(status.ResourcePathIllegalErr)
	}
	partitionInfo, err := utils.GetPartitionInfo(pathSlice[1], pathSlice[2])
	if err != nil {
		return
	}
	// 判断是不是系统分区&容量有没有超出限制
	if partitionInfo.PoolName == types.LvmSystemDefaultName && folderInfo.Name == types.LvmSystemDefaultName {
		// 90%的限制
		partitionInfo.FreeSize = partitionInfo.FreeSize / 10 * 9
		if partitionInfo.FreeSize <= int64(uploadSize) {
			// 系统分区超限制了
			err = errors.New(status.ResourceUploadLimitExceeded)
			return
		}
	} else if partitionInfo.FreeSize <= int64(uploadSize) {
		// 其它分区超限制了
		err = errors.New(status.ResourceUploadLimitExceeded)
		return
	}

	return
}
