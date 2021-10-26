package share

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"path/filepath"
	"time"
)

type GetShareReq struct {
	Page 	   int    `form:"page"`
	PageSize   int    `form:"page_size"`
	PageOffset int
}

type Info struct {
	ID       		int    `json:"id"`
	Name     		string `json:"name"`
	Path     		string `json:"path"`
	FromUser 		string `json:"from_user"`
	Read     		int    `json:"read"`
	Write    		int    `json:"write"`
	Deleted  		int    `json:"deleted"`
	IsFamilyPath 	int	   `json:"is_family_path"`
}

func GetShareList(c *gin.Context) {
	var (
		req 	 GetShareReq
		err  	 error
		list     []Info
		totalRow int64
	)

	defer func() {
		if len(list) == 0 {
			list = make([]Info, 0)
		}
		response.HandleResponseList(c, err, &list, totalRow)
	}()
	if err = c.BindQuery(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	user := session.Get(c)
	// 初始化家庭文件夹
	err = initGroupFolder(user)
	if err != nil {
		return
	}
	req.PageOffset = utils.GetPageOffset(req.Page, req.PageSize)

	list, totalRow, err = req.GetSharesForSelf(user)

	if err != nil {
		return
	}
	return
}

// initGroupFolder 初始化家庭文件夹
func initGroupFolder(user *session.User) (err error) {
	fb := filebrowser.GetFB()
	// 查询folder是否存在家庭文件夹数据
	tmpFolderInfo := entity.QueryFolderByUid(types.FolderFamilyDirUid)
	if tmpFolderInfo.ID == 0 {
		// 如果目录不存在，则新建一个
		homePath := filepath.Join("/", config.AppSetting.PoolName, config.AppSetting.PartitionName, types.FolderFamilyDir)
		if err = fb.Mkdir(homePath); err != nil {
			return
		}
		// 写入folder表
		tmpFolderInfo = &entity.FolderInfo{
			Uid:           types.FolderFamilyDirUid,
			AbsPath:       homePath,
			Name:          user.AreaName, // 固定为家庭名称
			PoolName:      config.AppSetting.PoolName,
			PartitionName: config.AppSetting.PartitionName,
			Mode:          types.FolderShareDir,
			CreatedAt:     time.Now().Unix(),
		}
		_, err = entity.CreateFolder(entity.GetDB(), tmpFolderInfo)
		if err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
	}

	// 如果家庭目录的名称不一致，则需要调整家庭名称
	if tmpFolderInfo.Name != user.AreaName {
		_ = entity.UpdateFolderInfo(entity.GetDB(), tmpFolderInfo.ID, map[string]interface{}{"name": user.AreaName})
	}

	// 写入folderAuth表  查看登陆用户是否存在权限，如果不存在，则需要写入
	folderAuth, _ := entity.GetFolderAuthByUidAndFolderId(user.UserID, tmpFolderInfo.ID)
	if folderAuth == nil || folderAuth.ID == 0 {
		tmpFolderAuth := entity.FolderAuth{
			Uid:      user.UserID,
			Nickname: user.Nickname,
			FolderId: tmpFolderInfo.ID,
			IsShare:  1,
			Read:     1,
			Deleted:  1,
			Write:    1,
		}
		if err = entity.InsertAuth(entity.GetDB(), tmpFolderAuth); err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
	}

	return
}

// GetSharesForSelf 获取别人共享给自己的文件
func (req *GetShareReq) GetSharesForSelf(user *session.User) (list []Info, totalRow int64, err error) {
	whereStr := fmt.Sprintf("auth.uid = %d and auth.read = 1 and auth.is_share = 1", user.UserID)
	folderList, err := entity.GetRelateFolderList(whereStr, req.PageOffset, req.PageSize)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}

	for _, folderRow := range folderList {
		isFamilyPath := 0
		if filepath.Base(folderRow.AbsPath) == types.FolderFamilyDir {
			isFamilyPath = 1
		}
		list = append(list, Info{
			ID:       folderRow.Id,
			Name:     folderRow.Name,
			Path:     fmt.Sprintf("/s/%d", folderRow.Id),
			FromUser: folderRow.FromUser,
			Read:     folderRow.Read,
			Write:    folderRow.Write,
			Deleted:  folderRow.Deleted,
			IsFamilyPath: isFamilyPath,
		})
	}

	totalRow, _ = entity.GetRelateFolderCount(whereStr)
	return
}
