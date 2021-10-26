package disk

import (
	"context"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/disk-manager.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"google.golang.org/grpc"
)

// ListResp 出参中list结构体
type ListResp struct {
	Id          string  `json:"id"`
	Name        string  `json:"name"`
	VGName      string  `json:"vg_name"`
	Capacity    int64   `json:"capacity"`
}

type ListReq struct {

}

func GetDiskList(c *gin.Context)  {
	var (
		list     []*ListResp
		req      ListReq
		err      error
		totalRow int64
	)

	defer func() {
		if len(list) == 0 {
			list = make([]*ListResp, 0)
		}
		response.HandleResponseList(c, err, &list, totalRow)
	}()

	if err = c.BindQuery(&req); err != nil {
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

	VList, err := client.PhysicalVolumeList(ctx, &proto.Empty{})
	if err != nil {
		err = errors.HandleLvmError(err)
		return
	}

	for _, pv := range VList.PVS {
		if pv.VGName == "" {
			info := &ListResp{
				Id:          pv.UUID,
				Name:        pv.Name,
				VGName:      pv.VGName,
				Capacity:    pv.Size,
			}
			list = append(list, info)
		}
	}

	totalRow = int64(len(list))
}