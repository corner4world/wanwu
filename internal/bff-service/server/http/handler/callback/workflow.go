package callback

import (
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	"github.com/UnicomAI/wanwu/pkg/constant"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetWorkflowList
//
//	@Tags			callback
//	@Summary		根据userId和spaceId获取Workflow
//	@Description	根据userId和spaceId获取Workflow
//	@Accept			json
//	@Produce		json
//	@Param			userId	query		string	true	"获取工作流参数userId"
//	@Param			orgId	query		string	true	"获取工作流参数orgId"
//	@Success		200		{object}	response.Response
//	@Router			/workflow/list [get]
func GetWorkflowList(ctx *gin.Context) {
	var req request.GetWorkflowListReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetAppList(ctx, req.UserId, req.OrgId, constant.AppTypeWorkflow)
	gin_util.Response(ctx, resp, err)
}

// GetChatflowList
//
//	@Tags			callback
//	@Summary		根据userId和spaceId获取Chatflow
//	@Description	根据userId和spaceId获取Chatflow
//	@Accept			json
//	@Produce		json
//	@Param			userId	query		string	true	"获取工作流参数userId"
//	@Param			orgId	query		string	true	"获取工作流参数orgId"
//	@Success		200		{object}	response.Response
//	@Router			/chatflow/list [get]
func GetChatflowList(ctx *gin.Context) {
	var req request.GetWorkflowListReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetAppList(ctx, req.UserId, req.OrgId, constant.AppTypeChatflow)
	gin_util.Response(ctx, resp, err)
}

// GetWorkflowCustomTool
//
//	@Tags			callback
//	@Summary		获取自定义工具详情
//	@Description	获取自定义工具详情
//	@Accept			json
//	@Produce		json
//	@Param			customToolId	query		string	true	"customToolId"
//	@Success		200				{object}	response.Response{data=response.CustomToolDetail}
//	@Router			/workflow/tool/custom [get]
func GetWorkflowCustomTool(ctx *gin.Context) {
	resp, err := service.GetCustomTool(ctx, "", "", ctx.Query("customToolId"))
	gin_util.Response(ctx, resp, err)
}

// GetWorkflowSquareTool
//
//	@Tags			callback
//	@Summary		获取内置工具详情
//	@Description	获取内置工具详情
//	@Accept			json
//	@Produce		json
//	@Param			toolSquareId	query		string	true	"toolSquareId"
//	@Param			userID			query		string	true	"用户ID"
//	@Param			orgID			query		string	true	"组织ID"
//	@Success		200				{object}	response.Response{data=response.ToolSquareDetail}
//	@Router			/workflow/tool/square [get]
func GetWorkflowSquareTool(ctx *gin.Context) {
	resp, err := service.GetToolSquareDetail(ctx, "", "", ctx.Query("toolSquareId"))
	gin_util.Response(ctx, resp, err)
}
