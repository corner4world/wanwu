package openapi

import (
	"net/http"

	"github.com/UnicomAI/wanwu/internal/bff-service/server/http/handler/openapi"
	"github.com/UnicomAI/wanwu/internal/bff-service/server/http/middleware"
	"github.com/UnicomAI/wanwu/pkg/constant"
	mid "github.com/UnicomAI/wanwu/pkg/gin-util/mid-wrap"
	"github.com/gin-gonic/gin"
)

func Register(openAPI *gin.RouterGroup) {
	// agent
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent", http.MethodPost, openapi.CreateAgent, "创建智能体OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent/publish", http.MethodPost, openapi.PublishAgent, "发布智能体OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent/info", http.MethodGet, openapi.GetAgentInfo, "获取智能体详情OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent/config", http.MethodPut, openapi.UpdateAgentConfig, "更新智能体配置OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthModelByUuid([]string{"modelConfig.modelId", "rerankConfig.modelId", "recommendConfig.modelConfig.modelId"}))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent/conversation", http.MethodPost, openapi.CreateAgentConversation, "智能体创建对话OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent/chat", http.MethodPost, openapi.ChatAgent, "智能体问答OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordFromReq))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/agent/chat/draft", http.MethodPost, openapi.DraftChatAgent, "智能体草稿态对话OpenAPI", constant.OpenAPITypeAgent, middleware.AuthOpenAPIKey(constant.OpenAPITypeAgent), middleware.APIKeyRecord(middleware.RecordFromReq))
	// rag
	mid.Sub("openapi").RegWithAPIType(openAPI, "/rag/chat", http.MethodPost, openapi.ChatRag, "文本问答OpenAPI", constant.OpenAPITypeRag, middleware.AuthOpenAPIKey(constant.OpenAPITypeRag), middleware.APIKeyRecord(middleware.RecordFromReq))
	// workflow
	mid.Sub("openapi").RegWithAPIType(openAPI, "/workflow/run", http.MethodPost, openapi.WorkflowRun, "工作流OpenAPI", constant.OpenAPITypeWorkflow, middleware.AuthOpenAPIKey(constant.OpenAPITypeWorkflow), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/workflow/file/upload", http.MethodPost, openapi.WorkflowFileUpload, "工作流OpenAPI文件上传", constant.OpenAPITypeChatflow, middleware.AuthOpenAPIKey(constant.OpenAPITypeChatflow), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	// chatflow
	mid.Sub("openapi").RegWithAPIType(openAPI, "/chatflow/conversation", http.MethodPost, openapi.CreateChatflowConversation, "对话流创建对话OpenAPI", constant.OpenAPITypeChatflow, middleware.AuthOpenAPIKey(constant.OpenAPITypeChatflow), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/chatflow/conversation/message/list", http.MethodPost, openapi.GetConversationMessageList, "对话流根据conversationId获取历史对话", constant.OpenAPITypeChatflow, middleware.AuthOpenAPIKey(constant.OpenAPITypeChatflow), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/chatflow/chat", http.MethodPost, openapi.ChatflowChat, "对话流OpenAPI", constant.OpenAPITypeChatflow, middleware.AuthOpenAPIKey(constant.OpenAPITypeChatflow), middleware.APIKeyRecord(middleware.RecordStreamType))
	// knowledge
	mid.Sub("openapi").RegWithAPIType(openAPI, "/file/upload/direct", http.MethodPost, openapi.DirectUploadFiles, "直接上传文件", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge", http.MethodPost, openapi.CreateKnowledge, "新建知识库", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthModelByUuid([]string{"embeddingModelInfo.modelId", "knowledgeGraph.llmModelId"}))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge", http.MethodPut, openapi.UpdateKnowledge, "更新知识库", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeSystem))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge", http.MethodDelete, openapi.DeleteKnowledge, "删除知识库", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeSystem))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/select", http.MethodPost, openapi.GetKnowledgeSelect, "查询知识库列表", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc/config", http.MethodGet, openapi.GetDocConfig, "获取文档配置信息", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeView))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc/list", http.MethodPost, openapi.GetDocList, "获取文档列表", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeView))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc/import", http.MethodPost, openapi.ImportDoc, "上传文档", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeEdit))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc/update/config", http.MethodPost, openapi.UpdateDocConfig, "更新文档配置", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeEdit))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc/import/tip", http.MethodGet, openapi.GetDocImportTip, "获取知识库文档上传状态", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeView))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc/export", http.MethodPost, openapi.ExportKnowledgeDoc, "知识库文档导出", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeEdit))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/export/record/list", http.MethodGet, openapi.GetKnowledgeExportRecordList, "获取知识库导出记录列表", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeView))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/export/record", http.MethodDelete, openapi.DeleteKnowledgeExportRecord, "删除知识库库导出记录", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeEdit))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/doc", http.MethodDelete, openapi.DeleteDoc, "删除文档", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthKnowledge("knowledgeId", middleware.KnowledgeEdit))
	mid.Sub("openapi").RegWithAPIType(openAPI, "/knowledge/hit", http.MethodPost, openapi.KnowledgeHit, "知识库命中测试", constant.OpenAPITypeKnowledge, middleware.AuthOpenAPIKey(constant.OpenAPITypeKnowledge), middleware.APIKeyRecord(middleware.RecordNonStreamType), middleware.AuthModelByUuid([]string{"knowledgeMatchParams.rerankModelId"}))
	// mcp server
	mid.Sub("openapi").Reg(openAPI, "/mcp/server/sse", http.MethodGet, openapi.GetMCPServerSSE, "新建MCP服务sse连接", middleware.AuthAppKeyByQuery(constant.AppTypeMCPServer))
	mid.Sub("openapi").Reg(openAPI, "/mcp/server/message", http.MethodPost, openapi.GetMCPServerMessage, "获取MCP服务sse消息", middleware.AuthAppKeyByQuery(constant.AppTypeMCPServer))
	mid.Sub("openapi").Reg(openAPI, "/mcp/server/streamable", http.MethodGet, openapi.GetMCPServerStreamable, "获取MCP服务streamable消息(GET)", middleware.AuthAppKeyByQuery(constant.AppTypeMCPServer))
	mid.Sub("openapi").Reg(openAPI, "/mcp/server/streamable", http.MethodPost, openapi.GetMCPServerStreamable, "获取MCP服务streamable消息(POST)", middleware.AuthAppKeyByQuery(constant.AppTypeMCPServer))

	// oauth
	mid.Sub("openapi").Reg(openAPI, "/oauth/jwks", http.MethodGet, openapi.OAuthJWKS, "JWT公钥")
	mid.Sub("openapi").Reg(openAPI, "/oauth/login", http.MethodGet, openapi.OAuthLogin, "OAuth登录授权")
	mid.Sub("openapi").Reg(openAPI, "/oauth/code/authorize", http.MethodGet, openapi.OAuthAuthorize, "获取授权码")
	mid.Sub("openapi").Reg(openAPI, "/oauth/code/token", http.MethodPost, openapi.OAuthToken, "授权码获取token")
	mid.Sub("openapi").Reg(openAPI, "/oauth/code/token/refresh", http.MethodPost, openapi.OAuthRefresh, "刷新Access Token")
	mid.Sub("openapi").Reg(openAPI, "/.well-known/openid-configuration", http.MethodGet, openapi.OAuthConfig, "返回Endpoint配置")
	// oauth user
	mid.Sub("openapi").Reg(openAPI, "/oauth/userinfo", http.MethodGet, openapi.OAuthGetUserInfo, "OAuth获取用户信息", middleware.JWTOAuthAccess)
}
