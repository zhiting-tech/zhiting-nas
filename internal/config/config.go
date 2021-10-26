package config

import (
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/logger"
	"time"
)

var (
	ServerSetting *ServerSettingS
	AppSetting *AppSettingS
	ExtServerSetting *ExtServerSettingS
	Logger *logger.Logger
)

// ServerSettingS 服务基础配置项
type ServerSettingS struct {
	RunMode string
	HttpPort string
	ReadTimeout time.Duration
	WriteTimeout time.Duration
}

// AppSettingS 应用配置项
type AppSettingS struct {
	DefaultPageSize int // 默认每页大小
	MaxPageSize int // 每页最大数量
	LogSavePath string // 日志保存目录
	LogFileName string // 日志文件名称
	LogFileExt string // 日志保存后缀
	UploadSavePath string // 文件保存路径
	DbSavePath string // 数据库保存路径
	PoolName   string // 默认保存存储池
	PartitionName string // 默认保存存储池分区
	IsAutoDel int // 成员退出后是否自动删除文件
}

// ExtServerSettingS 外部服务配置项
type ExtServerSettingS struct {
	LvmServer string // lvm的服务
	SaServer string // sa的服务
	SaHttp   string // sa的http协议
}