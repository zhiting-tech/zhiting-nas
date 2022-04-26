package entity

import (
	"fmt"
	"gorm.io/gorm"
)

// FolderInfo 共享目录相关的数据表
type FolderInfo struct {
	ID             int    `gorm:"primary_key"`
	Uid            int    // 创建人
	AbsPath        string `gorm:"index"` // 绝对路径
	Name           string // 文件/文件夹名称
	Hash           string // 文件hash
	Mode           int    // 文件夹类型：1私人文件夹 2共享文件夹
	Type           int    // 类型：0文件夹 1文件
	IsEncrypt      int    // 是否加密
	Cipher         string // 加密后的密钥
	PoolName       string // 储存池ID
	PartitionName  string // 储存池分区ID
	Persons        string // 可访问成员，冗余用
	CreatedAt      int64  // 创建时间
	Identification string // 备份文件唯一标识
}

func (folder FolderInfo) TableName() string {
	return "folder"
}

// GetFolderList 获取本地文件夹
func GetFolderList(uid, pageOffset, pageSize int) ([]*FolderInfo, error) {
	var folderList []*FolderInfo
	db := GetDB()
	// 只需要带出存储池分区第一层数据
	likeStr := "/%/%/%"
	notLikeStr := "/%/%/%/%"
	// db = db.Where("uid = ?", uid)
	db = db.Where("abs_path like ? and abs_path not like ?", likeStr, notLikeStr)

	if pageOffset >= 0 && pageSize > 0 {
		db = db.Offset(pageOffset).Limit(pageSize)
	}

	err := db.Find(&folderList).Error

	if err != nil {
		return nil, err
	}

	return folderList, nil
}

func GetFolderListByAbsPath(absPath string) ([]*FolderInfo, error) {
	var folderList []*FolderInfo
	db := GetDB()
	// 只需要带出存储池分区第一层数据
	likeStr := "/%"
	notLikeStr := "/%/%"
	db = db.Where("abs_path like ? and abs_path not like ?", fmt.Sprintf("%s%s", absPath, likeStr), fmt.Sprintf("%s%s", absPath, notLikeStr))

	err := db.Find(&folderList).Error

	if err != nil {
		return nil, err
	}

	return folderList, nil
}

// GetFolderCount 获取某个目录下的所有文件
func GetFolderCount(uid int) int64 {
	var count int64
	db := GetDB()
	// 只需要带出存储池分区第一层数据
	likeStr := "/%/%/%"
	notLikeStr := "/%/%/%/%"
	// db = db.Where("uid = ?", uid)
	db = db.Model(FolderInfo{}).Where("abs_path like ? and abs_path not like ?", likeStr, notLikeStr)
	if err := db.Count(&count).Error; err != nil {
		return 0
	}

	return count
}

// FolderRow 获取关联的文件夹信息
type FolderRow struct {
	Id        int    // 主键
	UId       int    // UserId
	Name      string // 文件夹名称
	Type      int    // 类型0文件夹1文件
	AbsPath   string // 路径
	IsEncrypt int    // 是否加密
	FromUser  string // 分享者昵称
	Read      int    // 是否可读：1/0
	Write     int    // 是否可写：1/0
	Deleted   int    // 是否可删：1/0
	Hash      string
}

// GetRelateFolderList 获取文件夹列表, 使用auth跟folder做关联
func GetRelateFolderList(whereStr string, pageOffset, pageSize int) ([]*FolderRow, error) {
	db := GetDB()

	fields := []string{"folder.id", "folder.name", "folder.type", "folder.abs_path", "folder.is_encrypt", "folder.hash"}
	fields = append(fields, []string{"auth.from_user", "auth.read", "auth.write", "auth.deleted"}...)

	if pageOffset >= 0 && pageSize > 0 {
		db = db.Offset(pageOffset).Limit(pageSize)
	}
	rows, err := db.Select(fields).Table(FolderInfo{}.TableName() + " as folder").
		Joins("inner join `" + FolderAuth{}.TableName() + "` as auth on folder.id = auth.folder_id").
		Where(whereStr).
		Order("created_at asc").
		Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var folderRows []*FolderRow
	for rows.Next() {
		r := &FolderRow{}
		if err := rows.Scan(&r.Id, &r.Name, &r.Type, &r.AbsPath, &r.IsEncrypt, &r.Hash, &r.FromUser, &r.Read, &r.Write, &r.Deleted); err != nil {
			return nil, err
		}
		folderRows = append(folderRows, r)
	}
	return folderRows, nil
}

// GetRelateFolderCount 获取文件夹总数, 使用auth跟folder做关联
func GetRelateFolderCount(whereStr string) (int64, error) {
	var count int64
	db := GetDB()
	err := db.Table(FolderInfo{}.TableName() + " as folder").
		Joins("inner join `" + FolderAuth{}.TableName() + "` as auth on folder.id = auth.folder_id").
		Where(whereStr).
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetRelateFolderInfo 根据folderId跟userId 获取文件夹详情
func GetRelateFolderInfo(folderId, userId int) (*FolderRow, error) {
	db := GetDB()
	fields := []string{"folder.id", "folder.name", "folder.type", "folder.abs_path", "folder.is_encrypt"}
	fields = append(fields, []string{"auth.read", "auth.write", "auth.deleted"}...)
	row := db.Select(fields).Table(FolderInfo{}.TableName()+" as folder").
		Joins("inner join `"+FolderAuth{}.TableName()+"` as auth on folder.id = auth.folder_id").
		Where("folder.id = ? and auth.uid = ?", folderId, userId).
		Row()
	r := &FolderRow{}
	if err := row.Scan(&r.Id, &r.Name, &r.Type, &r.AbsPath, &r.IsEncrypt, &r.Read, &r.Write, &r.Deleted); err != nil {
		return nil, err
	}

	return r, nil
}

// GetRelateFolderInfoByUid 根据folderUid和userId 获取文件夹详情
func GetRelateFolderInfoByUid(folderUid, userId int) (*FolderRow, error) {
	db := GetDB()
	fields := []string{"folder.id", "folder.name", "folder.type", "folder.abs_path", "folder.is_encrypt"}
	fields = append(fields, []string{"auth.read", "auth.write", "auth.deleted"}...)
	row := db.Select(fields).Table(FolderInfo{}.TableName()+" as folder").
		Joins("inner join `"+FolderAuth{}.TableName()+"` as auth on folder.id = auth.folder_id").
		Where("folder.uid = ? and auth.uid = ?", folderUid, userId).
		Row()
	r := &FolderRow{}
	if err := row.Scan(&r.Id, &r.Name, &r.Type, &r.AbsPath, &r.IsEncrypt, &r.Read, &r.Write, &r.Deleted); err != nil {
		return nil, err
	}
	return r, nil
}

// GetFolderInfo 获取文件夹详情
func GetFolderInfo(id int) (*FolderInfo, error) {
	folderInfo := FolderInfo{ID: id}
	if err := GetDB().First(&folderInfo).Error; err != nil {
		return nil, err
	}
	return &folderInfo, nil
}

// GetFolderInfoByAbsPath 根据路径获取目录详情
func GetFolderInfoByAbsPath(absPath string) (*FolderInfo, error) {
	folderInfo := FolderInfo{}
	if err := GetDB().Where("abs_path = ?", absPath).First(&folderInfo).Error; err != nil {
		return nil, err
	}
	return &folderInfo, nil
}

// GetFoldersLikeAbsPath 根据路径获取目录详情
func GetFoldersLikeAbsPath(absPath string) ([]FolderInfo, error) {
	var folderInfos []FolderInfo
	if err := GetDB().Where("abs_path like ?", absPath+"%").Find(&folderInfos).Error; err != nil {
		return nil, err
	}
	return folderInfos, nil
}

// GetFoldersLikeNameCount 根据名称获取相似名称数量
func GetFoldersLikeNameCount(name string) (int64, error) {
	var count int64
	db := GetDB()
	err := db.Table(FolderInfo{}.TableName()+" as folder").
		Where("name like ?", name+"%").
		Count(&count).
		Error
	if err != nil {
		return 0, err
	}

	return count, nil
}

// GetFolderByName 在同一分区内通过name查找文件夹
func GetFolderByName(poolName, partitionName string, name string) (*FolderInfo, error) {
	var folderInfo *FolderInfo
	if err := GetDB().Where("pool_name = ? and partition_name = ? and name =?", poolName, partitionName, name).Find(&folderInfo).Error; err != nil {
		return nil, err
	}
	return folderInfo, nil
}

// GetPrivateFolders 获取私人文件
func GetPrivateFolders(userIDs []int) ([]*FolderInfo, error) {
	var folderInfos []*FolderInfo
	if err := GetDB().Where("mode = 1 and uid IN ?", userIDs).Find(&folderInfos).Error; err != nil {
		return nil, err
	}
	return folderInfos, nil
}

// GetPrivateFolder 获取单个用户私人文件
func GetPrivateFolder(userIDs int) (*FolderInfo, error) {
	var folderInfos *FolderInfo
	if err := GetDB().Select("abs_path").Where("mode = 1 and uid IN ?", userIDs).First(&folderInfos).Error; err != nil {
		return nil, err
	}
	return folderInfos, nil
}

// CreateFolder 创建文件夹
func CreateFolder(tx *gorm.DB, FolderInfo *FolderInfo) (*FolderInfo, error) {
	if err := tx.Create(FolderInfo).Error; err != nil {
		return nil, err
	}
	return FolderInfo, nil
}

// DelFolder 删除文件夹数据
func DelFolder(tx *gorm.DB, absPath string) error {
	if err := tx.Delete(FolderInfo{}, "abs_path like ?", absPath+"%").Error; err != nil {
		return err
	}
	return nil
}

// DelFolderByAbsPaths 根据路径删除数据
func DelFolderByAbsPaths(tx *gorm.DB, absPaths []string) error {
	for _, v := range absPaths {
		if err := tx.Delete(&FolderInfo{}, "abs_path like ?", v+"%").Error; err != nil {
			return err
		}
	}
	return nil
}

// UpdateFolderInfo 更新文件夹
func UpdateFolderInfo(tx *gorm.DB, id int, values interface{}) error {
	return tx.Model(&FolderInfo{}).Where("id = ?", id).Updates(values).Error
}

// BatchInsertFolder 批量插入权限
func BatchInsertFolder(tx *gorm.DB, folders []*FolderInfo) error {
	return tx.Create(&folders).Error
}

// QueryFolderByUid 根据Uid查找数据
func QueryFolderByUid(uid int) *FolderInfo {
	var folderInfo FolderInfo
	GetDB().Where("uid = ?", uid).First(&folderInfo)
	return &folderInfo
}

// GetFolderIdentification 获取用户备份文件标识
func GetFolderIdentification(uid int) []string {
	var identifications []string
	var folderInfo []FolderInfo
	GetDB().Select("identification").Where("uid = ? and identification <> ?", uid, "").Find(&folderInfo)
	for _, v := range folderInfo {
		identifications = append(identifications, v.Identification)
	}
	return identifications
}

// DelAllFolderRecode 移除所有folder表数据
func DelAllFolderRecode() {
	GetDB().Exec("delete from folder")
}

// GetAbsPathByMode 获取文件夹目录
func GetAbsPathByMode(mode int) []string {
	var (
		modeStr    []string
		folderInfo []FolderInfo
	)
	GetDB().Select("abs_path").Where("mode = ?", mode).Find(&folderInfo)
	for _, v := range folderInfo {
		modeStr = append(modeStr, v.AbsPath)
	}
	return modeStr
}
