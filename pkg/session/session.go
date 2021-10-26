package session

import (
	"github.com/gin-gonic/gin"
)

// Get 根据SA 反向代理的HTTP头信息获取用户数据
func Get(c *gin.Context) *User {
	// 从上下文中获取session_user
	user, exists := c.Get("session_user")
	if !exists {
		return nil
	}

	return user.(*User)
}
