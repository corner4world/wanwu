package v1

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/UnicomAI/wanwu/pkg/gin-util/route"
	"github.com/gin-gonic/gin"
)

// GetAPIKeyStatistic
//
//	@Tags			app_observability.statistic
//	@Summary		获取API Key统计数据
//	@Description	获取API Key统计数据（概览+趋势）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.APIKeyStatisticReq	true	"获取API Key统计数据请求参数"
//	@Success		200		{object}	response.Response{data=response.APIKeyStatistic}
//	@Router			/statistic/api [post]
func GetAPIKeyStatistic(ctx *gin.Context) {
	var req request.APIKeyStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAPIKeyStatistic(ctx, &req, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetAPIKeyStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		获取API Key调用统计
//	@Description	获取API Key调用统计（分页，按名称+apikey+路径分组）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.APIKeyStatisticListReq	true	"获取API Key调用统计请求参数"
//	@Success		200		{object}	response.Response{data=response.PageResult{list=[]response.APIKeyStatisticItem}}
//	@Router			/statistic/api/list [post]
func GetAPIKeyStatisticList(ctx *gin.Context) {
	var req request.APIKeyStatisticListReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAPIKeyStatisticList(ctx, &req, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetAPIKeyStatisticRecord
//
//	@Tags			app_observability.statistic
//	@Summary		获取API Key调用记录
//	@Description	获取API Key调用记录（分页，单条记录）
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.APIKeyStatisticRecordReq	true	"获取API Key调用记录请求参数"
//	@Success		200		{object}	response.Response{data=response.PageResult{list=[]response.APIKeyStatisticRecordItem}}
//	@Router			/statistic/api/record [post]
func GetAPIKeyStatisticRecord(ctx *gin.Context) {
	var req request.APIKeyStatisticRecordReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetAPIKeyStatisticRecord(ctx, &req, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}

// ExportAPIKeyStatisticList
//
//	@Tags			app_observability.statistic
//	@Summary		导出API Key统计列表
//	@Description	导出API Key统计列表
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			data	body		request.APIKeyStatisticReq	true	"导出API Key统计列表请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/statistic/api/list/export [post]
func ExportAPIKeyStatisticList(ctx *gin.Context) {
	var req request.APIKeyStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	file, err := service.ExportAPIKeyStatisticList(ctx, &req, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	fileName := fmt.Sprintf("API Key统计列表_%v-%v.xlsx", req.StartDate, req.EndDate)
	ctx.Writer.WriteHeader(http.StatusOK)
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	if _, err := file.WriteTo(ctx.Writer); err != nil {
		gin_util.Response(ctx, nil, err)
	}
}

// ExportAPIKeyStatisticRecord
//
//	@Tags			app_observability.statistic
//	@Summary		导出API Key调用记录
//	@Description	导出API Key调用记录
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			data	body		request.APIKeyStatisticReq	true	"导出API Key调用记录请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/statistic/api/record/export [post]
func ExportAPIKeyStatisticRecord(ctx *gin.Context) {
	var req request.APIKeyStatisticReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	file, err := service.ExportAPIKeyStatisticRecord(ctx, &req, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	fileName := fmt.Sprintf("API Key调用记录_%v-%v.xlsx", req.StartDate, req.EndDate)
	ctx.Writer.WriteHeader(http.StatusOK)
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	if _, err := file.WriteTo(ctx.Writer); err != nil {
		gin_util.Response(ctx, nil, err)
	}
}

// GetApiKeyStatisticRoutes
//
//	@Tags			app_observability.statistic
//	@Summary		获取API Key统计路由列表
//	@Description	根据openApiType获取对应的路由信息，不传则返回所有OpenAPI路由
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			openApiType	query		string	false	"OpenAPI类型: agent, rag, workflow, chatflow, knowledge"
//	@Success		200			{object}	response.Response{data=response.ListResult{list=[]response.ApiKeyStatisticRouteItem}}
//	@Router			/statistic/api/routes [get]
func GetApiKeyStatisticRoutes(ctx *gin.Context) {
	routes := route.GetApiKeyStatisticRoutes(ctx.Query("openApiType"))
	gin_util.Response(ctx, routes, nil)
}

// GetStatisticAPIKeySelect
//
//	@Tags			app_observability.statistic
//	@Summary		获取API Key下拉列表
//	@Description	组织→用户→API Key名称级联
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.APIKeySelectReq	true	"获取API Key下拉列表请求参数"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.APIKeyDetailResponse}}
//	@Router			/statistic/api/select [post]
func GetStatisticAPIKeySelect(ctx *gin.Context) {
	var req request.APIKeySelectReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.GetStatisticAPIKeySelect(ctx, req.StatisticFilter, getUserID(ctx), getOrgID(ctx), isAdmin(ctx), isSystem(ctx))
	gin_util.Response(ctx, resp, err)
}
