package agent_chat_builder

import (
	"encoding/json"
	"fmt"
	"time"
	"unicode/utf8"

	"github.com/UnicomAI/wanwu/internal/agent-service/model"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/response"
	"github.com/cloudwego/eino/schema"
)

const (
	toolStartTitle        = `<tool>`
	toolStartTitleFormat  = `工具名：%s`
	toolParamsStartFormat = "\n\n```工具参数：\n"
	toolParamsEndFormat   = "\n```\n\n"
	toolEndFormat         = "\n\n```工具%s调用结果：\n %s \n```\n\n"
	toolEndTitle          = `</tool>`
)

type ToolMessageContent struct {
	Content      []string
	SubEventData *response.SubEventData
}

func (t ToolMessageContent) Empty() bool {
	return len(t.Content) == 0 && t.SubEventData == nil
}

type SingleAgentMessageBuilder struct {
}

func NewSingleBuilder() *SingleAgentMessageBuilder {
	return &SingleAgentMessageBuilder{}
}

func (*SingleAgentMessageBuilder) MessageType() MessageType {
	return SingleAgentMessage
}

func (*SingleAgentMessageBuilder) FilterMessage(respContext *response.AgentChatRespContext, chatMessage *schema.Message) bool {
	return filterMessage(respContext, chatMessage)
}

func (*SingleAgentMessageBuilder) BuildContent(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message, changeStyle *bool) ([]*response.AgentMessageContent, error) {
	return buildSingleAgentContent(req, respContext, chatMessage, changeStyle)
}

// buildSingleAgentContent
// 最极限情况，多智能体->单智能体->skill->多智能体->单智能体
func buildSingleAgentContent(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message, changeStyle *bool) ([]*response.AgentMessageContent, error) {
	var retContentList []*response.AgentMessageContent
	//构造思考内容
	thinkMessage := respContext.ThinkChatContext.ThinkMessage(buildChangeStyle(changeStyle, req), chatMessage, respContext)
	if len(thinkMessage) > 0 {
		retContentList = append(retContentList, thinkMessage...)
		if len(chatMessage.ReasoningContent) != 0 { //如果只有思考内容，则直接返回，说明是思考过程
			return retContentList, nil
		}
	}

	//技能消息
	skillMessage, err, directReturn := buildSkillMessage(req, respContext, chatMessage, changeStyle)
	if err != nil {
		return nil, err
	}
	if len(skillMessage) > 0 {
		retContentList = append(retContentList, skillMessage...)
	}
	//直接返回
	if directReturn {
		return retContentList, nil
	}

	req.AgentChatReq.NewStyle = buildChangeStyle(changeStyle, req)

	content := buildCommonSingleAgentContent(req, respContext, chatMessage)
	if len(content) > 0 {
		retContentList = append(retContentList, content...)
	}
	return retContentList, nil
}

func buildChangeStyle(changeStyle *bool, req *request.AgentChatContext) bool {
	//正常消息
	if changeStyle != nil { //新老版本兼容出现的情况，如果都切到新版本就没这种问题
		return *changeStyle
	} else {
		return req.AgentChatReq.OriginNewStyle
	}
}

func buildSkillMessage(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message, changeStyle *bool) ([]*response.AgentMessageContent, error, bool) {
	skillBuilder := NewSkillBuilder()
	if !skillBuilder.FilterMessage(respContext, chatMessage) { //是skill消息
		content, err := skillBuilder.BuildContent(req, respContext, chatMessage, changeStyle)
		return content, err, true

	}
	return nil, nil, false
}

func buildCommonSingleAgentContent(req *request.AgentChatContext, respContext *response.AgentChatRespContext, chatMessage *schema.Message) []*response.AgentMessageContent {
	stepsMap, toolIdList := buildToolStep(chatMessage, respContext)
	if len(stepsMap) == 0 { //没有工具处理
		if !respContext.ContentOutput {
			respContext.ContentOutput = true
			respContext.IncreaseOrder()
			respContext.ReplaceContent.Reset()
		}
		return buildNoToolContent(chatMessage, respContext, req.AgentChatReq.NewStyle)
	}
	if req.AgentChatReq.NewStyle { //新样式，工作流智能体前端处理完成后才能都切到新的样式
		return buildToolContentNewStyle(req, chatMessage, respContext, stepsMap, toolIdList)
	}
	return buildToolContent(chatMessage, respContext, stepsMap, toolIdList)
}

/*
*
目前工具调用有几种情况做处理
1.正常流式：先输出方法名，在流式分别输出方法对应的参数，再输出调用结果
2.并发流式：如果需要调用同一方法两次，先输出方法名，方法参数，再输出方法名方法参数，再输出结果1，再输出结果2
3.同步请求：请求一个事件，返回一个事件，没有流式
4.同步请求和返回：请求和返回都在同一个事件，没有流式
*/
func buildToolStep(chatMessage *schema.Message, respContext *response.AgentChatRespContext) (map[string][]response.ToolStep, []string) {
	messageTool := response.CreateMessageTool(chatMessage, respContext)
	var toolStepMap = make(map[string][]response.ToolStep)
	//构造toolId
	var toolId = messageTool.ToolId()

	var toolIdList []string
	if messageTool.ToolStart() {
		for _, tool := range chatMessage.ToolCalls {
			newTool := messageTool.NewTool(tool)
			if newTool { //新工具开始
				toolId = tool.ID
			}
			steps := toolStepMap[toolId]
			if len(tool.Function.Name) > 0 {
				steps = append(steps, response.ToolNameStep)
				if newTool {
					steps = append(steps, response.ToolParamStartStep)
				}
			}

			if len(tool.Function.Arguments) > 0 {
				steps = append(steps, response.ToolParamStep)
			}
			if messageTool.ToolParamsEnd() {
				steps = append(steps, response.ToolParamFinishStep)
			}
			if messageTool.ToolEnd() {
				steps = append(steps, response.ToolResultFinishStep)
			}
			toolStepMap[toolId] = steps
			toolIdList = append(toolIdList, toolId)
		}
	} else if messageTool.ToolParamsEnd() {
		steps := toolStepMap[toolId]
		steps = append(steps, response.ToolParamFinishStep)
		toolStepMap[toolId] = steps
		toolIdList = append(toolIdList, toolId)
	} else if messageTool.ToolEnd() {
		steps := toolStepMap[toolId]
		steps = append(steps, response.ToolResultFinishStep)
		toolStepMap[toolId] = steps
		toolIdList = append(toolIdList, toolId)
	}
	return toolStepMap, toolIdList
}

// buildNoToolContent 构造没有工具的内容
// case1：tool 有数据同时content内容；如果此时在工具的输出中还没有输出完，则不输出content的相关内容
// case2：在tool输出前会输出规划内容，但是会重复输出相同的规划内容，所以当内容数字大于10时，同时出现重复数据，则不输出
// case3：正式输出
func buildNoToolContent(chatMessage *schema.Message, respContext *response.AgentChatRespContext, newStyle bool) []*response.AgentMessageContent {
	notFinishList := response.FilerToolByStep(respContext, response.ToolResultFinishStep, false)
	if len(notFinishList) > 0 { //在工具期间，不输出任何content内容
		return []*response.AgentMessageContent{}
	}
	toolContext := respContext.AgentToolContext
	//替换内容准备(工具未开始，但是输出了内容, 有的模型会重复输出一样的话)
	if !toolContext.HasTool() {
		var content = chatMessage.Content
		if len(content) == 0 {
			content = chatMessage.ReasoningContent
		}
		if utf8.RuneCountInString(content) > 10 {
			var replaceContent = respContext.ReplaceContentStr
			if len(replaceContent) == 0 {
				replaceContent = respContext.ReplaceContent.String()
			}
			if replaceContent == content {
				respContext.ReplaceContentDone = true
				respContext.ReplaceContentStr = replaceContent
				return []*response.AgentMessageContent{}
			}
		}
		if !respContext.ReplaceContentDone {
			respContext.ReplaceContent.WriteString(content)
		}
	}
	return buildContent(chatMessage)
}

func buildContent(chatMessage *schema.Message) []*response.AgentMessageContent {
	var retContentList []*response.AgentMessageContent
	//构造正常内容
	if len(chatMessage.Content) > 0 || stopMessage(chatMessage) {
		retContentList = append(retContentList, &response.AgentMessageContent{
			ContentList: []string{chatMessage.Content},
		})
	}
	return retContentList
}

func stopMessage(chatMessage *schema.Message) bool {
	return chatMessage.ResponseMeta != nil && chatMessage.ResponseMeta.FinishReason == "stop"
}

// buildToolContent 构造有工具的内容输出
// 需要额外判断，如果此次输出的步骤不包含当前任务的步骤，同时之前工具有参数未完成的，则补充个参数结束的内容（处理并发调用工具的情况）
func buildToolContent(chatMessage *schema.Message, respContext *response.AgentChatRespContext, stepsMap map[string][]response.ToolStep, toolIdList []string) []*response.AgentMessageContent {
	toolContext := respContext.AgentToolContext
	steps := stepsMap[toolContext.GetCurrentToolId()]
	paramsNotFinishList := response.FilerToolByStep(respContext, response.ToolParamStep, true)
	var contentList []string
	if len(steps) == 0 && len(paramsNotFinishList) > 0 { //是新工具且之前工具处于参数处理未完成状态
		//增加参数处理完成结果，并更改状态
		for _, toolId := range paramsNotFinishList {
			tool := toolContext.GetTool(toolId)
			if tool == nil {
				continue
			}
			//更改状态
			tool.ToolStep = response.ToolParamFinishStep
			//输出结果，增加结束
			contentList = append(contentList, toolParamsEndFormat)
		}
	}
	//根据step循环构造输出的内容
	for _, toolId := range toolIdList {
		toolSteps := stepsMap[toolId]
		agentTool := toolContext.GetTool(toolId)
		if agentTool == nil {
			agentTool = toolContext.AddToolById(toolId)
		}
		for _, step := range toolSteps {
			agentTool.ToolStep = step
			toolContentList := buildContentByStep(chatMessage, step, toolId)
			if len(toolContentList) == 0 {
				continue
			}
			contentList = append(contentList, toolContentList...)
		}
	}
	return []*response.AgentMessageContent{{ContentList: contentList}}
}

// buildToolContentNewStyle 构造有工具的内容输出-新样式
// 需要额外判断，如果此次输出的步骤不包含当前任务的步骤，同时之前工具有参数未完成的，则补充个参数结束的内容（处理并发调用工具的情况）
func buildToolContentNewStyle(req *request.AgentChatContext, chatMessage *schema.Message, respContext *response.AgentChatRespContext, stepsMap map[string][]response.ToolStep, toolIdList []string) []*response.AgentMessageContent {
	toolContext := respContext.AgentToolContext
	steps := stepsMap[toolContext.GetCurrentToolId()]
	paramsNotFinishList := response.FilerToolByStep(respContext, response.ToolParamStep, true)
	var toolContentList []*response.AgentMessageContent
	if len(steps) == 0 && len(paramsNotFinishList) > 0 { //是新工具且之前工具处于参数处理未完成状态
		//增加参数处理完成结果，并更改状态
		for _, toolId := range paramsNotFinishList {
			tool := respContext.AgentToolContext.GetTool(toolId)
			if tool == nil {
				continue
			}
			//更改状态
			tool.ToolStep = response.ToolParamFinishStep
			toolContentList = append(toolContentList, &response.AgentMessageContent{
				SubEventData: response.BuildEndTool(tool),
				ContentList:  []string{toolParamsEndFormat},
			})
		}
	}
	//根据step循环构造输出的内容
	for _, toolId := range toolIdList {
		toolSteps := stepsMap[toolId]
		agentTool := respContext.AgentToolContext.GetTool(toolId)
		if agentTool == nil {
			agentTool = &response.AgentTool{ToolId: toolId, Order: respContext.Order, StartTime: time.Now().UnixMilli(), ToolIndex: response.BuildToolIndex(chatMessage)}
			respContext.AgentToolContext.AddTool(agentTool)
		}
		for _, step := range toolSteps {
			agentTool.ToolStep = step
			toolContent := buildNewContentByStep(respContext, req, agentTool, chatMessage, step, toolId)
			if toolContent.Empty() {
				continue
			}
			toolContentList = append(toolContentList, toolContent)
		}
	}
	return toolContentList
}

// buildContentByStep 根据当前步骤构造需要输出的内容,构造<tool></tool>数据以及markdown格式
func buildContentByStep(chatMessage *schema.Message, step response.ToolStep, toolId string) []string {
	var contentList []string
	switch step {
	case response.ToolNameStep:
		tool := buildMessageTool(chatMessage, toolId)
		if tool == nil {
			break
		}
		toolName := fmt.Sprintf(toolStartTitleFormat, tool.Function.Name)
		contentList = append(contentList, toolName)
	case response.ToolParamStartStep:
		contentList = append(contentList, toolStartTitle)
		contentList = append(contentList, toolParamsStartFormat)
	case response.ToolParamStep:
		tool := buildMessageTool(chatMessage, toolId)
		if tool == nil {
			break
		}
		contentList = append(contentList, tool.Function.Arguments)
	case response.ToolParamFinishStep:
		contentList = append(contentList, toolParamsEndFormat)
	case response.ToolResultFinishStep:
		if len(chatMessage.Content) > 0 {
			toolResult := fmt.Sprintf(toolEndFormat, chatMessage.ToolName, chatMessage.Content)
			contentList = append(contentList, toolResult)
		}
		contentList = append(contentList, toolEndTitle)
	}
	return contentList
}

// buildNewContentByStep 根据当前步骤构造需要输出的内容
func buildNewContentByStep(respContext *response.AgentChatRespContext, req *request.AgentChatContext, agentTool *response.AgentTool, chatMessage *schema.Message, step response.ToolStep, toolId string) *response.AgentMessageContent {
	var subEventData *response.SubEventData
	var contentList []string
	if agentTool.ToolType == response.KnowledgeEventType {
		return buildKnowledgeContentByStep(req, agentTool, chatMessage, step)
	}
	switch step {
	case response.ToolNameStep:
		tool := buildMessageTool(chatMessage, toolId)
		if tool == nil {
			break
		}
		respContext.ContentOutput = false
		respContext.IncreaseOrder()
		agentTool.ToolName = tool.Function.Name
		agentTool.ToolType = response.BuildEventTypeByTool(agentTool)
		agentTool.Avatar = buildToolAvatar(tool.Function.Name, req.ToolMap, agentTool.ToolType)
		subEventData = response.BuildStartTool(agentTool)
	case response.ToolParamStartStep:
		contentList = append(contentList, toolParamsStartFormat)
		subEventData = response.BuildProcessTool(agentTool)
	case response.ToolParamStep:
		tool := buildMessageTool(chatMessage, toolId)
		if tool == nil {
			break
		}
		contentList = append(contentList, tool.Function.Arguments)
		subEventData = response.BuildProcessTool(agentTool)
	case response.ToolParamFinishStep:
		contentList = append(contentList, toolParamsEndFormat)
		subEventData = response.BuildProcessTool(agentTool)
	case response.ToolResultFinishStep:
		if len(chatMessage.Content) > 0 {
			toolResult := fmt.Sprintf(toolEndFormat, "", chatMessage.Content)
			contentList = append(contentList, toolResult)
		}
		subEventData = response.BuildEndTool(agentTool)
	}
	return &response.AgentMessageContent{
		ContentList:  contentList,
		SubEventData: subEventData,
	}
}

func buildKnowledgeContentByStep(req *request.AgentChatContext, agentTool *response.AgentTool, chatMessage *schema.Message, step response.ToolStep) *response.AgentMessageContent {
	var subEventData *response.SubEventData
	var contentList []string
	switch step {
	case response.ToolNameStep, response.ToolParamStartStep, response.ToolParamStep, response.ToolParamFinishStep:
		break
	case response.ToolResultFinishStep:
		req.KnowledgeHitData = buildKnowledgeContent(chatMessage.Content)
		subEventData = response.BuildEndTool(agentTool)
	}
	return &response.AgentMessageContent{
		ContentList:  contentList,
		SubEventData: subEventData,
	}
}

// buildKnowledgeContent 构造知识内容数据
func buildKnowledgeContent(data string) *model.KnowledgeHitData {
	if len(data) == 0 {
		return nil
	}
	var knowledgeHitData = &model.KnowledgeHitData{}
	err := json.Unmarshal([]byte(data), knowledgeHitData)
	if err != nil {
		return nil
	}
	return knowledgeHitData
}

// buildMessageTool 构造消息工具内容数据
func buildMessageTool(chatMessage *schema.Message, toolId string) *schema.ToolCall {
	switch length := len(chatMessage.ToolCalls); length {
	case 0:
		return nil
	case 1:
		return &chatMessage.ToolCalls[0]
	}

	for _, call := range chatMessage.ToolCalls {
		if call.ID == toolId {
			return &call
		}
	}
	return nil
}

// buildToolAvatar 构建工具头像
func buildToolAvatar(toolName string, toolMap map[string]*request.ToolConfig, toolEventType int) string {
	if len(toolMap) == 0 {
		return response.BuildDefaultAvatarByType(toolEventType)
	}
	toolConfig := toolMap[toolName]
	if toolConfig == nil || toolConfig.Avatar == "" {
		return response.BuildDefaultAvatarByType(toolEventType)
	}
	return toolConfig.Avatar
}
