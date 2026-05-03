package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/internal/bff-service/service"
	"github.com/UnicomAI/wanwu/pkg/constant"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	sse_util "github.com/UnicomAI/wanwu/pkg/sse-util"
	"github.com/gin-gonic/gin"
)

//	@title		AI Agent Productivity Platform - Open API
//	@version	v0.0.1

//	@BasePath	/openapi/v1

// CreateAgent
//
//	@Tags			openapi
//	@Summary		创建智能体OpenAPI
//	@Description	创建智能体OpenAPI
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPICreateAgentRequest	true	"请求参数"
//	@Success		200		{object}	response.Response{data=response.OpenAPICreateAgentResponse}
//	@Router			/agent [post]
func CreateAgent(ctx *gin.Context) {
	var req request.OpenAPICreateAgentRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)
	assistantCreateResp, err := service.AssistantCreate(ctx, userID, orgID, request.AssistantCreateReq(req))
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	assistantInfo, err := service.GetAssistantInfo(ctx, userID, orgID, request.AssistantIdRequest{
		AssistantId: assistantCreateResp.AssistantId,
	}, false)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	gin_util.Response(ctx, response.OpenAPICreateAgentResponse{UUID: assistantInfo.UUID}, nil)
}

// CreateAgentConversation
//
//	@Tags			openapi
//	@Summary		智能体创建对话OpenAPI
//	@Description	智能体创建对话OpenAPI
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPIAgentCreateConversationRequest	true	"请求参数"
//	@Success		400		{object}	response.Response{data=response.OpenAPIAgentCreateConversationResponse}
//	@Router			/agent/conversation [post]
func CreateAgentConversation(ctx *gin.Context) {
	var req request.OpenAPIAgentCreateConversationRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)
	appID, err := service.GetAssistantIdByUuid(ctx, req.UUID)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	resp, err := service.ConversationCreate(ctx, userID, orgID, request.ConversationCreateRequest{
		AssistantId: appID,
		Prompt:      req.Title,
	}, constant.ConversationTypeOpenAPI)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	gin_util.Response(ctx, response.OpenAPIAgentCreateConversationResponse{ConversationID: resp.ConversationId}, nil)
}

// ChatAgent
//
//	@Tags			openapi
//	@Summary		智能体对话OpenAPI
//	@Description	智能体对话OpenAPI
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPIAgentChatRequest	true	"请求参数"
//	@Success		200		{object}	response.OpenAPIAgentChatResponse
//	@Success		400		{object}	response.Response
//	@Router			/agent/chat [post]
func ChatAgent(ctx *gin.Context) {
	var req request.OpenAPIAgentChatRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)
	appID, err := service.GetAssistantIdByUuid(ctx, req.UUID)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	// 流式
	if req.Stream {
		if err := service.AssistantConversionStream(ctx, userID, orgID, request.ConversionStreamRequest{
			AssistantId:    appID,
			ConversationId: req.ConversationID,
			Prompt:         req.Query,
			FileInfo:       req.FileInfo,
		}, true, constant.AppStatisticSourceOpenAPI); err != nil {
			gin_util.Response(ctx, nil, err)
		}
		return
	}
	// 非流式
	startTime := time.Now()
	chatCh, err := service.CallAssistantConversationStream(ctx, userID, orgID, request.ConversionStreamRequest{
		AssistantId:    appID,
		ConversationId: req.ConversationID,
		Prompt:         req.Query,
		FileInfo:       req.FileInfo,
	}, true)
	if err != nil {
		service.RecordAppStatistic(ctx.Request.Context(), userID, orgID, appID, constant.AppTypeAgent, false, false, 0, 0, constant.AppStatisticSourceOpenAPI)
		gin_util.Response(ctx, nil, err)
		return
	}
	var output string
	resp := &response.OpenAPIAgentChatResponse{}
	for chat := range chatCh {
		// 注意这里智能体的原始流式返回没有data:前缀
		if strings.TrimSpace(chat) == "" {
			continue
		}
		curr := &response.OpenAPIAgentChatResponse{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(chat, "data:")), curr); err != nil {
			log.Errorf("[Agent] %v conversation %v user %v org %v unmarshal %v err: %v", appID, req.ConversationID, userID, orgID, err)
			continue
		}
		resp = curr
		output += curr.Response
	}
	resp.Response = output
	costs := time.Since(startTime).Milliseconds()
	service.RecordAppStatistic(ctx.Request.Context(), userID, orgID, appID, constant.AppTypeAgent, true, false, 0, int64(costs), constant.AppStatisticSourceOpenAPI)
	b, _ := json.Marshal(resp)
	status := http.StatusOK
	ctx.Set(gin_util.STATUS, status)
	ctx.Set(gin_util.RESULT, string(b))
	ctx.JSON(status, resp)
}

// ChatRag
//
//	@Tags			openapi
//	@Summary		文本问答OpenAPI
//	@Description	文本问答OpenAPI
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPIRagChatRequest	true	"请求参数"
//	@Success		200		{object}	response.OpenAPIRagChatResponse
//	@Success		400		{object}	response.Response
//	@Router			/rag/chat [post]
func ChatRag(ctx *gin.Context) {
	var req request.OpenAPIRagChatRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)

	// 流式 —— openapi 固定走 legacy 格式（原 rag-service SSE JSON 透传），
	// 不跟随 web 的 AG-UI 协议改造，避免破坏外部集成方的解析逻辑
	if req.Stream {
		if err := service.ChatRagStreamLegacy(ctx, userID, orgID, request.ChatRagRequest{RagID: req.UUID, Question: req.Query, History: req.History}, true, constant.AppStatisticSourceOpenAPI); err != nil {
			gin_util.Response(ctx, nil, err)
		}
		return
	}
	// 非流式
	startTime := time.Now()
	chatCh, _, err := service.CallRagChatStream(ctx, userID, orgID, request.ChatRagRequest{RagID: req.UUID, Question: req.Query, History: req.History}, true)
	if err != nil {
		service.RecordAppStatistic(ctx.Request.Context(), userID, orgID, req.UUID, constant.AppTypeRag, false, false, 0, 0, constant.AppStatisticSourceOpenAPI)
		gin_util.Response(ctx, nil, err)
		return
	}
	var output string
	resp := &response.OpenAPIRagChatResponse{}
	for chat := range chatCh {
		if !strings.HasPrefix(chat, "data:") || strings.HasPrefix(chat, strings.TrimSpace(sse_util.DONE_MSG)) {
			continue
		}
		curr := &response.OpenAPIRagChatResponse{}
		if err := json.Unmarshal([]byte(strings.TrimPrefix(chat, "data:")), curr); err != nil {
			log.Errorf("[RAG] %v user %v org %v unmarshal %v err: %v", req.UUID, userID, orgID, err)
			continue
		}
		resp = curr
		output += curr.Data.Output
	}
	resp.Data.Output = output
	costs := time.Since(startTime).Milliseconds()
	service.RecordAppStatistic(ctx.Request.Context(), userID, orgID, req.UUID, constant.AppTypeRag, true, false, 0, int64(costs), constant.AppStatisticSourceOpenAPI)
	b, _ := json.Marshal(resp)
	status := http.StatusOK
	ctx.Set(gin_util.STATUS, status)
	ctx.Set(gin_util.RESULT, string(b))
	ctx.JSON(status, resp)
}

// DraftChatAgent
//
//	@Tags			openapi
//	@Summary		智能体草稿态对话OpenAPI
//	@Description	智能体草稿态对话OpenAPI，基于草稿配置进行问答，不要求智能体已发布，不计入统计
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPIAgentDraftChatRequest	true	"请求参数"
//	@Success		200		{object}	response.OpenAPIAgentChatResponse
//	@Success		400		{object}	response.Response
//	@Router			/agent/chat/draft [post]
func DraftChatAgent(ctx *gin.Context) {
	var req request.OpenAPIAgentDraftChatRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)
	appID, err := service.GetAssistantIdByUuid(ctx, req.UUID)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	assistantInfo, err := service.GetAssistantInfo(ctx, userID, orgID, request.AssistantIdRequest{AssistantId: appID}, false)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	if assistantInfo.Prologue == "" {
		gin_util.Response(ctx, nil, fmt.Errorf("开场白未配置，请先通过配置更新接口设置开场白"))
		return
	}
	if assistantInfo.ModelConfig.ModelId == "" {
		gin_util.Response(ctx, nil, fmt.Errorf("大模型未配置，请先通过配置更新接口设置大模型"))
		return
	}
	if err := service.AssistantConversionStream(ctx, userID, orgID, request.ConversionStreamRequest{
		AssistantId:    appID,
		ConversationId: req.ConversationID,
		Prompt:         req.Query,
		FileInfo:       req.FileInfo,
	}, false, constant.AppStatisticSourceDraft); err != nil {
		gin_util.Response(ctx, nil, err)
	}
}

// PublishAgent
//
//	@Tags			openapi
//	@Summary		智能体发布OpenAPI
//	@Description	发布智能体，发布后可通过智能体对话接口进行正式问答
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPIAgentPublishRequest	true	"请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/agent/publish [post]
func PublishAgent(ctx *gin.Context) {
	var req request.OpenAPIAgentPublishRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)
	appID, err := service.GetAssistantIdByUuid(ctx, req.AssistantUUID)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	assistantInfo, err := service.GetAssistantInfo(ctx, userID, orgID, request.AssistantIdRequest{AssistantId: appID}, false)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	if assistantInfo.Prologue == "" {
		gin_util.Response(ctx, nil, fmt.Errorf("开场白未配置，请先通过配置更新接口设置开场白"))
		return
	}
	if assistantInfo.ModelConfig.ModelId == "" {
		gin_util.Response(ctx, nil, fmt.Errorf("大模型未配置，请先通过配置更新接口设置大模型"))
		return
	}
	err = service.PublishApp(ctx, userID, orgID, request.PublishAppRequest{
		AppId:       appID,
		AppType:     constant.AppTypeAgent,
		Version:     req.Version,
		Desc:        req.Desc,
		PublishType: req.PublishType,
	})
	gin_util.Response(ctx, nil, err)
}

// GetAgentInfo
//
//	@Tags			openapi
//	@Summary		获取智能体详情OpenAPI
//	@Description	获取智能体详情，通过 published 参数控制返回草稿态或已发布态配置
//	@Accept			json
//	@Produce		json
//	@Param			uuid		query		string	true	"智能体UUID"
//	@Param			published	query		bool	false	"true=已发布态，false=草稿态（默认）"
//	@Success		200			{object}	response.Response{data=response.Assistant}
//	@Router			/agent/info [get]
func GetAgentInfo(ctx *gin.Context) {
	var req request.OpenAPIGetAgentInfoRequest
	if !gin_util.BindQuery(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)
	appID, err := service.GetAssistantIdByUuid(ctx, req.UUID)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	resp, err := service.GetAssistantInfo(ctx, userID, orgID, request.AssistantIdRequest{AssistantId: appID}, req.Published)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	if resp.ModelConfig.ModelId != "" {
		if uuid, convErr := service.GetModelUuidById(ctx, resp.ModelConfig.ModelId); convErr == nil {
			resp.ModelConfig.ModelId = uuid
		}
	}
	if resp.RerankConfig.ModelId != "" {
		if uuid, convErr := service.GetModelUuidById(ctx, resp.RerankConfig.ModelId); convErr == nil {
			resp.RerankConfig.ModelId = uuid
		}
	}
	if resp.RecommendConfig.ModelConfig.ModelId != "" {
		if uuid, convErr := service.GetModelUuidById(ctx, resp.RecommendConfig.ModelConfig.ModelId); convErr == nil {
			resp.RecommendConfig.ModelConfig.ModelId = uuid
		}
	}
	gin_util.Response(ctx, resp, nil)
}

// UpdateAgentConfig
//
//	@Tags			openapi
//	@Summary		更新智能体配置OpenAPI
//	@Description	更新智能体配置OpenAPI
//	@Accept			json
//	@Produce		json
//	@Param			data	body		request.OpenAPIAgentConfigUpdateRequest	true	"请求参数"
//	@Success		200		{object}	response.Response
//	@Router			/agent/config [put]
func UpdateAgentConfig(ctx *gin.Context) {
	var req request.OpenAPIAgentConfigUpdateRequest
	if !gin_util.Bind(ctx, &req) {
		return
	}
	userID := getUserID(ctx)
	orgID := getOrgID(ctx)

	if req.KnowledgeBaseConfig == nil {
		req.KnowledgeBaseConfig = &request.AppKnowledgebaseConfig{
			Knowledgebases: []request.AppKnowledgeBase{},
			Config: request.AppKnowledgebaseParams{
				MatchType:     "mix",
				PriorityMatch: 1,
				Threshold:     0.4,
				TopK:          5,
			},
		}
	}

	if req.ModelConfig != nil && req.ModelConfig.Config == nil && req.ModelConfig.ModelType == "llm" {
		thinkingEnable := true
		req.ModelConfig.Config = map[string]interface{}{
			"temperature":            0.7,
			"temperatureEnable":      true,
			"topP":                   1,
			"topPEnable":             true,
			"frequencyPenalty":       0,
			"frequencyPenaltyEnable": true,
			"presencePenalty":        0,
			"presencePenaltyEnable":  true,
			"maxTokens":              512,
			"maxTokensEnable":        true,
			"thinkingEnable":         &thinkingEnable,
		}
	}
	if req.RecommendConfig != nil && req.RecommendConfig.ModelConfig.Config == nil && req.RecommendConfig.ModelConfig.ModelType == "llm" {
		thinkingEnable := true
		req.RecommendConfig.ModelConfig.Config = map[string]interface{}{
			"temperature":            0.7,
			"temperatureEnable":      true,
			"topP":                   1,
			"topPEnable":             true,
			"frequencyPenalty":       0,
			"frequencyPenaltyEnable": true,
			"presencePenalty":        0,
			"presencePenaltyEnable":  true,
			"maxTokens":              512,
			"maxTokensEnable":        true,
			"thinkingEnable":         &thinkingEnable,
		}
	}

	assistantID, err := service.GetAssistantIdByUuid(ctx, req.AssistantUUID)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	if req.ModelConfig != nil && req.ModelConfig.ModelId != "" {
		modelID, convErr := service.GetModelIdByUuid(ctx, req.ModelConfig.ModelId)
		if convErr != nil {
			gin_util.Response(ctx, nil, convErr)
			return
		}
		cfg := *req.ModelConfig
		cfg.ModelId = modelID
		req.ModelConfig = &cfg
	}
	if req.RerankConfig != nil && req.RerankConfig.ModelId != "" {
		modelID, convErr := service.GetModelIdByUuid(ctx, req.RerankConfig.ModelId)
		if convErr != nil {
			gin_util.Response(ctx, nil, convErr)
			return
		}
		cfg := *req.RerankConfig
		cfg.ModelId = modelID
		req.RerankConfig = &cfg
	}
	if req.RecommendConfig != nil && req.RecommendConfig.ModelConfig.ModelId != "" {
		modelID, convErr := service.GetModelIdByUuid(ctx, req.RecommendConfig.ModelConfig.ModelId)
		if convErr != nil {
			gin_util.Response(ctx, nil, convErr)
			return
		}
		recCfg := *req.RecommendConfig
		recCfg.ModelConfig.ModelId = modelID
		req.RecommendConfig = &recCfg
	}

	_, err = service.AssistantConfigUpdate(ctx, userID, orgID, request.AssistantConfig{
		AssistantId:         assistantID,
		Prologue:            req.Prologue,
		Instructions:        req.Instructions,
		RecommendQuestion:   req.RecommendQuestion,
		ModelConfig:         req.ModelConfig,
		KnowledgeBaseConfig: req.KnowledgeBaseConfig,
		SafetyConfig:        req.SafetyConfig,
		RerankConfig:        req.RerankConfig,
		VisionConfig:        req.VisionConfig,
		MemoryConfig:        req.MemoryConfig,
		RecommendConfig:     req.RecommendConfig,
	})
	gin_util.Response(ctx, nil, err)
}

// --- internal ---

// 获取当前用户ID
func getUserID(ctx *gin.Context) string {
	return ctx.GetString(gin_util.USER_ID)
}

// 获取当前组织ID
func getOrgID(ctx *gin.Context) string {
	return ctx.GetString(gin_util.X_ORG_ID)
}

// 获取当前appID
func getAppID(ctx *gin.Context) string {
	return ctx.GetString(gin_util.APP_ID)
}
