package setting

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gorm.io/gorm"
	"strconv"
)

type UpdateResp struct {
}

type UpdateReq struct {
	PoolName    	string  `json:"pool_name"` // 系统存储池
	PartitionName   string  `json:"partition_name"` // 系统存储池分区
	IsAutoDel      	int  	`json:"is_auto_del"` // 成员退出是否自动删除
}


func UpdateSetting(c *gin.Context) {
	var (
		resp UpdateResp
		req  UpdateReq
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()
	if err = c.BindJSON(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	// 更新文件夹
	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		if err := entity.DropSetting(tx); err != nil {
			return errors.Wrap(err, status.SettingUpdateFailErr)
		}
		// 默认3个配置
		settings := make([]entity.Setting, 0, 3)
		settings = append(settings, entity.Setting{Name: "PoolName", Value: req.PoolName})
		settings = append(settings, entity.Setting{Name: "PartitionName", Value: req.PartitionName})
		settings = append(settings, entity.Setting{Name: "IsAutoDel", Value: strconv.Itoa(req.IsAutoDel)})
		if err := entity.BatchInsertSetting(tx, settings); err != nil {
			return errors.Wrap(err, status.SettingUpdateFailErr)
		}
		// 更新全局配置
		config.AppSetting.PoolName = req.PoolName
		config.AppSetting.PartitionName = req.PartitionName
		config.AppSetting.IsAutoDel = req.IsAutoDel

		return nil
	}); err != nil {
		return
	}
}

func (req *UpdateReq) validateRequest() (err error) {
	if req.PoolName == "" || req.PartitionName == ""  {
		err = errors.Wrap(err, status.SettingParamFailErr)
		return
	}
	return
}