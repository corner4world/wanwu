package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	err_code "github.com/UnicomAI/wanwu/api/proto/err-code"
	model_service "github.com/UnicomAI/wanwu/api/proto/model-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/internal/bff-service/pkg/ahocorasick"
	gin_util "github.com/UnicomAI/wanwu/pkg/gin-util"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	"github.com/UnicomAI/wanwu/pkg/log"
	mp "github.com/UnicomAI/wanwu/pkg/model-provider"
	mp_common "github.com/UnicomAI/wanwu/pkg/model-provider/mp-common"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

func ModelExperienceLLM(ctx *gin.Context, userId, orgId string, req *request.ModelExperienceLlmRequest) {
	// 敏感词检测 - 输入检测
	matchDicts, err := BuildSensitiveDict(ctx, nil, false)
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	matchResults, err := ahocorasick.ContentMatch(req.Content, matchDicts, true)
	if err != nil {
		gin_util.Response(ctx, nil, grpc_util.ErrorStatus(err_code.Code_BFFSensitiveWordCheck, err.Error()))
		return
	}
	if len(matchResults) > 0 {
		if matchResults[0].Reply != "" {
			gin_util.Response(ctx, nil, grpc_util.ErrorStatusWithKey(err_code.Code_BFFSensitiveWordCheck, "bff_sensitive_check_req", matchResults[0].Reply))
			return
		}
		gin_util.Response(ctx, nil, grpc_util.ErrorStatusWithKey(err_code.Code_BFFSensitiveWordCheck, "bff_sensitive_check_req_default_reply"))
		return
	}
	// model info
	modelInfo, err := model.GetModel(ctx.Request.Context(), &model_service.GetModelReq{ModelId: req.ModelId})
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	if !modelInfo.IsActive {
		gin_util.Response(ctx, nil, grpc_util.ErrorStatus(err_code.Code_BFFModelStatus, modelInfo.ModelId))
		return
	}

	// dialog records
	recordsResp, err := model.GetModelExperienceDialogRecords(ctx, &model_service.GetModelExperienceDialogRecordsReq{
		UserId: userId,
		OrgId:  orgId,
		// 常规模型对话记录（非模型对比时），modelExperienceId与sessionId非空
		// 模型对比时临时存储对话记录，modelExperienceId前端传空，sessionId非空
		ModelExperienceId: req.ModelExperienceId,
		SessionId:         req.SessionId,
	})
	if err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}
	var messages []mp_common.OpenAIReqMsg
	for _, record := range recordsResp.Records {
		content := record.HandledContent
		if content == "" {
			content = record.OriginalContent
		}
		messages = append(messages, mp_common.OpenAIReqMsg{
			Role:    mp_common.MsgRole(record.Role),
			Content: content,
		})
	}
	// 添加当前用户消息
	messages = append(messages, mp_common.OpenAIReqMsg{
		Role:    mp_common.MsgRoleUser,
		Content: req.Content,
	})

	// 构造LLM请求
	stream := true
	llmReq := &mp_common.LLMReq{
		Model:    modelInfo.Model,
		Messages: messages,
		Stream:   &stream,
	}
	if req.TemperatureEnable {
		llmReq.Temperature = &req.Temperature
	}
	if req.TopPEnable {
		llmReq.TopP = &req.TopP
	}
	if req.PresencePenaltyEnable {
		llmReq.PresencePenalty = &req.PresencePenalty
	}
	if req.FrequencyPenaltyEnable {
		llmReq.FrequencyPenalty = &req.FrequencyPenalty
	}
	if req.MaxTokensEnable {
		maxTokens := int(req.MaxTokens)
		llmReq.MaxTokens = &maxTokens
	}
	llmReq.EnableThinking = req.ThinkingEnable

	llm, err := mp.ToModelConfig(modelInfo.Provider, modelInfo.ModelType, modelInfo.ProviderConfig)
	if err != nil {
		recordModelStatistic(ctx, modelInfo, false, 0, 0, 0, 0, 0, false)
		gin_util.Response(ctx, nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, err.Error()))
		return
	}
	iLLM, ok := llm.(mp.ILLM)
	if !ok {
		recordModelStatistic(ctx, modelInfo, false, 0, 0, 0, 0, 0, false)
		gin_util.Response(ctx, nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, err.Error()))
		return
	}
	startTime := time.Now()

	// chat completions
	iLLMReq, err := iLLM.NewReq(llmReq)
	if err != nil {
		recordModelStatistic(ctx, modelInfo, false, 0, 0, 0, 0, 0, false)
		gin_util.Response(ctx, nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, err.Error()))
		return
	}
	// 使用 context.Background() 避免 ctx 被取消导致请求中断
	_, sseCh, err := iLLM.ChatCompletions(context.Background(), iLLMReq)
	if err != nil {
		recordModelStatistic(ctx, modelInfo, false, 0, 0, 0, 0, 0, false)
		gin_util.Response(ctx, nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, err.Error()))
		return
	}

	// save query
	if _, err := model.SaveModelExperienceDialogRecord(ctx.Request.Context(), &model_service.SaveModelExperienceDialogRecordReq{
		UserId:            userId,
		OrgId:             orgId,
		ModelExperienceId: req.ModelExperienceId,
		ModelId:           req.ModelId,
		SessionId:         req.SessionId,
		OriginalContent:   req.Content,
		Role:              string(mp_common.MsgRoleUser),
	}); err != nil {
		gin_util.Response(ctx, nil, err)
		return
	}

	// stream
	var answer string
	var reasonContent string
	var (
		firstTokenLatency int
		promptTokens      int
		completionTokens  int
		totalTokens       int
	)
	ctx.Header("Cache-Control", "no-cache")
	ctx.Header("Connection", "keep-alive")
	ctx.Header("Content-Type", "text/event-stream; charset=utf-8")

	// 消费 sseCh 时一边统计 token，一边转成 rawCh 交给敏感词处理。
	rawCh := make(chan string, 1024)
	go func() {
		defer util.PrintPanicStack()
		defer close(rawCh)
		firstTokenReceived := false
		for sseResp := range sseCh {
			resp, ok := sseResp.ConvertResp()
			if ok && resp != nil {
				if !firstTokenReceived {
					firstTokenReceived = true
					firstTokenLatency = int(time.Since(startTime).Milliseconds())
				}
				promptTokens = resp.Usage.PromptTokens
				completionTokens = resp.Usage.CompletionTokens
				totalTokens = resp.Usage.TotalTokens
			}
			// ConvertResp 失败的空行也是 SSE 事件分隔符，必须保留。
			dataStr := sseResp.String()
			if !ok {
				dataStr = fmt.Sprintf("%v\n", dataStr)
			}
			// 敏感词过滤结束后继续 drain sseCh 统计 token，但不再阻塞写 rawCh。
			select {
			case rawCh <- dataStr:
			default:
			}
		}
		recordModelStatistic(ctx, modelInfo, true,
			promptTokens, completionTokens, totalTokens, 0, firstTokenLatency, true)
	}()

	// 敏感词过滤（必须过滤，全局敏感词）
	outputCh := ProcessSensitiveWords(ctx, rawCh, matchDicts, &modelExperienceSensitiveService{})

	// 从过滤后的 channel 写入 SSE 并累加 answer
	for dataStr := range outputCh {
		if _, err = ctx.Writer.Write([]byte(dataStr)); err != nil {
			log.Errorf("model experience write sse err: %v", err)
		}
		ctx.Writer.Flush()

		// 解析 SSE 内容累加 answer
		if strings.HasPrefix(dataStr, "data:") {
			content := strings.TrimSpace(strings.TrimPrefix(dataStr, "data:"))
			if content == "[DONE]" {
				continue
			}
			var resp mp_common.LLMResp
			if err := json.Unmarshal([]byte(content), &resp); err == nil {
				if len(resp.Choices) > 0 && resp.Choices[0].Delta != nil {
					delta := resp.Choices[0].Delta
					answer = answer + delta.Content
					if delta.ReasoningContent != nil {
						reasonContent = reasonContent + *delta.ReasoningContent
					}
				}
			}
		}
	}
	// save answer
	if _, err := model.SaveModelExperienceDialogRecord(ctx.Request.Context(), &model_service.SaveModelExperienceDialogRecordReq{
		UserId:            userId,
		OrgId:             orgId,
		ModelExperienceId: req.ModelExperienceId,
		ModelId:           req.ModelId,
		SessionId:         req.SessionId,
		OriginalContent:   answer,
		ReasoningContent:  reasonContent,
		Role:              string(mp_common.MsgRoleAssistant),
	}); err != nil {
		log.Errorf("model experience save record err: %v", err)
		return
	}

	ctx.Set(gin_util.STATUS, http.StatusOK)
	ctx.Set(gin_util.RESULT, answer)

}

func SaveModelExperienceDialog(ctx *gin.Context, userId, orgId string, req *request.ModelExperienceDialogRequest) (*response.ModelExperienceDialog, error) {
	// 将interface{}类型的ModelSetting转换为 json string
	var modelSettingStr string
	if req.ModelSetting != nil {
		jsonBytes, err := json.Marshal(req.ModelSetting)
		if err != nil {
			return nil, grpc_util.ErrorStatus(err_code.Code_BFFGeneral, fmt.Sprintf("Model settings serialization error: err: %v", err))
		}
		modelSettingStr = string(jsonBytes)
	}
	dialog, err := model.SaveModelExperienceDialog(ctx.Request.Context(), &model_service.SaveModelExperienceDialogReq{
		UserId:       userId,
		OrgId:        orgId,
		ModelId:      req.ModelId,
		SessionId:    req.SessionId,
		ModelSetting: modelSettingStr,
		Title:        req.Title,
	})
	if err != nil {
		return nil, err
	}
	return toModelExperienceDialog(dialog), nil
}

func ListModelExperienceDialogs(ctx *gin.Context, userId, orgId string) (*response.ListResult, error) {
	resp, err := model.GetModelExperienceDialogs(ctx.Request.Context(), &model_service.ListModelExperienceDialogReq{
		UserId: userId,
		OrgId:  orgId,
	})
	if err != nil {
		return nil, err
	}

	// 收集所有唯一的模型ID
	modelIdMap := make(map[string]bool)
	for _, dialog := range resp.Dialogs {
		modelIdMap[dialog.ModelId] = true
	}

	// 提取唯一模型ID列表
	var uniqueModelIds []string
	for modelId := range modelIdMap {
		uniqueModelIds = append(uniqueModelIds, modelId)
	}

	// 批量检查模型权限
	authorizedModelIds, _ := CheckModelUserPermission(ctx, userId, orgId, uniqueModelIds)

	// 创建授权模型ID的映射，用于快速查找
	authorizedModelMap := make(map[string]bool)
	for _, modelId := range authorizedModelIds {
		authorizedModelMap[modelId] = true
	}

	// 过滤出用户有权限的对话
	var dialogs []*response.ModelExperienceDialog
	for _, dialog := range resp.Dialogs {
		if authorizedModelMap[dialog.ModelId] {
			dialogs = append(dialogs, toModelExperienceDialog(dialog))
		}
	}

	return &response.ListResult{
		List:  dialogs,
		Total: int64(len(resp.Dialogs)),
	}, nil
}

func DeleteModelExperienceDialog(ctx *gin.Context, userId, orgId, modelExperienceId string) error {
	_, err := model.DeleteModelExperienceDialog(ctx, &model_service.ModelExperienceDialogReq{
		ModelExperienceId: modelExperienceId,
		UserId:            userId,
		OrgId:             orgId,
	})
	return err
}

func ListModelExperienceDialogRecords(ctx *gin.Context, userId, orgId string, req *request.ModelExperienceDialogRecordRequest) (*response.ListResult, error) {
	recordsResp, err := model.GetModelExperienceDialogRecords(ctx, &model_service.GetModelExperienceDialogRecordsReq{
		UserId: userId,
		OrgId:  orgId,
		// 常规模型对话记录（非模型对比时），modelExperienceId非空，sessionId前端没传
		ModelExperienceId: req.ModelExperienceId,
		SessionId:         "",
	})
	if err != nil {
		return nil, err
	}
	var records []*response.ModelExperienceDialogRecord
	for _, record := range recordsResp.Records {
		records = append(records, &response.ModelExperienceDialogRecord{
			ModelExperienceId: record.ModelExperienceId,
			ModelId:           record.ModelId,
			SessionId:         record.SessionId,
			OriginalContent:   record.OriginalContent,
			ReasoningContent:  record.ReasoningContent,
			Role:              record.Role,
		})
	}
	return &response.ListResult{
		List:  records,
		Total: int64(len(records)),
	}, nil
}
func toModelExperienceDialog(dialog *model_service.ModelExperienceDialog) *response.ModelExperienceDialog {
	return &response.ModelExperienceDialog{
		ID:           dialog.ModelExperienceId,
		ModelId:      dialog.ModelId,
		SessionId:    dialog.SessionId,
		Title:        dialog.Title,
		ModelSetting: dialog.ModelSetting,
		CreatedAt:    dialog.CreatedAt,
	}
}

// --- modelExperienceSensitiveService: 实现 chatService 接口，供 ProcessSensitiveWords 使用 ---
type modelExperienceSensitiveService struct{}

func (m *modelExperienceSensitiveService) serviceType() string {
	return "ModelExperience"
}

func (m *modelExperienceSensitiveService) parseContent(raw string) (id, content string) {
	// SSE 数据格式为 "data: {...}\n\n"，解析出 content + reasoning_content
	raw = strings.TrimPrefix(raw, "data:")
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "[DONE]" {
		return "", ""
	}

	var resp mp_common.LLMResp
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		return "", ""
	}

	if len(resp.Choices) > 0 && resp.Choices[0].Delta != nil {
		delta := resp.Choices[0].Delta
		content = delta.Content
		if delta.ReasoningContent != nil && *delta.ReasoningContent != "" {
			content = content + *delta.ReasoningContent // 合并思维链和正文内容
		}
	}
	return resp.ID, content
}

func (m *modelExperienceSensitiveService) buildSensitiveResp(id, content string) []string {
	// 返回 OpenAI delta 格式的 SSE 数据（finish_reason: stop 表示流结束）
	resp := mp_common.LLMResp{
		ID: id,
		Choices: []mp_common.OpenAIRespChoice{
			{
				Index: 0,
				Delta: &mp_common.OpenAIMsg{
					Role:    mp_common.MsgRoleAssistant,
					Content: content,
				},
				FinishReason: "stop",
			},
		},
	}
	marshal, _ := json.Marshal(resp)
	return []string{"data: " + string(marshal) + "\n\n"}
}
