package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
)

type GetBackupsIdentificationResp struct {
	Identifications []string `json:"identifications"`
	PrivateFileId   int      `json:"private_file_id"`
}

func GetBackupsIdentification(c *gin.Context) {
	var (
		resp GetBackupsIdentificationResp
		err  error
	)

	defer func() {
		response.HandleResponse(c, err, &resp)
	}()
	// 获取登陆用户
	user := session.Get(c)
	//
	resp.Identifications = entity.GetFolderIdentification(user.UserID)
	folderOne, _ := entity.GetRelateFolderInfoByUid(types.FolderSelfDirUid, user.UserID)
	resp.PrivateFileId = folderOne.Id
}
