package partition

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
	"regexp"
)

type AddResp struct {
}

type AddReq struct {
	Name     string `json:"name"`
	Capacity int64  `json:"capacity"`
	Unit     string `json:"unit"`
	PoolName string `json:"pool_name"`
}

func AddPartition(c *gin.Context) {
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

	sign := fmt.Sprintf("%s_%s", req.PoolName, req.Name)
	task.GetTaskManager().Add(types.TaskAddPartition, sign, &req)

	return
}

func (req *AddReq) validateRequest() (err error) {
	if req.Name == "" || req.Capacity == 0 || req.Unit == "" || req.PoolName == "" {
		err = errors.Wrap(err, status.PartitionParamFailErr)
		return
	}
	if len(req.Name) > 50 {
		err = errors.Wrap(err, status.PartitionNameTooLongErr)
		return
	}
	// 匹配字符串是否存在除0-9a-zA-Z+_.-之外的字符
	reg := regexp.MustCompile("[^0-9a-zA-Z+_.-]")
	if reg.MatchString(req.Name) {
		err = errors.Wrap(err, status.PartitionNameParamErr)
		return
	}

	info, err := utils.GetPartitionInfo(req.PoolName, req.Name)
	if info != nil && err == nil {
		err = errors.Wrap(err, status.PartitionNameRepeatErr)
		return
	}

	// 判断申请容量是否超出可用容量
	err = req.checkPoolFreeSize()
	if err != nil {
		return
	}

	return
}

// changeUnit 转换单位,把单位转为MB
func (req *AddReq) changeUnit() int64 {
	switch req.Unit {
	case "GB":
		return req.Capacity * 1024
	case "TB":
		return req.Capacity * 1024 * 1024
	}

	return req.Capacity
}

func (req *AddReq) ExecTask() error {
	// 改变单位
	req.Capacity = req.changeUnit()

	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	param := proto.LogicalVolumeCreateReq{
		VGName: req.PoolName,
		LVName: req.Name,
		SizeM:  req.Capacity,
	}

	// 创建逻辑分区
	result, err := client.LogicalVolumeCreate(ctx, &param)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("LogicalVolumeCreate error: %v", err)
		return err
	}
	config.Logger.Infof("LogicalVolumeCreate info: %v", result.Data.Name)
	return nil
}

// CheckPoolFreeSize  获取存储池详情
func (req *AddReq) checkPoolFreeSize() (err error) {
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
	apply := req.changeUnit()
	apply = apply * 1024 * 1024
	var c float64
	for _, vg := range groups.VGS {
		var unit string
		if vg.Name == req.PoolName && apply > vg.FreeSize {
			if vg.FreeSize / 1024 / 1024 / 1024 / 1024 > 1 {
				c = float64(vg.FreeSize) / 1024 / 1024 / 1024 / 1024
				unit = "TB"
			} else if vg.FreeSize / 1024 / 1024 / 1024 > 1 {
				c = float64(vg.FreeSize) / 1024 / 1024 / 1024
				unit = "GB"
			} else if vg.FreeSize / 1024 / 1024 > 1 {
				c = float64(vg.FreeSize) / 1024 / 1024
				unit = "MB"
			} else {
				c = float64(vg.FreeSize) / 1024
				unit = "KB"
			}
			err = errors.Wrapf(err, status.PartitionCapacityExceededErr, c, unit)
			return
		}
	}
	return
}
