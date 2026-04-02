package agent_preprocessor

import (
	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/response"
	agent_message_processor "github.com/UnicomAI/wanwu/internal/agent-service/service/agent-message-processor"
	local_agent "github.com/UnicomAI/wanwu/internal/agent-service/service/local-agent"
	service_model "github.com/UnicomAI/wanwu/internal/agent-service/service/service-model"
	"github.com/UnicomAI/wanwu/pkg/log"
	safe_go_util "github.com/UnicomAI/wanwu/pkg/safe-go-util"
	"github.com/cloudwego/eino/adk"
	"github.com/gin-gonic/gin"
)

type AgentPreprocess struct {
	LocalAgentService local_agent.LocalAgentService
	AgentChatInfo     *service_model.AgentChatInfo
	CallDetail        bool
	GinContext        *gin.Context
}

// AgentPreProcess 预处理
func AgentPreProcess(agentPreprocess *AgentPreprocess, agentInput *adk.AgentInput, req *request.AgentChatParams) (*adk.AgentInput, *response.AgentChatRespContext) {
	ctx := agentPreprocess.GinContext
	iter, generator := adk.NewAsyncIteratorPair[*adk.AgentEvent]()
	safe_go_util.SafeGo(func() {
		defer func() {
			generator.Close()
		}()
		var err error
		agentInput, err = agentPreprocess.LocalAgentService.BuildAgentInput(ctx, req, agentPreprocess.AgentChatInfo, agentInput, generator)
		if err != nil {
			log.Errorf("failed to build agent input: %v", err)
			generator.Send(&adk.AgentEvent{Err: err})
		}
	})
	//此处阻塞读iter
	chatRespContext, _ := agent_message_processor.AgentMessage(ctx, iter, &request.AgentChatContext{AgentChatReq: req})
	return agentInput, chatRespContext
}
