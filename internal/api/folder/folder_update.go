package folder

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gorm.io/gorm"
	"path"
	"strconv"
	"strings"
	"unicode/utf8"
)

type UpdateResp struct {
}

type UpdateAuthReq struct {
	Uid      int    `json:"u_id"`
	Nickname string `json:"nickname"`
	Face     string `json:"face"`
	Read     int    `json:"read"`
	Write    int    `json:"write"`
	Deleted  int    `json:"deleted"`
}

type UpdateReq struct {
	ID            int             `uri:"id"`
	Name          string          `json:"name"`           // 文件/文件夹名称
	Mode          int             `json:"mode"`           // 文件夹类型：1私人文件夹 2共享文件夹
	PoolName      string          `json:"pool_name"`      // 储存池ID
	PartitionName string          `json:"partition_name"` // 储存池分区ID
	Auth          []UpdateAuthReq `json:"auth"`           // 可访问成员的权限
}

func UpdateFolder(c *gin.Context) {
	var (
		req  UpdateReq
		resp UpdateResp
		err  error
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
	// 请求参数校验
	oldInfo, err := req.validateRequest()
	if err != nil {
		return
	}
	// 权限数组,检查权限数组是否有重复用户（uid为判断标准）,并过滤掉重复的id
	var auths = make([]entity.FolderAuth, 0, len(req.Auth))
	var persons = make([]string, 0, len(req.Auth))
	var noRepeatUid = make(map[int]int)

	for _, auth := range req.Auth {
		if _, ok := noRepeatUid[auth.Uid]; !ok {
			noRepeatUid[auth.Uid] = auth.Uid
			auths = append(auths, entity.FolderAuth{
				Uid:      auth.Uid,
				Nickname: auth.Nickname,
				Face:     auth.Face,
				Read:     auth.Read,
				Write:    auth.Write,
				Deleted:  auth.Deleted,
			})
			persons = append(persons, auth.Nickname)
		}
	}

	// 更新文件夹
	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		values := map[string]interface{}{
			"name":           req.Name,
			"mode":           req.Mode,
			"Persons":        strings.Join(persons, "、"), // 可访问成员
		}
		if err = entity.UpdateFolderInfo(tx, req.ID, values); err != nil {
			return err
		}
		// 权限先删除
		if err = entity.DelFolderAuth(tx, req.ID); err != nil {
			return errors.Wrap(err, status.FolderUpdateFailErr)
		}
		// 判断编辑后的未加密文件夹是否为共享文件夹，Mode为2时，是共享文件夹，isShare=1
		isShare := 0
		if req.Mode == types.FolderShareDir {
			isShare = 1
		}
		// 权限再添加
		for key := range auths {
			auths[key].FolderId = req.ID
			auths[key].IsShare = isShare
		}
		if err = entity.BatchInsertAuth(tx, auths); err != nil {
			err = errors.Wrap(err, status.FolderUpdateFailErr)
			return err
		}
		// 处理文件夹下的目录问题
		if err = req.updateOldPath(tx, oldInfo); err != nil {
			err = errors.Wrap(err, status.FolderUpdateFailErr)
			return err
		}

		return nil
	}); err != nil {
		return
	}
}

// validateRequest 认证请求方式
func (req *UpdateReq) validateRequest() (oldInfo *entity.FolderInfo, err error) {
	// 校验必填
	if req.Name == "" || req.PoolName == "" || req.PartitionName == "" || req.Mode == 0 {
		err = errors.Wrap(err, status.FolderParamFailErr)
		return
	}

	// 校验名称长度
	if utf8.RuneCountInString(req.Name) > 100 {
		err = errors.Wrap(err, status.FolderNameTooLongErr)
		return
	}

	// 校验名称是否重复
	folderInfo, err := entity.GetFolderByName(req.PoolName, req.PartitionName, req.Name)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	if folderInfo.ID != 0 && folderInfo.ID != req.ID {
		// 存在同名文件夹
		err = errors.Wrap(err, status.FolderNameIsExistErr)
		return
	}

	// 查询旧数据是否存在
	oldInfo, err = entity.GetFolderInfo(req.ID)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	// 已加密文件夹不允许修改类型
	if req.Mode != folderInfo.Mode && folderInfo.IsEncrypt == 1 {
		err = errors.Wrap(err, status.FolderCannotModified)
		return
	}

	// 校验文件夹类型
	if req.Mode == types.FolderPrivateDir {
		// 私人文件夹
		if len(req.Auth) != 1 {
			err = errors.Wrap(err, status.FolderTooMuchMemberErr)
			return
		}
	} else if req.Mode == types.FolderShareDir {
		// 共享文件夹,成员大于等于1
		if len(req.Auth) < 1 {
			err = errors.Wrap(err, status.FolderTooFewMemberErr)
			return
		}
	}

	// 分区修改，判断空间是否足够 存储池和分区
	if oldInfo.PoolName != req.PoolName || oldInfo.PartitionName != req.PartitionName {
		folderSize, _ := utils.GetFolderSize(oldInfo.AbsPath)
		partitionInfo, err := utils.GetPartitionInfo(req.PoolName, req.PartitionName)
		if err != nil {
			err = errors.Wrap(err, status.PoolIsNotFoundErr)
			return nil, err
		}
		if folderSize > partitionInfo.FreeSize {
			err = errors.Wrap(err, status.FolderTargetTooSmallErr)
			return nil, err
		}
	}

	return
}

// updateOldPath 修改子目录的路径
func (req *UpdateReq) updateOldPath(tx *gorm.DB, oldInfo *entity.FolderInfo) error {
	// 如果名称没有修改
	if oldInfo.Name != req.Name {
		// 名称跟绝对路径名称相等才需要调整路径
		if oldInfo.Name == path.Base(oldInfo.AbsPath) {
			// 修改目录信息
			oldPath := fmt.Sprintf("/%s/%s/%s", oldInfo.PoolName, oldInfo.PartitionName, oldInfo.Name)
			newPath := fmt.Sprintf("/%s/%s/%s", oldInfo.PoolName, oldInfo.PartitionName, req.Name)
			if err := utils.UpdateFolderPath(tx, oldPath, newPath); err != nil {
				return err
			}
			// 修改文件夹名称，直接调整
			if err := filebrowser.GetFB().Rename(oldPath, newPath); err != nil {
				config.Logger.Errorf("update folder name fail %v", err)
				return err
			}
		} else {
			_ = entity.UpdateFolderInfo(tx, oldInfo.ID, map[string]interface{}{"name": req.Name})
		}
	}
	// 修改了分区名称，需要异步执行
	if oldInfo.PoolName != req.PoolName || oldInfo.PartitionName != req.PartitionName {
		task.GetTaskManager().Add(types.TaskMovingFolder, strconv.Itoa(req.ID), req)
	}

	return nil
}

// ExecTask 执行异步任务
func (req *UpdateReq) ExecTask() error {
	info, err := entity.GetFolderInfo(req.ID)
	if err != nil {
		return errors.New(errors.InternalServerErr)
	}
	// 不需要做迁移
	if info.PoolName == req.PoolName && info.PartitionName == req.PartitionName {
		return nil
	}

	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		// 修改数据库数据
		oldPath := fmt.Sprintf("%s", info.AbsPath)
		newPath := fmt.Sprintf("/%s/%s/%s", req.PoolName, req.PartitionName, path.Base(info.AbsPath))
		// 移动到对应的分区下
		if err = filebrowser.GetFB().CopyDir(oldPath, fmt.Sprintf("/%s/%s/", req.PoolName, req.PartitionName)); err != nil {
			config.Logger.Errorf("update folder name fail %v", err)
			return err
		}
		if err = filebrowser.GetFB().RemoveAll(oldPath); err != nil {
			config.Logger.Errorf("update folder name fail %v", err)
			return err
		}
		// 修改子目录的数据
		if err = utils.UpdateFolderPath(tx, oldPath, newPath); err != nil {
			return err
		}
		// 修改文件夹的pool数据和partition_name数据
		values := map[string]interface{}{
			"pool_name":      req.PoolName,
			"partition_name": req.PartitionName,
			"abs_Path":       newPath,
		}
		if err = entity.UpdateFolderInfo(tx, req.ID, values); err != nil {
			return err
		}

		return nil
	}); err != nil {
		return err
	}

	return nil
}


