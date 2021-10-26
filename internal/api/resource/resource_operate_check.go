package resource

import (
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"path/filepath"
	"strconv"
	"strings"
)

// validateRequest 验证请求参数
func (req *OperateResourceReq) validateRequest(userID int) (err error) {
	if req.Action != copyAction && req.Action != moveAction {
		err = errors.Newf(status.ParamsIllegalErr, "action")
		return
	}

	if req.Destination == "" || len(req.Sources) == 0 {
		err = errors.New(errors.BadRequest)
		return
	}

	// 检查参数路径是否合法
	if err = req.CheckParamPath(); err != nil {
		return
	}

	// 检查操作可行性
	if err = req.CheckWorkable(userID); err != nil {
		return
	}
	return nil
}

// CheckParamPath 判断参数路径是否合法
func (req *OperateResourceReq) CheckParamPath() (err error) {
	// TODO folderId 有可能不是根目录
	folderId, err := utils.GetFolderIdFromPath(req.Destination)
	if err != nil {
		return
	}
	// 查看是否存在异步任务
	if _, ok := task.GetTaskManager().GetTaskInfo(types.TaskMovingFolder, strconv.Itoa(folderId)); ok {
		err = errors.New(status.ResourceTargetOperatingErr)
		return
	} else if _, ok = task.GetTaskManager().GetTaskInfo(types.TaskDelFolder, strconv.Itoa(folderId)); ok {
		err = errors.New(status.ResourceTargetOperatingErr)
		return
	}

	// 获取实际的目标路径
	destination, err := utils.GetNewPath(req.Destination)
	if err != nil {
		return
	}

	fs := filebrowser.GetFB()
	isDir, err := fs.IsDir(destination)
	if err != nil {
		return
	}

	// 目标路径必须是目录
	if !isDir {
		err = errors.New(status.TargetPathMustBeDirErr)
		return
	}

	return
}

// CheckWorkable 检查操作可行性
func (req *OperateResourceReq) CheckWorkable(userID int) (err error) {
	fs := filebrowser.GetFB()
	var sourceResource *FolderResource

	// 获取目标目录信息
	destinationResource, err := req.getInfoByPath(userID, req.Destination)
	if err != nil {
		return
	}
	// 是否加密路径内移动，定义数组，作为判断
	req.IsEncryptMove = make([]int, len(req.Sources))
	// 需要移动文件的总大小
	var sourceSumSize int64
	// 检查源路径
	for i, source := range req.Sources {
		// 获取源目录信息
		sourceResource, err = req.getInfoByPath(userID, req.Sources[i])
		if err != nil {
			return
		}
		// 查看是否存在异步任务
		if _, ok := task.GetTaskManager().GetTaskInfo(types.TaskMovingFolder, strconv.Itoa(sourceResource.folderId)); ok {
			err = errors.New(status.ResourceSourceOperatingErr)
			return
		} else if _, ok = task.GetTaskManager().GetTaskInfo(types.TaskDelFolder, strconv.Itoa(sourceResource.folderId)); ok {
			err = errors.New(status.ResourceSourceOperatingErr)
			return
		}
		// 替换正式路径
		req.Sources[i] = sourceResource.realPath

		// 源文件所在文件夹与目标文件夹不能是同一文件夹
		srcDir := filepath.Dir(sourceResource.realPath)
		if destinationResource.realPath == srcDir {
			err = errors.New(status.TargetPathSameWithOriginalPathErr)
			return err
		}

		// 父级目录不能复制/移动到子级目录
		if strings.HasPrefix(req.Destination, source + "/") {
			err = errors.New(status.ResourceTargetSubdirectoryErr)
			return err
		}

		// 文件已存在目标文件或文件夹，不允许操作
		// 取source目录的最后一层名称，或者是文件名称
		src := source[strings.LastIndex(source, "/") + 1:]
		destSource := filepath.Join(destinationResource.realPath, src)
		_, err = fs.Open(destSource)
		if err == nil {
			err = errors.New(status.ResourceExistErr)
			return err
		}

		// 判断权限
		if err = req.checkAuth(sourceResource.folderAuth, destinationResource.folderAuth); err != nil {
			return
		}

		// 当源文件是私人加密文件或者他人共享文件时
		if err = req.sourceCheck(destinationResource.folderId, sourceResource); err != nil {
			return
		}
		// 目标路径是加密,需要输入密码,文件需要加密
		if destinationResource.folderInfo.IsEncrypt == 1 {
			if _, err = req.checkPwd(); err != nil {
				return
			}
			// 如果是加密路径内部自己的移动，不需要对目录进行加密
			if destinationResource.folderId == sourceResource.folderId {
				req.IsEncryptMove[i] = 1
			}
		}
		size, _ :=  utils.GetFolderSize(req.Sources[i])
		sourceSumSize += size
	}

	// 判断目标路径是否空间不足
	if err = req.destLimit(destinationResource.realPath, sourceSumSize); err != nil {
		return
	}

	return
}

// destLimit 目标路径的大小限制
func (req *OperateResourceReq) destLimit(destPath string, sourceSumSize int64) error {
	pathSlice := strings.Split(destPath, "/")
	if len(pathSlice) < 3 {
		return errors.New(status.ResourcePathIllegalErr)
	}
	partitionInfo, err := utils.GetPartitionInfo(pathSlice[1], pathSlice[2])
	if err != nil {
		return err
	}
	// 判断是不是系统分区&容量有没有超出限制
	if pathSlice[1] == types.LvmSystemDefaultName && pathSlice[2] == types.LvmSystemDefaultName {
		// 90%的限制
		partitionInfo.FreeSize = partitionInfo.FreeSize / 10 * 9
		if partitionInfo.FreeSize <= sourceSumSize {
			// 系统分区超限制了
			return errors.New(status.ResourceTargetLimitExceeded)
		}
	} else if partitionInfo.FreeSize <= sourceSumSize {
		// 其它分区超限制了
		return errors.New(status.ResourceTargetLimitExceeded)
	}

	return nil
}

// CheckAuth 判断权限
func (req OperateResourceReq) checkAuth(sourceFolderAuth, destinationFolderAuth *entity.FolderAuth) (err error) {
	// 判断是否有写权限
	if destinationFolderAuth.Write == 0 {
		if req.Action == moveAction {
			err = errors.Wrap(err, status.ResourceNotMoveErr)
			return
		} else if req.Action == copyAction {
			err = errors.Wrap(err, status.ResourceNotCopyErr)
			return
		}
	}

	// 移动时，需要有移出文件夹的删除权限
	if req.Action == moveAction {
		if sourceFolderAuth.Deleted == 0 {
			err = errors.Wrap(err, status.ResourceNotMoveErr)
			return
		}
	}
	return
}

func (req OperateResourceReq) sourceCheck(destinationFolderId int, sourceResource *FolderResource) (err error) {
	// source私人文件加密，目标路径在该私人文件下移动或复制, 源目录的根目录ID
	if sourceResource.folderInfo.IsEncrypt == 1 && sourceResource.folderInfo.Mode == 1 {
		if sourceResource.folderId != destinationFolderId {
			err = errors.New(status.ResourcePrivateErr)
			return
		}
	}

	// 别人共享的文件夹 只能在该共享文件夹下移动
	if req.Action == moveAction {
		if sourceResource.folderAuth.FromUser != "" && sourceResource.folderAuth.IsShare == 1 {
			if sourceResource.folderId != destinationFolderId {
				err = errors.New(status.ShareResourceBanMoveOtherDir)
				return
			}
		}
	}
	return
}

func (req OperateResourceReq) checkPwd() (string, error) {
	// 获取目录的密钥且校验密码，如果密钥为空
	return utils.GetFolderSecret(req.Destination, req.DestinationPwd)
}

func (req OperateResourceReq) getInfoByPath(userID int, path string) (*FolderResource, error) {
	// 实际路径
	realPath, err := utils.GetNewPath(path)
	if err != nil {
		return nil, err
	}

	// 目录的根目录ID
	folderId, err := utils.GetFolderIdFromPath(path)
	if err != nil {
		return nil, err
	}

	// 资源信息
	folderInfo, err := entity.GetFolderInfo(folderId)
	if err != nil {
		return nil, err
	}

	// 资源权限
	folderAuth, err := entity.GetFolderAuthByUidAndFolderId(userID, folderId)
	if err != nil {
		return nil, err
	}

	resource := &FolderResource{
		realPath:   realPath,
		folderInfo: folderInfo,
		folderId:   folderId,
		folderAuth: folderAuth,
	}

	return resource, nil
}