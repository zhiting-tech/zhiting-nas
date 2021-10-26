package disk

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
)

type AddResp struct {
}

type AddReq struct {
	PoolName string `json:"pool_name"` // 存储池名称
	DiskName string `json:"disk_name"` // 闲置硬盘名称（闲置的物理分区）
}

// AddDisk 添加物理分区到存储池
func AddDisk(c *gin.Context) {
	var (
		resp AddResp
		req  AddReq
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	if err = req.validateRequest(); err != nil {
		return
	}

	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return
	}
	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	createReq := proto.VolumeGroupCreateOrExtendReq{
		VGName: req.PoolName,
		PVName: req.DiskName,
	}

	// 添加物理分区到存储池
	result, err := client.VolumeGroupExtend(ctx, &createReq)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("VolumeGroupExtend error: %v", err)
		return
	}
	config.Logger.Infof("VolumeGroupExtend info: %v", result.Data.Name)

	return
}

func (req *AddReq) validateRequest() (err error) {
	if req.PoolName == "" || req.DiskName == "" || req.PoolName == types.LvmSystemDefaultName {
		err = errors.Wrap(err, status.DiskParamFailErr)
		return
	}
	return
}
