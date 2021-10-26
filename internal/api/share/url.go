package share

import "github.com/gin-gonic/gin"

func RegisterShareRouter(r gin.IRouter) {
	shareGroup := r.Group("shares")

	shareGroup.GET("", GetShareList)
	shareGroup.POST("", ResourcesShare)
}
