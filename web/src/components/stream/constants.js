export const AGENT_MESSAGE_CONFIG = {
  // 主智能体
  MAIN_AGENT: {
    EVENT_TYPE: 0,
    CONVERSATION_TYPE: '',
  },
  // 子智能体
  SUB_AGENT: {
    EVENT_TYPE: 1,
    CONVERSATION_TYPE: 'subAgent',
  },
  // 智能体-知识库
  AGENT_KNOWLEDGE: {
    EVENT_TYPE: 2,
    CONVERSATION_TYPE: 'agentKnowledge',
  },
  // 智能体-工具
  AGENT_TOOL: {
    EVENT_TYPE: 3,
    CONVERSATION_TYPE: 'agentTool',
  },
  // 智能体-skill
  AGENT_SKILL: {
    EVENT_TYPE: 4,
    CONVERSATION_TYPE: 'agentSkill',
  },
  // 智能体-思考
  AGENT_THINK: {
    EVENT_TYPE: 6,
    CONVERSATION_TYPE: 'agentThink',
  },
  // 智能体-子会话正文文本分段
  SUB_TEXT: {
    EVENT_TYPE: 20,
    CONVERSATION_TYPE: 'subText',
  },
};

export const AGENT_SSE_EVENT_TYPES = Object.fromEntries(
  Object.entries(AGENT_MESSAGE_CONFIG).map(([key, val]) => [
    key,
    val.EVENT_TYPE,
  ]),
);
