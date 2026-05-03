package request

import "fmt"

type OpenAPIAgentCreateConversationRequest struct {
	Title string `json:"title"`
	UUID  string `json:"uuid" validate:"required"`
}

func (req *OpenAPIAgentCreateConversationRequest) Check() error {
	return nil
}

type OpenAPIAgentChatRequest struct {
	UUID           string                 `json:"uuid" validate:"required"`
	ConversationID string                 `json:"conversation_id"`
	Query          string                 `json:"query" validate:"required"`
	Stream         bool                   `json:"stream"`
	FileInfo       []ConversionStreamFile `json:"file_info"`
}

func (req *OpenAPIAgentChatRequest) Check() error {
	return nil
}

type OpenAPIAgentDraftChatRequest struct {
	UUID           string                 `json:"uuid" validate:"required"`
	ConversationID string                 `json:"conversation_id"`
	Query          string                 `json:"query" validate:"required"`
	FileInfo       []ConversionStreamFile `json:"file_info"`
}

func (req *OpenAPIAgentDraftChatRequest) Check() error {
	return nil
}

type OpenAPIRagChatRequest struct {
	UUID    string     `json:"uuid" validate:"required"`
	Query   string     `json:"query" validate:"required"`
	Stream  bool       `json:"stream"`
	History []*History `json:"history"`
}

func (req *OpenAPIRagChatRequest) Check() error {
	return nil
}

type OpenAPICreateAgentRequest struct {
	Category int `json:"category"` // 1:单智能体 2:多智能体
	AppBriefConfig
}

func (req *OpenAPICreateAgentRequest) Check() error {
	return nil
}

type OpenAPIWorkflowRunReq struct {
	UUID       string         `json:"uuid" validate:"required"`
	Parameters map[string]any `json:"parameters"`
}

func (req *OpenAPIWorkflowRunReq) Check() error {
	return nil
}

type OpenAPIChatflowCreateConversationRequest struct {
	UUID             string `json:"uuid" validate:"required"`
	ConversationName string `json:"conversation_name"`
}

func (req *OpenAPIChatflowCreateConversationRequest) Check() error {
	return nil
}

type OpenAPIChatflowChatRequest struct {
	UUID           string         `json:"uuid" validate:"required"`
	ConversationId string         `json:"conversation_id" validate:"required"`
	Query          string         `json:"query" validate:"required"`
	Parameters     map[string]any `json:"parameters"`
}

func (req *OpenAPIChatflowChatRequest) Check() error {
	return nil
}

type OpenAPIChatflowGetConversationMessageListRequest struct {
	UUID           string `json:"uuid" validate:"required"`
	ConversationId string `json:"conversation_id" validate:"required"`
	Limit          string `json:"limit"`
}

func (req *OpenAPIChatflowGetConversationMessageListRequest) Check() error {
	return nil
}

type OpenAPIAgentConfigUpdateRequest struct {
	AssistantUUID       string                  `json:"assistantUuid" validate:"required"`
	Prologue            string                  `json:"prologue"`
	Instructions        string                  `json:"instructions"`
	RecommendQuestion   []string                `json:"recommendQuestion"`
	ModelConfig         *AppModelConfig         `json:"modelConfig"`
	KnowledgeBaseConfig *AppKnowledgebaseConfig `json:"knowledgeBaseConfig"`
	SafetyConfig        *AppSafetyConfig        `json:"safetyConfig"`
	RerankConfig        *AppModelConfig         `json:"rerankConfig"`
	VisionConfig        *VisionConfig           `json:"visionConfig"`
	MemoryConfig        *MemoryConfig           `json:"memoryConfig"`
	RecommendConfig     *RecommendConfig        `json:"recommendConfig"`
}

func (req *OpenAPIAgentConfigUpdateRequest) Check() error {
	return nil
}

type OpenAPIGetAgentInfoRequest struct {
	UUID      string `form:"uuid" validate:"required"`
	Published bool   `form:"published"`
}

func (req *OpenAPIGetAgentInfoRequest) Check() error {
	return nil
}

type OpenAPIAgentPublishRequest struct {
	AssistantUUID string `json:"assistantUuid" validate:"required"`
	Version       string `json:"version" validate:"required"`
	Desc          string `json:"desc" validate:"required"`
	PublishType   string `json:"publishType" validate:"required"` // public:系统公开发布 organization:组织公开发布 private:私密发布
}

func (req *OpenAPIAgentPublishRequest) Check() error {
	if !versionRegexp.MatchString(req.Version) {
		return fmt.Errorf("version must be in format 'vX.Y.Z'")
	}
	return nil
}
