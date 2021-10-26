package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	DiskParamFailErr = iota + 50000
)

func init() {
	errors.NewCode(DiskParamFailErr, "必填参数为空")
}