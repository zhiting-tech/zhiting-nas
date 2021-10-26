package setting

import "github.com/gin-gonic/gin"

func RegisterShareRouter(r gin.IRouter) {
	shareGroup := r.Group("settings")

	shareGroup.GET("", GetSettingList)
	shareGroup.POST("", UpdateSetting)
}