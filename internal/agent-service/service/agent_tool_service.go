package service

import (
	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	service_model "github.com/UnicomAI/wanwu/internal/agent-service/service/service-model"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/compose"
	"github.com/gin-gonic/gin"
)

// BuildAgentToolsConfig 构建智能体工具配置
func BuildAgentToolsConfig(ctx *gin.Context, req *request.AgentChatParams, chatInfo *service_model.AgentChatInfo) (adk.ToolsConfig, error) {
	params := req.ToolParams
	//无工具调用
	if params == nil {
		return adk.ToolsConfig{
			ToolsNodeConfig: compose.ToolsNodeConfig{},
		}, nil
	}
	//mcp 工具
	var toolList []tool.BaseTool
	mcpToolList, _ := GetToolsFromMCPServers(ctx, req.ToolParams.McpToolList)
	if len(mcpToolList) > 0 {
		toolList = append(toolList, mcpToolList...)
	}
	//plugin 工具
	pluginToolList, _ := GetToolsFromOpenAPISchema(ctx, req.ToolParams.PluginToolList)
	if len(pluginToolList) > 0 {
		toolList = append(toolList, pluginToolList...)
	}
	//skill 工具
	skillToolList, _ := GetToolsFromSkills(ctx, req.ToolParams.SkillToolList, req.Input, req.AgentBaseParams.Name, req.UploadFile, chatInfo)
	if len(skillToolList) > 0 {
		toolList = append(toolList, skillToolList...)
	}
	return adk.ToolsConfig{
		ToolsNodeConfig: compose.ToolsNodeConfig{
			Tools: toolList,
		},
	}, nil
}
