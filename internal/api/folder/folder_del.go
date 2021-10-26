package folder

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"gorm.io/gorm"
	"strconv"
)

type DelResp struct {
}

type DelReq struct {
	Id int `uri:"id"`
}

func DelFolder(c *gin.Context) {
	var (
		req  DelReq
		resp DelResp
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()
	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	user := session.Get(c)
	_, err = req.validateRequest(user.UserID)
	if err != nil {
		return
	}

	// 异步执行删除
	task.GetTaskManager().Add(types.TaskDelFolder, strconv.Itoa(req.Id), &req)
}

// validateRequest 认证请求方式
func (req *DelReq) validateRequest(UserID int) (oldInfo *entity.FolderInfo, err error) {
	// 查询旧数据是否存在
	oldInfo, err = entity.GetFolderInfo(req.Id)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	// 需要判断是否是自己的文件夹
	//if oldInfo.Uid != UserID {
	//	err = errors.Wrap(err, errors.InternalServerErr)
	//	return
	//}

	return
}

// ExecTask 执行异步任务
func (req *DelReq) ExecTask() error {
	// 查询旧数据是否存在
	oldInfo, err := entity.GetFolderInfo(req.Id)
	if err != nil {
		return errors.Wrap(err, errors.InternalServerErr)
	}

	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := entity.DelFolder(tx, oldInfo.AbsPath); err != nil {
			return errors.Wrap(err, status.FolderDelFailErr)
		}

		if err = entity.DelFolderAuth(tx, req.Id); err != nil {
			return errors.Wrap(err, status.FolderDelFailErr)
		}

		if err = filebrowser.GetFB().RemoveAll(oldInfo.AbsPath); err != nil {
			return errors.Wrap(err, status.FolderDelFailErr)
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}
