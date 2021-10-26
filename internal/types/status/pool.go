package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	PoolParamIsNullErr = iota + 30000
	PoolNameParamErr
	PoolNameTooLongErr
	PoolNameIsExistErr
	PoolIsNotFoundErr
	PoolIsNotPermission
)

func init() {
	errors.NewCode(PoolParamIsNullErr, "名称或者硬盘名称为空")
	errors.NewCode(PoolNameParamErr, "名称仅可输入英文字母大小写、数字及+ _ . -")
	errors.NewCode(PoolNameTooLongErr, "名称不能超过50个字符")
	errors.NewCode(PoolNameIsExistErr, "存储池名称不能重复")
	errors.NewCode(PoolIsNotFoundErr, "存储池不存在")
	errors.NewCode(PoolIsNotPermission, "没有管理员权限")
}
