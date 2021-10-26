package api

import (
	"fmt"
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/disk"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/folder"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/middleware"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/partition"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/pool"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/resource"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/setting"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/share"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
)

func LoadModules(r gin.IRouter) {
	r.Use(middleware.RequireAccount)
	rwp := r.Group(fmt.Sprintf("/api/plugin/%s", types.PluginName))

	resource.RegisterResourceRouter(rwp)
	share.RegisterShareRouter(rwp)
	folder.RegisterFolderRouter(rwp)
	pool.RegisterPoolRouter(rwp)
	partition.RegisterPartitionRouter(rwp)
	disk.RegisterDiskRouter(rwp)
	setting.RegisterShareRouter(rwp)
	task.RegisterTaskRouter(rwp)
}
