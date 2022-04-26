package task

import (
	"github.com/gin-gonic/gin"
)

func RegisterTaskRouter(r gin.IRouter) {
	group := r.Group("tasks")
	{
		group.PUT(":id", RestartTask) // 编辑存储池
		group.DELETE(":id", DelTask) // 删除存储池
	}
}