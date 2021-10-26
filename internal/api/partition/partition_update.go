package partition

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
	"regexp"
)

type UpdateResp struct {
}

type UpdateReq struct {
	Name     string `uri:"name"`       // 储存池分区名称
	PoolName string `json:"pool_name"` // 存储池名称
	Capacity int64  `json:"capacity"`  // 容量
	Unit     string `json:"unit"`      // 单位：MB
	NewName  string `json:"new_name"`  // 名称
}

func UpdatePartition(c *gin.Context) {
	var (
		resp UpdateResp
		req  UpdateReq
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
	if err = req.validateRequest(); err != nil {
		return
	}

	sign := fmt.Sprintf("%s_%s", req.PoolName, req.Name)
	// 当只修改文件夹名时，启用同步任务执行，当容量做修改时，无论文件夹名改不改，都启用异步任务执行
	info, _ := utils.GetPartitionInfo(req.PoolName,req.Name)
	if err = req.rename(); err != nil {
		config.Logger.Errorf("Partition Rename Failed,Error Is %v",err)
	}
	if req.changeUnit() * 1024 * 1024 != info.Size {
		task.GetTaskManager().Add(types.TaskUpdatePartition, sign, &req)
	}

	return
}

// validateRequest 校验参数
func (req *UpdateReq) validateRequest() (err error) {
	if req.Name == "" || req.PoolName == "" || req.Capacity == 0 || req.Unit == "" || req.NewName == "" {
		err = errors.Wrap(err, status.PartitionParamFailErr)
		return
	}
	if len(req.NewName) > 50 {
		err = errors.Wrap(err, status.PartitionNameTooLongErr)
		return
	}
	// 匹配字符串是否存在除0-9a-zA-Z+_.-之外的字符
	reg := regexp.MustCompile("[^0-9a-zA-Z+_.-]")
	if reg.MatchString(req.NewName) {
		err = errors.Wrap(err, status.PartitionNameParamErr)
		return
	}

	info, err := utils.GetPartitionInfo(req.PoolName, req.Name)
	if err != nil {
		return
	}
	// 不能缩小空间, 转换成B进行对比
	capacity := req.changeUnit() * 1024 * 1024
	if info.Size > capacity  {
		err = errors.Wrap(err, status.PartitionCapacityShrinkingErr)
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
func (req *UpdateReq) changeUnit() int64 {
	switch req.Unit {
	case "GB":
		return req.Capacity * 1024
	case "TB":
		return req.Capacity * 1024 * 1024
	}

	return req.Capacity
}

// ExecTask 执行异步任务
func (req *UpdateReq) ExecTask() error {
	if err := req.extend(); err != nil {
		return err
	}
	return nil
}

// rename 重命名
func (req *UpdateReq) rename() error {
	if req.NewName == req.Name {
		return nil
	}
	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()

	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	param := proto.LogicalVolumeRenameReq{
		VGName:    req.PoolName,
		LVName:    req.Name,
		NewLVName: req.NewName,
	}

	result, err := client.LogicalVolumeRename(ctx, &param)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("LogicalVolumeRename error: %v", err)
		return err
	}
	config.Logger.Infof("LogicalVolumeRename info: %v", result.Data.Name)

	oldPath := fmt.Sprintf("/%s/%s", req.PoolName, req.Name)
	newPath := fmt.Sprintf("/%s/%s", req.PoolName, req.NewName)
	if err = utils.UpdateFolderPath(entity.GetDB(), oldPath, newPath); err != nil {
		config.Logger.Infof("LogicalVolumeRename update path : %v", err)
		return err
	}

	// 如果修改的是配置项的存储池名称
	if config.AppSetting.PoolName == req.PoolName && config.AppSetting.PartitionName == req.Name {
		entity.UpdatePartitionNameSetting(req.NewName)
		config.AppSetting.PartitionName = req.NewName
	}

	return nil
}

// extend 扩容
func (req *UpdateReq) extend() error {
	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return err
	}
	defer conn.Close()
	// Unit 转换
	req.Capacity = req.changeUnit()

	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	param := proto.LogicalVolumeExtendReq{
		VGName:   req.PoolName,
		LVName:   req.NewName,
		NewSizeM: req.Capacity,
	}

	result, err := client.LogicalVolumeExtend(ctx, &param)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("LogicalVolumeExtend error: %v", err)
		return err
	}
	config.Logger.Infof("LogicalVolumeExtend info: %v", result.Data.Name)

	return nil
}

// CheckPoolFreeSize  获取存储池详情
func (req *UpdateReq) checkPoolFreeSize() (err error) {
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