package main

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/api/resource"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/entity"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/task"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/filebrowser"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/logger"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/proto"
	"gopkg.in/natefinch/lumberjack.v2"
	"log"
	"math/rand"
	"net/http"
	"strconv"
	"time"
)

func init() {
	rand.Seed(time.Now().UnixNano())

	err := setupSetting()
	if err != nil {
		log.Fatalf("init.setupSetting err: %v", err)
	}

	// 初始化数据库表
	err = setupDBMigrate()
	if err != nil {
		log.Fatalf("init.setupDBMigrate err: %v", err)
	}

	err = setupLogger()
	if err != nil {
		log.Fatalf("init.setupLogger err: %v", err)
	}

	err = setupProjectSetting()
	if err != nil {
		log.Fatalf("init.setupProjectSetting err: %v", err)
	}

	if err = InitFileType(); err != nil {
		log.Fatalf("init.InitFileType err: %v", err)
	}
}

func main() {
	_ = filebrowser.GetFB() // 提前报错
	go task.GetTaskManager().Start()
	go proto.SANotifyEvent()
	gin.SetMode(config.ServerSetting.RunMode)
	engine := gin.New()
	api.LoadModules(engine)
	s := &http.Server{
		Addr:           ":" + config.ServerSetting.HttpPort,
		Handler:        engine,
		ReadTimeout:    config.ServerSetting.ReadTimeout,
		WriteTimeout:   config.ServerSetting.WriteTimeout,
		MaxHeaderBytes: 1 << 20,
	}

	s.ListenAndServe()
}

func setupDBMigrate() error {
	return entity.AutoMigrate()
}

func setupSetting() error {

	setting, err := config.NewSetting()
	if err != nil {
		return err
	}
	err = setting.ReadSection("Server", &config.ServerSetting)
	if err != nil {
		return err
	}

	err = setting.ReadSection("App", &config.AppSetting)
	if err != nil {
		return err
	}

	err = setting.ReadSection("ExtServer", &config.ExtServerSetting)
	if err != nil {
		return err
	}

	config.FileTypeSetting.PhotoType = []string{"heic", "jpg", "jpeg", "png", "gif", "psd", "pdd", "psdt", "psb", "bmp", "rle", "dib", "dcm", "dc3", "dic", "eps", "iff", "tdi", "jpf", "jpx", "jp2", "j2c", "j2k", "jpc", "jps", "pcx", "pdp", "raw", "pxr", "pbm", "pgm", "ppm", "pnm", "pfm", "pam", "sct", "tga", "vda", "icb", "vst", "tif", "tiff", "mpo", "webp", "ico"}
	config.FileTypeSetting.VideoType = []string{"mp4", "m4v", "avi", "mkv", "mov", "mpg", "mpeg", "vob", "ram", "rm", "rmvb", "asf", "wmv", "webm", "m2ts", "movie"}
	config.FileTypeSetting.OfficeToHtmlType = []string{"docx", "doc", "pptx"}
	config.FileTypeSetting.OfficeToPdfType = []string{"xlsx"}

	config.ServerSetting.ReadTimeout *= time.Second
	config.ServerSetting.WriteTimeout *= time.Second

	return nil
}

func setupLogger() error {
	config.Logger = logger.NewLogger(&lumberjack.Logger{
		Filename:  config.AppSetting.LogSavePath + "/" + config.AppSetting.LogFileName + config.AppSetting.LogFileExt,
		MaxSize:   600,
		MaxAge:    10,
		LocalTime: true,
	}, "", log.LstdFlags)

	return nil
}

func setupProjectSetting() error {
	list, err := entity.GetSettingList()
	if err != nil {
		return err
	}
	// 如果返回空列表，初始化配置
	if len(list) == 0 {
		_ = entity.InitSetting()
		list, _ = entity.GetSettingList()
	}
	// 初始化默认配置
	for _, val := range list {
		switch val.Name {
		case "PoolName":
			config.AppSetting.PoolName = val.Value
		case "PartitionName":
			config.AppSetting.PartitionName = val.Value
		case "IsAutoDel":
			config.AppSetting.IsAutoDel, _ = strconv.Atoi(val.Value)
		}
	}

	return nil
}

func InitFileType() error {
	for _, v := range config.FileTypeSetting.PhotoType {
		resource.FileTypeMap[v] = types.FolderPhoto
	}
	for _, v := range config.FileTypeSetting.VideoType {
		resource.FileTypeMap[v] = types.FolderVideo
	}
	for _, v := range config.FileTypeSetting.OfficeToPdfType {
		resource.FileTypeMap[v] = types.FolderOfficeWordPPt
	}
	for _, v := range config.FileTypeSetting.OfficeToHtmlType {
		resource.FileTypeMap[v] = types.FolderOfficeExcel
	}
	return nil
}
