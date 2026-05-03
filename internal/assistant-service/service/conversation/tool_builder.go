package conversation

import "github.com/UnicomAI/wanwu/internal/assistant-service/client/model"

var tool = &Tool{}

type Tool struct {
}

func init() {
	InitBuilder(tool)
}

func (*Tool) EventType() int {
	return ToolEventType
}
func (*Tool) Build(conversationResp *ConversationResp, conversation, searchResult string, agentChatResp *AgentChatResp) error {
	eventData := agentChatResp.EventData
	if eventData == nil {
		return nil
	}
	resp := conversationResp.ConversationEventMap[eventData.Id]
	if resp == nil {
		resp = CreateConversationResp()
		resp.Order = eventData.Order
		resp.EventType = ToolEventType
		conversationResp.ConversationEventMap[eventData.Id] = resp
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
