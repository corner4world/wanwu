package v1

import (
	"net/http"

	v1 "github.com/UnicomAI/wanwu/internal/bff-service/server/http/handler/v1"
	mid "github.com/UnicomAI/wanwu/pkg/gin-util/mid-wrap"
	"github.com/gin-gonic/gin"
)

func registerAgentSkill(apiV1 *gin.RouterGroup) {
	// 自定义 skill（CRUD）
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom/list", http.MethodGet, v1.GetCustomSkillList, "获取自定义skill列表")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom/detail", http.MethodGet, v1.GetCustomSkillDetail, "获取自定义skill详情")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom/check", http.MethodPost, v1.CheckCustomSkill, "校验自定义skill zip包")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom", http.MethodPost, v1.CreateCustomSkill, "创建自定义skill")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom", http.MethodDelete, v1.DeleteCustomSkill, "删除自定义skill")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom/config", http.MethodPost, v1.CreateCustomSkillConfig, "新增自定义skill配置")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom/config", http.MethodPut, v1.UpdateCustomSkillConfig, "编辑自定义skill配置")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/custom/config", http.MethodDelete, v1.DeleteCustomSkillConfig, "删除自定义skill配置")

	// 内置 skill
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/builtin/list", http.MethodGet, v1.GetBuiltinSkillList, "获取内置skill列表")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/builtin/detail", http.MethodGet, v1.GetBuiltinSkillDetail, "获取内置skill详情")
	mid.Sub("resource.skill").Reg(apiV1, "/builtin/skill/download", http.MethodGet, v1.DownloadBuiltinSkill, "下载内置skill")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/builtin/config", http.MethodPost, v1.CreateBuiltinSkillConfig, "新增内置skill配置")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/builtin/config", http.MethodPut, v1.UpdateBuiltinSkillConfig, "编辑内置skill配置")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/builtin/config", http.MethodDelete, v1.DeleteBuiltinSkillConfig, "删除内置skill配置")

	// 自定义与内建 skill
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/select", http.MethodGet, v1.GetSkillSelect, "智能体skills下拉列表")

	// 资源库 - 我添加的 skill（acquired skill）
	mid.Sub("resource.skill").Reg(apiV1, "/agent/acquired/skill/list", http.MethodGet, v1.GetAcquiredSkillList, "获取我添加的skill列表")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/acquired/skill", http.MethodDelete, v1.DeleteAcquiredSkill, "删除我添加的skill")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/acquired/skill/detail", http.MethodGet, v1.GetAcquiredSkillDetail, "获取我添加的skill详情")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/acquired/skill/config", http.MethodPost, v1.CreateAcquiredSkillConfig, "新增我添加的skill配置")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/acquired/skill/config", http.MethodPut, v1.UpdateAcquiredSkillConfig, "编辑我添加的skill配置")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/acquired/skill/config", http.MethodDelete, v1.DeleteAcquiredSkillConfig, "删除我添加的skill配置")

	// skills conversation
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation", http.MethodPost, v1.CreateSkillConversation, "创建Skill生成会话")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation", http.MethodDelete, v1.DeleteSkillConversation, "删除Skill生成会话")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation/clear", http.MethodDelete, v1.ClearSkillConversation, "清除Skill生成会话对话记录")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation/detail", http.MethodGet, v1.GetSkillConversationDetail, "获取Skill生成会话详情")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation/list", http.MethodGet, v1.GetSkillConversationList, "获取Skill生成会话列表")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation/chat", http.MethodPost, v1.SkillConversationChat, "Skill生成流式对话")
	mid.Sub("resource.skill").Reg(apiV1, "/agent/skill/conversation/save", http.MethodPost, v1.SkillConversationSave, "将Skill发送到资源库")
}
