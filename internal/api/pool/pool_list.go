package pool

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
	"sort"
)

// ListResp 出参中list结构体
type ListResp struct {
	Id          string            `json:"id"`           // 存储池唯一标识符
	Name        string            `json:"name"`         // 存储池名称
	Capacity    int64             `json:"capacity"`     // 容量（默认GB）
	UseCapacity int64             `json:"use_capacity"` // 已用容量（默认GB）
	Status      string			  `json:"status"`		// 异步任务状态
	TaskId		string			  `json:"task_id"`		// 异步任务ID
	Lv          []*LogicalVolume  `json:"lv"`           // 逻辑分区
	Pv          []*PhysicalVolume `json:"pv"`           // 物理分区
}

// LogicalVolume 逻辑分区
type LogicalVolume struct {
	Id          string `json:"id"`           // 存储池唯一标识符
	Name        string `json:"name"`         // 存储池名称
	Capacity    int64  `json:"capacity"`     // 容量（默认GB）
	UseCapacity int64  `json:"use_capacity"` // 已用容量（默认GB）
	Status      string `json:"status"`		 // 异步任务状态
	TaskId		string `json:"task_id"`		 // 异步任务ID
}

// PhysicalVolume 物理分区
type PhysicalVolume struct {
	Id       string `json:"id"`       // 存储池唯一标识符
	Name     string `json:"name"`     // 存储池名称
	Capacity int64  `json:"capacity"` // 容量（默认GB）
}

// ListReq 入参结构体
type ListReq struct {
	Page     int `form:"page"`
	PageSize int `form:"page_size"`
}

// GetPoolList 获取存储池列表
func GetPoolList(c *gin.Context) {
	var (
		list     []*ListResp
		req      ListReq
		err      error
		totalRow int64
	)
	// 注册延迟调用函数，延迟到当前方法所在函数返回时才会执行该函数
	defer func() {
		if len(list) == 0 {
			list = make([]*ListResp, 0)
		}
		response.HandleResponseList(c, err, &list, totalRow)
	}()

	if err = c.BindQuery(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return
	}
	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	groups, err := client.VolumeGroupList(ctx, &proto.Empty{})
	if err != nil {
		return
	}
	for _, vg := range groups.VGS {
		taskId, status := getPoolTaskInfo(vg.Name)
		info := &ListResp{
			Id:          vg.UUID,
			Name:        vg.Name,
			Capacity:    vg.Size,
			UseCapacity: vg.Size - vg.FreeSize,
			Status:      status,
			TaskId:      taskId,
		}
		// 逻辑分区
		info.Lv = make([]*LogicalVolume, len(vg.LVS))
		for key, lv := range vg.LVS {
			info.Lv[key] = &LogicalVolume{
				Id:          lv.UUID,
				Name:        lv.Name,
				Capacity:    lv.Size,
				UseCapacity: lv.Size - lv.FreeSize,
			}
		}
		// 排序
		sort.Slice(info.Lv, func(p, q int) bool {
			if info.Lv[q].Name == types.LvmSystemDefaultName {
				return false
			}
			return info.Lv[p].Name < info.Lv[q].Name
		})
		// 物理分区
		info.Pv = make([]*PhysicalVolume, len(vg.PVS))
		for key, pv := range vg.PVS {
			info.Pv[key] = &PhysicalVolume{
				Id:       pv.UUID,
				Name:     pv.Name,
				Capacity: pv.Size,
			}
		}
		list = append(list, info)
	}

	// 排序
	sort.Slice(list, func(p, q int) bool {
		if list[q].Name == types.LvmSystemDefaultName {
			return false
		}
		return list[p].Name < list[q].Name
	})
	totalRow = int64(len(list))
}

// getPoolTaskInfo 获取存储池分区的异步任务状态
func getPoolTaskInfo(poolName string) (taskId string, status string) {
	taskManager := task.GetTaskManager()

	if taskInfo, ok := taskManager.GetTaskInfo(types.TaskDelPool, poolName); ok {
		taskId = fmt.Sprintf("%s_%s", types.TaskDelPool, poolName)
		status = fmt.Sprintf("%s_%d", types.TaskDelPool, taskInfo.Status)
	}

	return
}