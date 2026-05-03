package conversation

import "github.com/UnicomAI/wanwu/internal/assistant-service/client/model"

var subAgentTool = &SubAgentTool{}

type SubAgentTool struct {
}

func init() {
	InitBuilder(subAgentTool)
}

func (*SubAgentTool) EventType() int {
	return SubAgentToolEventType
}
func (*SubAgentTool) Build(conversationResp *ConversationResp, conversation, searchResult string, agentChatResp *AgentChatResp) error {
	eventData := agentChatResp.EventData
	if eventData == nil {
		return nil
	}
	parentEvent := conversationResp.ConversationEventMap[eventData.ParentId]
	if parentEvent == nil {
		return nil

	}
	resp := parentEvent.ConversationEventMap[eventData.Id]
	if resp == nil {
		resp = CreateConversationResp()
		resp.Order = eventData.Order
		parentEvent.ConversationEventMap[eventData.Id] = resp
	}

	if len(conversation) > 0 {
		//保存对话
		resp.Write(conversation, eventData.Order)
	}
	//如果已经输出过结束态则不再输出
	if resp.EventData == nil || resp.EventData.Status != model.EventEndStatus {
		resp.EventData = eventData
	}
	return nil
}
