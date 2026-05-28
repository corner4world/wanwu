package agent_tool

import (
	"context"
	"encoding/json"
	"time"

	"github.com/UnicomAI/wanwu/internal/agent-service/model/response"
	agent_config "github.com/UnicomAI/wanwu/internal/agent-service/pkg/config"
	"github.com/UnicomAI/wanwu/internal/agent-service/pkg/http"
	service_model "github.com/UnicomAI/wanwu/internal/agent-service/service/service-model"
	http_client "github.com/UnicomAI/wanwu/pkg/http-client"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/cloudwego/eino/components/tool"
	"github.com/cloudwego/eino/schema"
)

const (
	limitTokenRate = 0.7
)

type ChatDocParams struct {
	UploadFileUrl string `json:"upload_file_url"`
	MaxToken      int    `json:"max_token"`
}

// chatDocTool 实现了 tool.InvokableRun 接口
type chatDocTool struct {
	info     *schema.ToolInfo
	chatInfo *service_model.AgentChatInfo
}

// Info 返回工具的元信息
func (t *chatDocTool) Info(ctx context.Context) (*schema.ToolInfo, error) {
	marshal, _ := json.Marshal(t.info)
	log.Infof("chatDocTool %v", string(marshal))
	return t.info, nil
}

// InvokableRun 执行工具
func (t *chatDocTool) InvokableRun(ctx context.Context, argumentsInJSON string, opts ...tool.Option) (string, error) {
	doc, err := searchChatDoc(ctx, buildChatDocParams(argumentsInJSON, t.chatInfo))
	if err != nil {
		log.Errorf("chat doc request error,params %s, err: %v", argumentsInJSON, err)
		return response.ToolErrResp(err)
	}
	return doc, nil
}

// GetChatDocTool 构建chatDoc技能工具
func GetChatDocTool(chatInfo *service_model.AgentChatInfo) tool.BaseTool {
	//如果用户使用的不是多模态模型，但是又上传了文件，则通过工具对文件进行解析，
	//但是目前只支持一个文件，文件url具体处理逻在node_prompt_variables.go
	if chatInfo.FunctionCalling && chatInfo.UploadUrl && !chatInfo.ImageUpload {
		toolInfo := buildChatDocToolInfo()
		if toolInfo != nil {
			return &chatDocTool{
				info:     toolInfo,
				chatInfo: chatInfo,
			}
		}
	}
	return nil
}

func buildChatDocToolInfo() *schema.ToolInfo {
	templateConfig := agent_config.GetToolTemplateConfig()
	chatDocConfig, _ := templateConfig.GetToolByID(agent_config.DocParser)
	if chatDocConfig != nil {
		apiSchema, _ := GetEnioToolsFromOpenAPISchema(context.Background(), chatDocConfig)
		if len(apiSchema) > 0 {
			info := apiSchema[0]
			return &schema.ToolInfo{
				Name:        info.Name,
				Desc:        info.Desc,
				ParamsOneOf: info.ParamsOneOf,
			}
		}
	}
	return nil
}

// buildChatDocParams 构造chatDoc请求参数
func buildChatDocParams(argumentsInJSON string, chatInfo *service_model.AgentChatInfo) *ChatDocParams {
	var chatDocParams = &ChatDocParams{}
	_ = json.Unmarshal([]byte(argumentsInJSON), chatDocParams)
	chatDocParams.MaxToken = int(float64(chatInfo.ModelInfo.Config.ContextSize) * limitTokenRate)
	return chatDocParams
}

// searchChatDoc 查询chatDoc
func searchChatDoc(ctx context.Context, chatDocParams *ChatDocParams) (string, error) {
	toolServer := agent_config.GetConfig().ToolServer
	url := toolServer.Endpoint + "/v1/doc_parse"
	marshal, err := json.Marshal(chatDocParams)
	if err != nil {
		return "", err
	}
	result, err := http.GetClient().PostJson(ctx, &http_client.HttpRequestParams{
		Url:        url,
		Body:       marshal,
		Timeout:    5 * time.Minute,
		MonitorKey: "search_chat_doc",
		LogLevel:   http_client.LogAll,
	})
	if err != nil {
		return "", err
	}
	return string(result), nil
}
