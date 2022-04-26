package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	PartitionParamFailErr = iota + 40000
	PartitionNameParamErr
	PartitionNameTooLongErr
	PartitionAddErr
	PartitionDeleteErr
	PartitionInfoErr
	PartitionUpdateErr
	PartitionNameRepeatErr
	PartitionCapacityExceededErr
	PartitionCapacityShrinkingErr
)

func init() {
	errors.NewCode(PartitionParamFailErr, "必填参数为空")
	errors.NewCode(PartitionNameParamErr, "名称仅可输入英文字母大小写、数字及+ _ . -")
	errors.NewCode(PartitionNameTooLongErr, "名称不能超过50个字符")
	errors.NewCode(PartitionAddErr, "分区添加失败")
	errors.NewCode(PartitionDeleteErr, "分区删除失败")
	errors.NewCode(PartitionInfoErr, "分区详情获取失败")
	errors.NewCode(PartitionUpdateErr, "分区编辑失败")
	errors.NewCode(PartitionNameRepeatErr, "名称重复")
	errors.NewCode(PartitionCapacityExceededErr, "容量不能超出%.2f%s")
	errors.NewCode(PartitionCapacityShrinkingErr, "容量不能减少")
}
