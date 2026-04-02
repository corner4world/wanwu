package response

const (
	defaultKnowledgeAvatar = "/v1/static/icon/agent-knowledge-default-icon.png"
	defaultThinkingAvatar  = "/v1/static/icon/agent-thinking-default-icon.png"
	defaultToolAvatar      = "/v1/static/icon/agent-tool-default-icon.png"
	defaultSkillAvatar     = "/v1/static/icon/agent-skill-default-icon.png"
)

func BuildDefaultAvatarByType(toolEventType int) string {
	switch toolEventType {
	case KnowledgeEventType:
		return defaultKnowledgeAvatar
	case ToolEventType:
		return defaultToolAvatar
	case ThinkingEventType:
		return defaultThinkingAvatar
	case SkillEventType:
		return defaultSkillAvatar
	default:
		return defaultToolAvatar
	}
}
