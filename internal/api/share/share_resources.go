package share

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
)

type ResourcesShareReq struct {
	FromUser string   `json:"from_user"`
	ToUsers  []int    `json:"to_users"`
	Read     int      `json:"read"`
	Write    int      `json:"write"`
	Deleted  int      `json:"deleted"`
	Paths    []string `json:"paths"`
}

// ResourcesShare 分享目录
func ResourcesShare(c *gin.Context) {
	var (
		req ResourcesShareReq
		err error
	)

	defer func() {
		response.HandleResponse(c, err, nil)
	}()

	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	user := session.Get(c)

	if err = req.validateRequest(user.UserID); err != nil {
		return
	}

	if err = req.CreateShares(user.Nickname); err != nil {
		return
	}
}

// validateRequest 校验
func (req *ResourcesShareReq) validateRequest(userID int) (err error) {
	if len(req.ToUsers) == 0 || len(req.Paths) == 0 {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 不能共享文件给自己
	for _, uID := range req.ToUsers {
		if uID == userID {
			err = errors.New(status.ShareTargetUserIsSelfErr)
			return
		}

	}
	// 判断当前用户是否有写权限共享文件夹
	var folderId int
	for _, path := range req.Paths {
		folderId, err = utils.GetFolderIdFromPath(path)
		if err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
		var fileAuth *entity.FolderAuth
		fileAuth, err = entity.GetFolderAuthByUidAndFolderId(userID, folderId)
		if err != nil {
			err = errors.Wrap(err, status.ShareWithoutPermissionErr)
			return
		}
		if fileAuth.Write == 0 {
			err = errors.Wrap(err, status.ShareWithoutPermissionErr)
			return
		}

		var fileInfo *entity.FolderInfo
		fileInfo, err = entity.GetFolderInfo(folderId)
		if err != nil {
			err = errors.Wrap(err, status.ResourceNotExistErr)
			return
		}
		// 加密文件夹不能共享
		if fileInfo.IsEncrypt != 0 {
			err = errors.Wrap(err, status.FolderEncryptCannotShareErr)
			return
		}
	}

	return
}

// CreateShares 创建分享
func (req ResourcesShareReq) CreateShares(nickname string) (err error) {
	var folderId int
	var folderAuthCreates []entity.FolderAuth
	for _, path := range req.Paths {
		// path转换为实际路径
		folderId, err = utils.GetAbsFolderIdFromPath(path)
		if err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}

		for _, uID := range req.ToUsers {
			// 权限存在则删除
			err = entity.DelFolderAuthByUidAndFolderId(uID, folderId)
			if err != nil {
				err = errors.Wrap(err, errors.InternalServerErr)
				return
			}
			folderAuthCreate := entity.FolderAuth{
				Uid:      uID,
				FromUser: nickname,
				IsShare:  1,
				FolderId: folderId,
				Read:     req.Read,
				Write:    req.Write,
				Deleted:  req.Deleted,
			}
			folderAuthCreates = append(folderAuthCreates, folderAuthCreate)
		}
	}
	if folderAuthCreates != nil {
		if err = entity.BatchInsertAuth(entity.GetDB(), folderAuthCreates); err != nil {
			err = errors.Wrap(err, errors.InternalServerErr)
			return
		}
	}
	return
}
