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

// GetAppStatistic
//
//	@Tags			app_observability.statistic
//	@Summary		获取应用统计数据
//	@Description	获取应用统计数据
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.AppStatisticReq	true	"获取应用统计数据请求参数"
//	@Success		200		{object}	response.Response{data=response.AppStatistic}
//	@Router			/statistic/app [post]
func GetAppStatistic(ctx *gin.Context) {
	var req request.AppStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAppStatistic(ctx, req.StatisticFilter, req.StartDate, req.EndDate, req.Apps, req.AppType, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetAppStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		获取应用统计列表
//	@Description	获取应用统计列表（分页）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.AppStatisticListReq	true	"获取应用统计列表请求参数"
//	@Success		200		{object}	response.Response{data=response.PageResult{list=[]response.AppStatisticItem}}
//	@Router			/statistic/app/list [post]
func GetAppStatisticList(ctx *gin.Context) {
	var req request.AppStatisticListReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAppStatisticList(ctx, req.StatisticFilter, req.StartDate, req.EndDate, req.Apps, req.AppType, int32(req.PageNo), int32(req.PageSize), getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// ExportAppStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		导出应用统计列表
//	@Description	导出应用统计列表数据
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			data	body		request.AppStatisticReq	true	"导出应用统计列表请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/statistic/app/export [post]
func ExportAppStatisticList(ctx *gin.Context) {
	var req request.AppStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	file, err := service.ExportAppStatisticList(ctx, req.StatisticFilter, req.StartDate, req.EndDate, req.Apps, req.AppType, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	fileName := fmt.Sprintf("应用统计列表_%v-%v.xlsx", req.StartDate, req.EndDate)
	ctx.Writer.WriteHeader(http.StatusOK)
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	if _, err := file.WriteTo(ctx.Writer); err != nil {
		gin_util.Response(ctx, nil, err)
	}
}

// GetAppListSelect
//
//	@Tags			app_observability.statistic
//	@Summary		获取应用统计下拉列表
//	@Description	组织→用户→应用级联；获取筛选范围内的已发布应用
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.StatisticAppSelectReq	true	"获取应用统计下拉列表请求参数"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.MyAppItem}}
//	@Router			/statistic/app/select [post]
func GetAppListSelect(ctx *gin.Context) {
	var req request.StatisticAppSelectReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAppListSelect(ctx, req.StatisticFilter, req.AppType, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}
