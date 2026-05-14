/**
 * 消息聚合工具 - 将 AG-UI 事件聚合为消息格式
 */
import { formatDuration } from '@/utils/util';

/**
 * 将 AG-UI 事件聚合为消息 - 支持交错展示和 activity 嵌套
 */
export function aggregateEventsToMessages(events) {
  const messages = [];
  const toolCallMap = new Map();
  const activityStack = []; // 用于跟踪嵌套的 activity
  let currentActivity = null; // 当前的 activity

  const getCurrentActivity = () => currentActivity;

  const addFragment = fragment => {
    if (currentActivity) {
      currentActivity.fragments.push(fragment);
    } else {
      messages.push({
        ...fragment,
        id: fragment.id || generateId(),
        role: 'assistant',
        timestamp: Date.now(),
      });
    }
  };

  for (const event of events) {
    const eventTimestamp = event.timestamp
      ? new Date(event.timestamp).getTime()
      : Date.now();

    switch (event.type) {
      case 'RUN_STARTED': {
        if (event.input?.messages && Array.isArray(event.input.messages)) {
          event.input.messages.forEach(msg => {
            if (msg.role === 'user') {
              messages.push({
                id: msg.id || generateId(),
                role: 'user',
                content: formatContent(msg.content),
                files: extractFilesFromContent(msg.content),
                toolCalls: null,
                toolResults: null,
                toolCallId: null,
                reasoning: '',
                timestamp: eventTimestamp,
              });
            }
          });
        }
        break;
      }

      case 'ACTIVITY_SNAPSHOT': {
        const activityContent = event.content || {};
        if (event.activityType === 'sub_agent') {
          if (activityContent.status === 'started') {
            currentActivity = {
              type: 'activity',
              activityType: event.activityType,
              activityId: event.activityId,
              agentName: activityContent.agentName,
              fragments: [],
            };
            activityStack.push(currentActivity);
          } else if (activityContent.status === 'finished') {
            if (activityStack.length > 0) {
              const finishedActivity = activityStack.pop();
              if (activityStack.length > 0) {
                currentActivity = activityStack[activityStack.length - 1];
                currentActivity.fragments.push(finishedActivity);
              } else {
                messages.push({
                  id: event.messageId || generateId(),
                  role: 'assistant',
                  ...finishedActivity,
                  timestamp: eventTimestamp,
                });
                currentActivity = null;
              }
            }
          }
        } else if (
          event.activityType === 'workspace' &&
          activityContent.runId
        ) {
          addFragment({
            type: 'workspace',
            workspaceInfo: {
              fileCount: activityContent.fileCount || 0,
              totalSize: activityContent.totalSize || 0,
            },
            runId: activityContent.runId,
          });
        } else if (event.activityType === 'question') {
          // Human-in-the-Loop: 问题交互
          const questionId = activityContent.questionId;
          const status = activityContent.status;

          // 对于 answered/rejected 状态，更新已存在的 fragment
          if (status === 'answered' || status === 'rejected') {
            const fragments = currentActivity
              ? currentActivity.fragments
              : messages;
            // 递归查找匹配的 question fragment
            const findQuestionFragment = frags => {
              for (const frag of frags) {
                if (
                  frag.type === 'question' &&
                  frag.questionId === questionId
                ) {
                  return frag;
                }
                // 检查 activity 内部的 fragments
                if (frag.type === 'activity' && frag.fragments) {
                  const found = findQuestionFragment(frag.fragments);
                  if (found) return found;
                }
              }
              return null;
            };
            const existingFragment = findQuestionFragment(fragments);
            if (existingFragment) {
              existingFragment.status = status;
              existingFragment.answers =
                activityContent.answers || existingFragment.answers;
            }
            // answered/rejected 状态不创建新 fragment
          } else if (status === 'pending') {
            // 只有 pending 状态创建新 fragment
            addFragment({
              type: 'question',
              questionId: questionId,
              runId: activityContent.runId,
              status: status,
              questions: activityContent.questions || [],
              answers: activityContent.answers,
              timestamp: activityContent.timestamp || Date.now(),
            });
          }
        }
        break;
      }

      case 'REASONING_MESSAGE_START': {
        addFragment({
          type: 'reasoning',
          content: '',
          messageId: event.messageId,
          startTime: eventTimestamp,
        });
        break;
      }

      case 'REASONING_MESSAGE_CONTENT': {
        const activity = getCurrentActivity();
        if (activity) {
          const lastFragment =
            activity.fragments[activity.fragments.length - 1];
          if (lastFragment && lastFragment.type === 'reasoning') {
            lastFragment.content += event.delta;
          }
        } else {
          const lastMsg = messages[messages.length - 1];
          if (lastMsg && lastMsg.type === 'reasoning') {
            lastMsg.content += event.delta;
          }
        }
        break;
      }

      case 'REASONING_MESSAGE_END': {
        const activity = getCurrentActivity();
        if (activity) {
          const lastFragment =
            activity.fragments[activity.fragments.length - 1];
          if (
            lastFragment &&
            lastFragment.type === 'reasoning' &&
            lastFragment.startTime
          ) {
            lastFragment.duration = formatDuration(
              eventTimestamp - lastFragment.startTime,
            );
          }
        } else {
          const lastMsg = messages[messages.length - 1];
          if (lastMsg && lastMsg.type === 'reasoning' && lastMsg.startTime) {
            lastMsg.duration = formatDuration(
              eventTimestamp - lastMsg.startTime,
            );
          }
        }
        break;
      }

      case 'TEXT_MESSAGE_START': {
        addFragment({
          type: 'text',
          content: '',
          messageId: event.messageId,
        });
        break;
      }

      case 'TEXT_MESSAGE_CONTENT': {
        const activity = getCurrentActivity();
        if (activity) {
          const lastFragment =
            activity.fragments[activity.fragments.length - 1];
          if (lastFragment && lastFragment.type === 'text') {
            lastFragment.content += event.delta;
          }
        } else {
          const lastMsg = messages[messages.length - 1];
          if (lastMsg && lastMsg.type === 'text') {
            lastMsg.content += event.delta;
          }
        }
        break;
      }

      case 'TEXT_MESSAGE_END':
        break;

      case 'TOOL_CALL_START': {
        const toolCallData = {
          id: event.toolCallId,
          name: event.toolCallName,
          arguments: '',
          status: 'completed',
          result: '',
          startTime: eventTimestamp,
          executionTime: '',
        };
        toolCallMap.set(event.toolCallId, toolCallData);
        addFragment({
          type: 'tool_call',
          toolCall: toolCallData,
          messageId: event.messageId,
        });
        break;
      }

      case 'TOOL_CALL_ARGS': {
        if (toolCallMap.has(event.toolCallId)) {
          const toolCall = toolCallMap.get(event.toolCallId);
          toolCall.arguments += event.delta;
        }
        break;
      }

      case 'TOOL_CALL_END': {
        // 不删除 toolCallMap，等 TOOL_CALL_RESULT 处理
        break;
      }

      case 'TOOL_CALL_RESULT': {
        let executionTime = '';
        if (toolCallMap.has(event.toolCallId)) {
          const toolCall = toolCallMap.get(event.toolCallId);
          toolCall.result = event.content;
          toolCall.status = 'completed';
          if (toolCall.startTime && eventTimestamp) {
            executionTime = formatDuration(eventTimestamp - toolCall.startTime);
            toolCall.executionTime = executionTime;
          }
          toolCallMap.delete(event.toolCallId);
        }
        const activity = getCurrentActivity();
        if (activity) {
          const fragment = activity.fragments.find(
            f => f.type === 'tool_call' && f.toolCall?.id === event.toolCallId,
          );
          if (fragment && fragment.toolCall) {
            fragment.toolCall.result = event.content;
            fragment.toolCall.status = 'completed';
            fragment.toolCall.executionTime = executionTime;
          }
        } else {
          const toolCallMsg = messages.find(
            m => m.type === 'tool_call' && m.toolCall?.id === event.toolCallId,
          );
          if (toolCallMsg && toolCallMsg.toolCall) {
            toolCallMsg.toolCall.result = event.content;
            toolCallMsg.toolCall.status = 'completed';
            toolCallMsg.toolCall.executionTime = executionTime;
          }
        }
        break;
      }
    }
  }

  // 处理未关闭的 activity
  while (activityStack.length > 0) {
    const activity = activityStack.pop();
    if (activityStack.length > 0) {
      activityStack[activityStack.length - 1].fragments.push(activity);
    } else {
      messages.push({
        id: generateId(),
        role: 'assistant',
        ...activity,
        timestamp: Date.now(),
      });
    }
  }

  return mergeToFragments(messages);
}

/**
 * 将消息合并为带 fragments 的格式
 */
export function mergeToFragments(messages) {
  const result = [];
  let currentAssistant = null;

  for (const msg of messages) {
    if (msg.role === 'user') {
      if (currentAssistant) {
        result.push(currentAssistant);
        currentAssistant = null;
      }
      result.push(msg);
    } else if (msg.role === 'assistant') {
      if (!currentAssistant) {
        currentAssistant = {
          id: msg.id || generateId(),
          role: 'assistant',
          content: '',
          reasoning: '',
          toolCalls: [],
          fragments: [],
        };
      }

      if (msg.type === 'activity') {
        currentAssistant.fragments.push({
          type: 'activity',
          activityType: msg.activityType,
          agentName: msg.agentName,
          fragments: msg.fragments || [],
        });
      } else if (msg.type === 'reasoning') {
        currentAssistant.fragments.push({
          type: 'reasoning',
          content: msg.content,
          duration: msg.duration,
        });
        currentAssistant.reasoning = msg.content;
      } else if (msg.type === 'tool_call' && msg.toolCall) {
        currentAssistant.fragments.push({
          type: 'tool_call',
          toolCall: msg.toolCall,
        });
        currentAssistant.toolCalls.push(msg.toolCall);
      } else if (msg.type === 'workspace') {
        currentAssistant.fragments.push({
          type: 'workspace',
          workspaceInfo: msg.workspaceInfo,
          runId: msg.runId,
        });
      } else if (msg.type === 'question') {
        currentAssistant.fragments.push({
          type: 'question',
          questionId: msg.questionId,
          runId: msg.runId,
          status: msg.status,
          questions: msg.questions || [],
          answers: msg.answers,
        });
      } else if (msg.type === 'text' && msg.content) {
        currentAssistant.fragments.push({
          type: 'text',
          content: msg.content,
        });
        currentAssistant.content += msg.content;
      }
    }
  }

  if (currentAssistant) {
    result.push(currentAssistant);
  }

  return result;
}

/**
 * 格式化消息内容
 */
export function formatMessage(msg) {
  if (!msg) return null;

  // 如果已经是标准格式
  if (msg.role && (msg.content || msg.toolCalls || msg.reasoning)) {
    return {
      id: msg.id || generateId(),
      role: msg.role,
      content: formatContent(msg.content),
      toolCalls: msg.toolCalls,
      toolResults: msg.toolResults,
      toolCallId: msg.toolCallId,
      reasoning: msg.reasoning,
      reasoningDuration: msg.reasoningDuration,
      toolDuration: msg.toolDuration,
    };
  }

  // 处理 AG-UI 协议格式
  if (msg.type) {
    switch (msg.type) {
      case 'TEXT_MESSAGE':
      case 'text_message':
        return {
          id: msg.id || msg.messageId || generateId(),
          role: msg.role || 'assistant',
          content: formatContent(msg.content || msg.text),
          toolCalls: null,
          toolResults: null,
          toolCallId: null,
          reasoning: '',
          reasoningDuration: '',
          toolDuration: '',
        };
      case 'TOOL_CALL':
      case 'tool_call':
        return {
          id: msg.id || generateId(),
          role: 'tool',
          content: formatContent(msg.result || msg.content),
          toolCalls: null,
          toolResults: null,
          toolCallId: msg.toolCallId || msg.id,
          reasoning: '',
          reasoningDuration: '',
          toolDuration: '',
        };
      default:
        // 尝试从 message 字段获取内容
        if (msg.message) {
          return formatMessage(msg.message);
        }
        return null;
    }
  }

  // 尝试处理嵌套结构
  if (msg.message) {
    return formatMessage(msg.message);
  }

  // 跳过无效消息
  if (!msg.role && !msg.content && !msg.text) {
    return null;
  }

  return {
    id: msg.id || generateId(),
    role: msg.role || 'unknown',
    content: formatContent(msg.content || msg.text),
    toolCalls: msg.toolCalls,
    toolResults: msg.toolResults,
    toolCallId: msg.toolCallId,
    reasoning: msg.reasoning,
    reasoningDuration: msg.reasoningDuration,
    toolDuration: msg.toolDuration,
  };
}

/**
 * 格式化内容
 */
export function formatContent(content) {
  if (typeof content === 'string') return content;
  if (Array.isArray(content)) {
    return content
      .filter(item => item.type === 'text')
      .map(item => item.text)
      .join('\n');
  }
  if (typeof content === 'object' && content?.text) return content.text;
  return '';
}

/**
 * 从内容中提取文件
 */
export function extractFilesFromContent(content) {
  if (!Array.isArray(content)) return null;
  const files = content.filter(item => item.type === 'binary');
  if (files.length === 0) return null;
  return files.map(file => ({
    fileName: file.fileName || 'unknown',
    type: file.mimeType || 'application/octet-stream',
    url: file.url,
    displayUrl: file.url,
  }));
}

/**
 * 生成唯一ID
 */
export function generateId() {
  return 'msg_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9);
}
