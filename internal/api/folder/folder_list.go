package folder

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"strconv"
)

type Info struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	IsEncrypt int    `json:"is_encrypt"`
	Mode      int    `json:"mode"`
	Path      string `json:"path"`
	Type      int    `json:"type"`
	Persons   string `json:"persons"` // 可访问成员
	PoolName  string `json:"pool_name"` // 存储池名称
	Status    string `json:"status"` // 任务状态
	TaskId	  string `json:"task_id"` // 任务ID
}

type GetFolderListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// GetFolderList 获取列表
func GetFolderList(c *gin.Context) {
	var (
		err      error
		req      GetFolderListReq
		list     []*Info
		totalRow int64
	)

	defer func() {
		if len(list) == 0 {
			list = make([]*Info, 0)
		}
		response.HandleResponseList(c, err, &list, totalRow)
	}()

	if err = c.BindQuery(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 获取登陆用户
	user := session.Get(c)
	// 查询文件列表
	pageOffset := utils.GetPageOffset(req.Page, req.PageSize)
	folderInfos, err := entity.GetFolderList(user.UserID, pageOffset, req.PageSize)
	if err != nil {
		return
	}

	for _, folderInfo := range folderInfos {
		taskId, status := getFolderTaskInfo(folderInfo.ID)
		list = append(list, &Info{
			ID:        folderInfo.ID,
			Name:      folderInfo.Name,
			IsEncrypt: folderInfo.IsEncrypt,
			Mode:      folderInfo.Mode,
			Path:      folderInfo.AbsPath,
			Type:      folderInfo.Type,
			Persons:   folderInfo.Persons,
			PoolName:  fmt.Sprintf("%s-%s", folderInfo.PoolName, folderInfo.PartitionName),
			Status:	   status,
			TaskId:	   taskId,
		})
	}

	totalRow = entity.GetFolderCount(user.UserID)
}

// getFolderStatus 获取任务状态
func getFolderTaskInfo(folderId int) (taskId string, status string) {
	taskManager := task.GetTaskManager()
	if taskInfo, ok := taskManager.GetTaskInfo(types.TaskMovingFolder, strconv.Itoa(folderId)); ok {
		// 移动目录的任务
		taskId = fmt.Sprintf("%s_%s", types.TaskMovingFolder, strconv.Itoa(folderId))
		status = fmt.Sprintf("%s_%d", types.TaskMovingFolder, taskInfo.Status)
	} else if taskInfo, ok = taskManager.GetTaskInfo(types.TaskDelFolder, strconv.Itoa(folderId)); ok {
		// 删除文件夹的目录
		taskId = fmt.Sprintf("%s_%s", types.TaskDelFolder, strconv.Itoa(folderId))
		status = fmt.Sprintf("%s_%d", types.TaskDelFolder, taskInfo.Status)
	}

	return
}