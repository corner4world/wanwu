package agent_chat_builder

import (
	"encoding/json"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/cloudwego/eino/schema"
)

type AgentStep int

const (
	AgentNoneProcessStep AgentStep = 0 //无需处理，过滤
	AgentStartStep       AgentStep = 1 //智能体开始
	AgentChatStep        AgentStep = 2 //智能体会话
	AgentStopStep        AgentStep = 3 //智能体结束
	AgentAllFinishStep   AgentStep = 4 //智能体全完成，透传内容
)

type MultiAgentMessageBuilder struct {
}

func NewMultiBuilder() *MultiAgentMessageBuilder {
	return &MultiAgentMessageBuilder{}
}
func (*MultiAgentMessageBuilder) MessageType() MessageType {
	return MultiAgentMessage
}
func (*MultiAgentMessageBuilder) FilterMessage(respContext *response.AgentChatRespContext, chatMessage *schema.Message) bool {
	if filterMessage(respContext, chatMessage) {
		return true
	}
	multiAgentStep := response.CreateMultiAgentStep(respContext, chatMessage)
	//智能体切换完成消息|切换回主智能体开始消息|supervisor 结束时会以exit结束（设置enio时传入），模型流式输出exit工具参数时过滤消息
	if multiAgentStep.TransferFinish() || multiAgentStep.MainTransferStart() || multiAgentStep.ExitStart() {
		return true
	}
	return false
}
func (*MultiAgentMessageBuilder) BuildContent(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message, changeStyle *bool) ([]*response.AgentMessageContent, error) {
	step := buildAgentStep(req, chatMessage, respContext)
	marshal, _ := json.Marshal(chatMessage)
	log.Infof("step:%d, messageL %s", step, string(marshal))
	switch step {
	case AgentNoneProcessStep: //无需处理
		return buildSkipMessage(), nil
	case AgentAllFinishStep: //直接返回内容
		return buildMessageContent([]string{chatMessage.Content}, nil), nil
	case AgentChatStep: //智能体内容输出
		return buildChatMessage(req, respContext, chatMessage, buildSubAgentEvent(respContext, step), changeStyle)
	default: //智能体开始/结束
		if len(chatMessage.Content) > 0 {
			if req.AgentChatReq.NewStyle {
				respContext.IncreaseOrder()
			}
			return buildMessageContent([]string{chatMessage.Content}, buildSubAgentEvent(respContext, step)), nil
		}
		return buildMessageContent(nil, buildSubAgentEvent(respContext, step)), nil
	}
}

// buildAgentStep 构建智能体步骤
func buildAgentStep(req *request.AgentChatContext, chatMessage *schema.Message, respContext *response.AgentChatRespContext) AgentStep {
	multiAgentStep := response.CreateMultiAgentStep(respContext, chatMessage)
	//智能体切换消息
	if multiAgentStep.TransferStart() {
		agentTransferStart(respContext, chatMessage)
	}
	//智能体切换中处理智能体名称
	if respContext.MultiAgentContext.AgentChangeStart {
		skip, agentStep := buildAgentNameAndStep(req, chatMessage, respContext)
		if !skip {
			return agentStep
		}
	}
	//子智能体结束
	if multiAgentStep.SubAgentFinish() {
		return AgentStopStep
	}
	//主智能体结束
	if multiAgentStep.ExitFinish() {
		return AgentAllFinishStep
	}
	return AgentChatStep
}

// buildChatMessage 构造智能体对话消息
func buildChatMessage(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message, event *response.SubEventData, changeStyle *bool) ([]*response.AgentMessageContent, error) {
	contentList, err := NewSingleBuilder().BuildContent(req, respContext, chatMessage, changeStyle)
	if err != nil {
		return nil, err
	}
	for _, messageContent := range contentList {
		if event == nil {
			continue
		}
		if messageContent.SubEventData == nil {
			messageContent.SubEventData = event
		} else {
			messageContent.SubEventData.ParentId = event.Id
		}
		newStyle := buildChangeStyle(changeStyle, req)
		if !newStyle {
			messageContent.SubEventData.EventType = buildMultiAgentEventType(messageContent.SubEventData.EventType)
		}

		//messageContent.SubEventData = event
		//子智能体的结束消息，不需要输出stop
		if event.Status == response.EventEndStatus {
			messageContent.NotStop = true
		}
	}

	return contentList, nil
}

// buildMultiAgentEventType 构建多智能体事件类型
func buildMultiAgentEventType(eventType int) int {
	switch eventType {
	case response.ToolEventType:
		return response.SubAgentEventType
	default:
		return response.SubAgentEventType
	}
}

// agentTransferStart 智能体切换开始
func agentTransferStart(respContext *response.AgentChatRespContext, chatMessage *schema.Message) {
	respContext.MultiAgentContext.ChangeStart()
	respContext.ResetTool()
	respContext.AgentToolContext.DefaultToolId(chatMessage.ToolCalls[0].ID)
}

func buildSubAgentEvent(respContext *response.AgentChatRespContext, step AgentStep) *response.SubEventData {
	switch step {
	case AgentStartStep:
		//每切换一次智能体order + 1
		respContext.IncreaseOrder()
		respContext.MultiAgentContext.AgentOrder(respContext.Order)
		return response.BuildStartSubAgent(respContext)
	case AgentChatStep:
		return response.BuildProcessSubAgent(respContext)
	case AgentStopStep:
		subAgent := response.BuildEndSubAgent(respContext, util.NowSpanToHMS(respContext.MultiAgentContext.AgentStartTime))
		respContext.MultiAgentContext.ClearAgent()
		return subAgent
	}
	return nil
}

// buildAgentNameAndStep 构建智能体名称和步骤
func buildAgentNameAndStep(req *request.AgentChatContext, chatMessage *schema.Message, respContext *response.AgentChatRespContext) (bool, AgentStep) {
	stepsMap, toolIdList := buildToolStep(chatMessage, respContext)
	if len(toolIdList) > 0 {
		//根据step循环构造输出的内容
		for _, toolId := range toolIdList {
			toolSteps := stepsMap[toolId]
			for _, step := range toolSteps {
				finish := writeAgentName(req, chatMessage, step, respContext)
				if finish {
					return false, AgentStartStep
				}
			}
		}
		return false, AgentNoneProcessStep
	}
	//走到这里可能有bad case了
	return true, AgentNoneProcessStep
}

// buildAgentStep 根据当前步骤构造需要输出的内容,构造<tool></tool>数据以及markdown格式
func writeAgentName(req *request.AgentChatContext, chatMessage *schema.Message, step response.ToolStep, respContext *response.AgentChatRespContext) bool {
	var finish bool
	switch step {
	case response.ToolParamStep:
		respContext.MultiAgentContext.WriteAgentName(chatMessage)
	case response.ToolParamFinishStep:
		respContext.MultiAgentContext.CreateAgent(req.SubAgentMap)
		//智能体参数输出完成
		respContext.MultiAgentContext.ChangeFinish()
		finish = true
	}
	return finish
}
