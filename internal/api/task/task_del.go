package task

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
)

type DelResp struct {

}

type DelReq struct {
	Id    string `uri:"id"`
}

// DelTask 删除任务
func DelTask(c *gin.Context) {
	var (
		resp DelResp
		req  DelReq
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()

	// 入参有问题，返回BadRequest
	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	taskInfo, exist := task.GetTaskManager().GetTaskInfoByKey(req.Id)
	if !exist {
		err = errors.Wrap(err, status.TaskNotExistErr)
		return
	}

	if taskInfo.Status == types.TaskOnGoing {
		err = errors.Wrap(err, status.TaskStatusErr)
		return
	}

	task.GetTaskManager().DelByKey(req.Id)
	return
}
