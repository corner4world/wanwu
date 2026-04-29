package service

import (
	"regexp"
	"sync"

	assistant_service "github.com/UnicomAI/wanwu/api/proto/assistant-service"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/request"
	"github.com/UnicomAI/wanwu/internal/bff-service/model/response"
	"github.com/UnicomAI/wanwu/pkg/constant"
	"github.com/UnicomAI/wanwu/pkg/log"
	"github.com/UnicomAI/wanwu/pkg/util"
	"github.com/gin-gonic/gin"
)

// wgaResourceNameRegex 用于从文本中提取 @提及 的资源名称。
// 格式: @name，支持中文、英文字母、数字、下划线和连字符。
// 例如: "@工具名"、"@workflow-1"、"@skill_2"
var wgaResourceNameRegex = regexp.MustCompile(`@([\p{Han}a-zA-Z0-9_-]+)`)

func GetGeneralAgentResourceSelect(ctx *gin.Context, userId, orgId string, name string) ([]*response.GeneralAgentResourceSelectList, error) {
	result := make([]*response.GeneralAgentResourceSelectList, 0, 5)

	// 并发获取五种资源列表
	var wg sync.WaitGroup
	var mcpErr, workflowErr, skillErr, assistantErr, knowledgeErr error
	var mcpList []response.MCPSelect
	var workflowList []*response.ExplorationAppInfo
	var skillList []*response.SkillInfo
	var assistantList []*response.ExplorationAppInfo
	var knowledgeList *response.KnowledgeListResp

	// 获取 MCP 列表
	wg.Add(1)
	go func() {
		defer util.PrintPanicStack()
		defer wg.Done()
		resp, err := GetMCPSelect(ctx, userId, orgId, name)
		if err != nil {
			mcpErr = err
			return
		}
		if list, ok := resp.List.([]response.MCPSelect); ok {
			mcpList = list
		}
	}()

	// 获取 Workflow 列表
	wg.Add(1)
	go func() {
		defer util.PrintPanicStack()
		defer wg.Done()
		resp, err := GetWorkflowSelect(ctx, userId, orgId, request.GetExplorationAppListRequest{Name: name})
		if err != nil {
			workflowErr = err
			return
		}
		if list, ok := resp.List.([]*response.ExplorationAppInfo); ok {
			workflowList = list
		}
	}()

	// 获取 Skill 列表
	wg.Add(1)
	go func() {
		defer util.PrintPanicStack()
		defer wg.Done()
		resp, err := GetSkillSelect(ctx, userId, orgId, name, constant.SkillTypeCustom)
		if err != nil {
			skillErr = err
			return
		}
		if list, ok := resp.List.([]*response.SkillInfo); ok {
			skillList = list
		}
	}()

	// 获取 Assistant 列表
	wg.Add(1)
	go func() {
		defer util.PrintPanicStack()
		defer wg.Done()
		resp, err := GetAssistantSelect(ctx, userId, orgId, request.GetExplorationAppListRequest{Name: name})
		if err != nil {
			assistantErr = err
			return
		}
		if list, ok := resp.List.([]*response.ExplorationAppInfo); ok {
			assistantList = list
		}
	}()

	// 获取 Knowledge 列表
	wg.Add(1)
	go func() {
		defer util.PrintPanicStack()
		defer wg.Done()
		resp, err := SelectKnowledgeList(ctx, userId, orgId, &request.KnowledgeSelectReq{Name: name})
		if err != nil {
			knowledgeErr = err
			return
		}
		knowledgeList = resp
	}()

	wg.Wait()

	// 检查错误
	if mcpErr != nil {
		return nil, mcpErr
	}
	if workflowErr != nil {
		return nil, workflowErr
	}
	if skillErr != nil {
		return nil, skillErr
	}
	if assistantErr != nil {
		return nil, assistantErr
	}
	if knowledgeErr != nil {
		return nil, knowledgeErr
	}

	// 构建 MCP 列表
	mcpItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(mcpList))
	for _, item := range mcpList {
		mcpItems = append(mcpItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.MCPID,
			Name:   item.Name,
			Desc:   item.Description,
			Avatar: item.Avatar,
			Type:   item.Type,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "mcp",
		List:     mcpItems,
	})

	// 构建 Workflow 列表
	workflowItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(workflowList))
	for _, item := range workflowList {
		workflowItems = append(workflowItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.AppId,
			Name:   item.Name,
			Desc:   item.Desc,
			Avatar: item.Avatar,
			Type:   item.AppType,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "workflow",
		List:     workflowItems,
	})

	// 构建 Skill 列表
	skillItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(skillList))
	for _, item := range skillList {
		skillItems = append(skillItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.SkillId,
			Name:   item.SkillName,
			Desc:   item.Desc,
			Avatar: item.Avatar,
			Type:   item.SkillType,
			Author: item.Author,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "skill",
		List:     skillItems,
	})

	// 构建 Assistant 列表
	assistantItems := make([]*response.GeneralAgentResourceSelectItem, 0, len(assistantList))
	for _, item := range assistantList {
		assistantItems = append(assistantItems, &response.GeneralAgentResourceSelectItem{
			ID:     item.AppId,
			Name:   item.Name,
			Desc:   item.Desc,
			Avatar: item.Avatar,
			Type:   item.AppType,
		})
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "assistant",
		List:     assistantItems,
	})

	// 构建 Knowledge 列表
	knowledgeItems := make([]*response.GeneralAgentResourceSelectItem, 0)
	if knowledgeList != nil {
		knowledgeItems = make([]*response.GeneralAgentResourceSelectItem, 0, len(knowledgeList.KnowledgeList))
		for _, item := range knowledgeList.KnowledgeList {
			knowledgeItems = append(knowledgeItems, &response.GeneralAgentResourceSelectItem{
				ID:     item.KnowledgeId,
				Name:   item.Name,
				Desc:   item.Description,
				Avatar: item.Avatar,
				Type:   "knowledge",
			})
		}
	}
	result = append(result, &response.GeneralAgentResourceSelectList{
		ListType: "knowledge",
		List:     knowledgeItems,
	})

	return result, nil
}

// parseResourceMentions 从用户消息中解析 @ 提及的资源名称
// 格式: @资源名称 后面跟空格或消息结束
// 支持: "@mcp1 @workflow2 请帮我处理" -> ["mcp1", "workflow2"]
// 返回: 提取到的资源名称列表（去重后）
func parseWgaResourceMentions(content interface{}) []string {
	var text string
	switch v := content.(type) {
	case string:
		text = v
	case []interface{}:
		// 处理多部分消息，提取文本部分
		for _, item := range v {
			if m, ok := item.(map[string]interface{}); ok {
				if typ, _ := m["type"].(string); typ == "text" {
					if t, _ := m["text"].(string); t != "" {
						text += t + " "
					}
				}
			}
		}
	}

	// 使用正则提取 @name 格式，支持中文、英文、数字、下划线、连字符
	matches := wgaResourceNameRegex.FindAllStringSubmatch(text, -1)

	// 去重
	seen := make(map[string]bool)
	var names []string
	for _, m := range matches {
		if len(m) > 1 && !seen[m[1]] {
			seen[m[1]] = true
			names = append(names, m[1])
		}
	}

	return names
}

// wgaMentionResources @提及的资源列表
type wgaMentionResources struct {
	McpList       []*assistant_service.WgaConfigMcp
	WorkflowList  []*assistant_service.WgaConfigWorkflow
	SkillList     []*assistant_service.WgaConfigSkill
	AssistantList []*assistant_service.WgaConfigAssistant
	KnowledgeList []*assistant_service.WgaConfigKnowledge
	// 用于构建系统消息
	McpItems       []*response.GeneralAgentResourceSelectItem
	WorkflowItems  []*response.GeneralAgentResourceSelectItem
	SkillItems     []*response.GeneralAgentResourceSelectItem
	AssistantItems []*response.GeneralAgentResourceSelectItem
	KnowledgeItems []*response.GeneralAgentResourceSelectItem
}

// hasResources 检查是否有任何资源
func (r *wgaMentionResources) hasResources() bool {
	return len(r.McpItems) > 0 ||
		len(r.WorkflowItems) > 0 ||
		len(r.SkillItems) > 0 ||
		len(r.AssistantItems) > 0 ||
		len(r.KnowledgeItems) > 0
}

// fetchWgaMentionResources 获取@提及的资源列表
// 通过名称搜索下拉资源列表，返回各类型的资源列表
func fetchWgaMentionResources(ctx *gin.Context, userID, orgID string, mentionNames []string) *wgaMentionResources {
	result := &wgaMentionResources{
		McpList:        make([]*assistant_service.WgaConfigMcp, 0),
		WorkflowList:   make([]*assistant_service.WgaConfigWorkflow, 0),
		SkillList:      make([]*assistant_service.WgaConfigSkill, 0),
		AssistantList:  make([]*assistant_service.WgaConfigAssistant, 0),
		KnowledgeList:  make([]*assistant_service.WgaConfigKnowledge, 0),
		McpItems:       make([]*response.GeneralAgentResourceSelectItem, 0),
		WorkflowItems:  make([]*response.GeneralAgentResourceSelectItem, 0),
		SkillItems:     make([]*response.GeneralAgentResourceSelectItem, 0),
		AssistantItems: make([]*response.GeneralAgentResourceSelectItem, 0),
		KnowledgeItems: make([]*response.GeneralAgentResourceSelectItem, 0),
	}

	if len(mentionNames) == 0 {
		return result
	}

	// 对每个名称调用搜索接口
	for _, name := range mentionNames {
		searchResult, err := GetGeneralAgentResourceSelect(ctx, userID, orgID, name)
		if err != nil {
			log.Warnf("[wga] search resources by name '%s' failed: %v", name, err)
			continue
		}

		// 遍历资源列表，按类型收集
		for _, group := range searchResult {
			switch group.ListType {
			case "mcp":
				for _, item := range group.List {
					result.McpList = append(result.McpList, &assistant_service.WgaConfigMcp{
						McpId:   item.ID,
						McpType: item.Type,
					})
					result.McpItems = append(result.McpItems, item)
				}
			case "workflow":
				for _, item := range group.List {
					result.WorkflowList = append(result.WorkflowList, &assistant_service.WgaConfigWorkflow{
						WorkflowId: item.ID,
					})
					result.WorkflowItems = append(result.WorkflowItems, item)
				}
			case "skill":
				for _, item := range group.List {
					result.SkillList = append(result.SkillList, &assistant_service.WgaConfigSkill{
						SkillId:   item.ID,
						SkillType: constant.SkillTypeCustom,
					})
					result.SkillItems = append(result.SkillItems, item)
				}
			case "assistant":
				for _, item := range group.List {
					result.AssistantList = append(result.AssistantList, &assistant_service.WgaConfigAssistant{
						AssistantId:   item.ID,
						AssistantType: util.Int2Str(constant.AgentCategorySingle),
					})
					result.AssistantItems = append(result.AssistantItems, item)
				}
			case "knowledge":
				for _, item := range group.List {
					result.KnowledgeList = append(result.KnowledgeList, &assistant_service.WgaConfigKnowledge{
						KnowledgeId: item.ID,
					})
					result.KnowledgeItems = append(result.KnowledgeItems, item)
				}
			}
		}
	}

	return result
}
