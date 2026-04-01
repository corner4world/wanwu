package v1

import (
	"net/http"

	v1 "github.com/UnicomAI/wanwu/internal/bff-service/server/http/handler/v1"
	mid "github.com/UnicomAI/wanwu/pkg/gin-util/mid-wrap"
	"github.com/gin-gonic/gin"
)

func registerWGA(apiV1 *gin.RouterGroup) {

	// 通用智能体工具相关接口
	mid.Sub("wga").Reg(apiV1, "/general/agent/assistant/select", http.MethodGet, v1.GetGeneralAgentAssistantSelect, "通用智能体下拉列表接口")
	mid.Sub("wga").Reg(apiV1, "/general/agent/tool/select", http.MethodGet, v1.GetGeneralAgentToolSelect, "通用智能体工具下拉列表接口")
	mid.Sub("wga").Reg(apiV1, "/general/agent/tool/info", http.MethodGet, v1.GetGeneralAgentToolInfo, "通用智能体工具详情")
	mid.Sub("wga").Reg(apiV1, "/general/agent/config", http.MethodPut, v1.UpdateGeneralAgentConfig, "修改通用智能体配置")
	mid.Sub("wga").Reg(apiV1, "/general/agent/config", http.MethodGet, v1.GetGeneralAgentConfig, "通用智能体配置")

	// 通用智能体配置相关接口
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation", http.MethodPost, v1.CreateGeneralAgentConversation, "创建通用智能体对话")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation", http.MethodDelete, v1.DeleteGeneralAgentConversation, "删除通用智能体对话")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/list", http.MethodGet, v1.GetGeneralAgentConversationList, "通用智能体对话列表")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/detail", http.MethodGet, v1.GetGeneralAgentConversationDetail, "通用智能体对话详情")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/config", http.MethodGet, v1.GetGeneralAgentConversationConfig, "通用智能体对话配置")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/config", http.MethodPut, v1.UpdateGeneralAgentConversationConfig, "修改通用智能体对话配置")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/config/check", http.MethodPost, v1.CheckGeneralAgentConfig, "通用智能体配置检查")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/workspace/download", http.MethodGet, v1.GeneralAgentWorkspaceDownload, "通用智能体workspace下载")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/workspace/preview", http.MethodGet, v1.GeneralAgentWorkspacePreview, "通用智能体workspace预览")
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/workspace", http.MethodGet, v1.GeneralAgentWorkspaceInfo, "通用智能体workspace目录树")

	// 通用智能体对话相关接口
	mid.Sub("wga").Reg(apiV1, "/general/agent/conversation/chat", http.MethodPost, v1.GeneralAgentConversationChat, "通用智能体流式问答")
	mid.Sub("wga").Reg(apiV1, "/general/agent/copilotkit", http.MethodPost, v1.GeneralAgentCopilotRuntime, "CopilotRuntime协议端点")

}
