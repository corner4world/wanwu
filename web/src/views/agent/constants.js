export const SINGLE_AGENT = 1;
export const MULTIPLE_AGENT = 2;
export const AGENT_CONFIG_RECOMMEND_CONFIG_MODEL_CONFIG_DEFAULT_CONFIG = {
  temperature: 0.7,
  temperatureEnable: true,
  topP: 1,
  topPEnable: true,
  frequencyPenalty: 0,
  frequencyPenaltyEnable: true,
  presencePenalty: 0,
  presencePenaltyEnable: true,
  maxTokens: 512,
  maxTokensEnable: true,
};
export const AGENT_TOOL_TYPE = {
  TOOL: 'tool',
  MCP: 'mcp',
  WORKFLOW: 'workflow',
  SKILL: 'skill',
};
