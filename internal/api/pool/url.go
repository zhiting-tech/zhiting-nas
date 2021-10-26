package pool

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/middleware"
)

func RegisterPoolRouter(r gin.IRouter) {
	resourceGroup := r.Group("/pools")
	{
		resourceGroup.GET("", GetPoolList)      // 获取存储池列表
		resourceGroup.GET(":name", GetPoolInfo) // 获取存储池详情
		resourceGroup.POST("", middleware.RequireOwnerPermission(), AddPool)         // 添加存储池
		resourceGroup.PUT(":name", middleware.RequireOwnerPermission(), UpdatePool)  // 编辑存储池
		resourceGroup.DELETE(":name", middleware.RequireOwnerPermission(), DelPool)  // 删除存储池
	}
}
