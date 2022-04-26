package resource

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/middleware"
)

func RegisterResourceRouter(r gin.IRouter) {
	resourceGroup := r.Group("/resources")
	{
		resourceGroup.GET("*path", middleware.RequirePathPermission(), GetResourceInfo) // 目录下的文件/子目录列表
		resourceGroup.POST("*path", middleware.RequirePathPermission(), FileUpload)     // 上传文件/创建目录
		resourceGroup.PUT("*path", middleware.RequirePathPermission(), RenameResource)  // 重命名
		resourceGroup.PATCH("", OperateResource)                                        // 复制/移动文件/目录
		resourceGroup.DELETE("", DeleteResource)                                        // 删除文件/目录
	}

	r.GET("download/*path", middleware.RequirePathPermission(), FileDownload) // 下载文件
	r.GET("chunks/:hash", GetChunks)
	r.GET("backups", GetBackupsIdentification) // 获取用户备份文件标识
	r.DELETE("cache/*path", DeleteCache)
	// r.GET("preview/:id", FilePreview) libreoffice所用空间太大 该功能停用
}
