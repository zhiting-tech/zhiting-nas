package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	FolderParamFailErr = iota + 20000
	FolderInfoFailErr
	FolderConFirmPwdFailErr
	FolderPwdParamFailErr
	FolderSecretFailErr
	FolderKeyResolutionErr
	FolderAddFailErr
	FolderDelFailErr
	FolderUpdateFailErr
	FolderPwdFailErr
	FolderPathFailErr
	FolderNameTooLongErr
	FolderNameFormatErr
	FolderNameIsExistErr
	FolderTooMuchMemberErr
	FolderTooFewMemberErr
	FolderPwdConsistentErr
	FolderCannotModified
	FolderCannotEncryptErr
	FolderTargetTooSmallErr
	FolderPwdLessErr
	FolderPwdStringErr
	FolderRemoveErr
	FolderOldPwdFailErr
	FolderPwdNotInputErr
	FolderOldPwdErr
	FolderEncryptCannotShareErr
)

func init() {
	errors.NewCode(FolderParamFailErr, "必填参数为空")
	errors.NewCode(FolderInfoFailErr, "文件夹不存在")
	errors.NewCode(FolderConFirmPwdFailErr, "两次密码输入不一致")
	errors.NewCode(FolderPwdParamFailErr, "密码仅可使用数字、字母和 @ # & ！字符")
	errors.NewCode(FolderSecretFailErr, "密钥生成失败")
	errors.NewCode(FolderKeyResolutionErr, "密钥解析失败")
	errors.NewCode(FolderAddFailErr, "文件夹添加失败")
	errors.NewCode(FolderDelFailErr, "文件夹删除失败")
	errors.NewCode(FolderUpdateFailErr, "文件夹更新失败")
	errors.NewCode(FolderPwdFailErr, "文件夹密码错误")
	errors.NewCode(FolderPathFailErr, "文件夹路径错误")
	errors.NewCode(FolderNameTooLongErr, "文件夹名称长度过长")
	errors.NewCode(FolderNameFormatErr, "文件夹名称格式有误")
	errors.NewCode(FolderNameIsExistErr, "文件夹名称已存在")
	errors.NewCode(FolderTooMuchMemberErr, "列表成员过多")
	errors.NewCode(FolderTooFewMemberErr, "列表成员太少")
	errors.NewCode(FolderPwdConsistentErr, "新密码与原密码一致")
	errors.NewCode(FolderCannotModified, "文件夹类型不能修改")
	errors.NewCode(FolderCannotEncryptErr, "共享文件不能加密")
	errors.NewCode(FolderTargetTooSmallErr, "目标分区容量不足，不能迁移")
	errors.NewCode(FolderPwdLessErr, "密码不能少于6位")
	errors.NewCode(FolderPwdStringErr, "密码仅支持英文字母大小写、数字及 @ # & ！ ")
	errors.NewCode(FolderRemoveErr, "移除用户文件失败")
	errors.NewCode(FolderOldPwdFailErr, "文件夹密码错误")
	errors.NewCode(FolderPwdNotInputErr, "未输入密码")
	errors.NewCode(FolderOldPwdErr, "旧密码错误")
	errors.NewCode(FolderEncryptCannotShareErr, "加密文件夹不能共享")
}
