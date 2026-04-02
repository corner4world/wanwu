<template>
  <div class="general-agent-page">
    <!-- 左侧会话列表 - 可折叠 -->
    <div :class="['sidebar', { collapsed: sidebarCollapsed }]">
      <!-- 折叠状态下显示展开按钮 -->
      <div v-if="sidebarCollapsed" class="sidebar-collapsed-bar">
        <el-tooltip content="展开侧边栏" placement="right">
          <div class="expand-btn" @click="toggleSidebar">
            <i class="el-icon-s-unfold"></i>
          </div>
        </el-tooltip>
        <el-tooltip content="新建对话" placement="right">
          <div class="expand-btn" @click="createConversation">
            <i class="el-icon-plus"></i>
          </div>
        </el-tooltip>
      </div>

      <!-- 展开状态 -->
      <template v-else>
        <div class="sidebar-header">
          <el-button
            type="primary"
            class="new-chat-btn"
            @click="createConversation"
          >
            <i class="el-icon-plus"></i>
            新建对话
          </el-button>
          <div class="sidebar-toggle" @click="toggleSidebar">
            <i class="el-icon-s-fold"></i>
          </div>
        </div>

        <div class="sidebar-divider"></div>

        <div class="conversation-list">
          <div
            v-for="item in conversationList"
            :key="item.threadId"
            :class="[
              'conversation-item',
              { active: currentThreadId === item.threadId },
            ]"
            @click="selectConversation(item.threadId)"
          >
            <i class="el-icon-chat-dot-round"></i>
            <span class="conversation-title">{{ item.title || '新对话' }}</span>
            <el-dropdown trigger="click" @command="handleCommand($event, item)">
              <i class="el-icon-more" @click.stop></i>
              <el-dropdown-menu slot="dropdown">
                <el-dropdown-item command="delete">
                  <span style="color: #f56c6c">删除</span>
                </el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
          </div>
        </div>
      </template>
    </div>

    <!-- 主内容区 -->
    <div class="agent-main-content">
      <!-- 顶部标题栏 -->
      <div class="header">
        <div class="header-title">{{ currentTitle }}</div>
      </div>

      <!-- 消息区域 - 独立滚动 -->
      <div class="message-area" ref="messageArea">
        <!-- 空状态 -->
        <div
          v-if="messageList.length === 0 && !isStreaming"
          class="empty-state"
        >
          <div class="empty-icon">
            <i class="el-icon-chat-dot-round"></i>
          </div>
          <div class="empty-title">开始新对话</div>
          <div class="empty-tips">输入您的问题，开始与智能体对话</div>
        </div>

        <!-- 消息列表 -->
        <div v-else class="message-list">
          <message-item
            v-for="(msg, index) in messageList"
            :key="msg.id || index"
            :message="msg"
            :tool-results="getToolResultsForMessage(msg)"
            :is-last-message="index === messageList.length - 1"
            @regenerate="handleRegenerate"
          />
          <div
            v-if="isStreaming && !hasAssistantContent"
            class="typing-indicator"
          >
            <span></span>
            <span></span>
            <span></span>
            <span class="typing-text">思考中...</span>
          </div>
        </div>

        <div ref="scrollAnchor"></div>
      </div>

      <!-- 底部输入区 - 固定底部 -->
      <div class="input-area">
        <div class="input-container">
          <!-- 模型选择 -->
          <div class="model-selector">
            <el-select
              v-model="selectedModel"
              size="small"
              placeholder="选择模型"
              @change="handleModelChange"
            >
              <el-option
                v-for="model in modelList"
                :key="model.modelId"
                :label="model.modelName"
                :value="model.modelId"
              />
            </el-select>
          </div>

          <!-- 文件预览 -->
          <div v-if="uploadedFiles.length > 0" class="file-preview">
            <div
              v-for="(file, index) in uploadedFiles"
              :key="index"
              class="file-item"
            >
              <img
                v-if="file.type.startsWith('image/')"
                :src="file.url"
                class="file-thumb"
              />
              <div v-else class="file-icon">
                <i class="el-icon-document"></i>
              </div>
              <i
                class="el-icon-close file-remove"
                @click="removeFile(index)"
              ></i>
            </div>
          </div>

          <!-- 输入框 -->
          <div class="input-wrapper">
            <el-input
              v-model="inputMessage"
              type="textarea"
              :rows="1"
              :autosize="{ minRows: 1, maxRows: 6 }"
              placeholder="输入问题，按 Enter 发送，Shift+Enter 换行"
              @keydown.enter.native="handleKeyDown"
              :disabled="isStreaming"
            />
            <div class="input-actions">
              <el-upload
                action="#"
                :auto-upload="false"
                :show-file-list="false"
                :on-change="handleFileChange"
                multiple
              >
                <el-tooltip content="上传文件" placement="top">
                  <i class="el-icon-paperclip action-icon"></i>
                </el-tooltip>
              </el-upload>
              <el-button
                type="primary"
                size="small"
                circle
                :loading="isStreaming"
                :disabled="!canSend"
                @click="sendMessage"
              >
                <i v-if="!isStreaming" class="el-icon-top"></i>
              </el-button>
              <el-button
                v-if="isStreaming"
                type="danger"
                size="small"
                circle
                @click="stopStreaming"
              >
                <i class="el-icon-video-pause"></i>
              </el-button>
            </div>
          </div>
        </div>
        <div class="input-footer">
          <span>通用智能体 · 内容由 AI 生成，仅供参考</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import MessageItem from './components/MessageItem.vue';
import {
  getGeneralAgentConversationList,
  createGeneralAgentConversation,
  deleteGeneralAgentConversation,
  getGeneralAgentConversationDetail,
  getGeneralAgentConfig,
  updateGeneralAgentConfig,
  chatGeneralAgentConversation,
  getGeneralAgentToolSelect,
  getLlmModelSelect,
} from '@/api/generalAgent';
import { SSEEventParser } from './utils/sse-parser';

export default {
  name: 'GeneralAgent',
  components: {
    MessageItem,
  },
  data() {
    return {
      sidebarCollapsed: false,
      conversationList: [],
      currentThreadId: '',
      pageNo: 1,
      pageSize: 50,
      total: 0,

      messageList: [],
      inputMessage: '',
      uploadedFiles: [],
      isStreaming: false,
      abortController: null,

      selectedModel: '',
      selectedTools: [],
      selectedAssistants: [],
      modelList: [],
      toolList: [],

      currentRunId: '',
      currentStage: '',
      streamingMessage: null,
    };
  },
  computed: {
    currentTitle() {
      if (!this.currentThreadId) return '通用智能体';
      const conv = this.conversationList.find(
        c => c.threadId === this.currentThreadId,
      );
      return conv?.title || '新对话';
    },
    canSend() {
      return this.inputMessage.trim() || this.uploadedFiles.length > 0;
    },
    hasAssistantContent() {
      return this.messageList.some(
        m =>
          m.role === 'assistant' &&
          (m.content || m.reasoning || (m.toolCalls && m.toolCalls.length > 0)),
      );
    },
  },
  mounted() {
    this.fetchModelList();
    this.fetchConversationList();
    this.fetchToolList();
  },
  beforeDestroy() {
    this.stopStreaming();
  },
  methods: {
    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
    },

    async fetchModelList() {
      try {
        const res = await getLlmModelSelect();
        if (res.code === 0 && res.data?.list) {
          this.modelList = res.data.list.map(model => ({
            modelId: model.modelId || model.model,
            modelName: model.displayName || model.model,
            model: model.model,
            provider: model.provider,
            modelType: model.modelType,
            config: model.config,
          }));
        }
      } catch (error) {
        console.error('获取模型列表失败:', error);
      }
    },

    async fetchConversationList() {
      try {
        const res = await getGeneralAgentConversationList({
          pageNo: this.pageNo,
          pageSize: this.pageSize,
        });
        if (res.code === 0) {
          this.conversationList = res.data?.list || [];
          this.total = res.data?.total || 0;
          if (this.conversationList.length > 0 && !this.currentThreadId) {
            this.selectConversation(this.conversationList[0].threadId);
          }
        }
      } catch (error) {
        console.error('获取对话列表失败:', error);
      }
    },

    async fetchToolList() {
      try {
        const res = await getGeneralAgentToolSelect();
        if (res.code === 0 && res.data) {
          this.toolList = res.data || [];
        }
      } catch (error) {
        console.error('获取工具列表失败:', error);
      }
    },

    async createConversation() {
      try {
        const res = await createGeneralAgentConversation({
          title: '新对话',
        });
        if (res.code === 0) {
          const threadId = res.data?.threadId;
          if (threadId) {
            this.currentThreadId = threadId;
            this.messageList = [];
            this.selectedModel = '';
            this.selectedTools = [];
            this.conversationList.unshift({
              threadId,
              title: '新对话',
              createdAt: new Date().toISOString(),
            });
            await new Promise(resolve => setTimeout(resolve, 500));
            return true;
          } else {
            this.$message.error('创建对话失败：未返回对话ID');
          }
        } else {
          this.$message.error(res.msg || '创建对话失败');
        }
        return false;
      } catch (error) {
        console.error('创建对话失败:', error);
        this.$message.error('创建对话失败，请检查网络连接');
        return false;
      }
    },

    selectConversation(threadId) {
      if (this.currentThreadId === threadId) return;
      this.currentThreadId = threadId;
      this.fetchHistory();
    },

    async fetchHistory() {
      if (!this.currentThreadId) return;
      this.messageList = [];
      try {
        const res = await getGeneralAgentConversationDetail({
          threadId: this.currentThreadId,
          pageNo: 1,
          pageSize: 100,
        });
        console.log('fetchHistory response:', res);

        if (res.code === 0 && res.data?.list) {
          const allMessages = [];
          res.data.list.forEach(run => {
            console.log('run data:', run);
            // 后端返回的是 events 字段，需要聚合为消息
            if (run.events && Array.isArray(run.events)) {
              const messages = this.aggregateEventsToMessages(run.events);
              allMessages.push(...messages);
            }
            // 兼容旧格式 messages 字段
            if (run.messages && Array.isArray(run.messages)) {
              run.messages.forEach(msg => {
                const formatted = this.formatMessage(msg);
                if (formatted) {
                  allMessages.push(formatted);
                }
              });
            }
            if (run.runId) this.currentRunId = run.runId;
          });
          console.log('all messages:', allMessages);
          this.messageList = allMessages;
          this.$nextTick(() => this.scrollToBottom());
        }
        this.loadConfig();
      } catch (error) {
        console.error('获取历史消息失败:', error);
      }
    },

    // 将 AG-UI 事件聚合为消息 - 支持交错展示
    aggregateEventsToMessages(events) {
      const messages = [];
      const toolCallMap = new Map(); // 用于聚合工具调用参数
      let currentReasoningStart = null;
      let currentTextStart = null;

      for (const event of events) {
        const eventTimestamp = event.timestamp
          ? new Date(event.timestamp).getTime()
          : Date.now();

        switch (event.type) {
          // 开始新的对话
          case 'RUN_STARTED': {
            // 提取用户消息
            if (event.input?.messages && Array.isArray(event.input.messages)) {
              event.input.messages.forEach(msg => {
                if (msg.role === 'user') {
                  messages.push({
                    id: msg.id || this.generateId(),
                    role: 'user',
                    content: this.formatContent(msg.content),
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

          // 思考片段
          case 'REASONING_MESSAGE_START': {
            currentReasoningStart = eventTimestamp;
            // 创建思考片段
            messages.push({
              id: event.messageId || this.generateId(),
              role: 'assistant',
              type: 'reasoning',
              content: '',
              reasoning: '',
              toolCalls: null,
              isReasoningBlock: true,
              startTime: eventTimestamp,
              duration: '',
            });
            break;
          }

          case 'REASONING_MESSAGE_CONTENT': {
            // 追加到最后的思考片段
            const lastMsg = messages[messages.length - 1];
            if (lastMsg && lastMsg.isReasoningBlock) {
              lastMsg.reasoning += event.delta || '';
              if (lastMsg.startTime) {
                lastMsg.duration = this.formatDuration(
                  eventTimestamp - lastMsg.startTime,
                );
              }
            }
            break;
          }

          case 'REASONING_MESSAGE_END': {
            const lastMsg = messages[messages.length - 1];
            if (lastMsg && lastMsg.isReasoningBlock && lastMsg.startTime) {
              lastMsg.duration = this.formatDuration(
                eventTimestamp - lastMsg.startTime,
              );
            }
            currentReasoningStart = null;
            break;
          }

          // 文字片段
          case 'TEXT_MESSAGE_START': {
            currentTextStart = eventTimestamp;
            // 创建文字片段
            messages.push({
              id: event.messageId || this.generateId(),
              role: 'assistant',
              type: 'text',
              content: '',
              reasoning: '',
              toolCalls: null,
              startTime: eventTimestamp,
            });
            break;
          }

          case 'TEXT_MESSAGE_CONTENT': {
            // 追加到最后的文字片段
            const lastMsg = messages[messages.length - 1];
            if (
              lastMsg &&
              lastMsg.role === 'assistant' &&
              !lastMsg.isReasoningBlock &&
              !lastMsg.isToolCall
            ) {
              lastMsg.content += event.delta || '';
            }
            break;
          }

          case 'TEXT_MESSAGE_END': {
            currentTextStart = null;
            break;
          }

          // 工具调用
          case 'TOOL_CALL_START': {
            // 创建工具调用片段
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
            messages.push({
              id: event.toolCallId,
              role: 'assistant',
              type: 'tool_call',
              content: '',
              reasoning: '',
              toolCalls: [toolCallData],
              isToolCall: true,
              startTime: eventTimestamp,
            });
            break;
          }

          case 'TOOL_CALL_ARGS': {
            if (toolCallMap.has(event.toolCallId)) {
              const toolCall = toolCallMap.get(event.toolCallId);
              toolCall.arguments += event.delta || '';
            }
            break;
          }

          case 'TOOL_CALL_END': {
            if (toolCallMap.has(event.toolCallId)) {
              const toolCall = toolCallMap.get(event.toolCallId);
              if (toolCall.startTime) {
                toolCall.executionTime = this.formatDuration(
                  eventTimestamp - toolCall.startTime,
                );
              }
            }
            toolCallMap.delete(event.toolCallId);
            break;
          }

          case 'TOOL_CALL_RESULT': {
            // 找到对应的工具调用片段并更新结果
            const toolCallMsg = messages.find(m => m.id === event.toolCallId);
            if (
              toolCallMsg &&
              toolCallMsg.toolCalls &&
              toolCallMsg.toolCalls[0]
            ) {
              toolCallMsg.toolCalls[0].result = event.content || '';
              if (toolCallMsg.toolCalls[0].startTime) {
                toolCallMsg.toolCalls[0].executionTime = this.formatDuration(
                  eventTimestamp - toolCallMsg.toolCalls[0].startTime,
                );
              }
            }
            break;
          }
        }
      }

      // 过滤掉空的片段，合并相邻的同类型片段
      return this.mergeAssistantFragments(messages);
    },

    // 合并相邻的助手片段，保持展示简洁
    mergeAssistantFragments(messages) {
      const result = [];
      let currentAssistant = null;

      for (const msg of messages) {
        if (msg.role === 'user') {
          // 用户消息直接添加
          if (currentAssistant) {
            result.push(currentAssistant);
            currentAssistant = null;
          }
          result.push(msg);
        } else if (msg.role === 'assistant') {
          // 合并连续的文字片段到当前助手消息
          if (!currentAssistant) {
            currentAssistant = {
              id: msg.id,
              role: 'assistant',
              content: msg.content || '',
              reasoning: msg.reasoning || '',
              toolCalls: msg.toolCalls ? [...msg.toolCalls] : [],
              fragments: [], // 保存原始片段顺序
              reasoningDuration: '',
              toolDuration: '',
            };
          }

          // 记录片段（包含持续时间信息）
          if (msg.isReasoningBlock) {
            currentAssistant.fragments.push({
              type: 'reasoning',
              content: msg.reasoning,
              duration: msg.duration || '',
            });
            currentAssistant.reasoning = msg.reasoning;
            currentAssistant.reasoningDuration = msg.duration || '';
          } else if (msg.isToolCall && msg.toolCalls && msg.toolCalls[0]) {
            currentAssistant.fragments.push({
              type: 'tool_call',
              toolCall: msg.toolCalls[0],
            });
            if (!currentAssistant.toolCalls) {
              currentAssistant.toolCalls = [];
            }
            currentAssistant.toolCalls.push(msg.toolCalls[0]);
          } else if (msg.content) {
            currentAssistant.fragments.push({
              type: 'text',
              content: msg.content,
            });
            currentAssistant.content =
              (currentAssistant.content || '') + msg.content;
          }
        }
      }

      // 添加最后的助手消息
      if (currentAssistant) {
        // 计算工具总时间
        if (
          currentAssistant.toolCalls &&
          currentAssistant.toolCalls.length > 0
        ) {
          let totalToolTime = 0;
          currentAssistant.toolCalls.forEach(tc => {
            if (tc.executionTime) {
              const parsed = this.parseDuration(tc.executionTime);
              totalToolTime += parsed;
            }
          });
          if (totalToolTime > 0) {
            currentAssistant.toolDuration = this.formatDuration(totalToolTime);
          }
        }
        result.push(currentAssistant);
      }

      return result;
    },

    parseDuration(durationStr) {
      if (!durationStr) return 0;
      const match = durationStr.match(/(\d+)m\s*(\d+)s/);
      if (match) {
        return parseInt(match[1]) * 60000 + parseInt(match[2]) * 1000;
      }
      const seconds = durationStr.match(/(\d+)s/);
      if (seconds) {
        return parseInt(seconds[1]) * 1000;
      }
      const ms = durationStr.match(/(\d+)ms/);
      if (ms) {
        return parseInt(ms[1]);
      }
      return 0;
    },

    formatMessage(msg) {
      if (!msg) return null;

      // 如果已经是标准格式
      if (msg.role && (msg.content || msg.toolCalls || msg.reasoning)) {
        return {
          id: msg.id || this.generateId(),
          role: msg.role,
          content: this.formatContent(msg.content),
          toolCalls: msg.toolCalls || null,
          toolResults: msg.toolResults || null,
          toolCallId: msg.toolCallId || null,
          reasoning: msg.reasoning || '',
          reasoningDuration: msg.reasoningDuration || '',
          toolDuration: msg.toolDuration || '',
        };
      }

      // 处理 AG-UI 协议格式
      if (msg.type) {
        switch (msg.type) {
          case 'TEXT_MESSAGE':
          case 'text_message':
            return {
              id: msg.id || msg.messageId || this.generateId(),
              role: msg.role || 'assistant',
              content: this.formatContent(msg.content || msg.text),
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
              id: msg.id || this.generateId(),
              role: 'tool',
              content: this.formatContent(msg.result || msg.content),
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
              return this.formatMessage(msg.message);
            }
            return null;
        }
      }

      // 尝试处理嵌套结构
      if (msg.message) {
        return this.formatMessage(msg.message);
      }

      // 跳过无效消息
      if (!msg.role && !msg.content && !msg.text) {
        return null;
      }

      return {
        id: msg.id || this.generateId(),
        role: msg.role || 'unknown',
        content: this.formatContent(msg.content || msg.text || ''),
        toolCalls: msg.toolCalls || null,
        toolResults: msg.toolResults || null,
        toolCallId: msg.toolCallId || null,
        reasoning: msg.reasoning || '',
        reasoningDuration: msg.reasoningDuration || '',
        toolDuration: msg.toolDuration || '',
      };
    },

    async loadConfig() {
      if (!this.currentThreadId) return;
      try {
        const res = await getGeneralAgentConfig({
          threadId: this.currentThreadId,
        });
        if (res.code === 0 && res.data) {
          if (res.data.modelConfig) {
            const modelConfig = res.data.modelConfig;
            if (modelConfig.modelId) {
              this.selectedModel = modelConfig.modelId;
            }
          }
          if (res.data.toolList && Array.isArray(res.data.toolList)) {
            this.selectedTools = res.data.toolList.map(tool => ({
              toolId: tool.toolId,
              toolName: tool.toolName,
              toolType: tool.toolType,
              enable: tool.enable,
            }));
          }
          if (res.data.assistantList && Array.isArray(res.data.assistantList)) {
            this.selectedAssistants = res.data.assistantList.map(assistant => ({
              assistantId: assistant.agentId,
              name: assistant.name,
            }));
          }
        }
      } catch (error) {
        console.error('加载配置失败:', error);
      }
    },

    async saveModelConfig() {
      if (!this.currentThreadId) {
        this.$message.warning('请先创建对话');
        return;
      }
      if (!this.selectedModel) {
        this.$message.warning('请选择模型');
        return;
      }
      try {
        const selectedModelConfig = this.modelList.find(
          m => m.modelId === this.selectedModel,
        );
        const res = await updateGeneralAgentConfig({
          threadId: this.currentThreadId,
          modelConfig: {
            modelId: this.selectedModel,
            model: selectedModelConfig?.model || '',
            provider: selectedModelConfig?.provider || '',
            displayName: selectedModelConfig?.modelName || '',
            modelType: selectedModelConfig?.modelType || 'llm',
            config: selectedModelConfig?.config || {},
          },
          toolList: this.selectedTools.map(t => ({
            toolId: t.toolId,
            toolType: t.toolType,
          })),
        });
        if (res.code === 0) {
          this.$message.success('模型配置已保存');
        } else {
          this.$message.error(res.msg || '保存模型配置失败');
        }
      } catch (error) {
        console.error('保存模型配置失败:', error);
        this.$message.error('保存模型配置失败');
      }
    },

    formatContent(content) {
      if (typeof content === 'string') return content;
      if (Array.isArray(content)) {
        return content
          .filter(item => item.type === 'text')
          .map(item => item.text)
          .join('\n');
      }
      if (typeof content === 'object' && content?.text) return content.text;
      return '';
    },

    handleKeyDown(e) {
      if (e.shiftKey) return;
      e.preventDefault();
      this.sendMessage();
    },

    handleFileChange(file) {
      const reader = new FileReader();
      reader.onload = () => {
        this.uploadedFiles.push({
          name: file.name,
          type: file.raw.type,
          url: reader.result,
          data: reader.result.split(',')[1],
        });
      };
      reader.readAsDataURL(file.raw);
    },

    removeFile(index) {
      this.uploadedFiles.splice(index, 1);
    },

    handleModelChange() {
      this.saveModelConfig();
    },

    async sendMessage() {
      const content = this.inputMessage.trim();
      if (!content && this.uploadedFiles.length === 0) return;
      if (this.isStreaming) return;

      if (!this.currentThreadId) {
        const created = await this.createConversation();
        if (!created) {
          this.$message.error('创建对话失败，请重试');
          return;
        }
      }

      const userMessage = this.buildUserMessage(content);

      this.messageList.push({
        id: this.generateId(),
        role: 'user',
        content: content,
        files: [...this.uploadedFiles],
      });

      this.inputMessage = '';
      this.uploadedFiles = [];
      this.$nextTick(() => this.scrollToBottom());

      await this.startStreaming(userMessage);
    },

    buildUserMessage(content) {
      const message = { id: this.generateId(), role: 'user' };
      if (this.uploadedFiles.length === 0) {
        message.content = content;
      } else {
        const contentArray = [];
        if (content) contentArray.push({ type: 'text', text: content });
        this.uploadedFiles.forEach(file => {
          contentArray.push({
            type: 'binary',
            mimeType: file.type,
            url: file.url,
          });
        });
        message.content = contentArray;
      }
      return message;
    },

    async startStreaming(userMessage) {
      if (!this.currentThreadId) {
        this.$message.error('对话ID不存在，请刷新页面重试');
        this.isStreaming = false;
        return;
      }

      this.isStreaming = true;
      this.abortController = new AbortController();

      const assistantMessage = {
        id: this.generateId(),
        role: 'assistant',
        content: '',
        reasoning: '',
        toolCalls: [],
        toolResults: [],
        isStreaming: true,
        stageTimers: {
          thinking: { start: null, duration: '' },
          tool: { start: null, duration: '' },
        },
        reasoningDuration: '',
        toolDuration: '',
      };
      this.messageList.push(assistantMessage);
      this.streamingMessage = assistantMessage;
      this.currentStage = 'understanding';

      const parser = new SSEEventParser();

      try {
        await chatGeneralAgentConversation({
          threadId: this.currentThreadId,
          messages: [userMessage],
          onMessage: event => {
            this.handleSSEEvent(event, assistantMessage, parser);
          },
          onError: error => {
            console.error('SSE Error:', error);
            this.$message.error('对话请求失败');
            assistantMessage.isStreaming = false;
            this.isStreaming = false;
            this.currentStage = '';
          },
          signal: this.abortController.signal,
        });
      } catch (error) {
        console.error('Stream error:', error);
        if (error.name !== 'AbortError') {
          this.$message.error('发送消息失败: ' + (error.message || error));
        }
      } finally {
        assistantMessage.isStreaming = false;
        this.isStreaming = false;
        this.abortController = null;
        this.streamingMessage = null;
        this.currentStage = '';
      }
    },

    handleSSEEvent(event, assistantMessage, parser) {
      const parsed = parser.parse(event);
      if (!parsed) return;

      switch (parsed.type) {
        case 'RUN_STARTED':
          this.currentRunId = parsed.runId;
          this.currentStage = 'understanding';
          break;

        case 'TEXT_MESSAGE_START':
          assistantMessage.id = parsed.messageId;
          break;

        case 'TEXT_MESSAGE_CONTENT':
          if (!assistantMessage.content && this.currentStage !== 'generating') {
            this.currentStage = 'generating';
          }
          if (
            parsed.messageId === assistantMessage.id ||
            !assistantMessage.id
          ) {
            assistantMessage.id = parsed.messageId;
            assistantMessage.content += parsed.delta || '';
          }
          break;

        case 'REASONING_START':
        case 'REASONING_MESSAGE_START':
          if (this.currentStage !== 'thinking') {
            this.currentStage = 'thinking';
            assistantMessage.stageTimers.thinking.start = Date.now();
          }
          break;

        case 'REASONING_MESSAGE_CONTENT':
          if (!assistantMessage.stageTimers.thinking.start) {
            assistantMessage.stageTimers.thinking.start = Date.now();
          }
          assistantMessage.reasoning += parsed.delta || '';
          const thinkingElapsed =
            Date.now() - assistantMessage.stageTimers.thinking.start;
          assistantMessage.reasoningDuration =
            this.formatDuration(thinkingElapsed);
          break;

        case 'REASONING_END':
        case 'REASONING_MESSAGE_END':
          if (assistantMessage.stageTimers.thinking.start) {
            const elapsed =
              Date.now() - assistantMessage.stageTimers.thinking.start;
            assistantMessage.reasoningDuration = this.formatDuration(elapsed);
          }
          break;

        case 'TOOL_CALL_START':
          if (this.currentStage !== 'tool_calling') {
            this.currentStage = 'tool_calling';
            assistantMessage.stageTimers.tool.start = Date.now();
          }
          assistantMessage.toolCalls.push({
            id: parsed.toolCallId,
            name: parsed.toolCallName,
            arguments: '',
            status: 'running',
            startTime: Date.now(),
          });
          break;

        case 'TOOL_CALL_ARGS':
          const tool = assistantMessage.toolCalls.find(
            t => t.id === parsed.toolCallId,
          );
          if (tool) tool.arguments += parsed.delta || '';
          break;

        case 'TOOL_CALL_END':
          const toolToEnd = assistantMessage.toolCalls.find(
            t => t.id === parsed.toolCallId,
          );
          if (toolToEnd) {
            toolToEnd.status = 'completed';
            if (toolToEnd.startTime) {
              toolToEnd.executionTime = this.formatDuration(
                Date.now() - toolToEnd.startTime,
              );
            }
          }
          break;

        case 'TOOL_CALL_RESULT':
          if (!assistantMessage.toolResults) {
            assistantMessage.toolResults = [];
          }
          assistantMessage.toolResults.push({
            toolCallId: parsed.toolCallId,
            content: parsed.content,
          });
          if (assistantMessage.stageTimers.tool.start) {
            const toolElapsed =
              Date.now() - assistantMessage.stageTimers.tool.start;
            assistantMessage.toolDuration = this.formatDuration(toolElapsed);
          }
          break;

        case 'RUN_FINISHED':
          // SSE 结束时不再重新加载历史，保留实时构建的消息
          // this.fetchHistory();
          break;
      }
      this.$nextTick(() => this.scrollToBottom());
    },

    formatDuration(ms) {
      if (ms < 1000) {
        return `${ms}ms`;
      }
      const seconds = Math.floor(ms / 1000);
      const minutes = Math.floor(seconds / 60);
      const secs = seconds % 60;
      if (minutes > 0) {
        return `${minutes}m ${secs}s`;
      }
      return `${secs}s`;
    },

    stopStreaming() {
      if (this.abortController) {
        this.abortController.abort();
        this.abortController = null;
      }
      this.isStreaming = false;
    },

    scrollToBottom() {
      const anchor = this.$refs.scrollAnchor;
      if (anchor) anchor.scrollIntoView({ behavior: 'smooth' });
    },

    generateId() {
      return (
        'msg_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
      );
    },

    getToolResultsForMessage(message) {
      if (message.toolResults && message.toolResults.length > 0) {
        return message.toolResults;
      }
      return [];
    },

    // 重新生成 - 找到上一条用户消息并重新发送
    handleRegenerate(message) {
      if (this.isStreaming) return;

      // 找到这条助手消息的索引
      const messageIndex = this.messageList.findIndex(m => m.id === message.id);
      if (messageIndex <= 0) return;

      // 找到上一条用户消息
      let userMessage = null;
      for (let i = messageIndex - 1; i >= 0; i--) {
        if (this.messageList[i].role === 'user') {
          userMessage = this.messageList[i];
          break;
        }
      }

      if (!userMessage) return;

      // 删除当前助手消息及之后的消息
      this.messageList = this.messageList.slice(0, messageIndex);

      // 重新构建用户消息
      const userContent = userMessage.content;
      const userFiles = userMessage.files || [];

      // 设置输入内容并触发发送
      this.inputMessage = typeof userContent === 'string' ? userContent : '';
      this.uploadedFiles = userFiles.length > 0 ? [...userFiles] : [];

      // 如果有内容，直接发送
      if (this.inputMessage.trim() || this.uploadedFiles.length > 0) {
        this.$nextTick(() => {
          this.sendMessage();
        });
      }
    },

    async handleCommand(command, item) {
      if (command === 'delete') {
        try {
          await this.$confirm('确定要删除这个对话吗？', '提示', {
            type: 'warning',
          });
          const res = await deleteGeneralAgentConversation({
            threadId: item.threadId,
          });
          if (res.code === 0) {
            this.$message.success('删除成功');
            if (this.currentThreadId === item.threadId) {
              this.currentThreadId = '';
              this.messageList = [];
            }
            this.fetchConversationList();
          }
        } catch (error) {
          if (error !== 'cancel') console.error('删除对话失败:', error);
        }
      }
    },
  },
};
</script>

<style lang="scss" scoped>
$claude-primary: #10a37f;
$claude-primary-light: #1ab38b;
$claude-primary-dark: #0d8a6a;
$claude-bg: #ffffff;
$claude-bg-secondary: #f7f7f8;
$claude-border: #e5e5e5;
$claude-text: #1a1a1a;
$claude-text-secondary: #666666;
$claude-text-muted: #999999;
$message-max-width: 900px;

.general-agent-page {
  display: flex;
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: $claude-bg;
  overflow: hidden;
}

.sidebar {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 240px;
  height: 100%;
  background: $claude-bg-secondary;
  border-right: 1px solid $claude-border;
  transition: all 0.3s ease;
  overflow: hidden;

  &.collapsed {
    width: 56px;
    min-width: 56px;

    .sidebar-collapsed-bar {
      display: flex;
      flex-direction: column;
      align-items: center;
      padding: 16px 0;
      gap: 12px;
    }

    .sidebar-header,
    .sidebar-divider,
    .conversation-list {
      display: none;
    }
  }

  .sidebar-collapsed-bar {
    display: none;
  }

  .expand-btn {
    width: 40px;
    height: 40px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-radius: 10px;
    cursor: pointer;
    color: $claude-text-muted;
    transition: all 0.2s;

    &:hover {
      background: rgba($claude-primary, 0.1);
      color: $claude-primary;
    }

    i {
      font-size: 18px;
    }
  }

  .sidebar-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px;
    border-bottom: 1px solid $claude-border;

    .new-chat-btn {
      flex: 1;
      margin-right: 12px;
      border-radius: 12px;
      background: $claude-primary;
      border-color: $claude-primary;
      font-weight: 500;

      &:hover {
        background: $claude-primary-dark;
        border-color: $claude-primary-dark;
      }
    }

    .sidebar-toggle {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 36px;
      height: 36px;
      border-radius: 8px;
      cursor: pointer;
      color: $claude-text-muted;
      transition: all 0.2s;
      flex-shrink: 0;

      &:hover {
        color: $claude-primary;
        background: rgba($claude-primary, 0.1);
      }

      i {
        font-size: 18px;
      }
    }
  }

  .sidebar-divider {
    height: 1px;
    background: $claude-border;
    flex-shrink: 0;
  }

  .conversation-list {
    flex: 1;
    overflow-y: auto;
    padding: 8px;
    min-height: 0;

    &::-webkit-scrollbar {
      width: 4px;
    }

    &::-webkit-scrollbar-track {
      background: transparent;
    }

    &::-webkit-scrollbar-thumb {
      background: #d1d5db;
      border-radius: 2px;
    }
  }

  .conversation-item {
    display: flex;
    align-items: center;
    padding: 12px 14px;
    border-radius: 10px;
    cursor: pointer;
    margin-bottom: 4px;
    transition: background-color 0.2s;

    &:hover {
      background: rgba($claude-primary, 0.08);

      .el-icon-more {
        opacity: 1;
      }
    }

    &.active {
      background: rgba($claude-primary, 0.12);

      .conversation-title {
        font-weight: 500;
      }
    }

    i:first-child {
      margin-right: 10px;
      color: $claude-text-muted;
      font-size: 16px;
    }

    .conversation-title {
      flex: 1;
      font-size: 14px;
      color: $claude-text;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .el-icon-more {
      opacity: 0;
      color: $claude-text-muted;
      padding: 4px;
      transition: opacity 0.2s;

      &:hover {
        color: $claude-primary;
      }
    }
  }
}

.agent-main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  position: relative;
  overflow: hidden;
}

.header {
  flex: none;
  height: 56px;
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: $claude-bg;
  border-bottom: 1px solid $claude-border;

  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: $claude-text;
  }
}

.message-area {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: $claude-bg;

  .empty-state {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    height: 100%;
    color: $claude-text-muted;
    padding: 24px;

    .empty-icon {
      font-size: 64px;
      margin-bottom: 16px;
      color: #d1d5db;
    }

    .empty-title {
      font-size: 20px;
      color: $claude-text;
      font-weight: 500;
      margin-bottom: 8px;
    }

    .empty-tips {
      font-size: 14px;
      color: $claude-text-secondary;
    }
  }

  .message-list {
    max-width: $message-max-width;
    margin: 0 auto;
    padding: 24px;
    min-height: 100%;
  }
}

.typing-indicator {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 24px;

  span:not(.typing-text) {
    width: 8px;
    height: 8px;
    background: $claude-primary;
    border-radius: 50%;
    animation: bounce 1.4s infinite ease-in-out;

    &:nth-child(1) {
      animation-delay: 0s;
    }
    &:nth-child(2) {
      animation-delay: 0.2s;
    }
    &:nth-child(3) {
      animation-delay: 0.4s;
    }
  }

  .typing-text {
    margin-left: 8px;
    color: $claude-text-muted;
    font-size: 14px;
  }
}

@keyframes bounce {
  0%,
  60%,
  100% {
    transform: translateY(0);
    opacity: 0.6;
  }
  30% {
    transform: translateY(-6px);
    opacity: 1;
  }
}

.input-area {
  flex: none;
  background: $claude-bg;
  border-top: 1px solid $claude-border;
  padding: 16px 24px 24px;

  .input-container {
    max-width: $message-max-width;
    margin: 0 auto;
    background: $claude-bg-secondary;
    border-radius: 16px;
    border: 1px solid $claude-border;
    padding: 12px 16px;
    transition:
      border-color 0.2s,
      box-shadow 0.2s;

    &:focus-within {
      border-color: $claude-primary;
      box-shadow: 0 0 0 2px rgba($claude-primary, 0.1);
    }
  }

  .model-selector {
    display: flex;
    align-items: center;
    margin-bottom: 12px;
    padding-bottom: 12px;
    border-bottom: 1px solid $claude-border;

    ::v-deep .el-select {
      width: 160px;

      .el-input__inner {
        background: transparent;
        border: none;
        padding-left: 0;
        font-size: 13px;
        color: $claude-text;
      }
    }
  }

  .file-preview {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 12px;

    .file-item {
      position: relative;
      width: 48px;
      height: 48px;

      .file-thumb {
        width: 100%;
        height: 100%;
        object-fit: cover;
        border-radius: 8px;
      }

      .file-icon {
        width: 100%;
        height: 100%;
        display: flex;
        align-items: center;
        justify-content: center;
        background: #e5e7eb;
        border-radius: 8px;
        color: $claude-text-secondary;
      }

      .file-remove {
        position: absolute;
        top: -4px;
        right: -4px;
        width: 18px;
        height: 18px;
        background: #ef4444;
        color: #fff;
        border-radius: 50%;
        display: flex;
        align-items: center;
        justify-content: center;
        cursor: pointer;
        font-size: 10px;
        transition: transform 0.2s;

        &:hover {
          transform: scale(1.1);
        }
      }
    }
  }

  .input-wrapper {
    display: flex;
    align-items: flex-end;
    gap: 12px;

    ::v-deep .el-textarea {
      flex: 1;

      .el-textarea__inner {
        background: transparent;
        border: none;
        padding: 0;
        resize: none;
        font-size: 15px;
        line-height: 1.6;
        color: $claude-text;

        &::placeholder {
          color: $claude-text-muted;
        }
      }
    }

    .input-actions {
      display: flex;
      align-items: center;
      gap: 8px;

      .action-icon {
        font-size: 20px;
        color: $claude-text-muted;
        cursor: pointer;
        padding: 4px;
        border-radius: 6px;
        transition: all 0.2s;

        &:hover {
          color: $claude-primary;
          background: rgba($claude-primary, 0.1);
        }
      }

      .el-button--primary {
        background: $claude-primary;
        border-color: $claude-primary;

        &:hover {
          background: $claude-primary-dark;
          border-color: $claude-primary-dark;
        }

        &:disabled {
          background: #d1d5db;
          border-color: #d1d5db;
        }
      }

      .el-button--danger {
        background: #ef4444;
        border-color: #ef4444;

        &:hover {
          background: #dc2626;
          border-color: #dc2626;
        }
      }
    }
  }

  .input-footer {
    text-align: center;
    font-size: 12px;
    color: $claude-text-muted;
    margin-top: 12px;
  }
}
</style>
