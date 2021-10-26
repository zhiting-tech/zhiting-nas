package pool

import (
	"context"
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
	"regexp"
)

type UpdateResp struct {
}

type UpdateReq struct {
	Name    string `uri:"name"`      // 存储池名称
	NewName string `json:"new_name"` // 存储池名称
}

// UpdatePool 更新存储池名称
func UpdatePool(c *gin.Context) {
	var (
		resp UpdateResp
		req  UpdateReq
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()

	// 参数绑定
	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}
	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 参数校验
	err = req.validateRequest()
	if err != nil {
		return
	}

	conn, err := grpc.Dial(config.ExtServerSetting.LvmServer, grpc.WithInsecure())
	if err != nil {
		return
	}

	defer conn.Close()
	client := proto.NewDiskManagerClient(conn)
	ctx := context.Background()

	updateReq := proto.VolumeGroupRenameReq{
		OldName: req.Name,
		NewName: req.NewName,
	}

	// 重命名存储池
	result, err := client.VolumeGroupRename(ctx, &updateReq)
	if err != nil {
		err = errors.HandleLvmError(err)
		config.Logger.Errorf("VolumeGroupRename error: %v", err)
		return
	}
	config.Logger.Infof("VolumeGroupRename info: %v", result.Data.Name)

	oldPath := fmt.Sprintf("/%s", req.Name)
	newPath := fmt.Sprintf("/%s", req.NewName)
	if err = utils.UpdateFolderPath(entity.GetDB(), oldPath, newPath); err != nil {
		config.Logger.Infof("VolumeGroupRename update path : %v", err)
		return
	}

	if req.Name == config.AppSetting.PoolName {
		entity.UpdatePoolNameSetting(req.NewName)
		config.AppSetting.PoolName = req.NewName
	}

	return
}

func (req *UpdateReq) validateRequest() (err error) {
	// 判断大小0-50
	if req.NewName == "" {
		err = errors.Wrap(err, status.PoolParamIsNullErr)
		return
	}
	if len(req.NewName) > 50 {
		err = errors.Wrap(err, status.PoolNameTooLongErr)
		return
	}
	// 匹配字符串是否存在除0-9a-zA-Z+_.-之外的字符
	reg := regexp.MustCompile("[^0-9a-zA-Z+_.-]")
	if reg.MatchString(req.NewName) {
		err = errors.Wrap(err, status.PoolNameParamErr)
		return
	}

	return
}
