package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	ResourceNotExistErr = iota + 10000
	ResourceExistErr
	TargetPathSameWithOriginalPathErr
	ShareRecordNotExistErr
	ParamsIllegalErr
	ResourcePathIllegalErr
	ShareTargetUserIsSelfErr
	TargetPathMustBeDirErr
	ShareResourceMustBeDirErr
	ShareResourceBanMoveOtherDir
	ResourceNotChosenErr
	HashNotExistErr
	NameAlreadyExistErr
	ParamFileHashError
	ChunkFileNotExistErr
	ChunkFileSizeBeyondErr
	AuthorizationInvalid
	ShareWithoutPermissionErr
	ResourceNotAuthErr
	ResourceNotReadAuthErr
	ResourceNotWriteAuthErr
	ResourceNotDeleteAuthErr
	ResourceNotMoveErr
	ResourceNotCopyErr
	ResourcePrivateErr
	ResourceTargetSubdirectoryErr
	ResourceSourceOperatingErr
	ResourceTargetOperatingErr
	ResourceUploadLimitExceeded
	ResourceNameTooLongErr
	ResourceTargetLimitExceeded
	ResourceHashInputNil
)

func init() {
	errors.NewCode(ResourceNotExistErr, "文件/文件夹不存在")
	errors.NewCode(ResourceExistErr, "该文件/文件夹已存在，如不是同一个，请修改名称后再操作")
	errors.NewCode(TargetPathSameWithOriginalPathErr, "目标路径与源路径相同")
	errors.NewCode(ShareRecordNotExistErr, "该共享记录不存在")
	errors.NewCode(ParamsIllegalErr, "参数%s不合法")
	errors.NewCode(ResourcePathIllegalErr, "路径不合法")
	errors.NewCode(ShareTargetUserIsSelfErr, "共享文件的目标不能是自己")
	errors.NewCode(TargetPathMustBeDirErr, "移动或复制的目标路径必须是目录")
	errors.NewCode(ShareResourceMustBeDirErr, "只能共享文件夹")
	errors.NewCode(ShareResourceBanMoveOtherDir, "共享文件不能移动到其他文件夹")
	errors.NewCode(ResourceNotChosenErr, "请选择要上传的文件")
	errors.NewCode(HashNotExistErr, "hash文件不存在")
	errors.NewCode(NameAlreadyExistErr, "名称不能重复")
	errors.NewCode(ParamFileHashError, "合并文件后hash不匹配")
	errors.NewCode(ChunkFileNotExistErr, "合并文件不存在或total_chunks大小错误")
	errors.NewCode(ChunkFileSizeBeyondErr, "文件大小超出限制")
	errors.NewCode(AuthorizationInvalid, "无效的授权，请重新授权")
	errors.NewCode(ShareWithoutPermissionErr, "您没有权限共享该文件夹")
	errors.NewCode(ResourceNotAuthErr, "文件/文件夹没有权限")
	errors.NewCode(ResourceNotReadAuthErr, "文件/文件夹没有可读权限")
	errors.NewCode(ResourceNotWriteAuthErr, "文件/文件夹没有可写权限")
	errors.NewCode(ResourceNotDeleteAuthErr, "文件/文件夹没有删除权限")
	errors.NewCode(ResourceNotMoveErr, "无移入权限")
	errors.NewCode(ResourceNotCopyErr, "无权限")
	errors.NewCode(ResourcePrivateErr, "只能在该私人文件夹下移动或复制")
	errors.NewCode(ResourceTargetSubdirectoryErr, "不能复制/移动到到子目录")
	errors.NewCode(ResourceSourceOperatingErr, "源路径存在移动文件夹或删除文件夹动作，不能操作")
	errors.NewCode(ResourceTargetOperatingErr, "目标路径存在移动文件夹或删除文件夹动作，不能操作")
	errors.NewCode(ResourceUploadLimitExceeded, "上传文件大小超出限制")
	errors.NewCode(ResourceNameTooLongErr, "不能超过255个字符")
	errors.NewCode(ResourceTargetLimitExceeded, "空间不足，操作失败")
	errors.NewCode(ResourceHashInputNil, "TotalChunks或者Hash为空")
}
