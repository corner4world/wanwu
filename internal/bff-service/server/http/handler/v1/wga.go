package v1

import (
	"net/http"
	"net/url"

	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/gin-gonic/gin"
)

// GetGeneralAgentSubList
//
//	@Tags			wga
//	@Summary		获取通用智能体子智能体列表
//	@Description	获取通用智能体支持的子智能体列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.GetGeneralAgentSubListResp}
//	@Router			/general/agent/sub/list [get]
func GetGeneralAgentSubList(ctx *gin.Context) {
	resp, err := service.GetGeneralAgentSubList(ctx)
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentUploadLimit
//
//	@Tags			wga
//	@Summary		获取通用智能体上传文件格式限制
//	@Description	获取通用智能体所支持的上传文件格式及大小限制
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.GeneralAgentUploadLimitResp}
//	@Router			/general/agent/upload/limit [get]
func GetGeneralAgentUploadLimit(ctx *gin.Context) {
	resp, err := service.GetGeneralAgentUploadLimit(ctx, getUserID(ctx), getOrgID(ctx))
	gin_util.Response(ctx, resp, err)
}

// UpdateGeneralAgentConfig
//
//	@Tags			wga
//	@Summary		修改通用智能体配置
//	@Description	更新通用智能体工具配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.UpdateGeneralAgentConfigReq	true	"更新通用智能体配置请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/general/agent/config [put]
func UpdateGeneralAgentConfig(ctx *gin.Context) {
	var req request.UpdateGeneralAgentConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.UpdateGeneralAgentConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// GetGeneralAgentConfig
//
//	@Tags			wga
//	@Summary		获取通用智能体配置
//	@Description	获取通用智能体配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.GetGeneralAgentConfigResp}
//	@Router			/general/agent/config [get]
func GetGeneralAgentConfig(ctx *gin.Context) {
	resp, err := service.GetGeneralAgentConfig(ctx, getUserID(ctx), getOrgID(ctx))
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentAssistantSelect
//
//	@Tags			wga
//	@Summary		通用智能体智能体选择，只返回单智能体
//	@Description	获取通用智能体智能体选择，只返回单智能体
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"智能体名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.ExplorationAppInfo}}
//	@Router			/general/agent/assistant/select [get]
func GetGeneralAgentAssistantSelect(ctx *gin.Context) {
	req := request.GetExplorationAppListRequest{
		Name: ctx.Query("name"),
	}
	resp, err := service.GetAssistantSelect(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentToolSelect
//
//	@Tags			wga
//	@Summary		通用智能体工具选择
//	@Description	获取通用智能体工具选择，用于用户选择工具进行对话
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			agentId	query		string	true	"智能体ID，默认为空"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.GetGeneralAgentToolSelectResp}}
//	@Router			/general/agent/tool/select [get]
func GetGeneralAgentToolSelect(ctx *gin.Context) {
	resp, err := service.GetGeneralAgentToolSelect(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("agentId"))
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentToolInfo
//
//	@Tags			wga
//	@Summary		通用智能体工具详情
//	@Description	获取通用智能体工具详情，用于工具调用
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			toolId		query		string	true	"工具ID"
//	@Param			toolType	query		string	true	"工具类型"
//	@Success		200			{object}	response.Response{data=response.GeneralAgentToolInfoResp}
//	@Router			/general/agent/tool/info [get]
func GetGeneralAgentToolInfo(ctx *gin.Context) {
	resp, err := service.GetGeneralAgentToolInfo(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("toolId"), ctx.Query("toolType"))
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentMCPSelect
//
//	@Tags			wga
//	@Summary		通用智能体MCP下拉接口列表
//	@Description	获取通用智能体MCP下拉接口列表，用于用户选择MCP进行对话
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"MCP名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.MCPSelect}}
//	@Router			/general/agent/mcp/select [get]
func GetGeneralAgentMCPSelect(ctx *gin.Context) {
	resp, err := service.GetMCPSelect(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentWorkflowSelect
//
//	@Tags			wga
//	@Summary		通用智能体工作流列表
//	@Description	获取通用智能体工作流列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"workflow名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.ExplorationAppInfo}}
//	@Router			/general/agent/workflow/select [get]
func GetGeneralAgentWorkflowSelect(ctx *gin.Context) {
	req := request.GetExplorationAppListRequest{
		Name: ctx.Query("name"),
	}
	resp, err := service.GetWorkflowSelect(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentSkillSelect
//
//	@Tags			wga
//	@Summary		获取通用智能体skill列表
//	@Description	获取通用智能体skill列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			name	query		string	false	"skill名称"
//	@Success		200		{object}	response.Response{data=response.ListResult{list=[]response.CustomSkillDetail}}
//	@Router			/general/agent/skill/select [get]
func GetGeneralAgentSkillSelect(ctx *gin.Context) {
	resp, err := service.GetCustomSkillList(ctx, getUserID(ctx), getOrgID(ctx), ctx.Query("name"))
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentKnowledgeSelect
//
//	@Tags			knowledge
//	@Summary		通用智能体知识库下拉接口列表
//	@Description	获取通用智能体知识库下拉接口列表
//	@Security		JWT
//	@Accept			json
//	@Param			data	body	request.KnowledgeSelectReq	true	"查询知识库列表"
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.KnowledgeListResp}
//	@Router			/general/agent/knowledge/select [post]
func GetGeneralAgentKnowledgeSelect(ctx *gin.Context) {
	userId, orgId := getUserID(ctx), getOrgID(ctx)
	var req request.KnowledgeSelectReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.SelectKnowledgeList(ctx, userId, orgId, &req)
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentResourceSelect
//
//	@Tags			wga
//	@Summary		通用智能体资源选择列表
//	@Description	获取通用智能体资源选择列表，包括MCP、Workflow、Skill、智能体四种类型
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	response.Response{data=response.GeneralAgentResourceSelectList}
//	@Router			/general/agent/resource/select [get]
func GetGeneralAgentResourceSelect(ctx *gin.Context) {
	resp, err := service.GetGeneralAgentResourceSelect(ctx, getUserID(ctx), getOrgID(ctx), "")
	gin_util.Response(ctx, resp, err)
}

// CreateGeneralAgentConversation
//
//	@Tags			wga
//	@Summary		创建通用智能体对话
//	@Description	创建通用智能体对话
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.CreateGeneralAgentConversationReq	true	"创建通用智能体对话请求参数"
//	@Success		200		{object}	response.Response{data=response.CreateGeneralAgentConversationResp}
//	@Router			/general/agent/conversation [post]
func CreateGeneralAgentConversation(ctx *gin.Context) {
	var req request.CreateGeneralAgentConversationReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.CreateGeneralAgentConversation(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// DeleteGeneralAgentConversation
//
//	@Tags			wga
//	@Summary		删除通用智能体对话
//	@Description	删除通用智能体对话
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.DeleteGeneralAgentConversationReq	true	"删除通用智能体对话请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/general/agent/conversation [delete]
func DeleteGeneralAgentConversation(ctx *gin.Context) {
	var req request.DeleteGeneralAgentConversationReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.DeleteGeneralAgentConversation(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// GetGeneralAgentConversationList
//
//	@Tags			wga
//	@Summary		通用智能体对话列表
//	@Description	获取通用智能体对话历史列表
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			page		query		int	false	"页码，默认1"
//	@Param			pageSize	query		int	false	"每页数量，默认20"
//	@Success		200			{object}	response.Response{data=response.ListResult{list=[]response.GeneralAgentConversationInfo}}
//	@Router			/general/agent/conversation/list [get]
func GetGeneralAgentConversationList(ctx *gin.Context) {
	var req request.GetGeneralAgentConversationListReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetGeneralAgentConversationList(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentConversationDetail
//
//	@Tags			wga
//	@Summary		通用智能体对话详情
//	@Description	获取指定会话的对话详情，包括对话标题、创建时间等信息
//	@Security		JWT
//	@Produce		json
//	@Param			threadId	query		string	true	"会话ID"
//	@Success		200			{object}	response.Response{data=response.ListResult{list=[]response.GeneralAgentConversationDetailInfo}}
//	@Router			/general/agent/conversation/detail [get]
func GetGeneralAgentConversationDetail(ctx *gin.Context) {
	var req request.GetGeneralAgentConversationDetailReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetGeneralAgentConversationDetail(ctx, getUserID(ctx), getOrgID(ctx), req.ThreadID)
	gin_util.Response(ctx, resp, err)
}

// GetGeneralAgentConversationConfig
//
//	@Tags			wga
//	@Summary		通用智能体对话配置
//	@Description	获取指定会话的对话配置信息
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			threadId	query		string	true	"会话ID"
//	@Success		200			{object}	response.Response{data=response.GetGeneralAgentConversationConfigResp}
//	@Router			/general/agent/conversation/config [get]
func GetGeneralAgentConversationConfig(ctx *gin.Context) {
	var req request.GetGeneralAgentConversationConfigReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GetGeneralAgentConversationConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// UpdateGeneralAgentConversationConfig
//
//	@Tags			wga
//	@Summary		修改通用智能体对话配置
//	@Description	修改通用智能体对话配置
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.UpdateGeneralAgentConversationConfigReq	true	"修改通用智能体对话配置请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/general/agent/conversation/config [put]
func UpdateGeneralAgentConversationConfig(ctx *gin.Context) {
	var req request.UpdateGeneralAgentConversationConfigReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.UpdateGeneralAgentConversationConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, nil, err)
}

// CheckGeneralAgentConversationConfig
//
//	@Tags			wga
//	@Summary		通用智能体配置检查接口
//	@Description	通用智能体配置检查接口
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.GeneralAgentConfigCheckRequest	true	"通用智能体配置检查请求参数"
//	@Success		200		{object}	response.Response{data=response.GeneralAgentConfigCheckResponse}
//	@Router			/general/agent/conversation/config/check [post]
func CheckGeneralAgentConversationConfig(ctx *gin.Context) {
	var req request.GeneralAgentConfigCheckRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	resp, err := service.CheckGeneralAgentConversationConfig(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// GeneralAgentWorkspaceDownload
//
//	@Tags			wga
//	@Summary		通用智能体workspace下载
//	@Description	通用智能体workspace下载接口
//	@Security		JWT
//	@Accept			json
//	@Produce		application/octet-stream
//	@Param			data	query	request.GeneralAgentWorkspaceDownloadReq	true	"workspace下载请求参数"
//	@Success		200		{file}	stream
//	@Router			/general/agent/conversation/workspace/download [get]
func GeneralAgentWorkspaceDownload(ctx *gin.Context) {
	var req request.GeneralAgentWorkspaceDownloadReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	fileName, data, err := service.GeneralAgentWorkspaceDownload(ctx, getUserID(ctx), getOrgID(ctx), req)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	ctx.Header("Content-Disposition", "attachment; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", "application/octet-stream")
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Data(http.StatusOK, "application/octet-stream", data)
}

// GeneralAgentWorkspacePreview
//
//	@Tags			wga
//	@Summary		通用智能体workspace预览
//	@Description	通用智能体workspace预览接口，查看所给path的文件内容，返回文件内容用于前端预览
//	@Security		JWT
//	@Accept			json
//	@Produce		*/*
//	@Param			data	query	request.GeneralAgentWorkspacePreviewReq	true	"workspace预览请求参数"
//	@Success		200		{file}	stream
//	@Router			/general/agent/conversation/workspace/preview [get]
func GeneralAgentWorkspacePreview(ctx *gin.Context) {
	var req request.GeneralAgentWorkspacePreviewReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	fileName, data, contentType, err := service.GeneralAgentWorkspacePreview(ctx, getUserID(ctx), getOrgID(ctx), req)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	ctx.Header("Content-Disposition", "inline; filename*=utf-8''"+url.QueryEscape(fileName))
	ctx.Header("Content-Type", contentType)
	ctx.Header("Access-Control-Expose-Headers", "Content-Disposition")
	ctx.Data(http.StatusOK, contentType, data)
}

// GeneralAgentWorkspaceInfo
//
//	@Tags			wga
//	@Summary		通用智能体workspace目录树
//	@Description	通用智能体workspace目录树接口，查看所给path的层级目录，返回目录结构与文件名等信息，类似于linux的tree命令
//	@Security		JWT
//	@Accept			json
//	@Produce		json
//	@Param			data	query		request.GeneralAgentWorkspaceReq	true	"workspace目录树请求参数"
//	@Success		200		{object}	response.Response{data=response.GeneralAgentWorkspaceResp}
//	@Router			/general/agent/conversation/workspace [get]
func GeneralAgentWorkspaceInfo(ctx *gin.Context) {
	var req request.GeneralAgentWorkspaceReq
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	resp, err := service.GeneralAgentWorkspaceInfo(ctx, getUserID(ctx), getOrgID(ctx), req)
	gin_util.Response(ctx, resp, err)
}

// GeneralAgentConversationChat
//
//	@Tags			wga
//	@Summary		通用智能体对话流
//	@Description	通用智能体对话流，用于实时接收用户输入和获取智能体回复，SSE流式返回
//	@Security		JWT
//	@Accept			json
//	@Produce		text/event-stream
//	@Param			data	body		request.GeneralAgentConversationChatReq	true	"通用智能体对话流请求参数"
//	@Success		200		{object}	string									"SSE流式返回"
//	@Router			/general/agent/conversation/chat [post]
func GeneralAgentConversationChat(ctx *gin.Context) {
	var req request.GeneralAgentConversationChatReq
	if !gin_util.Bind(ctx, &req) {
		return
	}
	err := service.GeneralAgentConversationChat(ctx, getUserID(ctx), getOrgID(ctx), req)
	if err != nil {
		gin_util.Response(ctx, nil, err)
	}
}
