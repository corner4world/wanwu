package request

const (
	// StatisticFilterAll orgIds/userIds 数组中的哨兵值，表示该维度在角色可见范围内「全部」。
	StatisticFilterAll = "ALL"
)

// StatisticFilter 统计看板组织/用户筛选（嵌入各统计与下拉请求）。
//
// 约定：
//   - orgIds/userIds 未传或 []：该维度未扩大；统计查询为 JWT 当前 userId + orgId；
//   - 含 "ALL"：仅管理员；组织按 IAM 展开，用户为已解析组织下的全部用户；
//   - 具体 id：仅查所列 id（系统管理员原样；组织管理员须在可管理范围内）。
type StatisticFilter struct {
	OrgIds  []string `json:"orgIds" `  // 空=未扩大；["ALL"]=可见全部组织；["id",...]=指定组织
	UserIds []string `json:"userIds" ` // 空=未扩大用户维度；["ALL"]=已解析组织下全部用户；["id",...]=指定用户
}

func (f *StatisticFilter) Check() error { return nil }

// --- 应用统计相关请求 ---

type AppStatisticReq struct {
	StatisticFilter
	StartDate string   `json:"startDate" validate:"required"` // 开始时间（格式yyyy-mm-dd）
	EndDate   string   `json:"endDate" validate:"required"`   // 结束时间（格式yyyy-mm-dd）
	Apps      []string `json:"apps"`                          // 应用ID列表
	AppType   string   `json:"appType"`                       // 应用类型（默认agent）
}

func (r *AppStatisticReq) Check() error { return nil }

type AppStatisticListReq struct {
	AppStatisticReq
	PageNo   int `json:"pageNo" validate:"required"`
	PageSize int `json:"pageSize" validate:"required"`
}

func (r *AppStatisticListReq) Check() error { return nil }

type StatisticAppSelectReq struct {
	StatisticFilter
	AppType string `json:"appType"` // 应用类型
}

func (r *StatisticAppSelectReq) Check() error { return nil }

// --- 模型统计相关请求 ---

type ModelStatisticReq struct {
	StatisticFilter
	StartDate string   `json:"startDate" validate:"required"`
	EndDate   string   `json:"endDate" validate:"required"`
	Models    []string `json:"models"`
	ModelType string   `json:"modelType" validate:"required"`
}

func (r *ModelStatisticReq) Check() error { return nil }

type ModelStatisticListReq struct {
	ModelStatisticReq
	PageNo   int `json:"pageNo"`
	PageSize int `json:"pageSize"`
}

func (r *ModelStatisticListReq) Check() error { return nil }

type StatisticModelSelectReq struct {
	StatisticFilter
	ModelType string `json:"modelType"` // 模型类型
}

func (r *StatisticModelSelectReq) Check() error { return nil }

// --- API Key 统计相关请求 ---

type APIKeyStatisticReq struct {
	StatisticFilter
	StartDate   string   `json:"startDate" validate:"required"` // 开始时间（格式yyyy-mm-dd）
	EndDate     string   `json:"endDate" validate:"required"`   // 结束时间（格式yyyy-mm-dd）
	APIKeyIds   []string `json:"apiKeyIds"`                     // API Key 列表
	MethodPaths []string `json:"methodPaths"`                   // OpenAPI方法+路径（例如：POST-/agent/chat）
}

func (r *APIKeyStatisticReq) Check() error { return nil }

type APIKeyStatisticListReq struct {
	APIKeyStatisticReq
	PageNo   int `json:"pageNo" validate:"required"`
	PageSize int `json:"pageSize" validate:"required"`
}

func (r *APIKeyStatisticListReq) Check() error { return nil }

type APIKeyStatisticRecordReq struct {
	APIKeyStatisticReq
	PageNo   int `json:"pageNo" validate:"required"`
	PageSize int `json:"pageSize" validate:"required"`
}

func (r *APIKeyStatisticRecordReq) Check() error { return nil }

type StatisticAPIKeySelectReq struct {
	StatisticFilter
}

func (r *StatisticAPIKeySelectReq) Check() error { return nil }
