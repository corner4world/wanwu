/**
 * SSE 事件解析器
 * 用于解析 AG-UI 协议的 SSE 事件
 */

// 事件类型常量
export const EventType = {
  // 运行生命周期
  RUN_STARTED: 'RUN_STARTED',
  RUN_FINISHED: 'RUN_FINISHED',

  // 文本消息
  TEXT_MESSAGE_START: 'TEXT_MESSAGE_START',
  TEXT_MESSAGE_CONTENT: 'TEXT_MESSAGE_CONTENT',
  TEXT_MESSAGE_END: 'TEXT_MESSAGE_END',

  // 工具调用
  TOOL_CALL_START: 'TOOL_CALL_START',
  TOOL_CALL_ARGS: 'TOOL_CALL_ARGS',
  TOOL_CALL_END: 'TOOL_CALL_END',
  TOOL_CALL_RESULT: 'TOOL_CALL_RESULT',

  // 推理过程
  REASONING_START: 'REASONING_START',
  REASONING_MESSAGE_START: 'REASONING_MESSAGE_START',
  REASONING_MESSAGE_CONTENT: 'REASONING_MESSAGE_CONTENT',
  REASONING_MESSAGE_END: 'REASONING_MESSAGE_END',
  REASONING_END: 'REASONING_END',

  // 活动快照
  ACTIVITY_SNAPSHOT: 'ACTIVITY_SNAPSHOT',
};

// 活动类型
export const ActivityType = {
  SUB_AGENT: 'sub_agent',
  WORKSPACE: 'workspace',
};

/**
 * SSE 事件解析器类
 */
export class SSEEventParser {
  constructor() {
    // 消息状态
    this.currentMessageId = null;
    this.currentToolCallId = null;
    this.isReasoning = false;
    this.isReasoningMessage = false;
  }

  /**
   * 解析 SSE 事件
   * @param {object} event - 原始事件对象
   * @returns {object|null} 解析后的事件对象
   */
  parse(event) {
    if (!event || !event.type) {
      return null;
    }

    const baseEvent = {
      type: event.type,
      raw: event,
    };

    switch (event.type) {
      case EventType.RUN_STARTED:
      case EventType.RUN_FINISHED:
        return {
          ...baseEvent,
          threadId: event.threadId,
          runId: event.runId,
        };

      case EventType.TEXT_MESSAGE_START:
        this.currentMessageId = event.messageId;
        return {
          ...baseEvent,
          messageId: event.messageId,
          role: event.role || 'assistant',
        };

      case EventType.TEXT_MESSAGE_CONTENT:
        return {
          ...baseEvent,
          messageId: event.messageId,
          delta: event.delta || '',
        };

      case EventType.TEXT_MESSAGE_END:
        this.currentMessageId = null;
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.TOOL_CALL_START:
        this.currentToolCallId = event.toolCallId;
        return {
          ...baseEvent,
          toolCallId: event.toolCallId,
          toolCallName: event.toolCallName || '',
          parentMessageId: event.parentMessageId,
        };

      case EventType.TOOL_CALL_ARGS:
        return {
          ...baseEvent,
          toolCallId: event.toolCallId,
          delta: event.delta || '',
        };

      case EventType.TOOL_CALL_END:
        this.currentToolCallId = null;
        return {
          ...baseEvent,
          toolCallId: event.toolCallId,
        };

      case EventType.TOOL_CALL_RESULT:
        return {
          ...baseEvent,
          messageId: event.messageId,
          toolCallId: event.toolCallId,
          content: event.content || '',
        };

      case EventType.REASONING_START:
        this.isReasoning = true;
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.REASONING_MESSAGE_START:
        this.isReasoningMessage = true;
        return {
          ...baseEvent,
          messageId: event.messageId,
          role: 'reasoning',
        };

      case EventType.REASONING_MESSAGE_CONTENT:
        return {
          ...baseEvent,
          messageId: event.messageId,
          delta: event.delta || '',
        };

      case EventType.REASONING_MESSAGE_END:
        this.isReasoningMessage = false;
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.REASONING_END:
        this.isReasoning = false;
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.ACTIVITY_SNAPSHOT:
        return {
          ...baseEvent,
          messageId: event.messageId,
          activityType: event.activityType,
          content: event.content,
        };

      default:
        return baseEvent;
    }
  }

  /**
   * 重置解析器状态
   */
  reset() {
    this.currentMessageId = null;
    this.currentToolCallId = null;
    this.isReasoning = false;
    this.isReasoningMessage = false;
  }
}

/**
 * 消息聚合器
 * 用于将 SSE 事件聚合为消息对象
 */
export class MessageAggregator {
  constructor() {
    this.messages = [];
    this.currentMessage = null;
    this.currentToolCall = null;
  }

  /**
   * 添加事件并更新消息状态
   * @param {object} event - 解析后的事件对象
   */
  addEvent(event) {
    if (!event) return;

    switch (event.type) {
      case EventType.TEXT_MESSAGE_START:
        this.currentMessage = {
          id: event.messageId,
          role: event.role,
          content: '',
          toolCalls: [],
        };
        break;

      case EventType.TEXT_MESSAGE_CONTENT:
        if (this.currentMessage && event.messageId === this.currentMessage.id) {
          this.currentMessage.content += event.delta;
        }
        break;

      case EventType.TEXT_MESSAGE_END:
        if (this.currentMessage && event.messageId === this.currentMessage.id) {
          this.messages.push({ ...this.currentMessage });
          this.currentMessage = null;
        }
        break;

      case EventType.TOOL_CALL_START:
        if (this.currentMessage) {
          this.currentToolCall = {
            id: event.toolCallId,
            name: event.toolCallName,
            arguments: '',
            parentMessageId: event.parentMessageId,
          };
        }
        break;

      case EventType.TOOL_CALL_ARGS:
        if (
          this.currentToolCall &&
          event.toolCallId === this.currentToolCall.id
        ) {
          this.currentToolCall.arguments += event.delta;
        }
        break;

      case EventType.TOOL_CALL_END:
        if (this.currentToolCall && this.currentMessage) {
          this.currentMessage.toolCalls.push({ ...this.currentToolCall });
          this.currentToolCall = null;
        }
        break;

      case EventType.TOOL_CALL_RESULT:
        this.messages.push({
          id: event.messageId,
          role: 'tool',
          toolCallId: event.toolCallId,
          content: event.content,
        });
        break;

      case EventType.REASONING_MESSAGE_START:
        this.currentMessage = {
          id: event.messageId,
          role: 'reasoning',
          content: '',
        };
        break;

      case EventType.REASONING_MESSAGE_CONTENT:
        if (this.currentMessage && event.messageId === this.currentMessage.id) {
          this.currentMessage.content += event.delta;
        }
        break;

      case EventType.REASONING_MESSAGE_END:
        if (this.currentMessage && event.messageId === this.currentMessage.id) {
          this.messages.push({ ...this.currentMessage });
          this.currentMessage = null;
        }
        break;
    }
  }

  /**
   * 获取所有消息
   * @returns {Array} 消息列表
   */
  getMessages() {
    return [...this.messages];
  }

  /**
   * 重置聚合器
   */
  reset() {
    this.messages = [];
    this.currentMessage = null;
    this.currentToolCall = null;
  }
}

/**
 * 格式化工具调用参数
 * @param {string} args - JSON 字符串参数
 * @returns {object} 解析后的参数对象
 */
export function formatToolArgs(args) {
  if (!args) return {};
  try {
    return JSON.parse(args);
  } catch {
    return { raw: args };
  }
}

/**
 * 格式化工具调用结果
 * @param {string} content - 结果内容
 * @param {number} maxLength - 最大显示长度
 * @returns {string} 格式化后的内容
 */
export function formatToolResult(content, maxLength = 500) {
  if (!content) return '';

  // 尝试解析 JSON
  try {
    const parsed = JSON.parse(content);
    content = JSON.stringify(parsed, null, 2);
  } catch {
    // 不是 JSON，保持原样
  }

  if (content.length > maxLength) {
    return content.substring(0, maxLength) + '...';
  }
  return content;
}
