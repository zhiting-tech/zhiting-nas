package utils

import (
	"encoding/json"
	"fmt"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/types"
	"io/ioutil"
	"net/http"
)

// SaUserInfo SA-Server "用户详情" 反序列化结构体
type SaUserInfo struct {
	Status int    `json:"status"`
	Reason string `json:"reason"`
	SaData `json:"data"`
}

type SaData struct {
	UserId   int    `json:"user_id"`
	Nickname string `json:"nickname"`
	IsOwner  bool   `json:"is_owner"`
	SaArea   `json:"area"`
}

type SaArea struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetRequestSaServer(api, scopeToken string) (*SaUserInfo, error) {
	// 通过过来的token，验证token是否正确
	apiUrl := fmt.Sprint(config.ExtServerSetting.SaHttp, "://", config.ExtServerSetting.SaServer, api)
	request, err := http.NewRequest("GET", apiUrl, nil)
	if err != nil {
		return nil, err
	}

	// 设置GET请求头部参数
	request.Header[types.ScopeTokenKey] = []string{scopeToken}
	resp, err := (&http.Client{}).Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var userInfo SaUserInfo
	tmpUserInfo, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// 反序列化获得http请求body中所需的数据
	err = json.Unmarshal(tmpUserInfo, &userInfo)
	if err != nil {
		return nil, err
	}
	return &userInfo, nil
}
