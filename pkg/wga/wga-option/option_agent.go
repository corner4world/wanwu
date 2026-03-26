package wga_option

import (
	"sort"

	"github.com/UnicomAI/wanwu/pkg/wga/internal/config"
)

// ToolCategoryCondition 工具类别条件
type ToolCategoryCondition = config.ToolCategoryCondition

const (
	ToolCategoryConditionNone     = config.ToolCategoryConditionNone     // 无需检查，该类别下的工具都是可选项
	ToolCategoryConditionOptional = config.ToolCategoryConditionOptional // 该类别下至少有一个工具完成配置
	ToolCategoryConditionRequired = config.ToolCategoryConditionRequired // 该类别下所有工具完成配置
)

// ToolCategoryInfo 工具类别信息
type ToolCategoryInfo struct {
	Category  string
	Condition ToolCategoryCondition
	Tools     []ToolInfo
}

// ToolInfo 工具信息
type ToolInfo struct {
	Title        string
	Description  string
	AuthRequired bool
	Operations   []string
}

// CollectToolCategories 递归收集 agent 及其 sub agents 的 tool categories，扁平合并。
// 合并规则：
//   - 相同 category 的 condition 取最严格：required > optional > none
//   - 相同 tool 的 authRequired 取 true（任一需要即需要）
//   - 相同 tool 的 operations 合并去重
//
// 排序规则：
//   - Category: required → optional → none，然后按 category name
//   - Tool: need auth → no auth，然后按 tool title
//   - Operations: 按 string 排序
func CollectToolCategories(agent *config.Agent) []ToolCategoryInfo {
	categoryMap := make(map[string]*ToolCategoryInfo)

	collectToolCategoriesRecursive(agent, categoryMap)

	result := make([]ToolCategoryInfo, 0, len(categoryMap))
	for _, cat := range categoryMap {
		sortTools(cat.Tools)
		result = append(result, *cat)
	}

	sort.Slice(result, func(i, j int) bool {
		ci := conditionPriority(result[i].Condition)
		cj := conditionPriority(result[j].Condition)
		if ci != cj {
			return ci < cj
		}
		return result[i].Category < result[j].Category
	})

	return result
}

// collectToolCategoriesRecursive 递归收集 agent 及其 sub agents 的 tool categories 到 categoryMap
func collectToolCategoriesRecursive(agent *config.Agent, categoryMap map[string]*ToolCategoryInfo) {
	for _, tc := range agent.ToolCategories {
		existing, ok := categoryMap[string(tc.Category)]
		if !ok {
			// 首次遇到该 category，直接创建
			existing = &ToolCategoryInfo{
				Category:  string(tc.Category),
				Condition: tc.Condition,
				Tools:     make([]ToolInfo, 0),
			}
			categoryMap[string(tc.Category)] = existing
		} else {
			// 已存在，合并 condition（取最严格）
			existing.Condition = mergeCondition(existing.Condition, tc.Condition)
		}

		// 构建已存在 tool 的索引，便于查找和合并
		toolMap := make(map[string]*ToolInfo)
		for i := range existing.Tools {
			toolMap[existing.Tools[i].Title] = &existing.Tools[i]
		}

		for _, tool := range tc.Tools {
			title := tool.Doc.Info.Title
			if existingTool, toolOk := toolMap[title]; toolOk {
				// 已存在，合并
				if tool.AuthRequired {
					existingTool.AuthRequired = true
				}
				// 合并 operations
				opMap := make(map[string]bool)
				for _, op := range existingTool.Operations {
					opMap[op] = true
				}
				for _, op := range tool.Operations {
					if !opMap[op.OperationID] {
						existingTool.Operations = append(existingTool.Operations, op.OperationID)
						opMap[op.OperationID] = true
					}
				}
			} else {
				// 首次遇到，创建
				ops := make([]string, 0, len(tool.Operations))
				for _, op := range tool.Operations {
					ops = append(ops, op.OperationID)
				}
				description := ""
				if tool.Doc.Info.Description != "" {
					description = tool.Doc.Info.Description
				}
				newTool := ToolInfo{
					Title:        title,
					Description:  description,
					AuthRequired: tool.AuthRequired,
					Operations:   ops,
				}
				existing.Tools = append(existing.Tools, newTool)
				toolMap[title] = &existing.Tools[len(existing.Tools)-1]
			}
		}
	}

	// 递归处理 sub agents
	for _, subAgent := range agent.SubAgents {
		collectToolCategoriesRecursive(subAgent, categoryMap)
	}
}

// mergeCondition 合并两个 condition，返回更严格的那个
func mergeCondition(c1, c2 ToolCategoryCondition) ToolCategoryCondition {
	if c1 == ToolCategoryConditionRequired || c2 == ToolCategoryConditionRequired {
		return ToolCategoryConditionRequired
	}
	if c1 == ToolCategoryConditionOptional || c2 == ToolCategoryConditionOptional {
		return ToolCategoryConditionOptional
	}
	return ToolCategoryConditionNone
}

// conditionPriority 返回 condition 的排序优先级（数值越小越靠前）
func conditionPriority(c ToolCategoryCondition) int {
	switch c {
	case ToolCategoryConditionRequired:
		return 0
	case ToolCategoryConditionOptional:
		return 1
	default:
		return 2
	}
}

// sortTools 对工具列表排序：need auth → no auth，然后按 title
func sortTools(tools []ToolInfo) {
	sort.Slice(tools, func(i, j int) bool {
		if tools[i].AuthRequired != tools[j].AuthRequired {
			return tools[i].AuthRequired
		}
		return tools[i].Title < tools[j].Title
	})
	for i := range tools {
		sort.Strings(tools[i].Operations)
	}
}
