import service from '@/utils/request';
import { SERVICE_API } from '@/utils/requestConstants';
import { store } from '@/store';

// 基础路径
const BASE_URL = `${SERVICE_API}/general/agent`;

// ==================== 工具选择 ====================

/**
 * 获取工具选择列表
 * 返回按类别分组的工具列表
 * @param {string} agentId - 模式ID（可选）
 */
export const getGeneralAgentToolSelect = params => {
  return service({
    url: `${BASE_URL}/tool/select`,
    method: 'get',
    params,
  });
};

/**
 * 获取工具详情
 * @param {string} toolId - 工具ID
 * @param {string} toolType - 工具类型 (builtin/custom)
 */
export const getGeneralAgentToolInfo = params => {
  return service({
    url: `${BASE_URL}/tool/info`,
    method: 'get',
    params,
  });
};

/**
 * 获取全局配置选择列表
 */
export const getGeneralAgentResourceSelect = params => {
  return service({
    url: `${BASE_URL}/resource/select`,
    method: 'get',
    params,
  });
};

/**
 * 获取智能体选择列表
 * @param {string} name - 智能体名称（可选）
 */
export const getGeneralAgentAssistantSelect = params => {
  return service({
    url: `${BASE_URL}/assistant/select`,
    method: 'get',
    params,
  });
};

/**
 * 获取 MCP 选择列表
 */
export const getGeneralAgentMcpSelect = () => {
  return service({
    url: `${BASE_URL}/mcp/select`,
    method: 'get',
  });
};

/**
 * 获取工作流选择列表
 */
export const getGeneralAgentWorkflowSelect = () => {
  return service({
    url: `${BASE_URL}/workflow/select`,
    method: 'get',
  });
};

/**
 * 获取Skills选择列表
 */
export const getGeneralAgentSkillSelect = () => {
  return service({
    url: `${BASE_URL}/skill/select`,
    method: 'get',
  });
};

/**
 * 获取可选模式列表
 */
export const getGeneralAgentSubList = () => {
  return service({
    url: `${BASE_URL}/sub/list`,
    method: 'get',
  });
};

/**
 * 更新全局配置
 * @param {Array} toolList - 工具列表 [{ toolId, toolType }]
 * @param {Array} assistantList - 智能体列表 [{ assistantId, assistantType }]
 */
export const updateGeneralAgentGlobalConfig = data => {
  return service({
    url: `${BASE_URL}/config`,
    method: 'put',
    data,
  });
};

/**
 * 获取全局配置
 */
export const getGeneralAgentGlobalConfig = () => {
  return service({
    url: `${BASE_URL}/config`,
    method: 'get',
  });
};

// ==================== 对话管理 ====================

/**
 * 创建对话
 * @param {string} title - 对话标题（必填）
 */
export const createGeneralAgentConversation = data => {
  return service({
    url: `${BASE_URL}/conversation`,
    method: 'post',
    data,
  });
};

/**
 * 删除对话
 * @param {string} threadId - 对话ID（必填）
 */
export const deleteGeneralAgentConversation = data => {
  return service({
    url: `${BASE_URL}/conversation`,
    method: 'delete',
    data,
  });
};

/**
 * 获取对话列表
 * @param {number} page - 页码，默认1
 * @param {number} pageSize - 每页数量，默认20
 */
export const getGeneralAgentConversationList = params => {
  return service({
    url: `${BASE_URL}/conversation/list`,
    method: 'get',
    params,
  });
};

/**
 * 获取对话详情（含历史消息）
 * @param {string} threadId - 对话ID
 * @param {number} pageNo - 页码，默认1
 * @param {number} pageSize - 每页数量，默认1000
 */
export const getGeneralAgentConversationDetail = params => {
  return service({
    url: `${BASE_URL}/conversation/detail`,
    method: 'get',
    params,
  });
};

// ==================== 对话配置 ====================

/**
 * 获取对话配置
 * 返回：threadId, modelConfig, toolList, assistantList
 * @param {string} threadId - 对话ID（必填）
 */
export const getGeneralAgentConversationConfig = params => {
  return service({
    url: `${BASE_URL}/conversation/config`,
    method: 'get',
    params,
  });
};

/**
 * 修改对话配置
 * @param {string} threadId - 对话ID（必填）
 * @param {object} modelConfig - 模型配置 { modelId, model, provider, displayName, modelType, config }
 * @param {Array} toolList - 工具列表 [{ toolId, toolType }]
 * @param {Array} assistantList - 智能体列表 [{ assistantId, assistantType }]
 */
export const updateGeneralAgentConversationConfig = data => {
  return service({
    url: `${BASE_URL}/conversation/config`,
    method: 'put',
    data,
  });
};

// ==================== SSE 对话 ====================

/**
 * SSE 流式对话
 * @param {string} threadId - 对话ID（必填）
 * @param {string} agentId - 模式ID（选填）
 * @param {Array} messages - 消息列表 [{ id, role, content }]（必填）
 * @param {function} onMessage - 消息回调
 * @param {function} onError - 错误回调
 * @param {function} onOpen - 连接建立回调
 * @param {AbortSignal} signal - 取消信号
 * @param {number} timeout - 超时时间（毫秒），默认 5 分钟
 */
export const chatGeneralAgentConversation = async ({
  threadId,
  agentId,
  messages,
  onMessage,
  onError,
  onOpen,
  signal,
  timeout = 5 * 60 * 1000,
}) => {
  const token = store.getters['user/token'] || '';
  const user = store.getters['user/userInfo'] || {};
  const url = `${window.location.origin}${BASE_URL}/conversation/chat`;

  // 创建超时控制器
  const timeoutController = new AbortController();
  const timeoutId = setTimeout(() => {
    timeoutController.abort();
  }, timeout);

  // 合并外部 signal 和超时 signal
  const combinedSignal = signal
    ? AbortSignal.any
      ? AbortSignal.any([signal, timeoutController.signal])
      : timeoutController.signal
    : timeoutController.signal;

  let response;
  try {
    response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
        'x-user-id': user.uid || '',
        'x-org-id': user.orgId || '',
      },
      body: JSON.stringify({ threadId, agentId, messages }),
      signal: combinedSignal,
    });
  } catch (error) {
    clearTimeout(timeoutId);
    if (error.name === 'AbortError') {
      // 判断是超时还是用户取消
      if (timeoutController.signal.aborted) {
        onError?.(new Error('请求超时，请重试'));
      } else {
        // 用户主动取消，不报错
      }
    } else {
      onError?.(error);
    }
    return;
  }

  clearTimeout(timeoutId);

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
    try {
      const text = await response.text();
      // 尝试解析 JSON 错误
      try {
        const json = JSON.parse(text);
        errorMessage = json.msg || json.message || json.error || text;
      } catch {
        // 不是 JSON，直接使用文本
        if (text) {
          errorMessage = text;
        }
      }
    } catch (e) {
      // 无法读取响应内容
    }
    onError?.(new Error(errorMessage));
    return;
  }

  onOpen?.();

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data.trim()) {
            try {
              const event = JSON.parse(data);
              onMessage?.(event);
            } catch (e) {
              console.warn('Failed to parse SSE event:', data);
            }
          }
        }
      }
    }
  } catch (error) {
    if (error.name !== 'AbortError') {
      onError?.(error);
    }
  }
};

// ==================== Workspace ====================

/**
 * 获取 Workspace 目录树
 * @param {string} threadId - 对话ID
 * @param {string} runId - 运行ID
 * @param {string} path - 目录路径（可选，用于递归浏览）
 */
export const getGeneralAgentWorkspace = params => {
  return service({
    url: `${BASE_URL}/conversation/workspace`,
    method: 'get',
    params,
  });
};

/**
 * 下载 Workspace 文件
 * @param {string} threadId - 对话ID
 * @param {string} runId - 运行ID
 * @param {string} path - 文件路径
 */
export const downloadGeneralAgentWorkspace = params => {
  return service({
    url: `${BASE_URL}/conversation/workspace/download`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

/**
 * 预览 Workspace 文件
 * @param {string} threadId - 对话ID
 * @param {string} runId - 运行ID
 * @param {string} path - 文件路径
 */
export const previewGeneralAgentWorkspace = params => {
  return service({
    url: `${BASE_URL}/conversation/workspace/preview`,
    method: 'get',
    params,
    responseType: 'blob',
  });
};

/**
 * 检查对话配置是否满足条件
 * @param {string} agentId - 模式ID
 * @param {string} threadId - 对话ID
 */
export const checkGeneralAgentConversationConfig = data => {
  return service({
    url: `${BASE_URL}/conversation/config/check`,
    method: 'post',
    data,
  });
};

// ==================== Human-in-the-Loop ====================

/**
 * 回答问题
 * @param {string} runId - 运行ID
 * @param {string} questionId - 问题ID
 * @param {Array<Array<string>>} answers - 答案数组
 */
export const replyQuestion = data => {
  return service({
    url: `${BASE_URL}/question/reply`,
    method: 'post',
    data,
  });
};

/**
 * 拒绝问题
 * @param {string} runId - 运行ID
 * @param {string} questionId - 问题ID
 */
export const rejectQuestion = data => {
  return service({
    url: `${BASE_URL}/question/reject`,
    method: 'post',
    data,
  });
};

// ==================== skill相关 ====================
/**
 * 创建 Skill 专用对话
 * @param {string} title - 对话标题（必填）
 * @param {object} modelConfig - 模型配置（必填）
 */
export const createGeneralAgentSkillConversation = data => {
  return service({
    url: `${BASE_URL}/skill/conversation`,
    method: 'post',
    data,
  });
};

/**
 * 导入 Skill 专用对话
 * @param {string} zipUrl - Skill zip 文件地址（必填）
 * @param {object} modelConfig - 模型配置（必填）
 */
export const importGeneralAgentSkillConversation = data => {
  return service({
    url: `${BASE_URL}/skill/import/conversation`,
    method: 'post',
    data,
  });
};

/**
 * Skill SSE 流式对话
 * @param {string} customSkillId - Skill 工作区 ID（必填）
 * @param {string} threadId - 编辑对话 ID（必填）
 * @param {string} mode - 对话模式 (normal/import/convert/preview)
 * @param {string} previewId - 预览对话 ID（preview 模式必填）
 * @param {Array} messages - 消息列表 [{ id, role, content }]（必填）
 * @param {function} onMessage - 消息回调
 * @param {function} onError - 错误回调
 * @param {function} onOpen - 连接建立回调
 * @param {AbortSignal} signal - 取消信号
 * @param {number} timeout - 超时时间（毫秒），默认 5 分钟
 */
export const chatGeneralAgentSkillConversation = async ({
  customSkillId,
  threadId,
  mode,
  previewId,
  messages,
  onMessage,
  onError,
  onOpen,
  signal,
  timeout = 5 * 60 * 1000,
}) => {
  const token = store.getters['user/token'] || '';
  const user = store.getters['user/userInfo'] || {};
  const url = `${window.location.origin}${BASE_URL}/skill/conversation/chat`;

  const timeoutController = new AbortController();
  const timeoutId = setTimeout(() => {
    timeoutController.abort();
  }, timeout);

  const combinedSignal = signal
    ? AbortSignal.any
      ? AbortSignal.any([signal, timeoutController.signal])
      : timeoutController.signal
    : timeoutController.signal;

  let response;
  try {
    response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${token}`,
        'x-user-id': user.uid || '',
        'x-org-id': user.orgId || '',
      },
      body: JSON.stringify({
        customSkillId,
        threadId,
        mode,
        previewId,
        messages,
      }),
      signal: combinedSignal,
    });
  } catch (error) {
    clearTimeout(timeoutId);
    if (error.name === 'AbortError') {
      if (timeoutController.signal.aborted) {
        onError?.(new Error('请求超时，请重试'));
      }
    } else {
      onError?.(error);
    }
    return;
  }

  clearTimeout(timeoutId);

  if (!response.ok) {
    let errorMessage = `HTTP ${response.status}: ${response.statusText}`;
    try {
      const text = await response.text();
      try {
        const json = JSON.parse(text);
        errorMessage = json.msg || json.message || json.error || text;
      } catch {
        if (text) {
          errorMessage = text;
        }
      }
    } catch (e) {
      // 无法读取响应内容
    }
    onError?.(new Error(errorMessage));
    return;
  }

  onOpen?.();

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';

  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) break;

      buffer += decoder.decode(value, { stream: true });
      const lines = buffer.split('\n');
      buffer = lines.pop() || '';

      for (const line of lines) {
        if (line.startsWith('data: ')) {
          const data = line.slice(6);
          if (data.trim()) {
            try {
              const event = JSON.parse(data);
              onMessage?.(event);
            } catch (e) {
              console.warn('Failed to parse SSE event:', data);
            }
          }
        }
      }
    }
  } catch (error) {
    if (error.name !== 'AbortError') {
      onError?.(error);
    }
  }
};

/**
 * 获取 Skill preview 对话详情
 * @param {string} previewId - 预览对话 ID（必填）
 */
export const getGeneralAgentSkillPreviewConversationDetail = params => {
  return service({
    url: `${BASE_URL}/skill/preview/conversation/detail`,
    method: 'get',
    params,
  });
};

/**
 * 一键转化为 Skill 专用对话
 * @param {Object} data - 请求数据
 * @param {string} data.id - 待转化资源 ID（必填）
 * @param {string} data.type - 资源类型 (mcp/tool/agent/workflow/rag)（必填）
 * @param {Object} data.modelConfig - 模型配置（必填）
 * @param {string} [data.author] - 作者（选填）
 */
export const createConvertSkillConversation = data => {
  return service({
    url: `${BASE_URL}/skill/convert/conversation`,
    method: 'post',
    data,
  });
};

/**
 * 刷新 Skill 专用对话
 * @param {Object} data - 请求数据
 * @param {string} data.skillId - 已存在的 custom skill ID（必填）
 */
export const refreshGeneralAgentSkillConversation = data => {
  return service({
    url: `${BASE_URL}/skill/refresh/conversation`,
    method: 'post',
    data,
  });
};
