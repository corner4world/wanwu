package response

import (
	"time"

	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

type ThinkStep int

const (
	ThinkNone   ThinkStep = 0
	ThinkStart  ThinkStep = 1
	Thinking    ThinkStep = 2
	ThinkFinish ThinkStep = 3
)

type ThinkChatContext struct {
	Thinking     bool       // 思考中
	ThinkingTool *AgentTool // 思考工具
}

func NewThinkChatContext() *ThinkChatContext {
	return &ThinkChatContext{}
}

func (t *ThinkChatContext) ThinkContextPrepare(thinkStep ThinkStep, respContext *AgentChatRespContext) {
	switch thinkStep {
	case ThinkStart:
		t.Thinking = true
		respContext.ReplaceContent.Reset()
		respContext.IncreaseOrder()
		t.ThinkingTool = &AgentTool{
			Order:     respContext.Order,
			ToolId:    uuid.New().String(),
			ToolName:  "智能体思考",
			ToolType:  ThinkingEventType,
			Avatar:    BuildDefaultAvatarByType(ThinkingEventType),
			StartTime: time.Now().UnixMilli(),
		}
	case ThinkFinish:
		t.Thinking = false
		respContext.IncreaseOrder()
	}
}

func (t *ThinkChatContext) ThinkMessage(chatMessage *schema.Message, respContext *AgentChatRespContext) []*AgentMessageContent {
	thinkStep := t.ThinkStep(chatMessage)
	return t.ThinkMessageByStep(chatMessage, thinkStep, respContext)
}

func (t *ThinkChatContext) ThinkMessageByStep(chatMessage *schema.Message, thinkStep ThinkStep, respContext *AgentChatRespContext) []*AgentMessageContent {
	t.ThinkContextPrepare(thinkStep, respContext)
	return t.buildContentByStep(chatMessage, thinkStep)
}

func (t *ThinkChatContext) ThinkStep(chatMessage *schema.Message) ThinkStep {
	if len(chatMessage.ReasoningContent) > 0 {
		if !t.Thinking {
			return ThinkStart
		} else {
			return Thinking
		}
	} else if len(chatMessage.ReasoningContent) == 0 && t.Thinking {
		return ThinkFinish
	}
	return ThinkNone
}

func (t *ThinkChatContext) buildContentByStep(chatMessage *schema.Message, step ThinkStep) []*AgentMessageContent {
	return t.buildNewReasoningContent(chatMessage, step)
}

func (t *ThinkChatContext) buildNewReasoningContent(chatMessage *schema.Message, step ThinkStep) []*AgentMessageContent {
	var retContentList []*AgentMessageContent
	switch step {
	case ThinkStart:
		retContentList = append(retContentList, &AgentMessageContent{
			SubEventData: BuildStartTool(t.ThinkingTool),
			ContentList:  []string{chatMessage.ReasoningContent},
		})
	case Thinking:
		retContentList = append(retContentList, &AgentMessageContent{
			SubEventData: BuildProcessTool(t.ThinkingTool),
			ContentList:  []string{chatMessage.ReasoningContent},
		})
	case ThinkFinish:
		retContentList = append(retContentList, &AgentMessageContent{
			SubEventData: BuildEndTool(t.ThinkingTool),
		})
	}
	return retContentList
}
