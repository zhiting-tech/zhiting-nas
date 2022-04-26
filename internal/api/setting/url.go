package setting

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/middleware"
)

func RegisterShareRouter(r gin.IRouter) {
	shareGroup := r.Group("settings")
	shareGroup.Use(middleware.RequireAccount)

	shareGroup.GET("", GetSettingList)
	shareGroup.POST("", UpdateSetting)
}