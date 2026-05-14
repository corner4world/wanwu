<template>
  <div class="preview-chat">
    <div
      ref="messageArea"
      class="preview-message-area"
      :class="{ empty: isEmptyConversation }"
      @scroll="handleMessageAreaScroll"
    >
      <div v-if="messageList.length > 0 || isStreaming" class="message-list">
        <message-item
          v-for="(msg, index) in messageList"
          :key="msg.id || index"
          :message="msg"
          :is-last-message="index === messageList.length - 1"
          :thread-id="currentThreadId"
          @regenerate="handleRegenerate"
          @view-workspace="$emit('view-workspace', $event)"
          @download-all="$emit('download-all')"
        />
      </div>
      <div v-else class="preview-empty">
        <div class="empty-avatar">
          <i class="el-icon-cpu"></i>
        </div>
        <div class="empty-title">预览会话</div>
      </div>
    </div>

    <transition name="scroll-btn-fade">
      <button
        v-if="showScrollToBottom"
        class="scroll-to-bottom-btn"
        @click="handleScrollToBottomClick"
      >
        <i class="el-icon-arrow-down"></i>
      </button>
    </transition>

    <div class="preview-input-area">
      <div v-if="uploadedFiles.length > 0" class="file-preview">
        <div
          v-for="(file, index) in uploadedFiles"
          :key="index"
          class="echo-img-box"
        >
          <div class="echo-img-item">
            <el-image
              v-if="file.type && file.type.startsWith('image/')"
              class="echo-img"
              :src="file.displayUrl"
              :preview-src-list="[file.displayUrl]"
            ></el-image>
            <div v-else class="echo-doc-box">
              <img
                :src="require('@/assets/imgs/fileicon.png')"
                class="docIcon"
              />
              <div class="docInfo">
                <p class="docInfo_name">{{ file.fileName }}</p>
                <p class="docInfo_size">
                  {{
                    file.size > 1024
                      ? (file.size / (1024 * 1024)).toFixed(2) + ' MB'
                      : (file.size || 0) + ' bytes'
                  }}
                </p>
              </div>
            </div>
            <i class="el-icon-close echo-close" @click="removeFile(index)"></i>
          </div>
        </div>
      </div>

      <div class="input-container">
        <el-input
          ref="input"
          v-model="inputMessage"
          type="textarea"
          :autosize="{ minRows: 1, maxRows: 6 }"
          :placeholder="inputPlaceholder"
          :disabled="isStreaming || mainIsStreaming"
          resize="none"
          class="chat-textarea"
          @keydown.enter.native="handleKeyDown"
        />
        <div class="input-toolbar">
          <div class="toolbar-left"></div>
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
              <svg class="stop-icon" viewBox="0 0 24 24" width="16" height="16">
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
              <svg class="send-icon" viewBox="0 0 24 24" width="18" height="18">
                <path
                  fill="currentColor"
                  d="M2.01 21L23 12 2.01 3 2 10l15 2-15 2z"
                />
              </svg>
            </el-button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import MessageItem from '../MessageItem.vue';

import StreamUploadField from '@/components/stream/streamUploadField.vue';
import {
  chatGeneralAgentSkillConversation,
  getGeneralAgentSkillPreviewConversationDetail,
} from '@/api/generalAgent';

import { SSEEventParser } from '../../utils/sse-parser';
import streamStateManager from '../../mixins/streamStateManager';
import messageManager from '../../mixins/messageManager';
import fileManager from '../../mixins/fileManager';
import scrollController from '../../mixins/scrollController';
import skillManager from '../../mixins/skillManager';
import { aggregateEventsToMessages } from '../../utils/message-aggregator';

export default {
  name: 'PreviewChat',
  components: {
    MessageItem,
    StreamUploadField,
  },
  mixins: [
    skillManager,
    streamStateManager,
    messageManager,
    fileManager,
    scrollController,
  ],
  props: {
    skillPreviewParams: {
      type: Object,
      default: () => ({}),
    },
  },
  data() {
    return {
      currentThreadId: '',
      inputMessage: '',
      currentRunId: '',
      currentStage: '',
    };
  },
  computed: {
    isEmptyConversation() {
      return this.messageList.length === 0;
    },
    canSend() {
      const hasContent =
        this.inputMessage.trim() || this.uploadedFiles.length > 0;
      // 检查当前会话或预览面板是否正在流式传输
      return hasContent && !this.isStreaming && !this.mainIsStreaming;
    },
    inputPlaceholder() {
      return this.$t('generalAgent.skill.preview.placeholder');
    },
  },
  mounted() {
    this.initConversationFromProps();
  },
  watch: {
    'skillPreviewParams.previewId': {
      handler(val) {
        if (val) {
          this.initConversationFromProps();
        }
      },
    },
  },
  beforeDestroy() {
    this.cleanupAllStreams();
  },
  methods: {
    async initConversationFromProps() {
      const { previewId } = this.skillPreviewParams || {};
      if (previewId) {
        this.currentThreadId = previewId;
        await this.loadPreviewHistory();
      }
    },

    async loadPreviewHistory() {
      const { previewId } = this.skillPreviewParams || {};
      if (!previewId) return;

      this.isLoadingHistory = true;
      try {
        const res = await getGeneralAgentSkillPreviewConversationDetail({
          previewId,
        });
        if (res.code === 0 && res.data?.list) {
          this.clearMessages(this.currentThreadId);
          this.ensureMessageList(this.currentThreadId);
          const allMessages = [];
          // 聚合所有消息
          res.data.list.forEach(run => {
            // 后端返回的是 events 字段，需要聚合为消息
            if (run.events && Array.isArray(run.events)) {
              const messages = aggregateEventsToMessages(run.events);
              allMessages.push(...messages);
            }
            if (run.runId) this.currentRunId = run.runId;
          });
          this.$set(this.messagesMap, this.currentThreadId, allMessages);
          this.$nextTick(() => this.scrollToBottom(false));
        }
      } finally {
        this.isLoadingHistory = false;
      }
    },

    handleKeyDown(e) {
      if (e.shiftKey) return;
      e.preventDefault();
      this.sendMessage();
    },

    async sendMessage() {
      const content = this.inputMessage.trim();
      if (!content && this.uploadedFiles.length === 0) return;
      if (this.mainIsStreaming) return;
      if (this.isStreaming) return;

      if (!this.currentThreadId) {
        const { previewId } = this.skillPreviewParams || {};
        if (previewId) {
          this.currentThreadId = previewId;
        } else {
          this.$message.warning('预览 ID 未就绪，请稍后');
          return;
        }
      }

      const userMessage = this.buildUserMessage(content);
      this.ensureMessageList(this.currentThreadId);
      this.addUserMessage(this.currentThreadId, content, this.uploadedFiles);
      this.clearFiles();
      this.inputMessage = '';
      this.$nextTick(() => this.scrollToBottom(true));

      await this.startStreaming(userMessage);
    },

    async startStreaming(userMessage) {
      if (this.mainIsStreaming) return;

      if (!this.currentThreadId) {
        this.$message.error(
          this.$t('generalAgent.error.conversationIdNotExist'),
        );
        return;
      }

      const streamingThreadId = this.currentThreadId;
      const { abortController, assistantMessage } =
        this.initStreamState(streamingThreadId);

      this.addAssistantMessage(streamingThreadId, assistantMessage);
      this.currentStage = 'understanding';
      this.resetScrollState();

      const parser = new SSEEventParser();
      let isUserAborted = false;

      try {
        const { customSkillId, threadId, previewId } =
          this.skillPreviewParams || {};
        await chatGeneralAgentSkillConversation({
          customSkillId,
          threadId,
          previewId,
          mode: 'preview',
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
            console.error('Preview SSE Error:', error);
            this.$message.error(
              this.$t('generalAgent.error.chatRequestFailed'),
            );
            // 使用 mixin 的方法来清理状态
            this.cleanupStreamState(streamingThreadId);
            assistantMessage.isStreaming = false;
            this.setFragmentsNotStreaming(assistantMessage.fragments);
          },
          signal: abortController.signal,
        });
      } catch (error) {
        console.error('Preview stream error:', error);
        isUserAborted = error.name === 'AbortError';
        if (!isUserAborted) {
          this.$message.error(
            this.$t('generalAgent.error.sendMessageFailed') +
              (error.message || error),
          );
        }
      } finally {
        if (!isUserAborted) {
          // 使用 mixin 的方法来清理状态
          this.cleanupStreamState(streamingThreadId);
          assistantMessage.isStreaming = false;
          this.setFragmentsNotStreaming(assistantMessage.fragments);
          this.currentStage = '';
          this.resetScrollState();
          this.$nextTick(() => this.scrollToBottom(true));
        }
      }
    },

    handleRegenerate(message) {
      if (this.isStreaming || this.mainIsStreaming) return;

      const messageIndex = this.messageList.findIndex(m => m.id === message.id);
      if (messageIndex <= 0) return;

      let userMessage = null;
      for (let i = messageIndex - 1; i >= 0; i--) {
        if (this.messageList[i].role === 'user') {
          userMessage = this.messageList[i];
          break;
        }
      }
      if (!userMessage) return;

      this.removeMessage(this.currentThreadId, message.id);
      const requestMessage = this.buildRequestMessage(userMessage);
      this.$nextTick(() => this.startStreaming(requestMessage));
    },

    handleStopClick() {
      this.stopStreaming(this.currentThreadId);
    },

    handleWorkspaceActivity() {},

    showPanel() {},
  },
};
</script>

<style lang="scss" scoped>
$primary: #10a37f;
$text: #1a1a1a;
$text-secondary: #666;
$text-muted: #999;
$border: #e5e7eb;

.preview-chat {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #fff;
  position: relative;
}

.preview-message-area {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
  background: #fff;

  .message-list {
    padding: 18px 14px;
    min-height: 100%;
  }
}

.preview-empty {
  height: 100%;
  min-height: 260px;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  color: $text-muted;
}

.empty-avatar {
  width: 56px;
  height: 56px;
  border-radius: 16px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f7f7f8;
  color: $primary;
  font-size: 26px;
}

.empty-title {
  color: $text;
  font-size: 18px;
  font-weight: 600;
}

.preview-input-area {
  flex: none;
  padding: 12px 14px 16px;
  background: #fff;
}

.input-container {
  border: 1px solid $border;
  border-radius: 14px;
  padding: 12px;
  background: #fff;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.06);
  transition:
    border-color 0.2s,
    box-shadow 0.2s;

  &:focus-within {
    border-color: $primary;
    box-shadow: 0 4px 22px rgba(0, 0, 0, 0.1);
  }
}

.chat-textarea {
  ::v-deep .el-textarea__inner {
    border: none;
    padding: 0;
    font-family: inherit;
    font-size: 14px;
    line-height: 1.6;
    color: $text;
    background: transparent;
    &::placeholder {
      color: $text-muted;
    }
  }
}

.input-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 10px;
  padding-top: 10px;
  border-top: 1px solid #f3f4f6;
}

.toolbar-left,
.toolbar-right {
  display: flex;
  align-items: center;
  gap: 8px;
}

.action-icon {
  font-size: 18px;
  color: $text-muted;
  cursor: pointer;
  padding: 8px;
  border-radius: 8px;
  transition: all 0.2s;

  &:hover {
    color: $primary;
    background: rgba($primary, 0.08);
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

.file-preview {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 10px;
}

.echo-img-item {
  position: relative;
  display: inline-block;
}

.echo-img {
  width: 48px;
  height: 48px;
  border-radius: 8px;
  cursor: pointer;
}

.echo-doc-box {
  min-width: 180px;
  max-width: 280px;
  border: 1px solid #dcdfe6;
  border-radius: 6px;
  display: flex;
  align-items: center;
  padding: 8px 34px 8px 8px;
}

.docIcon {
  width: 28px;
  height: 28px;
  flex-shrink: 0;
}

.docInfo {
  flex: 1;
  margin-left: 8px;
  overflow: hidden;
}

.docInfo_name,
.docInfo_size {
  margin: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docInfo_name {
  color: #333;
  font-size: 13px;
}

.docInfo_size {
  color: #bbb;
  font-size: 12px;
  margin-top: 4px;
}

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
}

.scroll-to-bottom-btn {
  position: absolute;
  bottom: 120px;
  left: 50%;
  transform: translateX(-50%);
  width: 34px;
  height: 34px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #fff;
  color: $primary;
  border: 1px solid $primary;
  border-radius: 50%;
  cursor: pointer;
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.15);
  transition: all 0.2s ease;
  z-index: 2;
}

.scroll-btn-fade-enter-active,
.scroll-btn-fade-leave-active {
  transition: all 0.3s ease;
}

.scroll-btn-fade-enter,
.scroll-btn-fade-leave-to {
  opacity: 0;
  transform: translateY(20px);
}

::v-deep .message-item {
  .message-body {
    max-width: 100%;
  }

  .user-message-content {
    max-width: calc(100% - 44px);
  }
}
</style>
