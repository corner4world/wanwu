package agent_chat_builder

import (
	"encoding/json"
	"strconv"
	"strings"

	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/response"
	agent_util "github.com/UnicomAI/wanwu/internal/agent-service/pkg/util"
	"github.com/cloudwego/eino/schema"
)

type SkillMessageBuilder struct {
	RespContext *response.AgentChatRespContext
	ChatMessage *schema.Message
}

func NewSkillBuilder() *SkillMessageBuilder {
	return &SkillMessageBuilder{}
}

func (*SkillMessageBuilder) MessageType() MessageType {
	return SingleAgentMessage
}

func (*SkillMessageBuilder) FilterMessage(respContext *response.AgentChatRespContext, chatMessage *schema.Message) bool {
	skillChatStep := response.CreateSkillChatStep(respContext, chatMessage)
	return !skillChatStep.SkillNeedProcess()
}

func (*SkillMessageBuilder) BuildContent(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message) ([]*response.AgentMessageContent, error) {
	skillChatStep := response.CreateSkillChatStep(respContext, chatMessage)
	//到这里说明都是需要处理的skill消息
	step := skillChatStep.BuildSkillStep()
	message := buildMessageByStep(step, respContext, chatMessage)
	if message == nil {
		return make([]*response.AgentMessageContent, 0), nil
	}
	content, err := NewMultiBuilder().BuildContent(req, respContext, message)
	buildSkillEvent(content, respContext.Order)
	return content, err
}

func buildMessageByStep(step response.SkillStep, respContext *response.AgentChatRespContext, chatMessage *schema.Message) *schema.Message {
	switch step {
	case response.SkillTransferStartStep:
		toolCall := buildToolCall(chatMessage.ToolCalls[0])
		message := agent_util.BuildToolParamsMessage(toolCall)
		respContext.SkillChatContext.SkillTransferStart(toolCall[0].ID)
		return message
	case response.SkillTransferringStep:
		return nil
	case response.SkillTransferFinishStep:
		respContext.SkillChatContext.SkillTransferFinish()
		return nil
	case response.SkillResultStep:
		respContext.SkillChatContext.SkillTransferFinish()
		return rebuildSkillMessage(respContext, chatMessage)
	}
	return chatMessage
}

func rebuildSkillMessage(respContext *response.AgentChatRespContext, chatMessage *schema.Message) *schema.Message {
	message := &schema.Message{}
	err := json.Unmarshal([]byte(chatMessage.Content), message)
	if err == nil {
		//if len(message.ToolCalls) > 0 {
		//	message.Role = schema.Tool //目前openCode 不会返回结束类型所以通过这里处理
		//	message.ResponseMeta = &schema.ResponseMeta{
		//		FinishReason: "tool_calls",
		//	}
		//}
		if len(message.Extra) > 0 {
			fileListData := message.Extra["fileList"]
			if fileListData != nil {
				fileList := response.ParseDownloadFileInfoList(fileListData)
				if len(fileList) > 0 {
					respContext.DownloadContext.AddDownloadFile(respContext.SkillChatContext.SkillId, fileList)
				}
			}
		}
		if len(message.Content) > 0 {
			message.Content = strings.ReplaceAll(message.Content, "-ReplaceLocalFile", "")
		}
		return message
	}
	return chatMessage
}

func buildToolCall(call schema.ToolCall) []schema.ToolCall {
	info := response.MultiAgentNameInfo{AgentName: call.Function.Name}
	marshal, _ := json.Marshal(&info)
	return []schema.ToolCall{
		{
			ID:   call.ID,
			Type: "function",
			Function: schema.FunctionCall{
				Arguments: string(marshal),
				Name:      agent_util.AgentStartLabel,
			},
		},
	}
}

func buildSkillEvent(contentList []*response.AgentMessageContent, order int) {
	if len(contentList) > 0 {
		for _, content := range contentList {
			event := content.SubEventData
			if event != nil {
				if event.EventType == response.SubAgentEventType || event.EventType == response.MainAgentEventType {
					if event.Status == response.EventProcessStatus {
						event.EventType = response.SkillTextEventType
						event.ParentId = event.Id
						event.Id = event.Id + "_" + strconv.Itoa(order)
					} else {
						event.EventType = response.SkillEventType
					}
				}
			}
		}
	}
}
