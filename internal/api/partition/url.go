package partition

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/middleware"
)

func RegisterPartitionRouter(r gin.IRouter) {
	resourceGroup := r.Group("/partitions")
	{
		resourceGroup.POST("", middleware.RequireOwnerPermission(), AddPartition) // 添加存储池
		resourceGroup.PUT(":name", middleware.RequireOwnerPermission(), UpdatePartition) // 编辑存储池
		resourceGroup.DELETE(":name", middleware.RequireOwnerPermission(), DelPartition) // 删除存储池分区
	}
}
