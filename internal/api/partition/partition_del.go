package partition

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
)

type DelResp struct {
}

type DelReq struct {
	Name     string `uri:"name"`
	PoolName string `json:"pool_name"`
}

func DelPartition(c *gin.Context) {
	var (
		resp DelResp
		req  DelReq
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

	sign := fmt.Sprintf("%s_%s", req.PoolName, req.Name)
	task.GetTaskManager().Add(types.TaskDelPartition, sign, &req)

	return
}

func (req *DelReq) validateRequest() (err error) {
	if req.Name == "" || req.PoolName == "" {
		err = errors.Wrap(err, status.PartitionParamFailErr)
	}
	return
}

// ExecTask 执行异步任务
func (req *DelReq) ExecTask() error {
	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	delReq := proto.LogicalVolumeRemoveReq{
		VGName: req.PoolName, // 存储池名称
		LVName: req.Name,     // 逻辑分区名称
	}

	// 删除逻辑分区
	result, err := client.LogicalVolumeRemove(ctx, &delReq)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("VolumeGroupExtend error: %v", err)
		return err
	}
	config.Logger.Infof("VolumeGroupExtend info: %v", result)

	// 把对应文件夹信息删除
	absPath := fmt.Sprintf("/%s/%s", req.PoolName, req.Name)
	_ = entity.DelFolder(entity.GetDB(), absPath)

	// 如果删除的是配置的存储池，那么需要重置
	if req.PoolName == config.AppSetting.PoolName {
		entity.InitSettingPool()
		config.AppSetting.PoolName = types.LvmSystemDefaultName
		config.AppSetting.PartitionName = types.LvmSystemDefaultName
	}

	return nil
}