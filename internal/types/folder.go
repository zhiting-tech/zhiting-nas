package types

const (
	FolderTypeDir  = 0 // 目录类型
	FolderTypeFile = 1 // 文件类型

	FolderPrivateDir = 1 // 私人文件夹
	FolderShareDir   = 2 // 分享文件夹

	FolderEncryptExt = ".enc" // 文件加密后缀

	FolderFamilyDirUid = -100 // 家庭文件夹创建时的uid

	FolderSelfDirUid = 0 // 个人文件夹创建时的uid

	FolderFamilyDir  = "__family__"        // 共享家庭文件夹名称
	FolderDepartment = "__department_%d__" // 部门文件夹名称
	FolderSelfDir    = "__%d__"            // 私人文件夹名称

	AreaFamilyType  = 1 // 家庭类型
	AreaCompanyType = 2 // 公司类型

	SaAreaNotRemoveEvent    = 4 // 解散家庭/公司不移除文件事件
	SaAreaRemoveEvent       = 3 // 解散家庭/公司移除文件事件
	SaUserRemoveEvent       = 2 // 移除用户文件事件
	SaDepartmentRemoveEvent = 1 // 移除部门文件事件
	FolderPhoto             = 1 // 图片类型
	FolderVideo             = 2 // 视频类型
	FolderOfficeWordPPt     = 3 // ppt/word文件
	FolderOfficeExcel       = 4 // excel文件

	ChunkSize = 1024 * 1024 * 8
)
