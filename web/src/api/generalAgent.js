import service from '@/utils/request';
import { SERVICE_API } from '@/utils/requestConstants';
import { store } from '@/store/index';
import { USER_API } from '@/utils/requestConstants';

// 基础路径
const BASE_URL = `${SERVICE_API}/general/agent`;

// 获取 token
const getToken = () => {
  return store.getters['user/token'] || '';
};

// 获取用户信息
const getUserInfo = () => {
  const user = store.getters['user/userInfo'] || {};
  return {
    userId: user.uid || '',
    orgId: user.orgId || '',
  };
};

// ==================== 模型选择 ====================

/**
 * 获取LLM模型列表
 */
export const getLlmModelSelect = () => {
  return service({
    url: `${SERVICE_API}/model/select/llm`,
    method: 'get',
  });
};

// ==================== 智能体选择 ====================

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

// ==================== 工具选择 ====================

/**
 * 获取工具选择列表
 * 返回按类别分组的工具列表
 */
export const getGeneralAgentToolSelect = () => {
  return service({
    url: `${BASE_URL}/tool/select`,
    method: 'get',
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
 * 更新工具配置
 * @param {Array} toolList - 工具列表 [{ toolId, toolType }]
 * @param {Array} assistantList - 智能体列表 [{ assistantId, assistantType }]
 * @deprecated 请使用 updateGeneralAgentConfig
 */
export const updateGeneralAgentToolConfig = data => {
  return service({
    url: `${BASE_URL}/conversation/config`,
    method: 'put',
    data,
  });
};

/**
 * 获取工具配置
 * @deprecated 请使用 getGeneralAgentConfig
 */
export const getGeneralAgentToolConfig = () => {
  return service({
    url: `${BASE_URL}/conversation/config`,
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
export const getGeneralAgentConfig = params => {
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
export const updateGeneralAgentConfig = data => {
  return service({
    url: `${BASE_URL}/conversation/config`,
    method: 'put',
    data,
  });
};

/**
 * 检查配置有效性
 * @param {string} threadId - 对话ID（必填）
 */
export const checkGeneralAgentConfig = data => {
  return service({
    url: `${BASE_URL}/conversation/config/check`,
    method: 'post',
    data,
  });
};

// ==================== SSE 对话 ====================

/**
 * SSE 流式对话
 * @param {string} threadId - 对话ID（必填）
 * @param {Array} messages - 消息列表 [{ id, role, content }]（必填）
 * @param {function} onMessage - 消息回调
 * @param {function} onError - 错误回调
 * @param {function} onOpen - 连接建立回调
 * @param {AbortSignal} signal - 取消信号
 * @param {number} timeout - 超时时间（毫秒），默认 5 分钟
 */
export const chatGeneralAgentConversation = async ({
  threadId,
  messages,
  onMessage,
  onError,
  onOpen,
  signal,
  timeout = 5 * 60 * 1000,
}) => {
  const token = getToken();
  const { userId, orgId } = getUserInfo();
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
        Authorization: token ? `Bearer ${token}` : '',
        'x-user-id': userId,
        'x-org-id': orgId,
      },
      body: JSON.stringify({ threadId, messages }),
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

// ==================== CopilotKit ====================

/**
 * CopilotKit 协议端点
 * @param {string} method - 方法名 (info, agent/connect, agent/run, agent/stop)
 * @param {object} params - 参数
 * @param {object} body - 请求体
 */
export const generalAgentCopilotRuntime = data => {
  return service({
    url: `${BASE_URL}/copilotkit`,
    method: 'post',
    data,
  });
};

/**
 * 上传文件到通用智能体（直接上传）
 * @param {File} file - 文件对象
 * @param {function} onProgress - 进度回调 (percent) => {}
 * @returns {Promise<{code: number, data: {files: Array}, msg: string}>}
 */
export const uploadGeneralAgentFile = (file, onProgress) => {
  const formData = new FormData();
  formData.append('files', file);
  return service({
    url: `${SERVICE_API}/file/upload/direct`,
    method: 'post',
    data: formData,
    headers: {
      'Content-Type': 'multipart/form-data',
    },
    onUploadProgress: progressEvent => {
      if (onProgress && progressEvent.total) {
        const percent = Math.round(
          (progressEvent.loaded / progressEvent.total) * 100,
        );
        onProgress(percent);
      }
    },
  });
};
