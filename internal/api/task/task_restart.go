package task

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
)

type RestartResp struct {

}

type RestartReq struct {
	Id    string `uri:"id"`
}

// RestartTask 重新启动任务
func RestartTask(c *gin.Context) {
	var (
		resp RestartResp
		req  RestartReq
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

	task.GetTaskManager().RestartByKey(req.Id)

	return
}
