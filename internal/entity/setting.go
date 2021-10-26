package entity

import (
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"gorm.io/gorm"
)

type Setting struct {
	Name string
	Value string
}

func (setting Setting) TableName() string {
	return "setting"
}

func GetSettingList() ([]*Setting, error) {
	var settingList []*Setting
	if err := GetDB().Find(&settingList).Error; err != nil {
		return nil, err
	}
	return settingList, nil
}

// DropSetting 清空配置表
func DropSetting(tx *gorm.DB) error {
	return tx.Where("name <> ''").Delete(Setting{}).Error
}

// BatchInsertSetting 批量插入配置
func BatchInsertSetting(tx *gorm.DB, settings []Setting) error {
	return tx.Create(&settings).Error
}

// InitSetting 初始化配置
func InitSetting() error {
	settings := make([]Setting, 0, 3)
	settings = append(settings, Setting{Name: "PoolName", Value: types.LvmSystemDefaultName})
	settings = append(settings, Setting{Name: "PartitionName", Value: types.LvmSystemDefaultName})
	settings = append(settings, Setting{Name: "IsAutoDel", Value: "0"})

	return BatchInsertSetting(GetDB(), settings)
}

// InitSettingPool 初始化存储池相关的配置
func InitSettingPool() {
	// 初始化存储池相关的配置
	GetDB().Where("name = 'PoolName'").Updates(Setting{Value: types.LvmSystemDefaultName})
	GetDB().Where("name = 'PartitionName'").Updates(Setting{Value: types.LvmSystemDefaultName})
}

func UpdatePartitionNameSetting(partitionName string) {
	GetDB().Where("name = 'PartitionName'").Updates(Setting{Value: partitionName})
}

func UpdatePoolNameSetting(poolName string) {
	GetDB().Where("name = 'PoolName'").Updates(Setting{Value: poolName})
}