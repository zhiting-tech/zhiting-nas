package resource

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	utils2 "gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"strconv"
	"strings"

	"os"
	"path/filepath"
	"sort"
	"time"
)

const (
	GetOneFile = iota
	GetAllFile

	pageSize = 30 // 默认分页大小
)

type GetResourceInfoReq struct {
	Path       string `uri:"path"`
	Type       int    `form:"type"`
	Page       int    `form:"page"`
	PageSize   int    `form:"page_size"`
	PageOffset int
}

type Info struct {
	Name      string `json:"name"`
	Size      int64  `json:"size"`
	ModTime   int64  `json:"mod_time"`
	Type      int    `json:"type"`
	Path      string `json:"path"`
	IsEncrypt int    `json:"is_encrypt"` // 是否加密
	Read      int    `json:"read"`       // 是否可读：1/0
	Write     int    `json:"write"`      // 是否可写：1/0
	Deleted   int    `json:"deleted"`    // 是否可删：1/0
}

func GetResourceInfo(c *gin.Context) {

	var (
		err      error
		req      GetResourceInfoReq
		list     []Info
		totalRow int64
	)

	defer func() {
		if len(list) == 0 {
			list = make([]Info, 0)
		}
		response.HandleResponseList(c, err, &list, totalRow)
	}()

	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	if err = c.BindQuery(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	req.Path = filepath.Join(req.Path)
	// 处理路径, 把/s/:id，处理成实际的路径
	newPath, err := utils.GetNewPath(req.Path)
	if err != nil {
		return
	}

	// 验证请求参数
	err = req.validateRequest(c)
	if err != nil {
		return
	}
	req.PageOffset = utils2.GetPageOffset(req.Page, req.PageSize)
	list, totalRow, err = req.wrapResources(newPath, c)
	if err != nil {
		return
	}
}

// 验证参数
func (req *GetResourceInfoReq) validateRequest(c *gin.Context) error {
	if req.Type < GetOneFile || req.Type > GetAllFile {
		err := errors.Newf(status.ParamsIllegalErr, "type")
		return err
	}

	// 如果不是根目录，需要判断是否有权限进入
	if req.Path != "" && req.Path != "/" {
		if _, err := utils.GetFolderSecret(req.Path, c.GetHeader("pwd")); err != nil {
			return err
		}
	}

	return nil
}

// wrapResources 组织返回数据
func (req *GetResourceInfoReq) wrapResources(newPath string, c *gin.Context) (infos []Info, totalRow int64, err error) {
	// 如果是第一级文件，则从数据库表里查出文件夹
	user := session.Get(c)
	if newPath == "" {
		err = initPrivateFolder(user)
		if err != nil {
			return
		}
		// 获取私人文件夹 && 可访问权限包含自己的文件夹 && 非分享文件夹
		whereStr := fmt.Sprintf("auth.uid = %d and auth.read = 1 and folder.mode = 1 and auth.is_share = 0", user.UserID)
		folderList, _ := entity.GetRelateFolderList(whereStr, req.PageOffset, req.PageSize)
		totalRow, _ = entity.GetRelateFolderCount(whereStr)

		for _, folder := range folderList {
			infos = append(infos, Info{
				Name:      folder.Name,
				Type:      folder.Type,
				Path:      fmt.Sprintf("/s/%d", folder.Id),
				IsEncrypt: folder.IsEncrypt,
				Read:      folder.Read,
				Write:     folder.Write,
				Deleted:   folder.Deleted,
			})
		}
	} else {
		infos, err = req.GetResourceInfos(newPath, c)
		if err != nil {
			return
		}
		totalRow = int64(len(infos))
		// type不为1时处理分页
		if req.Type != GetAllFile {
			req.handlePage(infos)
			infos = infos[req.PageOffset:req.PageSize]
		}
	}

	// 如果能从path获取到对应的folderId，把路径改成/s/:id 格式
	folderId, _ := utils.GetFolderIdFromPath(req.Path)
	folderInfo, _ := entity.GetFolderInfo(folderId)
	if folderId != 0 {
		for i, rs := range infos {
			// 更换路径， 保留/s/:id, 格式
			infos[i].Path = fmt.Sprintf("/s/%d%s", folderId, strings.TrimPrefix(rs.Path, folderInfo.AbsPath))
		}
	}

	return
}

// GetResourceInfos 获取文件及子目录列表
func (req *GetResourceInfoReq) GetResourceInfos(newPath string, c *gin.Context) (resourceInfos []Info, err error) {
	fs := filebrowser.GetFB()
	file, err := fs.Open(newPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = errors.Wrap(err, status.ResourceNotExistErr)
		} else {
			err = errors.Wrap(err, errors.InternalServerErr)
		}
		return
	}

	// 读取该目录下所有信息
	fileInfos, err := file.Readdir(-1)
	if err != nil {
		return
	}

	// 文件排序， 倒叙
	sort.Slice(fileInfos, func(i, j int) bool {
		return fileInfos[i].ModTime().After(fileInfos[j].ModTime())
	})

	// 因为是二级目录，能从上下文中获取权限
	isEncrypt, _ := c.Get("is_encrypt")
	read, _ := c.Get("read")
	write, _ := c.Get("write")
	deleted, _ := c.Get("deleted")

	for _, fileInfo := range fileInfos {
		resourceInfo := Info{
			Name:      fileInfo.Name(),
			Size:      fileInfo.Size(),
			ModTime:   fileInfo.ModTime().Unix(),
			Path:      filepath.Join(newPath, fileInfo.Name()),
			IsEncrypt: isEncrypt.(int),
			Read:      read.(int),
			Write:     write.(int),
			Deleted:   deleted.(int),
		}

		if !fileInfo.IsDir() {
			resourceInfo.Type = types.FolderTypeFile
		} else if req.Type == GetAllFile {
			// 如果type 为1且是目录, 则继续获取该目录下的信息
			var resourceList []Info
			resourceList, err = req.GetResourceInfos(resourceInfo.Path, c)
			if err != nil {
				return nil, err
			}
			resourceInfos = append(resourceInfos, resourceList...)
		}
		resourceInfos = append(resourceInfos, resourceInfo)
	}

	return
}

// initPrivateFolder 初始化私人文件夹
func initPrivateFolder(user *session.User) (err error) {
	folderOne, _ := entity.GetRelateFolderInfoByUid(types.FolderSelfDirUid, user.UserID)
	if folderOne != nil && folderOne.Id > 0 {
		// 个人文件夹已存在, 直接返回
		return
	}

	// 私人文件夹的路径
	selfDirName := fmt.Sprintf(types.FolderSelfDir, user.UserID)
	folderPath := filepath.Join("/", config.AppSetting.PoolName, config.AppSetting.PartitionName, selfDirName)
	// 如果不存在，新建文件夹
	if err = createPrivateFolder(user, folderPath); err != nil {
		return
	}

	return
}

// createPrivateFolder 创建私人文件夹
func createPrivateFolder(user *session.User, folderPath string) (err error) {
	fb := filebrowser.GetFB()
	db := entity.GetDB()
	err = fb.Mkdir(folderPath)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	// 查询是否有相似Nickname的目录
	count, err := entity.GetFoldersLikeNameCount(user.Nickname)
	if err != nil {
		return
	}
	name := user.Nickname + "-文件"
	// 如果数量大于0 则表示存在 则需要在文件夹名称加上数量
	if count > 0 {
		name = name + strconv.FormatInt(count, 10)
	}

	// 写入 folder
	selfFolderInfo := entity.FolderInfo{
		Uid:           types.FolderSelfDirUid,
		AbsPath:       folderPath,
		Name:          name, // 初始化的名称，默认是用户名称
		PoolName:      config.AppSetting.PoolName,
		PartitionName: config.AppSetting.PartitionName,
		Mode:          types.FolderPrivateDir,
		CreatedAt:     time.Now().Unix(),
		Persons:       user.Nickname,
	}
	_, err = entity.CreateFolder(db, &selfFolderInfo)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	auth := entity.FolderAuth{
		Uid:      user.UserID,
		IsShare:  0,
		Nickname: user.Nickname,
		FolderId: selfFolderInfo.ID,
		Read:     1,
		Write:    1,
		Deleted:  1,
	}
	err = entity.InsertAuth(db, auth)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	return
}

// handlePage 处理分页
func (req *GetResourceInfoReq) handlePage(resourceInfos []Info) {
	if req.PageSize == 0 {
		req.PageSize = pageSize
	}
	resourceNum := len(resourceInfos)
	if resourceNum < req.PageSize {
		req.PageSize = resourceNum
	}

	if req.PageOffset > resourceNum {
		req.PageOffset = resourceNum
	}

	req.PageSize = req.PageSize + req.PageOffset
	if req.PageSize > resourceNum {
		req.PageSize = resourceNum
	}
}
