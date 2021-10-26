package setting

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"strconv"
)

// ListResp 出参中list结构体
type ListResp struct {
	PoolName    	string  `json:"pool_name"` // 系统存储池
	PartitionName   string  `json:"partition_name"` // 系统存储池分区
	IsAutoDel      	int  	`json:"is_auto_del"` // 成员退出是否自动删除
}

func GetSettingList(c *gin.Context) {
	var (
		resp ListResp
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()

	list, err := entity.GetSettingList()
	if err != nil {
		return
	}

	for _, val := range list {
		switch val.Name {
		case "PoolName":
			resp.PoolName = val.Value
		case "PartitionName":
			resp.PartitionName = val.Value
		case "IsAutoDel":
			resp.IsAutoDel, _ = strconv.Atoi(val.Value)
		}
	}
}
