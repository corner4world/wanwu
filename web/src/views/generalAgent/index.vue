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
            <div v-if="mode !== 'skill'" class="header-btns">
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
            </div>

            <div class="header-title">{{ currentTitle }}</div>
          </div>

          <div v-if="isSkillType" class="header-right">
            <button
              v-if="isSkillType && !workspacePanelVisible"
              class="header-icon-btn workspace-entry-btn"
              @click="handleViewSkillWorkspace"
            >
              <span class="btn-text">
                {{ $t('generalAgent.header.workspace') }}
              </span>
            </button>
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
              :is-last-message="index === messageList.length - 1"
              :thread-id="currentThreadId"
              @regenerate="handleRegenerate"
              @view-workspace="handleViewWorkspace"
              @download-all="handleDownloadAll"
              @question-reply="handleQuestionReply"
              @question-reject="handleQuestionReject"
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
            </div>
            <div class="welcome-title">
              {{ $t('generalAgent.header.welcomeTitle') }}
            </div>
          </div>

          <div class="input-container">
            <!-- 模型选择 -->
            <div style="margin-bottom: 12px">
              <ModelSelect
                v-model="selectedModel"
                :options="modelList"
                :placeholder="$t('common.model.select')"
                :loading="modelLoading"
                :filterable="true"
                @change="handleModelChange"
                class="model-select-inline"
              />
            </div>

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
                      <p class="docInfo_name">
                        {{ $t('knowledgeManage.fileName') }}：{{
                          file.fileName
                        }}
                      </p>
                      <p class="docInfo_size">
                        {{ $t('knowledgeManage.fileSize') }}：{{
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

            <!-- 输入框 -->
            <div class="input-wrapper">
              <MentionInput
                ref="mentionInput"
                v-model="inputMessage"
                :placeholder="inputPlaceholder"
                @keydown-enter="handleKeyDown"
                :disabled="isStreaming || previewIsStreaming"
              />
            </div>

            <!-- 底部工具栏：配置按钮 + 发送按钮 -->
            <div class="input-toolbar">
              <div class="toolbar-left">
                <div class="config-btn" @click="showConfigDialog = true">
                  <i class="el-icon-setting"></i>
                  <span>{{ $t('generalAgent.header.config') }}</span>
                </div>
                <!-- 导入Skill按钮 -->
                <div
                  v-if="isSkillType && !currentThreadId"
                  class="config-btn"
                  @click="openImportSkillDialog"
                >
                  <i class="el-icon-download"></i>
                  <span>{{ $t('generalAgent.header.importSkill') }}</span>
                </div>
                <!-- 已选模式标签 -->
                <div v-if="selectedMode" class="mode-btn mode-btn-selected">
                  <img
                    v-if="selectedMode.avatar"
                    :src="selectedMode.avatar"
                    alt=""
                    class="mode-avatar"
                  />
                  <i v-else :class="selectedMode.icon"></i>
                  <span>{{ selectedMode.label }}</span>
                  <i
                    v-if="!isSkillType || (isSkillType && !currentThreadId)"
                    class="el-icon-close"
                    @click="removeMode(selectedMode.value)"
                  ></i>
                </div>
              </div>
              <div class="toolbar-right">
                <StreamUploadField
                  :fileTypeArr="['doc/*', 'md', 'image/*']"
                  type="wga"
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
                  v-show="isStreaming"
                  class="send-btn stop-btn"
                  circle
                  @click="handleStopClick"
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
                  v-show="!isStreaming"
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
          <!-- 可选模式按钮区域 - 仅在未选择模式且有数据时显示 -->
          <div
            v-if="!selectedMode && Object.keys(modeOptions).length > 0"
            class="mode-buttons"
          >
            <!-- 模式按钮列表 -->
            <div
              v-for="(mode, key) in modeOptions"
              v-show="
                mode.value !== MAIN_CHAT_MODES.SKILL_MODE || !currentThreadId
              "
              :key="key"
              class="mode-btn"
              @click="addMode(mode.value)"
            >
              <img
                v-if="mode.avatar"
                :src="mode.avatar"
                class="mode-avatar"
                alt=""
              />
              <i v-else :class="mode.icon"></i>
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
      <transition name="workspace-slide">
        <file-preview-drawer
          v-if="previewVisible"
          :visible.sync="previewVisible"
          :file="previewFile"
          :file-path="previewFilePath"
          :blob="previewBlob"
          :loading="previewLoading"
          @close="previewVisible = false"
        />
      </transition>

      <!--
        预览会话面板生命周期说明：
        1. 切换会话或 previewId 变化时，skillTabsKey 会变化，组件会卸载重建，用于重置上一轮预览会话及 SSE 状态。
        2. 打开文件预览抽屉时，仅通过 v-show 隐藏面板，不卸载组件，避免中断正在进行的预览 SSE。
      -->
      <transition name="workspace-slide">
        <SkillTabs
          v-if="shouldMountSkillTabs"
          v-show="shouldShowSkillTabs"
          :key="skillTabsKey"
          :skillPreviewParams="skillPreviewParams"
          class="preview-chat-panel"
          @view-workspace="handleViewWorkspace"
        />
      </transition>

      <!-- 配置弹窗 -->
      <configDialog
        ref="configDialog"
        :visible.sync="showConfigDialog"
        :agent-id="selectedMode?.value ?? ''"
      />

      <!-- skill导入弹框 -->
      <SkillDialog
        ref="importSkillDialogRef"
        @submit="handleSkillImportSubmit"
      />
    </div>
  </div>
</template>

<script>
import MessageItem from './components/MessageItem.vue';
import WorkspacePanel from './components/WorkspacePanel.vue';
import FilePreviewDrawer from './components/FilePreviewDrawer.vue';
import ConfigDialog from './components/ConfigDialog.vue';
import MentionInput from './components/MentionInput.vue';
import SkillTabs from './components/skills/SkillTabs.vue';
import SkillDialog from './components/skills/SkillDialog.vue';
import ModelSelect from '@/components/modelSelect.vue';
import StreamUploadField from '@/components/stream/streamUploadField.vue';
import {
  chatGeneralAgentConversation,
  chatGeneralAgentSkillConversation,
  checkGeneralAgentConversationConfig,
  createGeneralAgentConversation,
  createGeneralAgentSkillConversation,
  deleteGeneralAgentConversation,
  downloadGeneralAgentWorkspace,
  getGeneralAgentConversationConfig,
  getGeneralAgentConversationDetail,
  getGeneralAgentConversationList,
  getGeneralAgentWorkspace,
  importGeneralAgentSkillConversation,
  previewGeneralAgentWorkspace,
  refreshGeneralAgentSkillConversation,
  updateGeneralAgentConversationConfig,
} from '@/api/generalAgent';
import { getCustomSkillInfo } from '@/api/templateSquare';
import { selectModelList } from '@/api/modelAccess';
import { avatarSrc, resDownloadFile } from '@/utils/util';
import { mapActions, mapGetters, mapState } from 'vuex';
import { SSEEventParser } from './utils/sse-parser';
// 引入工具函数
import { aggregateEventsToMessages } from './utils/message-aggregator';
// 引入 Mixins
import streamStateManager from './mixins/streamStateManager';
import messageManager from './mixins/messageManager';
import fileManager from './mixins/fileManager';
import scrollController from './mixins/scrollController';
import modeManager from './mixins/modeManager';
import skillManager from './mixins/skillManager';

// 引入常量
import { MAIN_CHAT_MODES } from './constants';

export default {
  name: 'GeneralAgent',
  components: {
    MessageItem,
    WorkspacePanel,
    FilePreviewDrawer,
    ConfigDialog: ConfigDialog,
    MentionInput,
    SkillTabs,
    ModelSelect,
    StreamUploadField,
    SkillDialog, // skill导入弹框
  },
  mixins: [
    skillManager,
    streamStateManager,
    messageManager,
    fileManager,
    scrollController,
    modeManager,
  ],
  props: {
    mode: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      MAIN_CHAT_MODES,
      sidebarCollapsed: true,
      conversationList: [],
      currentThreadId: '',
      pageNo: 1,
      pageSize: 50,
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

      // skill 相关
      chatType: '', // skill 则 selectedMode.value 为 Skill Chat Agent
      customSkillId: '',
      previewId: '',
      skillChatMode: 'normal', // normal | import | convert
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
      return (
        this.currentConversation?.title ||
        this.$t('generalAgent.index.newConversation')
      );
    },
    canSend() {
      const hasContent =
        this.inputMessage.trim() || this.uploadedFiles.length > 0;
      const hasModel = !!this.selectedModel;
      // 检查当前会话或预览面板是否正在流式传输
      return (
        hasContent && hasModel && !this.isStreaming && !this.previewIsStreaming
      );
    },
    isEmptyConversation() {
      return this.messageList.length === 0;
    },
    inputPlaceholder() {
      // 如果有选中的模式，使用模式的 placeholder
      if (this.selectedMode) {
        const modeConfig = this.modeOptions[this.selectedMode.value];
        if (modeConfig && modeConfig.placeholder) {
          return modeConfig.placeholder;
        }
      }
      // 默认 placeholder
      return this.$t('generalAgent.header.placeholder');
    },
    // skill预览相关参数
    skillPreviewParams() {
      return {
        customSkillId: this.customSkillId,
        previewId: this.previewId,
        threadId: this.currentThreadId,
      };
    },
    // 创建skill模式
    isSkillType() {
      return this.chatType === 'skill';
    },
    isActiveSkillMode() {
      return (
        this.isSkillType &&
        this.selectedMode?.value === MAIN_CHAT_MODES.SKILL_MODE
      );
    },
    // 控制组件是否挂载：会话身份存在时才挂载，切换会话/previewId 时配合 key 卸载重建以重置 SSE。
    shouldMountSkillTabs() {
      return (
        this.isActiveSkillMode &&
        !!this.currentThreadId &&
        !!this.customSkillId &&
        !!this.previewId
      );
    },
    // 控制组件是否可见：文件预览打开时只隐藏 SkillTabs，不卸载，避免中断内部预览 SSE。
    shouldShowSkillTabs() {
      return this.shouldMountSkillTabs && !this.previewVisible;
    },
    skillTabsKey() {
      return `${this.currentThreadId}_${this.previewId}`;
    },
    // 当前选中的会话对象
    currentConversation() {
      if (!this.currentThreadId) return null;
      return this.conversationList.find(
        c => c.threadId === this.currentThreadId,
      );
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
    this.initFromRoute();
    this.fetchModelList();
    this.fetchConversationList();
    this.initUserInfo();
    this.setupResizeObserver();
    this.fetchModeOptions().then(() => {
      if (this.isSkillType) {
        this.addMode(MAIN_CHAT_MODES.SKILL_MODE);
      }
    });
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

    initFromRoute() {
      const { chatType, customSkillId, chatMode, threadId } =
        this.$route.query || {};

      let isNewChatMode = false;
      let initialChatMode = '';

      if (chatType) {
        this.chatType = chatType;
      }
      if (customSkillId) {
        this.customSkillId = customSkillId;
      }
      if (chatMode) {
        isNewChatMode = true;
        initialChatMode = chatMode;
        this.skillChatMode = 'normal';

        // 防止页面刷新再次进入自动触发模式
        const newQuery = { ...this.$route.query };
        delete newQuery.chatMode;
        this.$router.replace({ query: newQuery }).catch(() => {});
      }

      // 根据 customSkillId 自动拉取并补全状态
      if (this.isSkillType && this.customSkillId) {
        // 如果是从某模式进入（如转换、导入），则由 fetchCustomSkillInfo 补全后触发启动
        this.fetchCustomSkillInfo(threadId || '', initialChatMode);
        return;
      }

      if (threadId) {
        // 普通 General Agent 路由恢复逻辑
        this.selectConversation(threadId);
      } else {
        this.initNewConversation();
      }
    },
    // 从会话列表设置skill参数
    setSkillParamsByConversation(c_data) {
      if (c_data.skillId) {
        this.customSkillId = c_data.skillId;
      }
      if (c_data.previewId) {
        this.previewId = c_data.previewId;
      }
    },

    startSkillPresetConversation(mode) {
      const promptKey =
        mode === 'convert'
          ? 'generalAgent.skill.convertPrompt'
          : mode === 'import'
            ? 'generalAgent.skill.importPrompt'
            : 'generalAgent.skill.defaultPrompt';
      const content = this.$t(promptKey);
      const userMessage = this.buildUserMessage(content);
      const currentConversation = this.conversationList.find(
        item => item.threadId === this.currentThreadId,
      );

      if (currentConversation) {
        this.$set(currentConversation, 'isSkillConversation', true);
        this.$set(currentConversation, 'skillId', this.customSkillId);
        this.$set(currentConversation, 'previewId', this.previewId);
      } else if (this.currentThreadId) {
        this.conversationList.unshift({
          threadId: this.currentThreadId,
          title: content.slice(0, 50),
          createdAt: new Date().toISOString(),
          isSkillConversation: true,
          skillId: this.customSkillId,
          previewId: this.previewId,
        });
      }

      this.ensureMessageList(this.currentThreadId);
      this.addUserMessage(this.currentThreadId, content);
      this.$nextTick(() => this.scrollToBottom());

      this.skillChatMode = mode;
      return this.startStreaming(userMessage);
    },

    /**
     * 根据 customSkillId 拉取详情并补全/自愈会话状态
     * @param {string} routeThreadId 路由中带入的 threadId（如果有）
     * @param {string} initialChatMode 预设模式（如 convert/import）
     */
    async fetchCustomSkillInfo(routeThreadId = '', initialChatMode = '') {
      if (!this.customSkillId) return;

      this.isLoadingHistory = true;
      try {
        const res = await getCustomSkillInfo({ skillId: this.customSkillId });

        if (res.code === 0 && res.data) {
          const { previewId, threadId } = res.data;

          // 1. 设置预览 ID
          if (previewId) this.previewId = previewId;

          // 优先级：接口活跃记录 > 路由穿透带入
          let targetThreadId = threadId;
          let isNewRefresh = false;

          // 若目标对话不存在（如刚转换完还没有任何会话），主动刷新获取新会话
          if (!targetThreadId) {
            targetThreadId = await this.refreshSkillConversation();
            isNewRefresh = true;
          }

          if (targetThreadId) {
            if (initialChatMode) {
              // 处理特殊模式（如 convert/import）的流式启动
              this.currentThreadId = targetThreadId;
              this.resetScrollState();
              this.clearModes();
              this.addMode(MAIN_CHAT_MODES.SKILL_MODE);
              this.hidePanel();
              this.clearMessages(targetThreadId);
              this.loadConfig();

              this.$nextTick(() => {
                this.startSkillPresetConversation(initialChatMode);
              });
            } else if (!isNewRefresh) {
              // 非新创建会话，则走正常加载逻辑
              this.selectConversation(targetThreadId);
            } else {
              // 刚刷新后的新会话且无 initialChatMode，激活模式并加载配置即可
              this.addMode(MAIN_CHAT_MODES.SKILL_MODE);
              this.loadConfig();
            }
          }
        } else {
          this.$message.error(
            this.$t('generalAgent.skill.invalidSkill') || '无效技能',
          );
          this.$router.go(-1);
        }
      } catch (error) {
        console.error('fetchCustomSkillInfo error:', error);
        this.$message.error(
          this.$t('generalAgent.skill.loadDetailError') || '获取技能详情失败',
        );
        this.$router.go(-1);
      } finally {
        this.isLoadingHistory = false;
      }
    },

    /**
     * 刷新 Skill 专用对话
     * 当 threadId 为空或对话历史丢失时，为已有 customSkillId 重新创建编辑对话
     * @returns {Promise<string>} 新创建的 threadId
     */
    async refreshSkillConversation() {
      if (!this.customSkillId) {
        console.warn('refreshSkillConversation: customSkillId 为空，无法刷新');
        return '';
      }

      this.isLoadingHistory = true;
      try {
        const res = await refreshGeneralAgentSkillConversation({
          skillId: this.customSkillId,
        });

        if (res.code === 0 && res.data) {
          const { customSkillId, threadId, previewId } = res.data;

          // 更新本地状态
          this.customSkillId = customSkillId || this.customSkillId;
          this.currentThreadId = threadId;
          if (previewId) this.previewId = previewId;

          // 清空消息列表，准备空白会话
          this.clearMessages(threadId);

          // 主动设置模型配置
          await this.syncModelConfigAfterRefresh(threadId);

          // 将新会话加入左侧会话列表
          this.conversationList.unshift({
            threadId,
            title: this.$t('generalAgent.index.newConversation'),
            createdAt: new Date().toISOString(),
            isSkillConversation: true,
            skillId: this.customSkillId,
            previewId: this.previewId,
          });

          // 注意：此处不再静默更新路由保持极简
          return threadId;
        } else {
          this.$message.error(res.msg || '刷新 Skill 对话失败');
          return '';
        }
      } catch (error) {
        console.error('refreshSkillConversation error:', error);
        this.$message.error('刷新 Skill 对话失败，请重试');
        return '';
      } finally {
        this.isLoadingHistory = false;
      }
    },

    /**
     * refresh 后主动同步模型配置到新对话
     * 使用当前已选模型或模型列表中的第一个模型
     */
    async syncModelConfigAfterRefresh(threadId) {
      const modelId = this.selectedModel || this.modelList[0]?.modelId;
      if (!modelId || !threadId) return;

      const modelConfig = this.modelList.find(m => m.modelId === modelId);
      if (!modelConfig) return;

      try {
        await updateGeneralAgentConversationConfig({
          threadId,
          modelConfig: {
            modelId: modelConfig.modelId,
            model: modelConfig.model,
            provider: modelConfig.provider,
            displayName: modelConfig.displayName,
            modelType: modelConfig.modelType || 'llm',
            config: modelConfig.config || {},
          },
        });
      } catch (error) {
        console.error('syncModelConfigAfterRefresh error:', error);
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

      const creationApi = this.isSkillType
        ? createGeneralAgentSkillConversation
        : createGeneralAgentConversation;

      const res = await creationApi({
        title: title || this.$t('generalAgent.index.newConversation'),
        modelConfig,
      });

      if (res.code === 0) {
        const threadId = res.data?.threadId;
        if (threadId) {
          this.currentThreadId = threadId;

          // 保存 skill 相关的额外 ID
          if (this.isSkillType) {
            this.customSkillId = res.data.customSkillId || '';
            this.previewId = res.data.previewId || '';
          }

          const oldMessages = this.messagesMap[''] || [];
          this.$set(this.messagesMap, threadId, oldMessages);
          this.$delete(this.messagesMap, '');

          this.selectedModel = modelConfig.modelId;
          this.conversationList.unshift({
            threadId,
            title: title || this.$t('generalAgent.index.newConversation'),
            createdAt: new Date().toISOString(),
            isSkillConversation: this.isSkillType,
            ...(this.isSkillType
              ? {
                  skillId: this.customSkillId,
                  previewId: this.previewId,
                }
              : {}),
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
      this.isLoadingHistory = true;
      this.resetScrollState();
      // 重置模式选择
      this.clearModes();
      // 检查当前会话是否是 skill 会话
      if (this.currentConversation?.isSkillConversation) {
        this.addMode(MAIN_CHAT_MODES.SKILL_MODE);
        this.setSkillParamsByConversation(this.currentConversation);
      }
      this.hidePanel();
      this.fetchHistory();
    },

    async fetchHistory() {
      if (!this.currentThreadId) return;

      const streaming = this.streamingMap[this.currentThreadId];
      if (streaming && streaming.isStreaming) {
        this.isLoadingHistory = false;
        this.loadConfig();
        return;
      }

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
            if (run.runId) this.currentRunId = run.runId;
          });

          this.$set(this.messagesMap, this.currentThreadId, allMessages);
          // 先关闭加载状态，让消息列表渲染
          this.isLoadingHistory = false;
          // 等待 DOM 渲染完成后滚动到底部
          this.$nextTick(() => {
            requestAnimationFrame(() => {
              this.scrollToBottom(true);
            });
          });
        } else if (res.code === 0 && !res.data?.list) {
          // Skill 会话且接口正常返回但历史为空（list 为 null）
          // 说明 threadId 对应的对话已被删除，调用 refresh 重建
          if (this.isSkillType && this.customSkillId) {
            this.isLoadingHistory = false;
            await this.refreshSkillConversation();
            return;
          }
          this.isLoadingHistory = false;
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
      if (this.previewIsStreaming) return;

      // 检查当前会话是否正在流式传输
      const currentStreaming = this.streamingMap[this.currentThreadId];
      if (currentStreaming && currentStreaming.isStreaming) return;

      // 在发送消息前进行本地配置校验
      await this.$refs.configDialog?.fetchToolList();

      // 先执行校验，检查是否有错误（不弹窗）
      const isValid = this.$refs.configDialog?.validateTools();

      if (!isValid) {
        // 有错误，打开弹窗显示错误提示
        this.showConfigDialog = true;
        await this.$nextTick();
        return;
      }

      if (!this.currentThreadId) {
        const title = content.slice(0, 50);
        const threadId = await this.createConversationWithTitle(title);
        if (!threadId) {
          this.$message.error(
            this.$t('generalAgent.error.createConversationFailed'),
          );
          return;
        }
      }

      // 检查配置是否满足条件（在发送消息前）
      const checkRes = await checkGeneralAgentConversationConfig({
        agentId: this.selectedMode?.value ?? '',
        threadId: this.currentThreadId,
      });

      if (checkRes.code === 0 && checkRes.data) {
        const { meet, modelMeet, toolsMeet } = checkRes.data;

        // 如果配置不满足，检查具体哪些项不满足
        if (!meet) {
          // 检查模型是否满足
          if (!modelMeet) {
            this.$message.warning(
              this.$t('generalAgent.error.modelNotAvailable'),
            );
            return;
          }

          // 检查工具是否满足
          if (toolsMeet && Array.isArray(toolsMeet)) {
            const unmetTools = toolsMeet.filter(category => !category.meet);
            if (unmetTools.length > 0) {
              this.showConfigDialog = true;
              this.$nextTick(async () => {
                await this.$refs.configDialog?.fetchToolList();
                this.$refs.configDialog?.validateTools();
              });
              return;
            }
          }
        }
      }

      if (this.previewIsStreaming) return;

      const userMessage = this.buildUserMessage(content);
      this.ensureMessageList(this.currentThreadId);
      this.addUserMessage(this.currentThreadId, content, this.uploadedFiles);

      this.clearFiles();
      this.$refs.mentionInput?.clear();
      this.$nextTick(() => this.scrollToBottom());

      await this.startStreaming(userMessage);
    },

    async startStreaming(userMessage) {
      if (this.previewIsStreaming) return;

      if (!this.currentThreadId) {
        this.$message.error(
          this.$t('generalAgent.error.conversationIdNotExist'),
        );
        return;
      }

      const streamingThreadId = this.currentThreadId;
      const agentId = this.selectedMode?.value ?? '';
      const currentMode = this.skillChatMode || 'normal';

      // 使用 mixin 初始化流式状态
      const { abortController, assistantMessage } =
        this.initStreamState(streamingThreadId);

      // 添加消息到对应会话的消息列表
      this.addAssistantMessage(streamingThreadId, assistantMessage);

      this.currentStage = 'understanding';
      this.resetScrollState();

      const parser = new SSEEventParser();
      let isUserAborted = false;

      const chatParams = {
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
          // 使用 mixin 的方法来清理状态，确保全局状态也被更新
          this.cleanupStreamState(streamingThreadId);
          assistantMessage.isStreaming = false;
          // 清理所有 fragments 的 isStreaming 状态
          this.setFragmentsNotStreaming(assistantMessage.fragments);
        },
        signal: abortController.signal,
      };

      try {
        if (this.isSkillType) {
          // Skill 专用对话流
          await chatGeneralAgentSkillConversation({
            ...chatParams,
            customSkillId: this.customSkillId,
            mode: currentMode,
          });
        } else {
          // 普通对话流
          await chatGeneralAgentConversation({
            ...chatParams,
            agentId,
          });
        }
      } catch (error) {
        console.error('Stream error:', error);
        // 判断是否是用户主动中止
        isUserAborted = error.name === 'AbortError';

        if (!isUserAborted && this.currentThreadId === streamingThreadId) {
          this.$message.error(
            this.$t('generalAgent.error.sendMessageFailed') +
              (error.message || error),
          );
        }
      } finally {
        if (this.isSkillType) {
          this.skillChatMode = 'normal';
        }
        // 只有非用户主动中止时才清理状态（用户中止由 stopStreaming 处理）
        if (!isUserAborted) {
          // 使用 mixin 的方法来清理状态，确保全局状态也被更新
          this.cleanupStreamState(streamingThreadId);
          assistantMessage.isStreaming = false;

          // 清理所有 fragments 的 isStreaming 状态
          this.setFragmentsNotStreaming(assistantMessage.fragments);
          this.currentStage = '';
          if (this.currentThreadId === streamingThreadId) {
            this.resetScrollState();
            this.$nextTick(() => this.scrollToBottom(true));
          }
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
        resDownloadFile(blob, this.$t('generalAgent.index.workspaceZip'));
        this.$message.success(
          this.$t('generalAgent.workspace.downloadSuccess'),
        );
      } catch (error) {
        console.error(
          this.$t('generalAgent.index.downloadWorkspaceFailed'),
          error,
        );
        this.$message.error(this.$t('generalAgent.workspace.downloadFailed'));
      }
    },

    // 重新生成 - 找到上一条用户消息并重新发送
    handleRegenerate(message) {
      if (this.isStreaming || this.previewIsStreaming) return;

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

    // Human-in-the-Loop: 用户回答问题
    handleQuestionReply(event) {
      // QuestionBlock 已处理 API 调用和状态更新
      // 此处仅用于日志或后续扩展
      console.log('[GeneralAgent] Question replied:', event);
    },

    // Human-in-the-Loop: 用户拒绝问题
    handleQuestionReject(event) {
      // QuestionBlock 已处理 API 调用和状态更新
      // 此处仅用于日志或后续扩展
      console.log('[GeneralAgent] Question rejected:', event);
    },

    async handleDeleteConversation(item) {
      await this.$confirm(
        this.$t('generalAgent.index.confirmDeleteConversation'),
        this.$t('common.button.tip'),
        {
          type: 'warning',
        },
      );
      const res = await deleteGeneralAgentConversation({
        threadId: item.threadId,
      });
      if (res.code === 0) {
        this.$message.success(this.$t('common.info.delete'));
        if (this.currentThreadId === item.threadId) {
          this.currentThreadId = '';
          this.messageList = [];
          this.hidePanel();
        }
        this.fetchConversationList();
      }
    },

    // 处理停止按钮点击
    handleStopClick() {
      this.stopStreaming(this.currentThreadId);
    },

    // 打开skill导入弹框
    openImportSkillDialog() {
      if (!this.selectedModel) {
        this.$message.warning(this.$t('generalAgent.skill.selectModelFirst'));
        return;
      }
      this.$refs.importSkillDialogRef?.openDialog();
    },
    async handleSkillImportSubmit(formData) {
      const dialogRef = this.$refs.importSkillDialogRef;
      dialogRef.btnLoading = true;
      try {
        const selectedModelConfig =
          this.modelList.find(m => m.modelId === this.selectedModel) ||
          this.modelList[0];
        const modelConfig = {
          modelId: selectedModelConfig?.modelId,
          model: selectedModelConfig?.model,
          provider: selectedModelConfig?.provider,
          displayName: selectedModelConfig?.displayName,
          modelType: selectedModelConfig?.modelType,
          config: selectedModelConfig?.config,
        };

        const { author, avatar, zipUrl } = formData;
        const res = await importGeneralAgentSkillConversation({
          author,
          avatar,
          zipUrl,
          modelConfig,
        });

        if (res.code === 0) {
          const { threadId, customSkillId, previewId } = res.data || {};
          this.$message.success(this.$t('common.message.success'));
          dialogRef.dialogVisible = false;

          if (threadId) {
            this.currentThreadId = threadId;
            this.customSkillId = customSkillId || '';
            this.previewId = previewId || '';
            this.startSkillPresetConversation('import');
          }
        }
      } catch (error) {
        console.error(error);
      } finally {
        dialogRef.btnLoading = false;
      }
    },
    // 打开skill工作空间
    handleViewSkillWorkspace() {
      // skill时runId可为空
      this.handleViewWorkspace({
        threadId: this.currentThreadId,
        runId: '',
        fileCount: 0,
        totalSize: 0,
      });
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
      min-width: 25vw;
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

  .header-btns {
    display: flex;
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

    &.workspace-entry-btn {
      width: auto;
      padding: 0 12px;
      gap: 6px;
      font-size: 13px;
      font-weight: 500;

      &:hover {
        background: $claude-primary;
        color: #fff;
        border-color: $claude-primary;
      }
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

  // 模式按钮基础样式
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

    .mode-avatar {
      width: 18px;
      height: 18px;
      border-radius: 50%;
      object-fit: cover;
    }

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

  // 已选模式样式
  .mode-btn-selected {
    background: rgba($claude-primary, 0.08);
    border-color: rgba($claude-primary, 0.3);
    color: $claude-primary;
    cursor: default;

    &:hover {
      background: rgba($claude-primary, 0.12);
      border-color: rgba($claude-primary, 0.4);
    }

    .el-icon-close {
      font-size: 12px;
      cursor: pointer;
      padding: 2px;
      border-radius: 50%;
      transition: all 0.2s;
      margin-left: 4px;

      &:hover {
        background: rgba($claude-primary, 0.2);
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

  .input-footer {
    text-align: center;
    font-size: 12px;
    color: $claude-text-muted;
    margin-top: 12px;
  }

  // 模式按钮容器
  .mode-buttons {
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    margin-top: 16px;
    flex-wrap: wrap;
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
