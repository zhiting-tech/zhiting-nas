package pool

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
)

type DelResp struct {
}

type DelReq struct {
	Name string `uri:"name"` //存储池name
}

func DelPool(c *gin.Context) {
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

	// 添加异步任务
	task.GetTaskManager().Add(types.TaskDelPool, req.Name, &req)

	return
}


// ExecTask 执行异步任务
func (req *DelReq) ExecTask() error {
	// 创建通道
	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return err
	}

	defer conn.Close()

	// 创建连接
	client := proto.NewDiskManagerClient(conn)

	param := proto.VolumeGroupRemoveReq{
		VGName: req.Name,
	}
	ctx := context.Background()
	// 调用服务端方法
	result, err := client.VolumeGroupRemove(ctx, &param)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Error("VolumeGroupRemove error: %v", err)
		return err
	}

	config.Logger.Infof("VolumeGroupRemove info: %v", result)

	// 把对应文件夹信息删除
	absPath := fmt.Sprintf("/%s", req.Name)
	_ = entity.DelFolder(entity.GetDB(), absPath)

	// 如果删除的是配置的存储池，那么需要重置
	if req.Name == config.AppSetting.PoolName {
		entity.InitSettingPool()
		config.AppSetting.PoolName = types.LvmSystemDefaultName
		config.AppSetting.PartitionName = types.LvmSystemDefaultName
	}

	return nil
}
