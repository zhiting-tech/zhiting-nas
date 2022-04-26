package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	LoginNotSaEntity = iota + 80000
)

func init() {
	errors.NewCode(LoginNotSaEntity, "无sa实体，请添加")
}