package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"path/filepath"
	"strconv"
)

// RequireAccount Userid+token真假验证
func RequireAccount(c *gin.Context) {
	// 获取用户ID
	strId := c.GetHeader(types.ScopeUserIdKey)
	if strId == "" {
		response.HandleResponse(c, errors.New(status.AuthorizationInvalid), nil)
		c.Abort()
		return
	}
	uid, err := strconv.Atoi(strId)
	if err != nil || uid <= 0 {
		response.HandleResponse(c, errors.New(status.AuthorizationInvalid), nil)
		c.Abort()
		return
	}

	// 通过过来的token，验证token是否正确
	// TODO 使用http的方式，可能会变得缓慢，后面改为其他方式验证
	apiUrl := fmt.Sprint( "/api/users/", uid)
	userInfo, err := utils.GetRequestSaServer(apiUrl, c)
	if err != nil {
		config.Logger.Errorf("GetRequestSaServer err %v", err)
		response.HandleResponse(c, errors.New(status.AuthorizationInvalid), nil)
		c.Abort()
		return
	}
	// 判断响应的状态码 为0则http请求成功
	if userInfo.Status != 0 {
		config.Logger.Errorf("GetRequestSaServer err status 0 %v", userInfo)
		response.HandleResponse(c, errors.New(status.AuthorizationInvalid), nil)
		c.Abort()
		return
	}

	user := &session.User{
		UserID: userInfo.UserId,
		Nickname: userInfo.Nickname,
		IsOwner: userInfo.IsOwner,
		ScopeToken: c.GetHeader(types.ScopeTokenKey),
		AreaName: userInfo.SaArea.Name,

	}
	c.Set("session_user", user)

	c.Next()
	return
}

// RequireOwnerPermission 判断是否有拥有者权限
func RequireOwnerPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := session.Get(c)
		if u == nil {
			response.HandleResponse(c, errors.New(status.PoolIsNotPermission), nil)
			c.Abort()
			return
		}
		if !u.IsOwner {
			response.HandleResponse(c, errors.New(status.PoolIsNotPermission), nil)
			c.Abort()
			return
		}
		c.Next()
		return
	}
}

// RequirePathPermission 判断是否有权限
func RequirePathPermission() gin.HandlerFunc {
	return func(c *gin.Context) {
		u := session.Get(c)
		if u == nil {
			response.HandleResponse(c, errors.New(errors.BadRequest), nil)
			c.Abort()
			return
		}
		path := filepath.Join(c.Param("path"))
		if !(path == "" || path == "/") {
			auth, err := utils.GetFilePathAuth(u.UserID, path)
			if err != nil || auth == nil {
				response.HandleResponse(c, errors.New(status.ResourceNotAuthErr), nil)
				c.Abort()
				return
			}
			// 没有只读权限
			if auth.Read == 0 {
				response.HandleResponse(c, errors.New(status.ResourceNotReadAuthErr), nil)
				c.Abort()
				return
			}
			// 把目录信息放入到上下文中
			c.Set("folder_id", auth.Id)
			c.Set("write", auth.Write)
			c.Set("deleted", auth.Deleted)
			c.Set("read", auth.Read)
			c.Set("is_encrypt", auth.IsEncrypt)
		}

		c.Next()
		return
	}
}
