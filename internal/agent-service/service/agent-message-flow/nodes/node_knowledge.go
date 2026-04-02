package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/UnicomAI/wanwu/internal/agent-service/model"
	"github.com/UnicomAI/wanwu/internal/agent-service/model/request"
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg/config"
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg/http"
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg/util"
	"github.com/UnicomAI/wanwu/internal/agent-service/service/agent-message-flow/prompt"
	http_client "github.com/UnicomAI/wanwu/pkg/http-client"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/cloudwego/eino/adk"
	"github.com/cloudwego/eino/schema"
	"github.com/google/uuid"
)

const (
	successCode = 0
)

type KnowledgeRetriever struct {
}

func (k *KnowledgeRetriever) Retrieve(ctx context.Context, reqContext *request.AgentChatContext) (string, error) {
	req := reqContext.AgentChatReq
	if req.KnowledgeParams == nil {
		return "", nil
	}
	req.KnowledgeParams.Question = req.Input
	req.KnowledgeParams.CustomModelInfo = &request.CustomModelInfo{
		LlmModelID: req.ModelParams.ModelId,
	}
	req.KnowledgeParams.AttachmentFiles = make([]*request.RagKnowledgeAttachment, 0)
	toolId := uuid.New().String()
	newStyle := reqContext.AgentChatReq.NewStyle
	sendKnowledgeMessage(reqContext.Generator, false, nil, toolId, newStyle)
	defer func() {
		sendKnowledgeMessage(reqContext.Generator, true, reqContext.KnowledgeHitData, toolId, newStyle)
	}()
	fileList := reqContext.AgentChatReq.UploadFile
	if len(fileList) > 0 {
		file := fileList[0]
		switch filepath.Ext(file) {
		case ".jpg", ".png", ".jpeg":
			req.KnowledgeParams.AttachmentFiles = append(req.KnowledgeParams.AttachmentFiles, &request.RagKnowledgeAttachment{
				FileType: "image",
				FileUrl:  file,
			})
		}
	}
	hit, _ := ragKnowledgeHit(ctx, req.KnowledgeParams)
	if hit == nil {
		return "", nil
	}
	reqContext.KnowledgeHitData = hit.Data
	packedRes := strings.Builder{}
	for idx, doc := range hit.Data.SearchList {
		if doc == nil {
			continue
		}
		number := idx + 1
		docLine := fmt.Sprintf("---\nrecall slice %d: 【%d】%s\n", number, number, doc.Snippet)
		packedRes.WriteString(docLine)
	}
	if packedRes.Len() > 0 {
		sliceCount := len(hit.Data.SearchList)
		knowledgeData := fmt.Sprintf(prompt.REACT_SYSTEM_PROMPT_KNOWLEDGE, sliceCount, packedRes.String())
		return knowledgeData, nil
	}
	//如果没有知识库时，尽量减少输入token大小
	return "", nil
}

// RagKnowledgeHit rag命中测试
func ragKnowledgeHit(ctx context.Context, knowledgeHitParams *request.KnowledgeParams) (*model.RagKnowledgeHitResp, error) {
	ragServer := config.GetConfig().RagServer
	url := ragServer.ProxyPoint + ragServer.KnowledgeHitUri
	paramsByte, err := json.Marshal(knowledgeHitParams)
	if err != nil {
		return nil, err
	}
	result, err := http.GetClient().PostJson(ctx, &http_client.HttpRequestParams{
		Url:        url,
		Body:       paramsByte,
		Timeout:    time.Duration(ragServer.Timeout) * time.Second,
		MonitorKey: "rag_knowledge_hit",
		LogLevel:   http_client.LogAll,
	})
	if err != nil {
		return nil, err
	}
	var resp model.RagKnowledgeHitResp
	if err := json.Unmarshal(result, &resp); err != nil {
		log.Errorf(err.Error())
		return nil, err
	}
	if resp.Code != successCode {
		return nil, errors.New(resp.Message)
	}
	return &resp, nil
}

// sendKnowledgeMessage 发送知识库消息
func sendKnowledgeMessage(generator *adk.AsyncGenerator[*adk.AgentEvent], finish bool, hitData *model.KnowledgeHitData, toolId string, newStyle bool) {
	if generator != nil && newStyle {
		message := buildKnowledgeMessage(finish, hitData, toolId)
		generator.Send(&adk.AgentEvent{
			Output: &adk.AgentOutput{
				MessageOutput: &adk.MessageVariant{
					IsStreaming: false,
					Message:     message,
					Role:        message.Role,
				},
			},
		})
	}
}

// buildKnowledgeMessage 构建知识库消息
func buildKnowledgeMessage(finish bool, hitData *model.KnowledgeHitData, toolId string) *schema.Message {
	if finish {
		return buildFinishMessage(hitData, toolId)
	}
	return buildStartMessage(toolId)
}

// buildStartMessage 构建开始消息
func buildStartMessage(toolId string) *schema.Message {
	return &schema.Message{
		Role:    schema.Assistant,
		Content: "",
		ToolCalls: []schema.ToolCall{
			{
				ID:   toolId,
				Type: "function",
				Function: schema.FunctionCall{
					Name:      util.AgentSearchKnowledgeName,
					Arguments: "",
				},
			},
		},
		ResponseMeta: &schema.ResponseMeta{
			FinishReason: "tool_calls",
			Usage:        &schema.TokenUsage{},
		},
	}
}

// buildFinishMessage 构建结束消息
func buildFinishMessage(hitData *model.KnowledgeHitData, toolId string) *schema.Message {
	marshal, err := json.Marshal(hitData)
	var message = ""
	if err == nil {
		message = string(marshal)
	}
	return &schema.Message{
		Role:       schema.Tool,
		Content:    message,
		ToolCallID: toolId,
		ToolName:   util.AgentSearchKnowledgeName,
	}
}
