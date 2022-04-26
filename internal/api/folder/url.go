package folder

import (
	"github.com/gin-gonic/gin"
)

func RegisterFolderRouter(r gin.IRouter) {
	resourceGroup := r.Group("/folders")
	{
		resourceGroup.GET("", GetFolderList)        // 获取文件夹列表
		resourceGroup.POST("", AddFolder)           // 添加文件夹
		resourceGroup.GET(":id", GetFolderInfo)     // 获取文件夹详情
		resourceGroup.PUT(":id", UpdateFolder)      // 编辑文件夹
		resourceGroup.DELETE(":id", DelFolder)      // 删除文件夹
		resourceGroup.PATCH("*path", DecryptFolder) // 解密文件夹
		resourceGroup.DELETE("", RemoveArea)        // 移除家庭
		//resourceGroup.DELETE("", DeleteFolderByIds) // 移除用户文件夹
	}

	r.POST("updateFolderPwd", ChangePwd) //修改文件夹密码
}
