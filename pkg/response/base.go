package response

import (
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/errors"
	"gitlab.yctc.tech/zhiting/wangpan.git/pkg/utils"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type BaseResponse struct {
	errors.Code
	Data interface{} `json:"data,omitempty"`
}

func getResponse(err error, resp interface{}) *BaseResponse {
	baseResult := BaseResponse{
		errors.GetCode(errors.OK),
		resp,
	}
	if err != nil {
		switch v := err.(type) {
		case errors.Error:
			baseResult.Code = v.Code
		default:
			baseResult.Code = errors.GetCode(errors.InternalServerErr)
		}
		log.Printf("%+v\n", err)
	}
	return &baseResult
}

func HandleResponse(ctx *gin.Context, err error, response interface{}) {
	HandleResponseWithStatus(ctx, http.StatusOK, err, response)
}

func HandleResponseList(ctx *gin.Context, err error, list interface{}, totalRow int64) {

	page := utils.GetPage(ctx)
	PageSize := utils.GetPageSize(ctx)
	pageOffset := utils.GetPageOffset(page, PageSize)
	// 是否还有更多数据
	hasMore := false
	if PageSize > 0 && int64(pageOffset+PageSize) < totalRow {
		hasMore = true
	}
	HandleResponseWithStatus(ctx, http.StatusOK, err, gin.H{
		"list": list,
		"pager": utils.Pager{
			Page:      page,
			PageSize:  PageSize,
			TotalRows: totalRow,
			HasMore:   hasMore,
		},
	})
}

func HandleResponseWithStatus(ctx *gin.Context, status int, err error, response interface{}) {
	baseResult := getResponse(err, response)
	ctx.JSON(status, baseResult)
}
