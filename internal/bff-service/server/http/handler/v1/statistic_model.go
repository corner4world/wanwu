package v1

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetModelStatistic
//
//	@Tags			app_observability.statistic
//	@Summary		获取模型统计数据
//	@Description	获取模型统计数据
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.ModelStatisticReq	true	"获取模型统计数据请求参数"
//	@Success		200		{object}	response.Response{data=response.ModelStatistic}
//	@Router			/statistic/model [post]
func GetModelStatistic(ctx *gin.Context) {
	var req request.ModelStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetModelStatistic(ctx, req.StatisticFilter, req.StartDate, req.EndDate, req.Models, req.ModelType, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetModelStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		获取模型统计列表
//	@Description	获取模型统计列表（分页）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.ModelStatisticListReq	true	"获取模型统计列表请求参数"
//	@Success		200		{object}	response.Response{data=response.PageResult{list=[]response.ModelStatisticItem}}
//	@Router			/statistic/model/list [post]
func GetModelStatisticList(ctx *gin.Context) {
	var req request.ModelStatisticListReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetModelStatisticList(ctx, req.StatisticFilter, req.StartDate, req.EndDate, req.Models, req.ModelType, int32(req.PageNo), int32(req.PageSize), getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// ExportModelStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		导出模型统计列表
//	@Description	导出模型统计列表数据
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			data	body		request.ModelStatisticReq	true	"导出模型统计列表请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/statistic/model/export [post]
func ExportModelStatisticList(ctx *gin.Context) {
	var req request.ModelStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	file, err := service.ExportModelStatisticList(ctx, req.StatisticFilter, req.StartDate, req.EndDate, req.Models, req.ModelType, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	fileName := fmt.Sprintf("模型统计列表_%v-%v.xlsx", req.StartDate, req.EndDate)
	ctx.Writer.WriteHeader(http.StatusOK)
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	if _, err := file.WriteTo(ctx.Writer); err != nil {
		gin_util.Response(ctx, nil, err)
	}
}

// GetStatisticModelSelect
//
//	@Tags			app_observability.statistic
//	@Summary		获取模型统计下拉列表
//	@Description	组织→用户→模型级联；用于模型 Tab 筛选，非统计 list 接口
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.ModelStatisticSelectReq	true	"获取模型统计下拉列表请求参数"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.ModelInfo}}
//	@Router			/statistic/model/select [post]
func GetStatisticModelSelect(ctx *gin.Context) {
	var req request.ModelStatisticSelectReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetStatisticModelSelect(ctx, req.ModelType, getUserID(ctx), getOrgID(ctx), &req.StatisticFilter, isAdmin(ctx), isSystem(ctx) && isAdmin(ctx))
	gin_util.Response(ctx, resp, err)
}
