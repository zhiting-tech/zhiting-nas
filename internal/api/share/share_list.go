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
	Page       int `form:"page"`
	PageSize   int `form:"page_size"`
	PageOffset int
}

type Info struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	Path         string `json:"path"`
	FromUser     string `json:"from_user"`
	Read         int    `json:"read"`
	Write        int    `json:"write"`
	Deleted      int    `json:"deleted"`
	IsFamilyPath int    `json:"is_family_path"`
}

func GetShareList(c *gin.Context) {
	var (
		req      GetShareReq
		err      error
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

	// 查询folder是否存在家庭文件夹数据
	tmpFamilyFolderInfo := entity.QueryFolderByUid(types.FolderFamilyDirUid)
	if tmpFamilyFolderInfo.ID == 0 {
		if err = createShareFolder(types.FolderFamilyDir, user.AreaName, types.FolderFamilyDirUid); err != nil {
			return
		}
	}

	// 如果家庭目录的名称不一致，则需要调整家庭名称
	if tmpFamilyFolderInfo.Name != user.AreaName {
		_ = entity.UpdateFolderInfo(entity.GetDB(), tmpFamilyFolderInfo.ID, map[string]interface{}{"name": user.AreaName})
	}

	// 写入folderAuth表  查看登陆用户是否存在权限，如果不存在，则需要写入
	tmpFamilyFolderInfo = entity.QueryFolderByUid(types.FolderFamilyDirUid)
	if err = createFolderAuth(user.UserID, tmpFamilyFolderInfo.ID, user.Nickname); err != nil {
		return
	}

	// 如果是公司类型则创建部门文件夹
	if user.AreaType == types.AreaCompanyType {
		var tmpDepartmentInfo *entity.FolderInfo
		for _, v := range user.DepartmentBaseInfos {
			// 查询folder是否存在部门文件夹数据
			tmpDepartmentInfo = entity.QueryFolderByUid(-int(v.DepartmentId))
			if tmpDepartmentInfo.ID == 0 {
				if err = createShareFolder(types.FolderDepartment, v.Name, -int(v.DepartmentId)); err != nil {
					return
				}
			}
			// 如果部门文件夹目录名称不一致， 则需要调整部门名称
			if tmpDepartmentInfo.Name != v.Name {
				_ = entity.UpdateFolderInfo(entity.GetDB(), tmpDepartmentInfo.ID, map[string]interface{}{"name": v.Name})
			}

			// 给用户部门的权限
			tmpDepartmentInfo = entity.QueryFolderByUid(-int(v.DepartmentId))
			if err = createFolderAuth(user.UserID, tmpDepartmentInfo.ID, user.Nickname); err != nil {
				return
			}
		}
	}

	return
}

func createFolderAuth(userId, folderId int, nickName string) (err error) {
	folderAuth, _ := entity.GetFolderAuthByUidAndFolderId(userId, folderId)
	if folderAuth == nil || folderAuth.ID == 0 {
		tmpFolderAuth := entity.FolderAuth{
			Uid:      userId,
			Nickname: nickName,
			FolderId: folderId,
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
	return nil
}

func createShareFolder(folderDir, name string, folderDirId int) (err error) {
	var homePath string
	fb := filebrowser.GetFB()
	// 如果目录不存在，则新建一个
	if folderDirId != types.FolderFamilyDirUid {
		homePath = filepath.Join("/", config.AppSetting.PoolName, config.AppSetting.PartitionName, fmt.Sprintf(folderDir, -folderDirId))
	} else {
		homePath = filepath.Join("/", config.AppSetting.PoolName, config.AppSetting.PartitionName, folderDir)
	}

	if err = fb.Mkdir(homePath); err != nil {
		return
	}
	// 写入folder表
	tmpFolderInfo := &entity.FolderInfo{
		Uid:           folderDirId,
		AbsPath:       homePath,
		Name:          name, // 固定为家庭名称
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

	return nil
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
			ID:           folderRow.Id,
			Name:         folderRow.Name,
			Path:         fmt.Sprintf("/s/%d", folderRow.Id),
			FromUser:     folderRow.FromUser,
			Read:         folderRow.Read,
			Write:        folderRow.Write,
			Deleted:      folderRow.Deleted,
			IsFamilyPath: isFamilyPath,
		})
	}

	totalRow, _ = entity.GetRelateFolderCount(whereStr)
	return
}
