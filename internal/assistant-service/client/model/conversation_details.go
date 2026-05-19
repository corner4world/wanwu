package model

type ConversationType string
type SubEventStatus int

const (
	AgentTool         ConversationType = "agentTool"         //主智能体工具
	AgentKnowledge    ConversationType = "agentKnowledge"    //主智能体知识库
	AgentThink        ConversationType = "agentThink"        //主智能体思考
	SubAgent          ConversationType = "subAgent"          //子智能体
	AgentSkill        ConversationType = "agentSkill"        //子智能体
	AgentSubText      ConversationType = "subText"           //智能体skill内容,或者子智能体内容
	SubAgentTool      ConversationType = "subAgentTool"      //子智能体工具
	SubAgentKnowledge ConversationType = "subAgentKnowledge" //子智能体只是库

	EventStartStatus   SubEventStatus = 1 //开始事件
	EventProcessStatus SubEventStatus = 2 //输出中
	EventEndStatus     SubEventStatus = 3 //结束事件
	EventFailStatus    SubEventStatus = 4 //子智能体失败
)

type FileInfo struct {
	FileName string `json:"fileName"`
	FileSize int64  `json:"fileSize"`
	FileUrl  string `json:"fileUrl"`
}

type AgentFile struct {
	Name     string     `json:"name"`
	Size     int        `json:"size"`
	FileUrl  string     `json:"fileUrl"`
	FileType string     `json:"fileType"`
	Metadata *AgentMeta `json:"metadata"`
}

type AgentMeta struct {
	Desc     string `json:"desc"`
	CreateAt string `json:"createAt"`
	Name     string `json:"name"`
}

type SubEventData struct {
	Status      SubEventStatus `json:"status"`
	Id          string         `json:"id"`
	ParentId    string         `json:"parentId"`
	Name        string         `json:"name"`
	Profile     string         `json:"profile"`
	TimeCost    string         `json:"timeCost"`
	Order       int            `json:"order"`
	DisplayMode int            `json:"displayMode"`
	ErrMessage  string         `json:"errMessage"`
}

func (s *SubEventData) Copy() *SubEventData {
	return &SubEventData{
		Status:     s.Status,
		Id:         s.Id,
		ParentId:   s.ParentId,
		Name:       s.Name,
		Profile:    s.Profile,
		TimeCost:   s.TimeCost,
		Order:      s.Order,
		ErrMessage: s.ErrMessage,
	}
}

type SubConversationDetail struct {
	BusinessId                string                   `json:"businessId"` //业务id，当conversationType 是AgentTool,SubAgentTool 则是toolId，SubAgent 则是agentId
	ConversationType          ConversationType         `json:"conversationType"`
	Content                   string                   `json:"content"`                   //内容
	Order                     int                      `json:"order"`                     //全局顺序
	SubConversationDetailList []*SubConversationDetail `json:"subConversationDetailList"` //子数据内容，对于多智能体，每个智能体又有多个工具详情，使用此处
	SearchList                string                   `json:"searchList"`
	EventData                 *SubEventData            `json:"eventData"`
}

type ConversationResponse struct {
	Response    string `json:"response"`
	ErrResponse string `json:"errResponse"`
	ErrMessage  string `json:"errMessage"`
	Order       int    `json:"order"`
}

func (c *ConversationResponse) Empty() bool {
	return len(c.Response) == 0 && len(c.ErrResponse) == 0 && len(c.ErrMessage) == 0
}

type ConversationDetails struct {
	Id                        string                   `json:"id"`
	AssistantId               string                   `json:"assistantId"`
	ConversationId            string                   `json:"conversationId"`
	Prompt                    string                   `json:"prompt"`
	SysPrompt                 string                   `json:"sysPrompt"`
	Response                  string                   `json:"response"`
	ResponseList              []*ConversationResponse  `json:"responseList"`
	SubConversationDetailList []*SubConversationDetail `json:"SubConversationDetailList"`
	SearchList                string                   `json:"searchList"`
	FileUrl                   string                   `json:"requestFileUrls"`
	FileSize                  int64                    `json:"fileSize"`
	FileName                  string                   `json:"fileName"`
	FileInfo                  []FileInfo               `json:"fileInfo"`
	UserId                    string                   `json:"userId"`
	OrgId                     string                   `json:"orgId"`
	CreatedAt                 int64                    `json:"createdAt"`
	UpdatedAt                 int64                    `json:"updatedAt"`
	ResponseFiles             []*AgentFile             `json:"responseFiles"`
}
