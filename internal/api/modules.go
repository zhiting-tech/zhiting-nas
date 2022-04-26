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
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"net/http"
)

func LoadModules(r gin.IRouter) {
	rwp := r.Group(fmt.Sprintf("/%s/api/", types.PluginName))
	file := fmt.Sprintf("%s/%s", config.AppSetting.UploadSavePath, "file")
	rwp.StaticFS("/file", http.Dir(file))
	rwp.Use(middleware.RequireAccount)
	resource.RegisterResourceRouter(rwp)
	share.RegisterShareRouter(rwp)
	folder.RegisterFolderRouter(rwp)
	pool.RegisterPoolRouter(rwp)
	partition.RegisterPartitionRouter(rwp)
	disk.RegisterDiskRouter(rwp)
	setting.RegisterShareRouter(rwp)
	task.RegisterTaskRouter(rwp)
}
