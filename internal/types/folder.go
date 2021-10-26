package types

const (
	FolderTypeDir = 0 // 目录类型
	FolderTypeFile = 1 // 文件类型

	FolderPrivateDir = 1 // 私人文件夹
	FolderShareDir = 2 // 分享文件夹

	FolderEncryptExt = ".enc" // 文件加密后缀

	FolderFamilyDirUid = -1 // 家庭文件夹创建时的uid
	FolderSelfDirUid = 0 // 个人文件夹创建时的uid

	FolderFamilyDir = "__family__" // 共享家庭文件夹名称
	FolderSelfDir = "__%d__" // 私人文件夹名称
)