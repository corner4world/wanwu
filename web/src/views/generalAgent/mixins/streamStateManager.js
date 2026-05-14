/**
 * 流式状态管理 Mixin - 管理多会话的 SSE 流式状态
 */
import { formatDuration } from '@/utils/util';
import { ActivityType, ActivityStatus } from '../utils/sse-parser';

export default {
  data() {
    return {
      // 每个会话独立的流式状态 { threadId: { isStreaming, abortController, streamingMessage, ... } }
      streamingMap: {},
    };
  },

  computed: {
    // 当前会话的流式状态
    currentStreaming() {
      return (
        this.streamingMap[this.currentThreadId] || {
          isStreaming: false,
          abortController: null,
          streamingMessage: null,
        }
      );
    },
    isStreaming() {
      return this.currentStreaming.isStreaming;
    },
    streamingMessage() {
      return this.currentStreaming.streamingMessage;
    },
  },

  watch: {
    // 监听 streamingMap 的变化，自动同步到 skillManager
    streamingMap: {
      handler(newVal) {
        // 检查是否有任何会话在流式传输
        const anyStreaming = Object.values(newVal).some(
          state => state?.isStreaming === true,
        );

        // 直接设置 previewIsStreaming 或 mainIsStreaming
        if (this.$options.name === 'GeneralAgent') {
          this.mainIsStreaming = anyStreaming;
        } else if (this.$options.name === 'PreviewChat') {
          this.previewIsStreaming = anyStreaming;
        }
      },
      deep: true,
    },
  },

  beforeDestroy() {
    // 清理所有会话的流式状态
    this.cleanupAllStreams();
  },

  methods: {
    /**
     * 初始化会话的流式状态
     */
    initStreamState(threadId) {
      const abortController = new AbortController();
      const assistantMessage = {
        id: this.generateId(),
        role: 'assistant',
        content: '',
        reasoning: '',
        toolCalls: [],
        toolResults: [],
        fragments: [],
        isStreaming: true,
        threadId: threadId,
      };

      this.$set(this.streamingMap, threadId, {
        isStreaming: true,
        abortController: abortController,
        streamingMessage: assistantMessage,
        activityStack: [],
        currentActivity: null,
        currentFragment: null,
        toolCallMap: new Map(),
      });

      return { abortController, assistantMessage };
    },

    /**
     * 清理单个会话的流式状态
     */
    cleanupStreamState(threadId) {
      const streaming = this.streamingMap[threadId];
      if (streaming) {
        if (streaming.abortController) {
          streaming.abortController.abort();
        }
        streaming.isStreaming = false;
        streaming.streamingMessage = null;
        streaming.abortController = null;
        this.$delete(this.streamingMap, threadId);
      }
    },

    /**
     * 清理所有会话的流式状态
     */
    cleanupAllStreams() {
      Object.keys(this.streamingMap).forEach(threadId => {
        this.cleanupStreamState(threadId);
      });
      this.streamingMap = {};
    },

    /**
     * 中止指定会话的流
     */
    stopStreaming(threadId) {
      const targetThreadId = threadId || this.currentThreadId;
      const streaming = this.streamingMap[targetThreadId];
      if (!streaming) {
        return;
      }

      if (!streaming.abortController) {
        return;
      }

      // 中止请求
      streaming.abortController.abort();

      // 清理流式消息的 isStreaming 状态
      if (streaming.streamingMessage) {
        streaming.streamingMessage.isStreaming = false;

        // 递归设置所有 fragments 的 isStreaming 为 false
        this.setFragmentsNotStreaming(streaming.streamingMessage.fragments);
      }

      // 使用 $set 确保响应式更新
      this.$set(streaming, 'isStreaming', false);
      this.$set(streaming, 'streamingMessage', null);
      this.$set(streaming, 'abortController', null);

      // 重置滚动状态和阶段
      if (targetThreadId === this.currentThreadId) {
        this.currentStage = '';
        this.resetScrollState();
        this.$nextTick(() => this.scrollToBottom(true));
      }
    },

    /**
     * 递归设置所有 fragments 的 isStreaming 为 false
     */
    setFragmentsNotStreaming(fragments) {
      if (!fragments || !Array.isArray(fragments)) return;

      fragments.forEach(fragment => {
        if (fragment.isStreaming !== undefined) {
          this.$set(fragment, 'isStreaming', false);
        }

        // 处理 activity 类型，递归设置其子 fragments
        if (fragment.type === 'activity' && fragment.fragments) {
          this.setFragmentsNotStreaming(fragment.fragments);
        }
      });
    },

    /**
     * 处理 SSE 事件
     */
    handleSSEEvent(event, assistantMessage, parser, streamingThreadId) {
      const parsed = parser.parse(event);
      if (!parsed) return;

      const streamState = this.streamingMap[streamingThreadId];
      if (!streamState) return;

      const getCurrentFragments = () => {
        if (streamState.currentActivity) {
          return streamState.currentActivity.fragments;
        }
        return assistantMessage.fragments;
      };

      const addFragment = fragment => {
        const fragments = getCurrentFragments();
        if (fragments) {
          fragments.push(fragment);
        }
      };

      switch (parsed.type) {
        case 'RUN_STARTED':
          this.currentRunId = parsed.runId;
          if (this.currentThreadId === streamingThreadId) {
            this.currentStage = 'understanding';
          }
          break;

        case 'ACTIVITY_SNAPSHOT': {
          const activityContent = parsed.content || {};
          if (parsed.activityType === ActivityType.SUB_AGENT) {
            if (activityContent.status === ActivityStatus.STARTED) {
              const activity = {
                type: 'activity',
                activityType: parsed.activityType,
                activityId: parsed.activityId,
                agentName: activityContent.agentName,
                fragments: [],
                isStreaming: true,
                startTime: Date.now(),
                duration: '',
              };
              // 先添加到父级 fragments，再设置 currentActivity
              const parentFragments = getCurrentFragments();
              if (parentFragments) {
                parentFragments.push(activity);
              }
              streamState.currentActivity = activity;
              streamState.activityStack.push(activity);
            } else if (activityContent.status === ActivityStatus.FINISHED) {
              if (streamState.activityStack.length > 0) {
                const finishedActivity = streamState.activityStack.pop();
                finishedActivity.isStreaming = false;
                if (finishedActivity.startTime) {
                  finishedActivity.duration = formatDuration(
                    Date.now() - finishedActivity.startTime,
                  );
                }
                if (streamState.activityStack.length > 0) {
                  streamState.currentActivity =
                    streamState.activityStack[
                      streamState.activityStack.length - 1
                    ];
                } else {
                  streamState.currentActivity = null;
                }
              }
            }
          } else if (
            parsed.activityType === ActivityType.WORKSPACE &&
            activityContent.runId
          ) {
            this.handleWorkspaceActivity({
              runId: activityContent.runId,
              threadId: activityContent.threadId || this.currentThreadId,
              fileCount: activityContent.fileCount || 0,
              totalSize: activityContent.totalSize || 0,
              timestamp: activityContent.timestamp || Date.now(),
            });
            addFragment({
              type: 'workspace',
              workspaceInfo: {
                fileCount: activityContent.fileCount || 0,
                totalSize: activityContent.totalSize || 0,
              },
              runId: activityContent.runId,
            });
            if (this.currentThreadId === streamingThreadId) {
              this.$notify({
                type: 'success',
                title: '工作空间已更新',
                message: `生成了 ${activityContent.fileCount || 0} 个文件`,
                duration: 3000,
                onClick: () => {
                  this.showPanel();
                },
              });
            }
          } else if (parsed.activityType === ActivityType.QUESTION) {
            const questionId = activityContent.questionId;
            const status = activityContent.status;

            // 对于 answered/rejected 状态，更新已存在的 fragment
            if (status === 'answered' || status === 'rejected') {
              const fragments = getCurrentFragments();
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
              // answered/rejected 状态不创建新 fragment，继续处理后续事件
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

        case 'REASONING_MESSAGE_START':
          streamState.currentFragment = {
            type: 'reasoning',
            content: '',
            messageId: parsed.messageId,
            startTime: Date.now(),
            isStreaming: true,
          };
          addFragment(streamState.currentFragment);
          if (this.currentThreadId === streamingThreadId) {
            this.currentStage = 'thinking';
          }
          break;

        case 'REASONING_MESSAGE_CONTENT':
          if (streamState.currentFragment?.type === 'reasoning') {
            streamState.currentFragment.content += parsed.delta;
            if (!streamState.currentActivity) {
              assistantMessage.reasoning += parsed.delta;
            }
          }
          break;

        case 'REASONING_MESSAGE_END':
          if (streamState.currentFragment?.type === 'reasoning') {
            if (streamState.currentFragment.startTime) {
              streamState.currentFragment.duration = formatDuration(
                Date.now() - streamState.currentFragment.startTime,
              );
            }
            streamState.currentFragment.isStreaming = false;
          }
          streamState.currentFragment = null;
          break;

        case 'TEXT_MESSAGE_START':
          streamState.currentFragment = {
            type: 'text',
            content: '',
            messageId: parsed.messageId,
            isStreaming: true,
          };
          addFragment(streamState.currentFragment);
          assistantMessage.id = parsed.messageId || assistantMessage.id;
          if (this.currentThreadId === streamingThreadId) {
            this.currentStage = 'generating';
          }
          break;

        case 'TEXT_MESSAGE_CONTENT':
          if (streamState.currentFragment?.type === 'text') {
            streamState.currentFragment.content += parsed.delta;
            if (!streamState.currentActivity) {
              assistantMessage.content += parsed.delta;
            }
          }
          break;

        case 'TEXT_MESSAGE_END':
          if (streamState.currentFragment?.type === 'text') {
            streamState.currentFragment.isStreaming = false;
          }
          streamState.currentFragment = null;
          break;

        case 'TOOL_CALL_START': {
          const toolCallData = {
            id: parsed.toolCallId,
            name: parsed.toolCallName,
            arguments: '',
            status: 'running',
            result: '',
            startTime: Date.now(),
            executionTime: '',
          };
          streamState.toolCallMap.set(parsed.toolCallId, toolCallData);
          assistantMessage.toolCalls.push(toolCallData);
          streamState.currentFragment = {
            type: 'tool_call',
            toolCall: toolCallData,
            messageId: parsed.messageId,
          };
          addFragment(streamState.currentFragment);
          if (this.currentThreadId === streamingThreadId) {
            this.currentStage = 'tool_calling';
          }
          break;
        }

        case 'TOOL_CALL_ARGS':
          if (streamState.toolCallMap.has(parsed.toolCallId)) {
            const toolCall = streamState.toolCallMap.get(parsed.toolCallId);
            toolCall.arguments += parsed.delta;
          }
          break;

        case 'TOOL_CALL_END':
          // 不设置 completed，不计算时间，等 TOOL_CALL_RESULT
          streamState.currentFragment = null;
          break;

        case 'TOOL_CALL_RESULT':
          if (streamState.toolCallMap.has(parsed.toolCallId)) {
            const toolCall = streamState.toolCallMap.get(parsed.toolCallId);
            toolCall.result = parsed.content;
            toolCall.status = 'completed';
            if (toolCall.startTime) {
              toolCall.executionTime = formatDuration(
                Date.now() - toolCall.startTime,
              );
            }
            streamState.toolCallMap.delete(parsed.toolCallId);
          }
          const tc = assistantMessage.toolCalls.find(
            t => t.id === parsed.toolCallId,
          );
          if (tc) {
            tc.result = parsed.content;
            tc.status = 'completed';
            if (tc.startTime) {
              tc.executionTime = formatDuration(Date.now() - tc.startTime);
            }
          }
          const fragments = getCurrentFragments();
          const toolCallFragment = fragments.find(
            f => f.type === 'tool_call' && f.toolCall?.id === parsed.toolCallId,
          );
          if (toolCallFragment?.toolCall) {
            toolCallFragment.toolCall.result = parsed.content;
            toolCallFragment.toolCall.status = 'completed';
            if (toolCallFragment.toolCall.startTime) {
              toolCallFragment.toolCall.executionTime = formatDuration(
                Date.now() - toolCallFragment.toolCall.startTime,
              );
            }
          }
          break;

        case 'RUN_FINISHED':
          while (streamState.activityStack.length > 0) {
            const activity = streamState.activityStack.pop();
            if (streamState.activityStack.length > 0) {
              streamState.activityStack[
                streamState.activityStack.length - 1
              ].fragments.push(activity);
            } else {
              assistantMessage.fragments.push(activity);
            }
          }
          streamState.currentActivity = null;
          streamState.currentFragment = null;
          break;
      }

      if (this.currentThreadId === streamingThreadId) {
        this.$nextTick(() => this.scrollToBottom());
      }
    },

    /**
     * 生成唯一ID
     */
    generateId() {
      return (
        'msg_' + Date.now() + '_' + Math.random().toString(36).slice(2, 11)
      );
    },
  },
};
