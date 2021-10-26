package pool

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
	"regexp"
)

type AddResp struct {
}

type AddReq struct {
	Name     string `json:"name"`      // 存储池名称
	DiskName string `json:"disk_name"` // 闲置硬盘名称（闲置的物理分区）
}

// AddPool 添加
func AddPool(c *gin.Context) {
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
		VGName: req.Name,
		PVName: req.DiskName,
	}
	// 选择物理分区，创建存储池
	result, err := client.VolumeGroupCreate(ctx, &createReq)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("VolumeGroupCreate error: %v", err)
		return
	}
	config.Logger.Infof("VolumeGroupCreate info: %v", result.Data.Name)

	return
}

func (req *AddReq) validateRequest() (err error) {
	// 判断大小0-50
	if req.Name == "" || req.DiskName == "" {
		err = errors.Wrap(err, status.PoolParamIsNullErr)
		return
	}
	if len(req.Name) > 50 {
		err = errors.Wrap(err, status.PoolNameTooLongErr)
		return
	}
	// 匹配字符串是否存在除0-9a-zA-Z+_.-之外的字符
	reg := regexp.MustCompile("[^0-9a-zA-Z+_.-]")
	if reg.MatchString(req.Name) {
		err = errors.Wrap(err, status.PoolNameParamErr)
		return
	}

	return
}
