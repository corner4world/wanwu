package service

import (
	"fmt"
	"net/url"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	errs "github.com/UnicomAI/wanwu/api/proto/err-code"
	"github.com/UnicomAI/wanwu/internal/bff-service/config"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	grpc_util "github.com/UnicomAI/wanwu/pkg/grpc-util"
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

// checkWgaOntologyConfig 校验 wga Ontology 配置（用于更新配置）
func checkWgaOntologyConfig(ctx *gin.Context, userId, orgId string, ontologyList []*assistant_service.WgaConfigOntologyKnowledge) error {
	if len(ontologyList) == 0 {
		return nil
	}

	ontologyIds := make([]string, 0, len(ontologyList))
	for _, o := range ontologyList {
		ontologyIds = append(ontologyIds, o.OntologyKnowledgeId)
	}

	validIds, err := getValidOntologyIds(ctx, userId, orgId, ontologyIds)
	if err != nil {
		return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr, "ontology not found")
	}

	for _, o := range ontologyList {
		if !validIds[o.OntologyKnowledgeId] {
			return grpc_util.ErrorStatus(errs.Code_WgaConfigCheckErr,
				fmt.Sprintf("ontology not found: %s", o.OntologyKnowledgeId))
		}
	}
	return nil
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
