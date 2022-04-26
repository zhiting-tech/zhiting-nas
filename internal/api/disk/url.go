package disk

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/middleware"
)

func RegisterDiskRouter(r gin.IRouter) {
	resourceGroup := r.Group("/disks")
	{
		// 物理分区接口
		resourceGroup.GET("", GetDiskList) // 获取硬盘列表列表
		resourceGroup.POST("", middleware.RequireOwnerPermission(), AddDisk) // 获取物理分区到存储池
	}
}
