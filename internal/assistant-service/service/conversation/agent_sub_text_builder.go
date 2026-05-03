package conversation

var agentSkillText = &AgentSkillText{}

type AgentSkillText struct {
}

func init() {
	InitBuilder(agentSkillText)
}

func (*AgentSkillText) EventType() int {
	return SubTextEventType
}
func (*AgentSkillText) Build(conversationResp *ConversationResp, conversation, searchResult string, agentChatResp *AgentChatResp) error {
	eventData := agentChatResp.EventData
	if eventData == nil {
		return nil
	}
	resp := conversationResp.ConversationEventMap[eventData.Id]
	if resp == nil {
		resp = CreateConversationResp()
		resp.Order = eventData.Order
		resp.EventType = SubTextEventType
		conversationResp.ConversationEventMap[eventData.Id] = resp
	}
	if len(conversation) > 0 {
		//保存对话
		resp.Write(conversation, eventData.Order)
	}
	//终态存储
	//if eventData.Status == model.EventEndStatus || eventData.Status == model.EventFailStatus {
	resp.EventData = eventData
	//}
	return nil
}
