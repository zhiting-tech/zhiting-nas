package folder

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"regexp"
)

type ChangePwdReq struct {
	Id         int    `json:"id"`
	OldPwd     string `json:"old_pwd"`
	NewPwd     string `json:"new_pwd"`
	ConfirmPwd string `json:"confirm_pwd"`
}

type ChangePwdResp struct {
}

func ChangePwd(c *gin.Context) {
	var (
		req  ChangePwdReq
		resp ChangePwdResp
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

	// 修改cipher
	// 获取密钥重新加密
	folderInfo, err := entity.GetFolderInfo(req.Id)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	secret, err := utils.DecryptString(req.OldPwd, folderInfo.Cipher)
	if err != nil {
		err = errors.Wrap(err, status.FolderKeyResolutionErr)
		return
	}
	cipher, err := utils.EncryptString(req.NewPwd, secret)
	if err != nil {
		err = errors.Wrap(err, status.FolderSecretFailErr)
		return
	}

	if err = entity.UpdateFolderInfo(entity.GetDB(), req.Id, entity.FolderInfo{Cipher: cipher}); err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
}

func (req *ChangePwdReq) validateRequest() (err error) {
	// 密码是否符合格式
	if req.NewPwd == "" || req.OldPwd == "" || req.ConfirmPwd == "" {
		err = errors.Wrap(err, status.FolderPwdNotInputErr)
		return
	}

	if len(req.NewPwd) < 6 {
		err = errors.Wrap(err, status.FolderPwdParamFailErr)
		return
	}

	// 旧密码和原密码是否一致
	folderInfo, err := entity.GetFolderInfo(req.Id)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	_, err = utils.DecryptString(req.OldPwd, folderInfo.Cipher)
	if err != nil {
		err = errors.Wrap(err, status.FolderOldPwdErr)
		return
	}

	// 正则表达式，匹配字符串是否存在除0-9a-zA-Z@#&!之外的字符
	reg := regexp.MustCompile("[^0-9a-zA-Z@#&!]")
	if reg.MatchString(req.NewPwd) {
		err = errors.Wrap(err, status.FolderPwdParamFailErr)
		return
	}

	// 新密码与确认密码是否一致
	if req.NewPwd != req.ConfirmPwd {
		err = errors.Wrap(err, status.FolderConFirmPwdFailErr)
		return
	}
	return
}
