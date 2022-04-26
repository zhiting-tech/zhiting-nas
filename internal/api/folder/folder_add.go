package folder

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"gorm.io/gorm"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

type AddResp struct {
}

type AddAuthReq struct {
	Uid      int    `json:"u_id"`
	Nickname string `json:"nickname"`
	Face     string `json:"face"`
	Read     int    `json:"read"`
	Write    int    `json:"write"`
	Deleted  int    `json:"deleted"`
}

type AddReq struct {
	Name          string       `json:"name"`          // 文件/文件夹名称
	Mode          int          `json:"mode"`          // 文件夹类型：1私人文件夹 2共享文件夹
	PoolName      string       `json:"pool_name"`     // 储存池ID
	PartitionName string       `json:"partition_name"`// 储存池分区ID
	IsEncrypt     int          `json:"is_encrypt"`    // 是否加密
	Pwd           string       `json:"pwd"`           // 密码
	ConfirmPwd    string       `json:"confirm_pwd"`   // 确认密码
	Auth          []AddAuthReq `json:"auth"`          // 可访问成员的权限
	Cipher        string
}

func AddFolder(c *gin.Context) {
	var (
		req  AddReq
		resp AddResp
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
	// 获取登陆用户
	user := session.Get(c)

	// 权限数组
	var auths = make([]entity.FolderAuth, len(req.Auth))
	var persons = make([]string, len(req.Auth))
	for key, auth := range req.Auth {
		auths[key] = entity.FolderAuth{
			Uid:      auth.Uid,
			Nickname: auth.Nickname,
			Face:     auth.Face,
			Read:     auth.Read,
			Write:    auth.Write,
			Deleted:  auth.Deleted,
		}
		persons[key] = auth.Nickname
	}
	if err = entity.GetDB().Transaction(func(tx *gorm.DB) error {
		// 创建文件夹
		err = filebrowser.GetFB().Mkdir(fmt.Sprintf("/%s/%s/%s", req.PoolName, req.PartitionName, req.Name))
		if err != nil {
			err = errors.Wrap(err, status.FolderAddFailErr)
			return err
		}
		folderInfo, err := entity.CreateFolder(tx, &entity.FolderInfo{
			Name:          req.Name,                                                           // 名称
			Uid:           user.UserID,                                                        // 创建人
			Mode:          req.Mode,                                                           // 文件夹类型
			PoolName:      req.PoolName,                                                       // 池名称
			PartitionName: req.PartitionName,                                                  // 分区名称
			IsEncrypt:     req.IsEncrypt,                                                      // 是否加密
			Cipher:        req.Cipher,                                                         // 加密的密钥
			Type:          types.FolderTypeDir,                                               // 文件夹
			CreatedAt:     time.Now().Unix(),                                                 // 创建时间
			Persons:       strings.Join(persons, "、"),                                   // 可访问成员
			AbsPath:       fmt.Sprintf("/%s/%s/%s", req.PoolName, req.PartitionName, req.Name),  // 目录存放到存储池/分区下
		})
		if err != nil {
			err = errors.Wrap(err, status.FolderAddFailErr)
			return err
		}

		// 判断添加文件夹是否为共享文件夹，Mode为2时，是共享文件夹，isShare=1
		isShare := 0
		if req.Mode == types.FolderShareDir {
			isShare = 1
		}
		for key := range req.Auth {
			auths[key].FolderId = folderInfo.ID
			auths[key].IsShare = isShare
		}
		if err = entity.BatchInsertAuth(tx, auths); err != nil {
			err = errors.Wrap(err, status.FolderAddFailErr)
			return err
		}

		return nil
	}); err != nil {
		return
	}
}

// validateRequest 认证请求方式
func (req *AddReq) validateRequest() (err error) {
	// 校验必填
	if req.Name == "" || req.PoolName == "" || req.PartitionName == "" || req.Mode == 0 {
		err = errors.Wrap(err, status.FolderParamFailErr)
		return
	}
	// 校验名称字符
	compile := regexp.MustCompile(`[/:*?"<>|\\]+`)
	if isLimit := compile.MatchString(req.Name); isLimit != false {
		err = errors.Wrap(err, status.FolderNameParamErr)
		return
	}

	// 校验名称长度
	if utf8.RuneCountInString(req.Name) > 100 {
		err = errors.Wrap(err, status.FolderNameTooLongErr)
		return
	}

	// 校验名称是否重复
	folderInfo, err := entity.GetFolderByName(req.PoolName, req.PartitionName, req.Name)
	if err != nil {
		err = errors.Wrap(err, errors.InternalServerErr)
		return
	}
	if folderInfo.ID != 0 {
		// 存在同名文件夹
		err = errors.Wrap(err, status.FolderNameIsExistErr)
		return
	}
	// 校验文件夹类型
	if req.Mode == 1 {
		// 私人文件夹
		if len(req.Auth) != 1 {
			err = errors.Wrap(err, status.FolderTooMuchMemberErr)
			return
		}
	} else if req.Mode == 2 {
		// 共享文件夹,成员大于等于1
		if len(req.Auth) < 1 {
			err = errors.Wrap(err, status.FolderTooFewMemberErr)
			return
		}
		// 共享文件不能加密
		if req.IsEncrypt == 1 {
			err = errors.Wrap(err, status.FolderCannotEncryptErr)
			return
		}
	}

	// 校验密码
	if req.IsEncrypt == 1 {
		if req.Pwd == "" || req.ConfirmPwd == "" {
			err = errors.Wrap(err, status.FolderParamFailErr)
			return
		}

		// 校验密码长度，不少于6位
		if len(req.Pwd) < 6 {
			err = errors.Wrap(err, status.FolderPwdLessErr)
			return
		}

		// 正则匹配密码 匹配字符串是否存在除0-9a-zA-Z@#&!之外的字符
		reg := regexp.MustCompile("[^0-9a-zA-Z@#&!]")
		if reg.MatchString(req.Pwd) {
			err = errors.Wrap(err, status.FolderPwdStringErr)
			return
		}
		// 两次输入的密码一致
		if req.Pwd != req.ConfirmPwd {
			err = errors.Wrap(err, status.FolderConFirmPwdFailErr)
			return
		}

		// 密钥生成
		secret := utils.EncodeMD5(strconv.FormatInt(time.Now().Unix(), 10))
		req.Cipher, err = utils.EncryptString(req.Pwd, secret)
		if err != nil {
			err = errors.Wrap(err, status.FolderSecretFailErr)
			return
		}
	}

	return
}