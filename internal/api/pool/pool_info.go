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
	"reflect"
	"sort"
	"strings"
)

// InfoResp 出参中data结构体
type InfoResp struct {
	Id          string    		 `json:"id"`           // 存储池唯一标识符
	Name        string    		 `json:"name"`         // 存储池名称
	Capacity    int64     		 `json:"capacity"`     // 容量（默认GB）
	UseCapacity int64     		 `json:"use_capacity"` // 已用容量（默认GB）
	Lv			[]*LogicalVolume `json:"lv"`     	   // 逻辑分区
	Pv			[]*PhysicalVolume`json:"pv"`     	   // 物理分区
}


type InfoReq struct {
	Name string `uri:"name"`
}

func GetPoolInfo(c *gin.Context) {
	var (
		resp InfoResp
		req  InfoReq
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

	// 获取正在进行中的存储池任务
	taskListMap := getPartitionAddTask()

	// 循环每一个存储池
	for _, vg := range groups.VGS {
		if vg.Name != req.Name {
			// 使用名称进行匹配
			continue
		}
		resp = InfoResp{
			Id:          vg.UUID,
			Name:        vg.Name,
			Capacity:    vg.Size,
			UseCapacity: vg.Size - vg.FreeSize,
		}

		// 初始化逻辑分区结构
		resp.Lv = make([]*LogicalVolume, len(vg.LVS))
		// 把逻辑分区的名称存出来，在后面查重
		lvMap := make(map[string]string, len(vg.LVS))

		for key, lv := range vg.LVS {
			taskId, status := getPartitionTaskInfo(req.Name, lv.Name)
			resp.Lv[key] = &LogicalVolume{
				Id: lv.UUID,
				Name: lv.Name,
				Capacity: lv.Size,
				UseCapacity: lv.Size - lv.FreeSize,
				Status: status,
				TaskId: taskId,
			}
			lvMap[lv.Name] = lv.UUID
		}

		if lvAddTaskList, ok := taskListMap[vg.Name]; ok {
			// 把任务里面的添加进去
			for _, value := range lvAddTaskList {
				// 如果已经存在，则丢弃
				if _, ok = lvMap[value.Name]; !ok {
					resp.Lv = append(resp.Lv, value)
				}
			}
		}

		// 物理分区
		resp.Pv = make([]*PhysicalVolume, len(vg.PVS))
		for key, pv := range vg.PVS {
			resp.Pv[key] = &PhysicalVolume{
				Id: pv.UUID,
				Name: pv.Name,
				Capacity: pv.Size,
			}
		}
		break
	}

	// 排序
	sort.Slice(resp.Lv, func(p, q int) bool {
		if resp.Lv[q].Name == types.LvmSystemDefaultName {
			return false
		}
		return resp.Lv[p].Name < resp.Lv[q].Name
	})
}

// getPartitionTaskStatus 获取存储池分区的异步任务状态
func getPartitionTaskInfo(poolName, partitionName string) (taskId string, status string) {
	taskManager := task.GetTaskManager()
	sign := fmt.Sprintf("%s_%s", poolName, partitionName)

	if taskInfo, ok := taskManager.GetTaskInfo(types.TaskAddPartition, sign); ok {
		// 添加分区
		taskId = fmt.Sprintf("%s_%s_%s",types.TaskAddPartition, poolName, partitionName)
		status = fmt.Sprintf("%s_%d", types.TaskAddPartition, taskInfo.Status)
	} else if taskInfo, ok = taskManager.GetTaskInfo(types.TaskUpdatePartition, sign); ok {
		// 更新分区
		taskId = fmt.Sprintf("%s_%s_%s",types.TaskUpdatePartition, poolName, partitionName)
		status = fmt.Sprintf("%s_%d", types.TaskUpdatePartition, taskInfo.Status)
	} else if taskInfo, ok = taskManager.GetTaskInfo(types.TaskDelPartition, sign); ok {
		// 删除分区
		taskId = fmt.Sprintf("%s_%s_%s",types.TaskDelPartition, poolName, partitionName)
		status = fmt.Sprintf("%s_%d", types.TaskDelPartition, taskInfo.Status)
	}

	return
}

// getPartitionAddTask 获取添加存储池分区的任务
func getPartitionAddTask() map[string][]*LogicalVolume {
	taskManager := task.GetTaskManager()
	result := make(map[string][]*LogicalVolume)

	// 添加存储池分区的key前缀
	prefix := fmt.Sprintf("%s_", types.TaskAddPartition)

	for key, value := range taskManager.Tasks {
		if !strings.HasPrefix(key, prefix) {
			continue
		}
		// 通过反射获取到结构体
		valueStruct :=  reflect.ValueOf(value.DetailInterface).Elem()
		// 获取存储池名称
		poolName := valueStruct.FieldByName("PoolName").String()
		// 获取存储池分区名称
		partitionName := valueStruct.FieldByName("Name").String()
		// 获取存储池容量
		unit := valueStruct.FieldByName("Unit").String()
		capacity := valueStruct.FieldByName("Capacity").Int()
		capacity = changeUnit(capacity, unit)
		// 状态
		taskId, status := getPartitionTaskInfo(poolName, partitionName)

		result[poolName] = append(result[poolName], &LogicalVolume{
			Id: "",
			Name: partitionName,
			Capacity: capacity,
			UseCapacity: 0,
			Status: status,
			TaskId: taskId,
		})
	}

	return result
}

// changeUnit 转换单位,把单位转为B
func changeUnit(capacity int64, unit string) int64 {
	switch unit {
	case "MB":
		return capacity * 1024 * 1024
	case "GB":
		return capacity * 1024  * 1024 * 1024
	case "T":
		return capacity * 1024 * 1024  * 1024 * 1024
	}

	return capacity
}