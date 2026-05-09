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

  // 活动快照 (包含 status: started/finished)
  ACTIVITY_SNAPSHOT: 'ACTIVITY_SNAPSHOT',
};

// 活动类型
export const ActivityType = {
  SUB_AGENT: 'sub_agent',
  WORKSPACE: 'workspace',
  QUESTION: 'question',
};

// 活动状态
export const ActivityStatus = {
  STARTED: 'started',
  FINISHED: 'finished',
};

/**
 * SSE 事件解析器类
 * @class SSEEventParser
 */
export class SSEEventParser {
  constructor() {
    /** @type {string|null} 当前消息ID */
    this.currentMessageId = null;
    /** @type {string|null} 当前工具调用ID */
    this.currentToolCallId = null;
  }

  /**
   * 解析 SSE 事件
   * @param {object} event - 原始事件对象
   * @param {string} event.type - 事件类型
   * @param {string} [event.messageId] - 消息ID
   * @param {string} [event.toolCallId] - 工具调用ID
   * @param {string} [event.toolCallName] - 工具名称
   * @param {string} [event.delta] - 增量内容
   * @param {string} [event.content] - 内容
   * @param {object} [event.content] - 活动内容
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
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.REASONING_MESSAGE_START:
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
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.REASONING_END:
        return {
          ...baseEvent,
          messageId: event.messageId,
        };

      case EventType.ACTIVITY_SNAPSHOT: {
        const content = event.content || {};
        return {
          ...baseEvent,
          messageId: event.messageId,
          activityId: event.activityId || content.activityId,
          activityType: event.activityType,
          status: content.status,
          agentName: content.agentName || '',
          instanceNum: content.instanceNum,
          content: content,
        };
      }

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
  }
}
// 从 @/utils/util 重新导出 formatDuration
export { formatDuration } from '@/utils/util';
