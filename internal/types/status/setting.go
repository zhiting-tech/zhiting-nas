package status

import "gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"

const (
	SettingParamFailErr = iota + 60000
	SettingUpdateFailErr
)

func init() {
	errors.NewCode(SettingParamFailErr, "必填参数为空")
	errors.NewCode(SettingUpdateFailErr, "更新配置失败")
}
