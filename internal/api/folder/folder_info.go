package folder

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
)

type AddAuthResp struct {
	Uid      int    `json:"u_id"`
	Nickname string `json:"nickname"`
	Face     string `json:"face"`
	Read     int    `json:"read"`
	Write    int    `json:"write"`
	Deleted  int    `json:"deleted"`
}

type InfoResp struct {
	ID            int           `json:"id"`
	Name          string        `json:"name"`
	IsEncrypt     int           `json:"is_encrypt"`
	Mode          int           `json:"mode"`
	Path          string        `json:"path"`
	Type          int           `json:"type"`
	PoolName      string        `json:"pool_name"`      // 储存池ID
	PartitionName string        `json:"partition_name"` // 储存池分区ID
	Auth        []AddAuthResp   `json:"auth"`         // 权限
}

type InfoReq struct {
	Id int `uri:"id"`
}

func GetFolderInfo(c *gin.Context) {
	var (
		req  InfoReq
		resp InfoResp
		err  error
	)
	defer func() {
		response.HandleResponse(c, err, &resp)
	}()
	if err = c.BindUri(&req); err != nil {
		err = errors.Wrap(err, errors.BadRequest)
		return
	}

	info, err := entity.GetFolderInfo(req.Id)
	if err != nil || info == nil {
		err = errors.Wrap(err, status.FolderInfoFailErr)
		return
	}
	folderAuthList, err := entity.GetFolderAuthByFolderId(req.Id)
	if err != nil {
		err = errors.Wrap(err, status.FolderInfoFailErr)
		return
	}

	resp.ID = info.ID
	resp.Name = info.Name
	resp.IsEncrypt = info.IsEncrypt
	resp.Mode = info.Mode
	resp.Type = info.Type
	resp.PoolName = info.PoolName
	resp.PartitionName = info.PartitionName

	for _, auth := range folderAuthList {
		resp.Auth = append(resp.Auth, AddAuthResp{
			Uid:      auth.Uid,
			Nickname: auth.Nickname,
			Face:     auth.Face,
			Read:     auth.Read,
			Write:    auth.Write,
			Deleted:  auth.Deleted,
		})
	}
}