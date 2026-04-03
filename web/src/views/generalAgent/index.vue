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
    <div
      class="agent-main-content"
      :class="{ 'has-workspace': panelVisible && activeWorkspace }"
    >
      <!-- 主消息区域 -->
      <div class="main-content-body">
        <!-- 顶部标题栏 -->
        <div class="header">
          <div class="header-left">
            <button
              v-if="!sidebarCollapsed"
              class="sidebar-toggle-btn"
              @click="toggleSidebar"
            >
              <i class="el-icon-s-fold"></i>
            </button>
            <div class="header-title">{{ currentTitle }}</div>
          </div>
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
              :thread-id="currentThreadId"
              @regenerate="handleRegenerate"
              @view-workspace="handleViewWorkspace"
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
            <!-- 模型选择和配置 -->
            <div class="model-config-row">
              <div class="model-selector">
                <el-select
                  v-model="selectedModel"
                  size="small"
                  placeholder="选择模型"
                  filterable
                  :filter-method="filterModel"
                  @change="handleModelChange"
                >
                  <el-option
                    v-for="model in filteredModelList"
                    :key="model.modelId"
                    :label="model.modelName"
                    :value="model.modelId"
                  >
                    <div class="model-option">
                      <span class="model-name">{{ model.modelName }}</span>
                      <span v-if="model.provider" class="model-provider">
                        {{ model.provider }}
                      </span>
                    </div>
                  </el-option>
                </el-select>
              </div>

              <!-- 配置按钮 -->
              <div
                class="config-btn"
                :class="{ 'has-selection': selectedTools.length > 0 }"
                @click="showConfigDrawer = true"
              >
                <i class="el-icon-setting"></i>
                <span>配置</span>
                <el-badge
                  v-if="selectedTools.length > 0"
                  :value="selectedTools.length"
                  type="primary"
                />
              </div>
            </div>

            <!-- 文件预览 -->
            <div v-if="uploadedFiles.length > 0" class="file-preview">
              <div
                v-for="(file, index) in uploadedFiles"
                :key="index"
                class="file-item"
                :class="{ 'is-uploading': file.uploading }"
              >
                <img
                  v-if="file.type.startsWith('image/')"
                  :src="file.displayUrl || file.url"
                  class="file-thumb"
                />
                <div v-else class="file-icon">
                  <i class="el-icon-document"></i>
                </div>
                <!-- 上传进度遮罩 -->
                <div v-if="file.uploading" class="upload-overlay">
                  <div class="upload-progress-bar">
                    <svg viewBox="0 0 36 36" width="36" height="36">
                      <circle class="progress-bg" cx="18" cy="18" r="15" />
                      <circle
                        class="progress-fill"
                        cx="18"
                        cy="18"
                        r="15"
                        :stroke-dasharray="94.2"
                        :stroke-dashoffset="
                          94.2 - (94.2 * (file.uploadProgress || 0)) / 100
                        "
                      />
                    </svg>
                    <span class="progress-text">
                      {{ file.uploadProgress || 0 }}
                    </span>
                  </div>
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

      <!-- Workspace 面板 -->
      <transition name="workspace-slide">
        <workspace-panel
          v-if="panelVisible && activeWorkspace"
          :thread-id="activeWorkspace.threadId"
          :run-id="activeWorkspace.runId"
          :initial-data="currentWorkspaceTree"
          @close="hidePanel"
        />
      </transition>

      <!-- 配置抽屉 -->
      <el-drawer
        :visible.sync="showConfigDrawer"
        direction="rtl"
        size="400px"
        :with-header="false"
        custom-class="config-drawer"
      >
        <div class="drawer-content">
          <div class="drawer-header">
            <h3>对话配置</h3>
            <i class="el-icon-close" @click="showConfigDrawer = false"></i>
          </div>

          <div class="drawer-body">
            <!-- 工具选择 -->
            <div class="drawer-section">
              <div class="section-header">
                <i class="el-icon-setting"></i>
                <span>工具选择</span>
                <el-tag size="mini" type="info">
                  {{ selectedTools.length }} 已选
                </el-tag>
              </div>

              <!-- 搜索框 -->
              <div class="tool-search">
                <el-input
                  v-model="toolSearchKeyword"
                  size="small"
                  placeholder="搜索工具..."
                  prefix-icon="el-icon-search"
                  clearable
                />
              </div>

              <div class="section-body">
                <div v-if="loadingTools" class="config-loading">
                  <i class="el-icon-loading"></i>
                  加载中...
                </div>
                <div
                  v-else-if="filteredToolList.length === 0"
                  class="config-empty"
                >
                  <i class="el-icon-search"></i>
                  <span>未找到匹配的工具</span>
                </div>
                <div v-else class="tool-categories">
                  <div
                    v-for="category in filteredToolList"
                    :key="category.category"
                    class="tool-category"
                  >
                    <div class="category-header">
                      <span class="category-name">{{ category.category }}</span>
                      <el-tag
                        size="mini"
                        :type="getConditionType(category.condition)"
                      >
                        {{ getConditionLabel(category.condition) }}
                      </el-tag>
                    </div>
                    <div class="tool-list">
                      <el-tooltip
                        v-for="tool in category.toolList"
                        :key="tool.toolId"
                        placement="top"
                        :open-delay="500"
                        :disabled="!tool.description && !tool.desc"
                        effect="light"
                        popper-class="tool-tooltip-popper"
                      >
                        <div slot="content" class="tool-detail-tooltip">
                          <div class="tooltip-title">{{ tool.toolName }}</div>
                          <div class="tooltip-desc">
                            {{
                              tool.description || tool.desc || '暂无详细描述'
                            }}
                          </div>
                        </div>
                        <div
                          :class="[
                            'tool-item',
                            { selected: isToolSelected(tool.toolId) },
                          ]"
                          @click="toggleTool(tool)"
                        >
                          <div class="tool-avatar">
                            <img
                              v-if="tool.avatar?.path"
                              :src="tool.avatar.path"
                            />
                            <i v-else class="el-icon-setting"></i>
                          </div>
                          <div class="tool-info">
                            <div class="tool-name">{{ tool.toolName }}</div>
                            <div class="tool-desc">{{ tool.desc }}</div>
                          </div>
                          <el-checkbox
                            :value="isToolSelected(tool.toolId)"
                            @click.native.stop
                            @change="toggleTool(tool)"
                          />
                        </div>
                      </el-tooltip>
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </el-drawer>
    </div>
  </div>
</template>

<script>
import MessageItem from './components/MessageItem.vue';
import WorkspacePanel from './components/WorkspacePanel.vue';
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
  getGeneralAgentWorkspace,
  uploadGeneralAgentFile,
} from '@/api/generalAgent';
import { SSEEventParser, EventType, ActivityType } from './utils/sse-parser';
import { mapState, mapActions, mapGetters } from 'vuex';

export default {
  name: 'GeneralAgent',
  components: {
    MessageItem,
    WorkspacePanel,
  },
  data() {
    return {
      sidebarCollapsed: false,
      conversationList: [],
      currentThreadId: '',
      pageNo: 1,
      pageSize: 50,
      total: 0,

      // 每个会话独立的消息列表 { threadId: messageList }
      messagesMap: {},
      inputMessage: '',
      uploadedFiles: [],
      // 每个会话独立的流式状态 { threadId: { isStreaming, abortController, streamingMessage } }
      streamingMap: {},

      selectedModel: '',
      selectedTools: [],
      selectedAssistants: [],
      modelList: [],
      modelSearchKeyword: '',
      toolList: [],
      loadingTools: false,
      showConfigDrawer: false,
      toolSearchKeyword: '',

      currentRunId: '',
      currentStage: '',

      // Workspace 相关
      workspacePanelVisible: false,
      workspaceLoading: false,
      workspaceInfo: null,
    };
  },
  computed: {
    ...mapState('workspace', ['activeWorkspace', 'panelVisible']),
    ...mapGetters('workspace', ['hasWorkspace', 'currentWorkspaceTree']),

    // 当前会话的消息列表
    messageList: {
      get() {
        return this.messagesMap[this.currentThreadId] || [];
      },
      set(val) {
        this.$set(this.messagesMap, this.currentThreadId, val);
      },
    },
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
    // 过滤后的工具列表
    filteredToolList() {
      if (!this.toolSearchKeyword.trim()) {
        return this.toolList;
      }
      const keyword = this.toolSearchKeyword.toLowerCase().trim();
      return this.toolList
        .map(category => {
          const filteredTools = category.toolList.filter(tool => {
            const name = (tool.toolName || '').toLowerCase();
            const desc = (tool.desc || '').toLowerCase();
            const description = (tool.description || '').toLowerCase();
            return (
              name.includes(keyword) ||
              desc.includes(keyword) ||
              description.includes(keyword)
            );
          });
          if (filteredTools.length === 0) return null;
          return {
            ...category,
            toolList: filteredTools,
          };
        })
        .filter(Boolean);
    },
    // 过滤后的模型列表
    filteredModelList() {
      if (!this.modelSearchKeyword.trim()) {
        return this.modelList;
      }
      const keyword = this.modelSearchKeyword.toLowerCase().trim();
      return this.modelList.filter(model => {
        const name = (model.modelName || '').toLowerCase();
        const provider = (model.provider || '').toLowerCase();
        const modelType = (model.modelType || '').toLowerCase();
        return (
          name.includes(keyword) ||
          provider.includes(keyword) ||
          modelType.includes(keyword)
        );
      });
    },
    // Workspace 相关
    workspaceThreadAndRun() {
      if (this.activeWorkspace && this.currentThreadId) {
        return {
          threadId: this.currentThreadId,
          runId: this.activeWorkspace.runId,
        };
      }
      return null;
    },
  },
  watch: {
    panelVisible(val) {
      this.workspacePanelVisible = val;
      if (val && this.activeWorkspace) {
        this.loadWorkspaceFiles();
      }
    },
  },
  mounted() {
    this.fetchModelList();
    this.fetchConversationList();
    this.fetchToolList();
  },
  beforeDestroy() {
    // 清理所有会话的流式状态
    Object.keys(this.streamingMap).forEach(threadId => {
      const streaming = this.streamingMap[threadId];
      if (streaming && streaming.abortController) {
        streaming.abortController.abort();
      }
    });
    this.streamingMap = {};
    this.reset();
  },
  methods: {
    ...mapActions('workspace', [
      'handleWorkspaceActivity',
      'showPanel',
      'hidePanel',
      'setWorkspaceTree',
      'setActiveWorkspace',
      'clearWorkspace',
      'reset',
    ]),

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
      this.loadingTools = true;
      try {
        const res = await getGeneralAgentToolSelect();
        if (res.code === 0 && res.data) {
          this.toolList = res.data || [];
        }
      } catch (error) {
        console.error('获取工具列表失败:', error);
      } finally {
        this.loadingTools = false;
      }
    },

    async createConversation() {
      try {
        // 检查模型列表是否已加载
        if (!this.modelList || this.modelList.length === 0) {
          this.$message.warning('模型列表加载中，请稍后重试');
          return false;
        }

        // 获取默认模型配置
        const defaultModel = this.modelList[0];
        const modelConfig = {
          modelId: defaultModel?.modelId || '',
          model: defaultModel?.model || '',
          provider: defaultModel?.provider || '',
          displayName: defaultModel?.modelName || '',
          modelType: defaultModel?.modelType || 'llm',
          config: defaultModel?.config || {},
        };

        const res = await createGeneralAgentConversation({
          title: '新对话',
          modelConfig,
        });
        if (res.code === 0) {
          const threadId = res.data?.threadId;
          if (threadId) {
            this.currentThreadId = threadId;
            // 初始化新会话的消息列表
            this.$set(this.messagesMap, threadId, []);
            this.selectedModel = modelConfig.modelId;
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
      // 切换会话时，只切换 currentThreadId，不中止 SSE 流
      // SSE 流会继续在后台运行，切换回来时能继续显示
      this.currentThreadId = threadId;
      this.fetchHistory();
    },

    async fetchHistory() {
      if (!this.currentThreadId) return;

      // 如果当前会话正在流式传输，不清空消息
      if (this.isStreaming) {
        console.log('[fetchHistory] 当前会话正在流式传输，跳过获取历史');
        return;
      }

      // 初始化当前会话的消息列表
      if (!this.messagesMap[this.currentThreadId]) {
        this.$set(this.messagesMap, this.currentThreadId, []);
      }

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
          // 使用 $set 确保响应式
          this.$set(this.messagesMap, this.currentThreadId, allMessages);
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

          // Workspace 活动快照
          case 'ACTIVITY_SNAPSHOT': {
            if (event.activityType === 'workspace' && event.content) {
              // 更新 workspace store
              this.handleWorkspaceActivity({
                runId: event.content.runId,
                threadId: event.content.threadId || this.currentThreadId,
                fileCount: event.content.fileCount || 0,
                totalSize: event.content.totalSize || 0,
                timestamp: event.content.timestamp || eventTimestamp,
              });

              // 创建 workspace 片段消息
              messages.push({
                id: event.messageId || this.generateId(),
                role: 'assistant',
                type: 'workspace',
                isWorkspaceActivity: true,
                workspaceInfo: {
                  fileCount: event.content.fileCount || 0,
                  totalSize: event.content.totalSize || 0,
                },
                runId: event.content.runId,
                toolCalls: null,
              });
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
          } else if (msg.isWorkspaceActivity) {
            // Workspace 活动片段
            currentAssistant.fragments.push({
              type: 'workspace',
              workspaceInfo: msg.workspaceInfo,
              runId: msg.runId,
            });
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
        console.log('loadConfig response:', res);
        if (res.code === 0 && res.data) {
          if (res.data.modelConfig) {
            const modelConfig = res.data.modelConfig;
            console.log('modelConfig:', modelConfig);
            // 使用 model 或 modelId，取决于后端返回
            this.selectedModel = modelConfig.modelId || modelConfig.model || '';
            console.log('selectedModel set to:', this.selectedModel);
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
              assistantId: assistant.agentId || assistant.assistantId,
              name: assistant.name,
            }));
          }
        }
      } catch (error) {
        console.error('加载配置失败:', error);
      }
    },

    async saveModelConfig(silent = false) {
      if (!this.currentThreadId) {
        if (!silent) {
          this.$message.warning('请先创建对话');
        }
        return;
      }
      if (!this.selectedModel) {
        if (!silent) {
          this.$message.warning('请选择模型');
        }
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
          if (!silent) {
            this.$message.success('配置已保存');
          }
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

    // 将内部服务地址转换为外部可访问地址
    convertToExternalUrl(url) {
      if (!url) return url;
      // 替换 minio 内部服务名为外部地址
      return url.replace(/minio-wanwu:9000/g, '192.168.0.21:9000');
    },

    async handleFileChange(file) {
      // 先显示本地预览
      const localUrl = URL.createObjectURL(file.raw);
      const tempFile = {
        name: file.name,
        type: file.raw.type,
        url: localUrl,
        localUrl: localUrl,
        uploading: true,
        uploadProgress: 0,
      };
      this.uploadedFiles.push(tempFile);

      // 上传文件到服务器
      try {
        const res = await uploadGeneralAgentFile(file.raw, percent => {
          // 更新进度
          const index = this.uploadedFiles.findIndex(
            f => f.localUrl === localUrl,
          );
          if (index !== -1) {
            this.$set(this.uploadedFiles[index], 'uploadProgress', percent);
          }
        });
        if (res.code === 0 && res.data?.files?.[0]?.filePath) {
          // 更新为服务器 URL（转换为外部可访问地址）
          const index = this.uploadedFiles.findIndex(
            f => f.localUrl === localUrl,
          );
          if (index !== -1) {
            // 使用 Vue.set 确保响应式更新
            this.$set(this.uploadedFiles, index, {
              name: file.name,
              type: file.raw.type,
              url: res.data.files[0].filePath, // 原始 minio URL，用于发送给后端
              displayUrl: this.convertToExternalUrl(res.data.files[0].filePath), // 转换后的 URL，用于前端显示
              uploading: false,
              uploadProgress: 100,
            });
          }
          URL.revokeObjectURL(localUrl);
        } else {
          // 上传失败，移除文件
          const index = this.uploadedFiles.findIndex(
            f => f.localUrl === localUrl,
          );
          if (index !== -1) {
            this.uploadedFiles.splice(index, 1);
          }
          this.$message.error(res.msg || '文件上传失败');
          URL.revokeObjectURL(localUrl);
        }
      } catch (error) {
        console.error('文件上传失败:', error);
        const index = this.uploadedFiles.findIndex(
          f => f.localUrl === localUrl,
        );
        if (index !== -1) {
          this.uploadedFiles.splice(index, 1);
        }
        this.$message.error('文件上传失败');
        URL.revokeObjectURL(localUrl);
      }
    },

    removeFile(index) {
      this.uploadedFiles.splice(index, 1);
    },

    handleModelChange() {
      this.saveModelConfig();
    },

    // 模型搜索过滤
    filterModel(keyword) {
      this.modelSearchKeyword = keyword || '';
    },

    isToolSelected(toolId) {
      return this.selectedTools.some(t => t.toolId === toolId);
    },

    async toggleTool(tool) {
      const index = this.selectedTools.findIndex(t => t.toolId === tool.toolId);
      if (index > -1) {
        this.selectedTools.splice(index, 1);
      } else {
        this.selectedTools.push({
          toolId: tool.toolId,
          toolName: tool.toolName,
          toolType: tool.toolType,
        });
      }
      // 静默保存配置，不显示消息
      await this.saveModelConfig(true);
    },

    getConditionLabel(condition) {
      const labels = {
        none: '可选',
        optional: '推荐',
        required: '必选',
      };
      return labels[condition] || condition;
    },

    getConditionType(condition) {
      const types = {
        none: 'info',
        optional: 'warning',
        required: 'danger',
      };
      return types[condition] || 'info';
    },

    async sendMessage() {
      const content = this.inputMessage.trim();
      if (!content && this.uploadedFiles.length === 0) return;

      // 检查当前会话是否正在流式传输
      const currentStreaming = this.streamingMap[this.currentThreadId];
      if (currentStreaming && currentStreaming.isStreaming) return;

      // 检查是否有文件正在上传
      const uploadingFiles = this.uploadedFiles.filter(f => f.uploading);
      if (uploadingFiles.length > 0) {
        this.$message.warning('请等待文件上传完成');
        return;
      }

      if (!this.currentThreadId) {
        const created = await this.createConversation();
        if (!created) {
          this.$message.error('创建对话失败，请重试');
          return;
        }
      }

      const userMessage = this.buildUserMessage(content);

      // 确保当前会话的消息列表存在
      if (!this.messagesMap[this.currentThreadId]) {
        this.$set(this.messagesMap, this.currentThreadId, []);
      }

      // 添加用户消息到当前会话
      const messages = this.messagesMap[this.currentThreadId];
      messages.push({
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

      // 如果没有文件，直接返回文本
      if (this.uploadedFiles.length === 0) {
        message.content = content;
        return message;
      }

      // 有文件时，构建多部分内容
      const contentArray = [];

      // 添加文本内容（如果有）
      if (content && content.trim()) {
        contentArray.push({ type: 'text', text: content.trim() });
      }

      // 添加文件内容 - 后端统一使用 type: 'binary'，根据 mimeType 判断具体类型
      this.uploadedFiles.forEach(file => {
        contentArray.push({
          type: 'binary',
          mimeType: file.type || 'application/octet-stream',
          url: file.url, // 使用服务器返回的 HTTP URL
        });
      });

      message.content = contentArray;
      return message;
    },

    async startStreaming(userMessage) {
      if (!this.currentThreadId) {
        this.$message.error('对话ID不存在，请刷新页面重试');
        return;
      }

      const streamingThreadId = this.currentThreadId;

      // 初始化该会话的流式状态
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
        stageTimers: {
          thinking: { start: null, duration: '' },
          tool: { start: null, duration: '' },
        },
        reasoningDuration: '',
        toolDuration: '',
        threadId: streamingThreadId,
      };

      // 设置该会话的流式状态
      this.$set(this.streamingMap, streamingThreadId, {
        isStreaming: true,
        abortController: abortController,
        streamingMessage: assistantMessage,
      });

      // 确保该会话的消息列表存在
      if (!this.messagesMap[streamingThreadId]) {
        this.$set(this.messagesMap, streamingThreadId, []);
      }

      // 添加消息到对应会话的消息列表
      const messages = this.messagesMap[streamingThreadId];
      messages.push(assistantMessage);

      this.currentStage = 'understanding';

      const parser = new SSEEventParser();

      try {
        await chatGeneralAgentConversation({
          threadId: streamingThreadId,
          messages: [userMessage],
          onMessage: event => {
            // 直接更新对应会话的消息，不检查当前会话
            this.handleSSEEvent(
              event,
              assistantMessage,
              parser,
              streamingThreadId,
            );
          },
          onError: error => {
            console.error('SSE Error:', error);
            // 只在对应会话显示错误提示
            if (this.currentThreadId === streamingThreadId) {
              this.$message.error('对话请求失败');
            }
            // 更新该会话的流式状态
            const streaming = this.streamingMap[streamingThreadId];
            if (streaming) {
              streaming.isStreaming = false;
              streaming.streamingMessage = null;
            }
            assistantMessage.isStreaming = false;
          },
          signal: abortController.signal,
        });
      } catch (error) {
        console.error('Stream error:', error);
        if (
          error.name !== 'AbortError' &&
          this.currentThreadId === streamingThreadId
        ) {
          this.$message.error('发送消息失败: ' + (error.message || error));
        }
      } finally {
        // 更新该会话的流式状态
        const streaming = this.streamingMap[streamingThreadId];
        if (streaming) {
          streaming.isStreaming = false;
          streaming.streamingMessage = null;
          streaming.abortController = null;
        }
        assistantMessage.isStreaming = false;
        this.currentStage = '';
      }
    },

    handleSSEEvent(event, assistantMessage, parser, streamingThreadId) {
      const parsed = parser.parse(event);
      if (!parsed) return;

      switch (parsed.type) {
        case 'RUN_STARTED':
          this.currentRunId = parsed.runId;
          if (this.currentThreadId === streamingThreadId) {
            this.currentStage = 'understanding';
          }
          break;

        case 'TEXT_MESSAGE_START':
          assistantMessage.id = parsed.messageId;
          break;

        case 'TEXT_MESSAGE_CONTENT':
          if (
            !assistantMessage.content &&
            this.currentThreadId === streamingThreadId &&
            this.currentStage !== 'generating'
          ) {
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
          if (
            this.currentThreadId === streamingThreadId &&
            this.currentStage !== 'thinking'
          ) {
            this.currentStage = 'thinking';
          }
          if (!assistantMessage.stageTimers.thinking.start) {
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
          if (
            this.currentThreadId === streamingThreadId &&
            this.currentStage !== 'tool_calling'
          ) {
            this.currentStage = 'tool_calling';
          }
          if (!assistantMessage.stageTimers.tool.start) {
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
          // SSE 结束时规范化 fragments，确保所有内容都被正确添加
          this.normalizeFragments(assistantMessage);
          break;

        case 'ACTIVITY_SNAPSHOT':
          console.log('[ACTIVITY_SNAPSHOT] Received:', parsed);
          this.handleActivitySnapshot(
            parsed,
            assistantMessage,
            streamingThreadId,
          );
          break;
      }
      // 只有当前会话才滚动到底部
      if (this.currentThreadId === streamingThreadId) {
        this.$nextTick(() => this.scrollToBottom());
      }
    },

    handleActivitySnapshot(event, assistantMessage, streamingThreadId) {
      console.log('[handleActivitySnapshot] event:', event);
      console.log('[handleActivitySnapshot] activityType:', event.activityType);
      console.log(
        '[handleActivitySnapshot] ActivityType.WORKSPACE:',
        ActivityType.WORKSPACE,
      );
      console.log('[handleActivitySnapshot] content:', event.content);
      console.log(
        '[handleActivitySnapshot] assistantMessage.fragments:',
        assistantMessage?.fragments,
      );

      const { activityType, content } = event;

      if (activityType === ActivityType.WORKSPACE) {
        console.log(
          '[handleActivitySnapshot] MATCH! Processing workspace activity',
        );

        // 处理 Workspace 活动
        const result = this.handleWorkspaceActivity({
          runId: content.runId || this.currentRunId,
          threadId: content.threadId || this.currentThreadId,
          fileCount: content.fileCount || 0,
          totalSize: content.totalSize || 0,
          timestamp: content.timestamp || Date.now(),
        });

        // 添加 workspace 片段到消息
        if (assistantMessage.fragments) {
          console.log('[handleActivitySnapshot] Adding fragment to message');
          assistantMessage.fragments.push({
            type: 'workspace',
            workspaceInfo: {
              fileCount: content.fileCount || 0,
              totalSize: content.totalSize || 0,
            },
            runId: content.runId || this.currentRunId,
          });
          console.log(
            '[handleActivitySnapshot] fragments after push:',
            assistantMessage.fragments,
          );
        } else {
          console.log(
            '[handleActivitySnapshot] WARNING: fragments is not initialized',
          );
        }

        // 只有当前会话时才显示通知
        if (this.currentThreadId === streamingThreadId) {
          this.$notify({
            type: 'success',
            title: '工作空间已更新',
            message: `生成了 ${content.fileCount || 0} 个文件`,
            duration: 3000,
            onClick: () => {
              this.showPanel();
            },
          });
        }

        // 如果面板已打开，刷新文件列表
        if (result && result.shouldRefresh) {
          this.loadWorkspaceFiles();
        }
      } else {
        console.log(
          '[handleActivitySnapshot] NOT MATCHED, activityType:',
          activityType,
        );
      }
    },

    // 规范化 fragments，确保所有内容都被正确添加
    normalizeFragments(assistantMessage) {
      if (!assistantMessage) return;

      const fragments = [];
      let hasReasoning =
        assistantMessage.reasoning && assistantMessage.reasoning.length > 0;
      let hasToolCalls =
        assistantMessage.toolCalls && assistantMessage.toolCalls.length > 0;
      let hasContent =
        assistantMessage.content && assistantMessage.content.length > 0;
      let hasWorkspace =
        assistantMessage.fragments &&
        assistantMessage.fragments.some(f => f.type === 'workspace');

      // 如果已经有 fragments 且包含 workspace，需要重新组织
      const existingWorkspaceFragments = (
        assistantMessage.fragments || []
      ).filter(f => f.type === 'workspace');

      // 添加思考片段
      if (hasReasoning) {
        fragments.push({
          type: 'reasoning',
          content: assistantMessage.reasoning,
          duration: assistantMessage.reasoningDuration || '',
        });
      }

      // 添加工具调用片段
      if (hasToolCalls) {
        assistantMessage.toolCalls.forEach(tc => {
          fragments.push({
            type: 'tool_call',
            toolCall: tc,
          });
        });
      }

      // 添加文本片段
      if (hasContent) {
        fragments.push({
          type: 'text',
          content: assistantMessage.content,
        });
      }

      // 添加工作空间片段
      existingWorkspaceFragments.forEach(ws => {
        fragments.push(ws);
      });

      // 只有当有内容时才更新 fragments
      if (fragments.length > 0) {
        assistantMessage.fragments = fragments;
      }
    },

    async loadWorkspaceFiles() {
      if (!this.activeWorkspace || !this.currentThreadId) return;

      this.workspaceLoading = true;
      try {
        const res = await getGeneralAgentWorkspace({
          threadId: this.currentThreadId,
          runId: this.activeWorkspace.runId,
        });
        if (res.code === 0 && res.data) {
          this.setWorkspaceTree({
            threadId: this.currentThreadId,
            runId: this.activeWorkspace.runId,
            data: res.data,
          });
        }
      } catch (error) {
        console.error('加载工作空间文件失败:', error);
      } finally {
        this.workspaceLoading = false;
      }
    },

    toggleWorkspacePanel() {
      if (this.panelVisible) {
        this.hidePanel();
      } else {
        this.showPanel();
        if (this.activeWorkspace) {
          this.loadWorkspaceFiles();
        }
      }
    },

    handleViewWorkspace(data) {
      console.log('[handleViewWorkspace] data:', data);
      console.log(
        '[handleViewWorkspace] currentThreadId:',
        this.currentThreadId,
      );
      console.log(
        '[handleViewWorkspace] activeWorkspace before:',
        this.activeWorkspace,
      );

      // 设置 activeWorkspace
      this.setActiveWorkspace({
        runId: data.runId,
        threadId: data.threadId || this.currentThreadId,
        fileCount: data.fileCount || 0,
        totalSize: data.totalSize || 0,
        timestamp: Date.now(),
      });

      console.log(
        '[handleViewWorkspace] activeWorkspace after:',
        this.activeWorkspace,
      );

      // 收起会话列表
      if (!this.sidebarCollapsed) {
        this.sidebarCollapsed = true;
      }
      // 打开工作空间面板
      this.showPanel();

      console.log('[handleViewWorkspace] panelVisible:', this.panelVisible);
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
      // 中止当前会话的 SSE 流
      const streaming = this.streamingMap[this.currentThreadId];
      if (streaming && streaming.abortController) {
        streaming.abortController.abort();
        streaming.isStreaming = false;
        streaming.streamingMessage = null;
        streaming.abortController = null;
      }
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
  min-width: 0;
  min-height: 0;
  position: relative;
  overflow: hidden;

  &.has-workspace {
    .main-content-body {
      flex: 1;
      min-width: 0;
    }
  }
}

.main-content-body {
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
  justify-content: space-between;
  padding: 0 24px;
  background: $claude-bg;
  border-bottom: 1px solid $claude-border;

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: $claude-text;
  }

  .sidebar-toggle-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid $claude-border;
    background: #fff;
    border-radius: 8px;
    cursor: pointer;
    color: $claude-text-muted;
    transition: all 0.2s;

    &:hover {
      border-color: $claude-primary;
      color: $claude-primary;
      background: rgba($claude-primary, 0.05);
    }

    i {
      font-size: 16px;
    }
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

  .model-config-row {
    display: flex;
    align-items: center;
    gap: 12px;
    margin-bottom: 12px;
    padding-bottom: 12px;
    border-bottom: 1px solid $claude-border;
  }

  .model-selector {
    display: flex;
    align-items: center;

    ::v-deep .el-select {
      width: 200px;

      .el-input__inner {
        background: transparent;
        border: none;
        padding-left: 0;
        font-size: 13px;
        color: $claude-text;
      }
    }
  }

  .model-option {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;

    .model-name {
      flex: 1;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .model-provider {
      flex-shrink: 0;
      margin-left: 8px;
      padding: 2px 6px;
      font-size: 11px;
      color: #666;
      background: #f5f5f5;
      border-radius: 4px;
    }
  }

  .config-btn {
    display: flex;
    align-items: center;
    gap: 4px;
    padding: 6px 12px;
    border-radius: 8px;
    cursor: pointer;
    font-size: 13px;
    color: $claude-text-secondary;
    background: transparent;
    border: 1px solid $claude-border;
    transition: all 0.2s;

    &:hover {
      background: rgba($claude-primary, 0.08);
      color: $claude-primary;
      border-color: rgba($claude-primary, 0.3);
    }

    &.has-selection {
      color: $claude-primary;
      border-color: rgba($claude-primary, 0.3);
      background: rgba($claude-primary, 0.05);
    }

    i {
      font-size: 16px;
    }

    .el-badge {
      margin-left: 4px;
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
        z-index: 10;

        &:hover {
          transform: scale(1.1);
        }
      }

      .upload-overlay {
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        background: rgba(0, 0, 0, 0.5);
        border-radius: 8px;
        display: flex;
        align-items: center;
        justify-content: center;
        z-index: 5;

        .upload-progress-bar {
          width: 32px;
          height: 32px;
          position: relative;
          display: flex;
          align-items: center;
          justify-content: center;

          svg {
            position: absolute;
            top: 0;
            left: 0;
            transform: rotate(-90deg);

            circle {
              fill: none;
              stroke-width: 3;
            }

            .progress-bg {
              stroke: rgba(255, 255, 255, 0.3);
            }

            .progress-fill {
              stroke: #fff;
              stroke-linecap: round;
              transition: stroke-dashoffset 0.3s ease;
            }
          }

          .progress-text {
            color: #fff;
            font-size: 9px;
            font-weight: 600;
            z-index: 1;
          }
        }
      }

      &.is-uploading {
        .file-remove {
          display: none;
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

// Workspace 面板过渡动画
.workspace-slide-enter-active,
.workspace-slide-leave-active {
  transition: all 0.3s ease;
}

.workspace-slide-enter,
.workspace-slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

// Workspace 面板容器（需要添加）
.workspace-panel {
  width: 320px;
  flex-shrink: 0;
}
</style>

<style lang="scss">
// 配置抽屉样式 - 需要非 scoped 才能覆盖 Element UI
.config-drawer {
  .drawer-content {
    height: 100%;
    display: flex;
    flex-direction: column;
  }

  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 16px 20px;
    border-bottom: 1px solid #e5e5e5;

    h3 {
      margin: 0;
      font-size: 16px;
      font-weight: 500;
      color: #1a1a1a;
    }

    .el-icon-close {
      font-size: 18px;
      color: #999;
      cursor: pointer;
      transition: color 0.2s;

      &:hover {
        color: #10a37f;
      }
    }
  }

  .drawer-body {
    flex: 1;
    overflow-y: auto;
    padding: 16px 20px;
  }

  .drawer-section {
    margin-bottom: 24px;

    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 16px;
      font-size: 14px;
      font-weight: 500;
      color: #1a1a1a;

      i {
        font-size: 16px;
        color: #10a37f;
      }
    }
  }

  .config-loading {
    text-align: center;
    color: #999;
    padding: 24px;
  }

  .tool-categories {
    .tool-category {
      margin-bottom: 16px;

      .category-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 10px;
        padding-bottom: 8px;
        border-bottom: 1px solid #f0f0f0;

        .category-name {
          font-size: 13px;
          font-weight: 500;
          color: #1a1a1a;
        }
      }
    }
  }

  .tool-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .tool-item {
    display: flex;
    align-items: center;
    padding: 10px 12px;
    border-radius: 10px;
    cursor: pointer;
    transition: all 0.2s;
    border: 1px solid transparent;

    &:hover {
      background: #f5f7fa;
      border-color: #e4e7ed;
    }

    &.selected {
      background: rgba(16, 163, 127, 0.08);
      border-color: rgba(16, 163, 127, 0.2);
    }

    .tool-avatar {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      margin-right: 12px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: #f0f0f0;
      overflow: hidden;
      flex-shrink: 0;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }

      i {
        font-size: 18px;
        color: #999;
      }
    }

    .tool-info {
      flex: 1;
      min-width: 0;

      .tool-name {
        font-size: 14px;
        font-weight: 500;
        color: #1a1a1a;
        margin-bottom: 2px;
      }

      .tool-desc {
        font-size: 12px;
        color: #666;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }

    .el-checkbox {
      margin-left: 8px;
    }
  }

  .tool-search {
    margin-bottom: 16px;

    .el-input {
      .el-input__inner {
        border-radius: 8px;
      }
    }
  }

  .config-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 32px;
    color: #999;

    i {
      font-size: 32px;
      margin-bottom: 8px;
    }

    span {
      font-size: 14px;
    }
  }
}

// 工具详情 tooltip
.tool-tooltip-popper {
  max-width: 360px !important;
  padding: 0 !important;
  border: 1px solid #e4e7ed !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12) !important;
  border-radius: 8px !important;
}

.tool-detail-tooltip {
  padding: 12px 14px;

  .tooltip-title {
    font-size: 14px;
    font-weight: 600;
    color: #1a1a1a;
    margin-bottom: 8px;
    padding-bottom: 8px;
    border-bottom: 1px solid #f0f0f0;
  }

  .tooltip-desc {
    font-size: 13px;
    color: #666;
    line-height: 1.6;
    white-space: pre-wrap;
    max-height: 200px;
    overflow-y: auto;
  }
}
</style>
