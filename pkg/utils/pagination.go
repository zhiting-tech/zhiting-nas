package utils

import (
	"github.com/gin-gonic/gin"
	"gitlab.yctc.tech/zhiting/wangpan.git/internal/config"
)

type Pager struct {
	Page      int   `json:"page"`
	PageSize  int   `json:"page_size"`
	TotalRows int64 `json:"total_rows"`
	HasMore   bool  `json:"has_more"`
}

func GetPage(c *gin.Context) int {
	page := StrTo(c.Query("page")).MustInt()
	//if page <= 0 {
	//	return 1
	//}
	return page
}

func GetPageSize(c *gin.Context) int {
	pageSize := StrTo(c.Query("page_size")).MustInt()
	//if pageSize <= 0 {
	//	return global.AppSetting.DefaultPageSize
	//}
	if pageSize > config.AppSetting.MaxPageSize {
		return config.AppSetting.MaxPageSize
	}

	return pageSize
}

func GetPageOffset(page, pageSize int) int {
	result := 0
	if page > 0 {
		result = (page - 1) * pageSize
	}

	return result
}
