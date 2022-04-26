package entity

import (
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"sync"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var db *gorm.DB
var once sync.Once

func GetDB() *gorm.DB {
	once.Do(func() {
		loadDB()
	})
	return db
}

func loadDB() {
	sqlDB, err := gorm.Open(sqlite.Open(config.AppSetting.DbSavePath), &gorm.Config{})
	if err != nil {
		panic(fmt.Errorf("数据库连接失败 %v", err.Error()))
	}

	db = sqlDB.Session(&gorm.Session{Logger: logger.Default.LogMode(logger.Warn)})
}

func AutoMigrate() error {
	return GetDB().AutoMigrate(FolderInfo{}, FolderAuth{}, Setting{})
}
