package conversation

import (
	"github.com/UnicomAI/wanwu/internal/assistant-service/client/model"
	"strconv"
)

var agentSkill = &AgentSkill{}

type AgentSkill struct {
}

func init() {
	InitBuilder(agentSkill)
}

func (*AgentSkill) EventType() int {
	return SkillEventType
}
func (*AgentSkill) Build(conversationResp *ConversationResp, conversation, searchResult string, agentChatResp *AgentChatResp) error {
	eventData := agentChatResp.EventData
	if eventData == nil {
		return nil
	}
	resp := conversationResp.ConversationEventMap[eventData.Id]
	eventId := eventData.Id + "_" + strconv.Itoa(agentChatResp.Order)
	textResp := conversationResp.ConversationEventMap[eventId]
	if resp == nil {
		resp = CreateConversationResp()
		resp.Order = eventData.Order
		resp.EventType = SkillEventType
		conversationResp.ConversationEventMap[eventData.Id] = resp
	} else if resp.Order != agentChatResp.Order {
		if textResp == nil {
			textResp = CreateConversationResp()
			textResp.Order = agentChatResp.Order
			textResp.EventType = SkillTextEventType
			conversationResp.ConversationEventMap[eventId] = textResp
		}
		if len(conversation) > 0 {
			//保存对话
			textResp.Write(conversation, agentChatResp.Order)
		}
	}

	//终态存储
	if eventData.Status == model.EventEndStatus || eventData.Status == model.EventFailStatus {
		resp.EventData = eventData
		if textResp != nil {
			textResp.EventData = eventData.Copy()
			textResp.EventData.ParentId = eventData.Id
			textResp.EventData.Id = eventId
		}
	}
	return nil
}
