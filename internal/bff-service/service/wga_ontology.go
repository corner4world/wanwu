package service

import (
	"fmt"
	"net/url"
	"strings"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
	wga_option "github.com/UnicomAI/wanwu/pkg/wga/wga-option"
	"github.com/cloudwego/eino/schema"
	"github.com/gin-gonic/gin"
	"github.com/go-resty/resty/v2"
)

// --- internal wga ontology ---

// ontologyKnowledgeNetworkListResp 知识网络列表响应
type ontologyKnowledgeNetworkListResp struct {
	KnowledgeNetworks []ontologyKnowledgeNetwork `json:"entries"`
	TotalCount        int                        `json:"total_count"`
}

// ontologyKnowledgeNetwork 知识网络
type ontologyKnowledgeNetwork struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Comment    string `json:"comment"`
	ModuleType string `json:"module_type"`
}

// getOntologyKnowledgeSelect 获取知识网络选择列表
func getOntologyKnowledgeSelect(ctx *gin.Context, userId, orgId, name string) ([]*response.GeneralAgentResourceSelectItem, error) {
	if config.Cfg().Ontology.Enable == 0 {
		return nil, nil
	}
	// 构建URL
	endpoint := config.Cfg().Ontology.Endpoint
	listUri := config.Cfg().Ontology.KnowledgeNetworkListUri
	requestUrl, err := url.JoinPath(endpoint, listUri)
	if err != nil {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_ontology_knowledge_select", fmt.Sprintf("build url failed: %v", err))
	}

	// 构建请求参数
	queryParams := map[string]string{
		"direction": "desc",
		"sort":      "update_time",
		"offset":    "0",
		"limit":     "999", // 默认拉取前999条，满足大部分场景需求
	}
	if name != "" {
		queryParams["name"] = name
	}

	// 发送HTTP请求
	ret := &ontologyKnowledgeNetworkListResp{}
	resp, err := resty.New().
		R().
		SetContext(ctx.Request.Context()).
		SetHeader("Content-Type", "application/json").
		SetHeader("Accept", "application/json").
		SetHeader("X-Account-Id", userId).
		SetHeaders(workflowHttpReqHeader(ctx)).
		SetQueryParams(queryParams).
		SetResult(ret).
		Get(requestUrl)

	if err != nil {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_ontology_knowledge_select", err.Error())
	}
	if resp.StatusCode() >= 300 {
		return nil, grpc_util.ErrorStatusWithKey(errs.Code_BFFGeneral, "bff_ontology_knowledge_select", fmt.Sprintf("[%d] http error", resp.StatusCode()))
	}

	// 转换为统一的响应格式
	items := make([]*response.GeneralAgentResourceSelectItem, 0, len(ret.KnowledgeNetworks))
	for _, kn := range ret.KnowledgeNetworks {
		items = append(items, &response.GeneralAgentResourceSelectItem{
			ID:     kn.ID,
			Name:   kn.Name,
			Desc:   kn.Comment,
			Type:   kn.ModuleType,
			Avatar: request.Avatar{Path: config.Cfg().DefaultIcon.KnowledgeIcon},
		})
	}

	return items, nil
}

// checkWgaOntologyKnowledgeConfig 校验 wga Ontology 配置（用于更新配置）
func checkWgaOntologyKnowledgeConfig(ctx *gin.Context, userId, orgId string, ontologyKnowledgeList []*assistant_service.WgaConfigOntologyKnowledge) error {
	if len(ontologyKnowledgeList) == 0 {
		return nil
	}

	ontologyIds := make([]string, 0, len(ontologyKnowledgeList))
	for _, o := range ontologyKnowledgeList {
		ontologyIds = append(ontologyIds, o.OntologyKnowledgeId)
	}

	validIds, err := getValidOntologyIds(ctx, userId, orgId, ontologyIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "ontology not found")
	}

	for _, o := range ontologyKnowledgeList {
		if !validIds[o.OntologyKnowledgeId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr,
				fmt.Sprintf("ontology not found: %s", o.OntologyKnowledgeId))
		}
	}
	return nil
}

// buildWgaOntologyKnowledgeOptions 构建 wga Ontology 配置选项
//
// 当 ontology 功能开启时，根据知识网络配置（@提及优先、配置表兜底）为 Agent 注入
//  1. 系统提示：告知 userId 和知识网络 ID，提示 agent 如何使用这些信息
//  2. 数据查询技能：smart-data-analysis / smart-search-tables / smart-ask-data / ontology-core
func buildWgaOntologyKnowledgeOptions(ctx *gin.Context, userId, orgId, agentId string,
	ontologyKnowledgeList, ontologyKnowledgeMentions []*assistant_service.WgaConfigOntologyKnowledge) ([]wga_option.Option, *schema.Message, error) {
	if config.Cfg().Ontology.Enable == 0 {
		return nil, nil, nil
	}

	var contentBuilder strings.Builder
	_, _ = fmt.Fprintf(&contentBuilder, "【当前用户ID为 %s】如果技能参数需要 userId、user-id、accountId、x-account-id 等相关信息，可以使用此用户ID进行查询。", userId)

	// Ontology Agent 模式：只用于创建本体知识网络，不加载问数技能，仅透传 userId 供 CLI 命令使用
	if agentId == "Ontology Agent" {
		return nil, &schema.Message{
			Role:    schema.System,
			Content: contentBuilder.String(),
		}, nil
	}

	// 非 Ontology Agent 模式：可用于本体知识网络问数，确定知识网络 ID：@提及 > 配置表 > 无配置则只返回 userId 提示；最多只有一个知识网络 ID
	var ontologyKnowledgeId string
	if len(ontologyKnowledgeMentions) > 0 {
		ontologyKnowledgeId = ontologyKnowledgeMentions[0].OntologyKnowledgeId
	} else if len(ontologyKnowledgeList) > 0 {
		ontologyKnowledgeId = ontologyKnowledgeList[0].OntologyKnowledgeId
	} else {
		return nil, &schema.Message{
			Role:    schema.System,
			Content: contentBuilder.String(),
		}, nil
	}

	_, _ = fmt.Fprintf(&contentBuilder, "\n\n【当前配置的知识网络ID为 %s】如果需要用到本体智能体知识网络的智能问答能力，可以使用此知识网络ID作为 kn_id 或 KN ID 等相关参数进行查询。", ontologyKnowledgeId)

	// 从配置读取数据查询技能列表，动态注入
	smartSkills := config.Cfg().Ontology.SmartDataSkills
	var opts []wga_option.Option
	if len(smartSkills) > 0 {
		_, _ = contentBuilder.WriteString("\n\n已加载以下数据查询相关技能，可以使用这些技能访问知识网络中的数据：")
		opts = make([]wga_option.Option, 0, len(smartSkills))
		for _, s := range smartSkills {
			_, _ = fmt.Fprintf(&contentBuilder, "\n- %s: %s", s.Name, s.Desc)
			opts = append(opts, wga_option.WithSkill(wga_option.Skill{
				Dir: s.SkillPath,
			}))
		}
	}

	return opts, &schema.Message{
		Role:    schema.System,
		Content: contentBuilder.String(),
	}, nil
}

// getValidOntologyIds 批量获取有效的 Ontology ID 映射
func getValidOntologyIds(ctx *gin.Context, userId, orgId string, ontologyIds []string) (map[string]bool, error) {
	if len(ontologyIds) == 0 {
		return make(map[string]bool), nil
	}

	// 获取所有 ontology 列表
	items, err := getOntologyKnowledgeSelect(ctx, userId, orgId, "")
	if err != nil {
		return nil, err
	}

	// 构建有效 ID 映射
	validIds := make(map[string]bool)
	for _, item := range items {
		validIds[item.ID] = true
	}

	return validIds, nil
}
