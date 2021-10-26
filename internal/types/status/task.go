package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	TaskNotExistErr = iota + 70000
	TaskStatusErr
)

func init() {
	errors.NewCode(TaskNotExistErr, "任务不存在")
	errors.NewCode(TaskStatusErr, "当前任务状态不支持操作")
}
