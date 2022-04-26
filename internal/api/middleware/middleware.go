package middleware

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/utils"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types/status"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/proto"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/response"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/session"
	"path/filepath"
)

type userInfo struct {
	ScopeToken string `json:"scope_token"`
}

func getHeader(c *gin.Context) (userInfo, error) {
	// 获取用户ID
	var uInfo userInfo
	uInfo.ScopeToken = c.GetHeader(types.ScopeTokenKey)
	if uInfo.ScopeToken == "" {
		return uInfo, fmt.Errorf("getHeader scopeToken is nil")
	}
	return uInfo, nil
}

func getQuery(c *gin.Context) (userInfo, error) {
	// 获取用户ID
	var (
		uInfo   userInfo
		isExist bool
	)

	uInfo.ScopeToken, isExist = c.GetQuery(types.ScopeTokenKey)
	if isExist == false {
		return uInfo, fmt.Errorf("getQuery scopeToken is nil")
	}
	return uInfo, nil
}

// RequireAccount Userid+token真假验证
func RequireAccount(c *gin.Context) {
	// 获取用户ID
	info, err := getHeader(c)
	if err != nil {
		info, err = getQuery(c)
		if err != nil {
			response.HandleResponse(c, errors.New(status.AuthorizationInvalid), nil)
			c.Abort()
			return
		}
	}

	userInfo, err := proto.GetUserInfo(info.ScopeToken)
	config.Logger.Info(fmt.Sprintf("%s_%s", "GetUserInfo", userInfo))
	if err != nil || userInfo.AreaInfo.AreaType == 0 {
		response.HandleResponse(c, errors.New(status.AuthorizationInvalid), nil)
		c.Abort()
		return
	}

	user := &session.User{
		UserID:     int(userInfo.UserInfo.UserId),
		Nickname:   userInfo.UserInfo.NickName,
		IsOwner:    userInfo.UserInfo.IsOwner,
		ScopeToken: info.ScopeToken,
		AreaName:   userInfo.AreaInfo.Name,
		AreaType:   int(userInfo.AreaInfo.AreaType),
	}

	if userInfo.AreaInfo.AreaType == types.AreaCompanyType {
		for _, v := range userInfo.DepartmentInfos {
			tmpInfo := session.DepartmentBaseInfo{
				DepartmentId: v.DepartmentId,
				Name:         v.Name,
				Role:         int(v.CompanyRole),
			}
			user.DepartmentBaseInfos = append(user.DepartmentBaseInfos, tmpInfo)
		}

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
