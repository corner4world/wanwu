<template>
  <div class="general-agent-page">
    <!-- 左侧会话列表 - 可折叠 -->
    <div :class="['sidebar', { collapsed: sidebarCollapsed }]">
      <div class="sidebar-content">
        <div class="sidebar-header">
          <el-button
            type="primary"
            class="new-chat-btn"
            @click="initNewConversation"
          >
            <i class="el-icon-plus"></i>
            {{ $t('generalAgent.sidebar.newChat') }}
          </el-button>
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
            <span class="conversation-title">{{ item.title }}</span>
            <i
              class="el-icon-delete conversation-delete"
              @click.stop="handleDeleteConversation(item)"
            ></i>
          </div>
        </div>
      </div>
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
            <button class="header-icon-btn" @click="toggleSidebar">
              <i
                :class="
                  sidebarCollapsed ? 'el-icon-s-unfold' : 'el-icon-s-fold'
                "
              ></i>
            </button>
            <button class="header-icon-btn" @click="initNewConversation">
              <i class="el-icon-plus"></i>
            </button>
            <div class="header-title">{{ currentTitle }}</div>
          </div>
        </div>

        <!-- 消息区域 - 独立滚动 -->
        <div
          :class="[
            'message-area',
            { empty: isEmptyConversation && !isLoadingHistory },
          ]"
          ref="messageArea"
          @scroll="handleMessageAreaScroll"
        >
          <!-- 加载历史记录中 -->
          <div v-if="isLoadingHistory" class="history-loading">
            <i class="el-icon-loading"></i>
            <span>{{ $t('generalAgent.sidebar.loading') }}</span>
          </div>

          <!-- 消息列表 -->
          <div
            v-else-if="messageList.length > 0 || isStreaming"
            class="message-list"
          >
            <message-item
              v-for="(msg, index) in messageList"
              :key="msg.id || index"
              :message="msg"
              :tool-results="getToolResultsForMessage(msg)"
              :is-last-message="index === messageList.length - 1"
              :thread-id="currentThreadId"
              @regenerate="handleRegenerate"
              @view-workspace="handleViewWorkspace"
              @download-all="handleDownloadAll"
            />
          </div>

          <div ref="scrollAnchor"></div>
        </div>

        <!-- 滚动到底部按钮 -->
        <transition name="scroll-btn-fade">
          <button
            v-if="showScrollToBottom"
            class="scroll-to-bottom-btn"
            @click="handleScrollToBottomClick"
          >
            <svg
              viewBox="0 0 24 24"
              width="16"
              height="16"
              fill="none"
              stroke="currentColor"
              stroke-width="2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <polyline points="6,9 12,15 18,9"></polyline>
            </svg>
          </button>
        </transition>

        <!-- 底部输入区 -->
        <div
          :class="[
            'input-area',
            { 'is-centered': isEmptyConversation && !isLoadingHistory },
          ]"
        >
          <!-- 欢迎词 - 仅居中时显示 -->
          <div
            v-if="isEmptyConversation && !isLoadingHistory"
            class="welcome-section"
          >
            <div class="welcome-avatar">
              <img
                v-if="assistantAvatar"
                :src="assistantAvatar"
                alt="Assistant"
              />
              <i v-else class="el-icon-cpu"></i>
            </div>
            <div class="welcome-title">
              {{ $t('generalAgent.header.welcomeTitle') }}
            </div>
          </div>

          <div class="input-container">
            <!-- 文件预览 -->
            <div v-if="uploadedFiles.length > 0" class="file-preview">
              <!-- 图片文件 -->
              <div
                v-for="(file, index) in uploadedFiles"
                :key="index"
                class="echo-img-box"
                :class="{ 'is-uploading': file.uploading }"
              >
                <div class="echo-img-item">
                  <!-- 图片类型 -->
                  <el-image
                    v-if="file.type && file.type.startsWith('image/')"
                    class="echo-img"
                    :src="file.displayUrl"
                    :preview-src-list="[file.displayUrl]"
                  ></el-image>
                  <!-- 文档类型 -->
                  <div v-else class="echo-doc-box">
                    <img
                      :src="require('@/assets/imgs/fileicon.png')"
                      class="docIcon"
                    />
                    <div class="docInfo">
                      <p class="docInfo_name">文件名：{{ file.fileName }}</p>
                      <p class="docInfo_size">
                        文件大小：{{
                          file.size > 1024
                            ? (file.size / (1024 * 1024)).toFixed(2) + ' MB'
                            : (file.size || 0) + ' bytes'
                        }}
                      </p>
                    </div>
                  </div>
                  <!-- 删除按钮 -->
                  <i
                    class="el-icon-close echo-close"
                    @click="removeFile(index)"
                  ></i>
                </div>
              </div>
            </div>

            <!-- 模型选择 -->
            <div style="margin-bottom: 12px">
              <ModelSelect
                v-model="selectedModel"
                :options="modelList"
                placeholder="选择模型"
                :loading="modelLoading"
                :filterable="true"
                @change="handleModelChange"
                class="model-select-inline"
              />
            </div>

            <!-- 输入框 -->
            <div class="input-wrapper">
              <el-input
                v-model="inputMessage"
                type="textarea"
                :rows="1"
                :autosize="{ minRows: 1, maxRows: 6 }"
                :placeholder="inputPlaceholder"
                @keydown.enter.native="handleKeyDown"
                :disabled="isStreaming"
              />
            </div>

            <!-- 底部工具栏：配置按钮 + 发送按钮 -->
            <div class="input-toolbar">
              <div class="toolbar-left">
                <div class="config-btn" @click="showConfigDialog = true">
                  <i class="el-icon-setting"></i>
                  <span>{{ $t('generalAgent.header.config') }}</span>
                </div>
                <!-- 已选模式标签 -->
                <div
                  v-for="mode in selectedModes"
                  :key="mode.value"
                  class="mode-tag"
                >
                  <i :class="mode.icon"></i>
                  <span>{{ mode.label }}</span>
                  <i class="el-icon-close" @click="removeMode(mode.value)"></i>
                </div>
              </div>
              <div class="toolbar-right">
                <StreamUploadField
                  :fileTypeArr="['doc/*', 'image/*']"
                  type="agentChat"
                  @setFileId="handleSetFileId"
                >
                  <template #default="{ openDialog }">
                    <el-tooltip
                      :content="$t('generalAgent.header.uploadFile')"
                      placement="top"
                    >
                      <i
                        class="action-icon el-icon-paperclip"
                        @click="openDialog"
                      ></i>
                    </el-tooltip>
                  </template>
                </StreamUploadField>
                <el-button
                  v-if="isStreaming"
                  class="send-btn stop-btn"
                  circle
                  @click="stopStreaming"
                >
                  <svg
                    class="stop-icon"
                    viewBox="0 0 24 24"
                    width="16"
                    height="16"
                  >
                    <rect x="6" y="6" width="12" height="12" rx="2" />
                  </svg>
                </el-button>
                <el-button
                  v-else
                  type="primary"
                  class="send-btn"
                  circle
                  :disabled="!canSend"
                  @click="sendMessage"
                >
                  <svg
                    class="send-icon"
                    viewBox="0 0 24 24"
                    width="18"
                    height="18"
                  >
                    <path
                      fill="currentColor"
                      d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"
                    />
                  </svg>
                </el-button>
              </div>
            </div>
          </div>
          <div v-if="selectedModes.length === 0" class="mode-buttons">
            <div
              v-for="(mode, key) in modeOptions"
              :key="key"
              class="mode-btn"
              @click="addMode(mode.value)"
            >
              <i :class="mode.icon"></i>
              <span>{{ mode.label }}</span>
            </div>
          </div>
          <div class="input-footer">
            <span>{{ $t('generalAgent.header.footer') }}</span>
          </div>
        </div>
      </div>

      <!-- Workspace 面板 -->
      <transition name="workspace-slide">
        <workspace-panel
          v-if="panelVisible && activeWorkspace"
          ref="workspacePanel"
          :thread-id="activeWorkspace.threadId"
          :run-id="activeWorkspace.runId"
          :initial-data="currentWorkspaceTree"
          @close="hidePanel"
          @preview-file="handlePreviewFile"
        />
      </transition>

      <!-- 文件预览抽屉 -->
      <file-preview-drawer
        :visible.sync="previewVisible"
        :file="previewFile"
        :file-path="previewFilePath"
        :blob="previewBlob"
        :loading="previewLoading"
        :panel-style="previewPanelStyle"
      />

      <!-- 配置弹窗 -->
      <configDialog :visible.sync="showConfigDialog" />
    </div>
  </div>
</template>

<script>
import MessageItem from './components/MessageItem.vue';
import WorkspacePanel from './components/WorkspacePanel.vue';
import FilePreviewDrawer from './components/FilePreviewDrawer.vue';
import ConfigDialog from './components/ConfigDialog.vue';
import ModelSelect from '@/components/modelSelect.vue';
import StreamUploadField from '@/components/stream/streamUploadField.vue';
import {
  chatGeneralAgentConversation,
  createGeneralAgentConversation,
  deleteGeneralAgentConversation,
  downloadGeneralAgentWorkspace,
  getGeneralAgentConversationConfig,
  getGeneralAgentConversationDetail,
  getGeneralAgentConversationList,
  getGeneralAgentWorkspace,
  previewGeneralAgentWorkspace,
  updateGeneralAgentConversationConfig,
} from '@/api/generalAgent';
import { selectModelList } from '@/api/modelAccess';
import { avatarSrc, resDownloadFile } from '@/utils/util';
import { mapActions, mapGetters, mapState } from 'vuex';
import { SSEEventParser } from './utils/sse-parser';
// 引入工具函数
import {
  aggregateEventsToMessages,
  formatMessage,
} from './utils/message-aggregator';
// 引入 Mixins
import streamStateManager from './mixins/streamStateManager';
import messageManager from './mixins/messageManager';
import fileManager from './mixins/fileManager';
import scrollController from './mixins/scrollController';
import modeManager from './mixins/modeManager';

export default {
  name: 'GeneralAgent',
  components: {
    MessageItem,
    WorkspacePanel,
    FilePreviewDrawer,
    ConfigDialog: ConfigDialog,
    ModelSelect,
    StreamUploadField,
  },
  mixins: [
    streamStateManager,
    messageManager,
    fileManager,
    scrollController,
    modeManager,
  ],
  data() {
    return {
      sidebarCollapsed: true,
      conversationList: [],
      currentThreadId: '',
      pageNo: 1,
      pageSize: 50,
      isNewConversation: false,
      isLoadingHistory: false,

      inputMessage: '',
      selectedModel: '',
      modelList: [],
      modelLoading: false,
      showConfigDialog: false,

      currentRunId: '',
      currentStage: '',

      // Workspace 相关
      workspacePanelVisible: false,
      workspaceLoading: false,
      workspaceInfo: null,

      // 文件预览
      previewVisible: false,
      previewLoading: false,
      previewFile: null,
      previewFilePath: '',
      previewBlob: null, // 只存储 blob
      workspaceRect: null,
      resizeObserver: null,
    };
  },
  computed: {
    ...mapState('workspace', ['activeWorkspace', 'panelVisible']),
    ...mapGetters('workspace', ['hasWorkspace', 'currentWorkspaceTree']),
    ...mapGetters('user', ['commonInfo']),

    assistantAvatar() {
      return avatarSrc(this.commonInfo?.data?.tab?.logo?.path);
    },

    currentTitle() {
      if (!this.currentThreadId) return '';
      const conv = this.conversationList.find(
        c => c.threadId === this.currentThreadId,
      );
      return conv?.title || '新对话';
    },
    canSend() {
      const hasContent =
        this.inputMessage.trim() || this.uploadedFiles.length > 0;
      const hasModel = !!this.selectedModel;
      return hasContent && hasModel;
    },
    previewPanelStyle() {
      // 计算宽度：屏幕宽度的一半，最小 500px
      const screenWidth = window.screen.width;
      const halfScreenWidth = Math.floor(screenWidth / 2);
      const width = Math.max(500, halfScreenWidth);

      if (!this.workspaceRect) {
        // 默认情况：workspace 未显示时，使用计算值
        const sidebarWidth = this.sidebarCollapsed ? 0 : 240;
        const sidebarMargin = this.sidebarCollapsed ? 0 : 16;
        const workspaceWidth = 400;
        const pagePadding = 16;
        return {
          right: `${pagePadding + sidebarWidth + sidebarMargin + workspaceWidth}px`,
          width: `${width}px`,
        };
      }
      // workspace 显示时，紧贴其左边缘
      const rightEdge = window.innerWidth - this.workspaceRect.left;
      return {
        right: `${rightEdge}px`,
        width: `${width}px`,
      };
    },
    isEmptyConversation() {
      return this.messageList.length === 0;
    },
    inputPlaceholder() {
      // 如果有选中的模式，使用第一个模式的 placeholder
      if (this.selectedModes && this.selectedModes.length > 0) {
        const firstMode = this.selectedModes[0];
        const modeConfig = this.modeOptions[firstMode.value];
        if (modeConfig && modeConfig.placeholder) {
          return modeConfig.placeholder;
        }
      }
      // 默认 placeholder
      return this.$t('generalAgent.header.placeholder');
    },
  },
  watch: {
    panelVisible(val) {
      this.workspacePanelVisible = val;
      if (val && this.activeWorkspace) {
        this.loadWorkspaceFiles();
        this.$nextTick(() => this.updateWorkspaceRect());
      } else if (!val) {
        // 工作空间关闭时，关闭文件预览
        this.previewVisible = false;
      }
    },
    previewVisible(val) {
      if (val) {
        this.$nextTick(() => this.updateWorkspaceRect());
      }
    },
  },
  mounted() {
    this.initNewConversation();
    this.fetchModelList();
    this.fetchConversationList();
    this.initUserInfo();
    this.setupResizeObserver();
  },
  beforeDestroy() {
    this.reset();
    if (this.resizeObserver) {
      this.resizeObserver.disconnect();
      this.resizeObserver = null;
    }
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
    ...mapActions('user', ['getPermissionInfo', 'getCommonInfo']),

    async initUserInfo() {
      if (localStorage.getItem('access_cert')) {
        await this.getPermissionInfo();
        await this.getCommonInfo();
      }
    },

    setupResizeObserver() {
      if (typeof ResizeObserver === 'undefined') return;
      this.resizeObserver = new ResizeObserver(() => {
        this.updateWorkspaceRect();
      });
      // 监听整个页面容器
      const pageEl = this.$el;
      if (pageEl) {
        this.resizeObserver.observe(pageEl);
      }
      // 也监听 sidebar 变化
      const sidebar = pageEl?.querySelector('.sidebar');
      if (sidebar) {
        this.resizeObserver.observe(sidebar);
      }
    },

    updateWorkspaceRect() {
      this.$nextTick(() => {
        const workspaceEl = this.$refs.workspacePanel?.$el;
        if (workspaceEl) {
          this.workspaceRect = workspaceEl.getBoundingClientRect();
        } else {
          // 如果 workspace 不可见，使用 mainContent 的右边界
          const mainContent = this.$el?.querySelector('.agent-main-content');
          if (mainContent) {
            const rect = mainContent.getBoundingClientRect();
            this.workspaceRect = { left: rect.right };
          }
        }
      });
    },

    toggleSidebar() {
      this.sidebarCollapsed = !this.sidebarCollapsed;
    },

    async fetchModelList() {
      this.modelLoading = true;
      try {
        const res = await selectModelList();
        if (res.code === 0 && res.data?.list) {
          this.modelList = res.data.list.map(model => ({
            modelId: model.modelId,
            displayName: model.displayName,
            model: model.model,
            provider: model.provider,
            modelType: model.modelType,
            config: model.config,
            avatar: model.avatar,
            tags: model.tags || [],
          }));
          if (!this.selectedModel)
            this.selectedModel = this.modelList[0].modelId;
        }
      } finally {
        this.modelLoading = false;
      }
    },

    async fetchConversationList() {
      const res = await getGeneralAgentConversationList({
        pageNo: this.pageNo,
        pageSize: this.pageSize,
      });
      if (res.code === 0) {
        this.conversationList = res.data?.list || [];
      }
    },

    initNewConversation() {
      this.currentThreadId = '';
      this.isNewConversation = true;
      this.clearMessages('');
      // 重置滚动状态
      this.resetScrollState();
      // 重置模式选择
      this.clearModes();
      // 关闭工作区面板
      this.hidePanel();
      this.$nextTick(() => {
        if (this.modelList && this.modelList.length > 0) {
          const defaultModel = this.modelList[0];
          this.selectedModel = defaultModel?.modelId || '';
        }
      });
    },

    async createConversationWithTitle(title) {
      if (!this.modelList || this.modelList.length === 0) {
        this.$message.warning(this.$t('generalAgent.error.modelListLoading'));
        return null;
      }

      // 使用用户选择的模型，如果没有选择则使用第一个模型
      const selectedModelConfig = this.selectedModel
        ? this.modelList.find(m => m.modelId === this.selectedModel)
        : this.modelList[0];

      const modelConfig = {
        modelId: selectedModelConfig?.modelId,
        model: selectedModelConfig?.model,
        provider: selectedModelConfig?.provider,
        displayName: selectedModelConfig?.displayName,
        modelType: selectedModelConfig?.modelType,
        config: selectedModelConfig?.config,
      };

      const res = await createGeneralAgentConversation({
        title: title || '新对话',
        modelConfig,
      });

      if (res.code === 0) {
        const threadId = res.data?.threadId;
        if (threadId) {
          this.currentThreadId = threadId;
          this.isNewConversation = false;

          const oldMessages = this.messagesMap[''] || [];
          this.$set(this.messagesMap, threadId, oldMessages);
          this.$delete(this.messagesMap, '');

          this.selectedModel = modelConfig.modelId;
          this.conversationList.unshift({
            threadId,
            title: title || '新对话',
            createdAt: new Date().toISOString(),
          });
          return threadId;
        } else {
          this.$message.error(this.$t('generalAgent.error.createFailed'));
        }
      } else {
        this.$message.error(
          res.msg || this.$t('generalAgent.error.createError'),
        );
      }
      return null;
    },

    selectConversation(threadId) {
      if (this.currentThreadId === threadId) return;
      // 切换会话时，只切换 currentThreadId，不中止 SSE 流
      // SSE 流会继续在后台运行，切换回来时能继续显示
      this.currentThreadId = threadId;
      this.isNewConversation = false;
      this.isLoadingHistory = true;
      this.resetScrollState();
      // 重置模式选择
      this.clearModes();
      this.hidePanel();
      this.fetchHistory();
    },

    async fetchHistory() {
      if (!this.currentThreadId) return;

      try {
        const res = await getGeneralAgentConversationDetail({
          threadId: this.currentThreadId,
          pageNo: 1,
          pageSize: 100,
        });

        if (res.code === 0 && res.data?.list) {
          const allMessages = [];
          res.data.list.forEach(run => {
            // 后端返回的是 events 字段，需要聚合为消息
            if (run.events && Array.isArray(run.events)) {
              const messages = aggregateEventsToMessages(run.events);
              allMessages.push(...messages);
            }
            // 兼容旧格式 messages 字段
            if (run.messages && Array.isArray(run.messages)) {
              run.messages.forEach(msg => {
                const formatted = formatMessage(msg);
                if (formatted) {
                  allMessages.push(formatted);
                }
              });
            }
            if (run.runId) this.currentRunId = run.runId;
          });
          console.log('allMessages', allMessages);

          // 检查是否有正在进行的流式传输
          const streaming = this.streamingMap[this.currentThreadId];
          const hasStreamingMessage =
            streaming && streaming.isStreaming && streaming.streamingMessage;

          // 如果有正在进行的流式消息，保留它
          if (hasStreamingMessage) {
            // 将历史消息和流式消息合并
            const streamingMsg = streaming.streamingMessage;
            // 检查流式消息是否已经在历史消息中（通过 messageId 或内容匹配）
            const isDuplicate = allMessages.some(
              msg =>
                msg.id === streamingMsg.id ||
                (msg.role === 'assistant' &&
                  msg.content === streamingMsg.content),
            );

            if (!isDuplicate) {
              allMessages.push(streamingMsg);
            } else {
              // 如果已存在，用流式消息替换历史消息（保留最新的流式内容）
              const index = allMessages.findIndex(
                msg =>
                  msg.id === streamingMsg.id ||
                  (msg.role === 'assistant' &&
                    msg.content === streamingMsg.content),
              );
              if (index !== -1) {
                allMessages[index] = streamingMsg;
              }
            }
          }

          // 使用 $set 确保响应式
          this.$set(this.messagesMap, this.currentThreadId, allMessages);
          // 先关闭加载状态，让消息列表渲染
          this.isLoadingHistory = false;
          // 等待 DOM 渲染完成后滚动到底部
          this.$nextTick(() => {
            requestAnimationFrame(() => {
              this.scrollToBottom(true);
            });
          });
        } else {
          this.isLoadingHistory = false;
        }
        this.loadConfig();
      } finally {
        this.isLoadingHistory = false;
      }
    },

    async loadConfig() {
      if (!this.currentThreadId) return;
      const res = await getGeneralAgentConversationConfig({
        threadId: this.currentThreadId,
      });
      if (res.code === 0 && res.data) {
        if (res.data.modelConfig) {
          const modelConfig = res.data.modelConfig;
          this.selectedModel = modelConfig.modelId || modelConfig.model;
        }
      }
    },

    handleKeyDown(e) {
      if (e.shiftKey) return;
      e.preventDefault();
      this.sendMessage();
    },

    handleModelChange(value) {
      if (!this.currentThreadId) {
        return;
      }
      const selectedModelConfig = this.modelList.find(m => m.modelId === value);
      const res = updateGeneralAgentConversationConfig({
        threadId: this.currentThreadId,
        modelConfig: {
          modelId: value,
          model: selectedModelConfig?.model,
          provider: selectedModelConfig?.provider,
          displayName: selectedModelConfig?.displayName,
          modelType: selectedModelConfig?.modelType || 'llm',
          config: selectedModelConfig?.config || {},
        },
      });
      if (res.code === 0) {
        this.$message.success(this.$t('generalAgent.config.saveSuccess'));
      }
    },

    async sendMessage() {
      const content = this.inputMessage.trim();
      if (!content && this.uploadedFiles.length === 0) return;

      // 检查当前会话是否正在流式传输
      const currentStreaming = this.streamingMap[this.currentThreadId];
      if (currentStreaming && currentStreaming.isStreaming) return;

      if (this.isNewConversation || !this.currentThreadId) {
        const title = content.slice(0, 50);
        const threadId = await this.createConversationWithTitle(title);
        if (!threadId) {
          this.$message.error(
            this.$t('generalAgent.error.createConversationFailed'),
          );
          return;
        }
      }

      const userMessage = this.buildUserMessage(content);
      this.ensureMessageList(this.currentThreadId);
      this.addUserMessage(this.currentThreadId, content, this.uploadedFiles);

      this.inputMessage = '';
      this.clearFiles();
      this.$nextTick(() => this.scrollToBottom());

      await this.startStreaming(userMessage);
    },

    async startStreaming(userMessage) {
      if (!this.currentThreadId) {
        this.$message.error(
          this.$t('generalAgent.error.conversationIdNotExist'),
        );
        return;
      }

      const streamingThreadId = this.currentThreadId;

      // 使用 mixin 初始化流式状态
      const { abortController, assistantMessage } =
        this.initStreamState(streamingThreadId);

      // 添加消息到对应会话的消息列表
      this.addAssistantMessage(streamingThreadId, assistantMessage);

      this.currentStage = 'understanding';
      this.resetScrollState();

      const parser = new SSEEventParser();

      try {
        await chatGeneralAgentConversation({
          threadId: streamingThreadId,
          messages: [userMessage],
          onMessage: event => {
            this.handleSSEEvent(
              event,
              assistantMessage,
              parser,
              streamingThreadId,
            );
          },
          onError: error => {
            console.error('SSE Error:', error);
            if (this.currentThreadId === streamingThreadId) {
              this.$message.error(
                this.$t('generalAgent.error.chatRequestFailed'),
              );
            }
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
          this.$message.error(
            this.$t('generalAgent.error.sendMessageFailed') +
              (error.message || error),
          );
        }
      } finally {
        const streaming = this.streamingMap[streamingThreadId];
        if (streaming) {
          streaming.isStreaming = false;
          streaming.streamingMessage = null;
          streaming.abortController = null;
        }
        assistantMessage.isStreaming = false;
        this.currentStage = '';
        if (this.currentThreadId === streamingThreadId) {
          this.resetScrollState();
          this.$nextTick(() => this.scrollToBottom(true));
        }
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
      } finally {
        this.workspaceLoading = false;
      }
    },

    handleViewWorkspace(data) {
      this.setActiveWorkspace({
        runId: data.runId,
        threadId: data.threadId || this.currentThreadId,
        fileCount: data.fileCount || 0,
        totalSize: data.totalSize || 0,
        timestamp: Date.now(),
      });
      this.showPanel();
    },

    async handlePreviewFile(data) {
      const { file, filePath, threadId, runId } = data;

      this.previewFile = file;
      this.previewFilePath = filePath;
      this.previewVisible = true;
      this.previewLoading = true;
      this.previewBlob = null;

      try {
        this.previewBlob = await previewGeneralAgentWorkspace({
          threadId,
          runId,
          path: filePath,
        });
      } finally {
        this.previewLoading = false;
      }
    },

    // 下载整个工作空间
    async handleDownloadAll() {
      try {
        const blob = await downloadGeneralAgentWorkspace({
          threadId: this.currentThreadId,
          runId: this.currentRunId,
          path: '',
        });
        resDownloadFile(blob, '工作空间.zip');
        this.$message.success(
          this.$t('generalAgent.workspace.downloadSuccess'),
        );
      } catch (error) {
        console.error('下载工作空间失败:', error);
        this.$message.error(this.$t('generalAgent.workspace.downloadFailed'));
      }
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

      // 删除当前助手消息
      this.removeMessage(this.currentThreadId, message.id);

      // 构建请求消息
      const requestMessage = this.buildRequestMessage(userMessage);

      // 直接调用 startStreaming
      this.$nextTick(() => {
        this.startStreaming(requestMessage);
      });
    },

    async handleDeleteConversation(item) {
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
          this.isNewConversation = true;
          this.messageList = [];
          this.hidePanel();
        }
        this.fetchConversationList();
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
  background: #f5f7fa;
  overflow: hidden;
  padding: 16px;
  box-sizing: border-box;
}

.sidebar {
  display: flex;
  flex-direction: column;
  flex-shrink: 0;
  width: 240px;
  height: 100%;
  background: #fff;
  border-radius: 12px;
  overflow: hidden;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
  transition:
    width 0.3s ease,
    margin-right 0.3s ease;
  margin-right: 16px;

  &.collapsed {
    width: 0;
    margin-right: 0;
    box-shadow: none;
  }

  .sidebar-content {
    display: flex;
    flex-direction: column;
    width: 240px;
    height: 100%;
    flex-shrink: 0;
  }

  .sidebar-header {
    flex-shrink: 0;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 16px;
    border-bottom: 1px solid #f0f0f0;

    .new-chat-btn {
      width: 100%;
      border-radius: 12px;
      background: $claude-primary;
      border-color: $claude-primary;
      font-weight: 500;

      &:hover {
        background: $claude-primary-dark;
        border-color: $claude-primary-dark;
      }
    }
  }

  .sidebar-divider {
    height: 1px;
    background: #f0f0f0;
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

      .conversation-delete {
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

    .conversation-delete {
      opacity: 0;
      color: $claude-text-muted;
      padding: 4px;
      font-size: 16px;
      transition: all 0.2s;
      cursor: pointer;

      &:hover {
        color: #f56c6c;
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
  background: #fff;
  border-radius: 12px;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);

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
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
  border-radius: 12px 12px 0 0;

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 12px;
  }

  .header-title {
    font-size: 16px;
    font-weight: 600;
    color: $claude-text;
  }

  .header-icon-btn {
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
  background: #fff;
  position: relative;

  &.empty {
    flex: none;
    min-height: 0;
    height: 0;
    overflow: hidden;
  }

  .message-list {
    max-width: $message-max-width;
    margin: 0 auto;
    padding: 24px;
    min-height: 100%;
  }

  .history-loading {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    color: #909399;
    font-size: 14px;
    gap: 12px;
    background: #fff;
    z-index: 10;

    i {
      font-size: 32px;
      color: #10a37f;
    }
  }
}

.scroll-to-bottom-btn {
  position: absolute;
  bottom: 120px;
  left: 50%;
  transform: translateX(-50%);
  width: 36px;
  height: 36px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  color: #10a37f;
  border: 1px solid #10a37f;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transition: all 0.2s ease;
  z-index: 100;

  &:hover {
    background: #10a37f;
    color: #fff;
    transform: translateX(-50%) translateY(-2px);
    box-shadow: 0 4px 12px rgba(16, 163, 127, 0.4);
  }

  svg {
    width: 16px;
    height: 16px;
  }
}

.scroll-btn-fade-enter-active,
.scroll-btn-fade-leave-active {
  transition: all 0.3s ease;
}

.scroll-btn-fade-enter,
.scroll-btn-fade-leave-to {
  opacity: 0;
  transform: translateX(-50%) translateY(20px);
}

.input-area {
  flex: none;
  background: #fff;
  padding: 16px 24px 24px;
  border-radius: 0 0 12px 12px;

  &.is-centered {
    flex: 1;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    border-top: none;
    padding: 0 24px;

    .input-container {
      max-width: 800px;
      width: 100%;
    }

    .welcome-section {
      display: flex;
      flex-direction: column;
      align-items: center;
      margin-bottom: 32px;

      .welcome-avatar {
        width: 72px;
        height: 72px;
        border-radius: 20px;
        display: flex;
        align-items: center;
        justify-content: center;
        margin-bottom: 20px;
        background: #fff;
        box-shadow: 0 4px 16px rgba(0, 0, 0, 0.08);
        overflow: hidden;

        img {
          width: 100%;
          height: 100%;
          border-radius: 20px;
          object-fit: cover;
        }

        i {
          font-size: 32px;
          color: #10a37f;
        }
      }

      .welcome-title {
        font-size: 28px;
        color: $claude-text;
        font-weight: 600;
      }
    }
  }

  &:not(.is-centered) {
    border-top: none;
  }

  .input-container {
    max-width: $message-max-width;
    margin: 0 auto;
    background: #fff;
    border-radius: 16px;
    border: 1px solid #e5e7eb;
    padding: 16px;
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
    transition:
      border-color 0.2s,
      box-shadow 0.2s;

    &:focus-within {
      border-color: $claude-primary;
      box-shadow: 0 4px 24px rgba(0, 0, 0, 0.1);
    }
  }

  .file-preview {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    margin-bottom: 12px;

    .echo-img-box {
      position: relative;

      .echo-img-item {
        position: relative;
        display: inline-block;

        // 图片样式
        .echo-img {
          width: 48px;
          height: 48px;
          border-radius: 8px;
          cursor: pointer;
        }

        // 文档样式
        .echo-doc-box {
          background: #fff;
          min-width: 200px;
          max-width: 300px;
          border: 1px solid #dcdfe6;
          border-radius: 5px;
          display: flex;
          justify-content: space-between;
          align-items: center;
          padding: 10px 50px 10px 5px;

          .docIcon {
            width: 30px;
            height: 30px;
            flex-shrink: 0;
          }

          .docInfo {
            flex: 1;
            margin-left: 8px;
            overflow: hidden;

            .docInfo_name {
              color: #333;
              font-size: 13px;
              margin: 0;
              white-space: nowrap;
              overflow: hidden;
              text-overflow: ellipsis;
            }

            .docInfo_size {
              color: #bbbbbb;
              font-size: 12px;
              margin: 4px 0 0 0;
            }
          }
        }

        // 关闭按钮
        .echo-close {
          position: absolute;
          top: -6px;
          right: -6px;
          width: 18px;
          height: 18px;
          background: #ef4444;
          color: #fff;
          border-radius: 50%;
          display: flex;
          align-items: center;
          justify-content: center;
          cursor: pointer;
          font-size: 12px;
          transition: transform 0.2s;
          z-index: 10;

          &:hover {
            transform: scale(1.1);
          }
        }

        // 加载图标
        .loading-icon {
          position: absolute;
          top: 50%;
          left: 50%;
          transform: translate(-50%, -50%);
          font-size: 20px;
          color: #409eff;
          z-index: 5;
        }

        // 上传遮罩层
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
            width: 36px;
            height: 36px;
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
          .echo-close {
            display: none;
          }
        }
      }
    }
  }

  .input-wrapper {
    ::v-deep .el-textarea {
      .el-textarea__inner {
        background: transparent;
        border: none;
        padding: 0;
        resize: none;
        font-size: 16px;
        line-height: 1.6;
        color: $claude-text;

        &::placeholder {
          color: #9ca3af;
        }
      }
    }
  }

  .input-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: 12px;
    padding-top: 12px;
    border-top: 1px solid #f3f4f6;

    .toolbar-left {
      display: flex;
      align-items: center;
      gap: 8px;

      .model-select-inline {
        min-width: 200px;

        ::v-deep .el-input__inner {
          background: transparent;
          border: none;
          padding-left: 32px;
          font-size: 13px;
          color: $claude-text;
        }

        ::v-deep .el-input__prefix {
          left: 8px;
        }
      }
    }

    .toolbar-right {
      display: flex;
      align-items: center;
      gap: 8px;

      .action-icon {
        font-size: 18px;
        color: $claude-text-muted;
        cursor: pointer;
        padding: 8px;
        border-radius: 8px;
        transition: all 0.2s;

        &:hover {
          color: $claude-primary;
          background: rgba($claude-primary, 0.08);
        }
      }

      .send-btn {
        padding: 10px;
        border: none;
        color: #5147ff;
        line-height: 0;
        display: inline-flex;
        justify-content: center;
        align-items: center;

        &:hover {
          background-color: rgba(87, 104, 161, 0.08);
        }

        .send-icon {
          width: 18px;
          height: 18px;
          fill: currentColor;
        }

        .stop-icon {
          width: 16px;
          height: 16px;
          fill: currentColor;
        }
      }

      .stop-btn {
        background: #fff !important;
        border: 1px solid #333 !important;
        color: #333 !important;

        &:hover,
        &:focus {
          background: #333 !important;
          border-color: #333 !important;
          color: #fff !important;
        }
      }
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

  // 模式标签样式
  .mode-tag {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px 4px 12px;
    border-radius: 16px;
    background: rgba($claude-primary, 0.08);
    border: 1px solid rgba($claude-primary, 0.2);
    color: $claude-primary;
    font-size: 13px;
    font-weight: 500;
    animation: mode-tag-enter 0.2s ease;

    i:first-child {
      font-size: 14px;
    }

    .el-icon-close {
      font-size: 12px;
      cursor: pointer;
      padding: 2px;
      border-radius: 50%;
      transition: all 0.2s;

      &:hover {
        background: rgba($claude-primary, 0.2);
      }
    }
  }

  @keyframes mode-tag-enter {
    from {
      opacity: 0;
      transform: scale(0.9);
    }
    to {
      opacity: 1;
      transform: scale(1);
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

  .input-footer {
    text-align: center;
    font-size: 12px;
    color: $claude-text-muted;
    margin-top: 12px;
  }

  // 模式按钮样式
  .mode-buttons {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: 16px;
    flex-wrap: wrap;

    .mode-btn {
      display: inline-flex;
      align-items: center;
      gap: 6px;
      padding: 6px 14px;
      border-radius: 20px;
      border: 1px solid $claude-border;
      background: #fff;
      color: $claude-text-secondary;
      font-size: 13px;
      cursor: pointer;
      transition: all 0.2s;
      user-select: none;

      i {
        font-size: 14px;
      }

      &:hover {
        background: rgba($claude-primary, 0.06);
        border-color: rgba($claude-primary, 0.3);
        color: $claude-primary;
      }

      .el-icon--right {
        margin-left: 0;
        font-size: 12px;
      }
    }
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
