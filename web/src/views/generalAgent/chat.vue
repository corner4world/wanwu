<template>
  <div class="chat-container">
    <!-- 顶部配置区 -->
    <div class="chat-header">
      <div class="header-left">
        <el-input
          v-model="conversationTitle"
          placeholder="对话标题"
          class="title-input"
          @blur="updateTitle"
        />
      </div>
      <div class="header-right">
        <el-button size="small" @click="showConfigPanel = true">
          <i class="el-icon-setting"></i>
          配置
        </el-button>
        <el-button size="small" @click="showWorkspace = !showWorkspace">
          <i class="el-icon-folder"></i>
          工作空间
        </el-button>
      </div>
    </div>

    <!-- 主体区域 -->
    <div class="chat-main">
      <!-- 消息列表 -->
      <div class="message-list" ref="messageList">
        <div v-if="loadingHistory" class="loading-more">
          <i class="el-icon-loading"></i>
          加载历史消息...
        </div>
        <message-item
          v-for="(msg, index) in messageList"
          :key="msg.id || index"
          :message="msg"
        />
        <div v-if="isStreaming" class="streaming-indicator">
          <i class="el-icon-loading"></i>
          正在思考...
        </div>
      </div>

      <!-- 工作空间面板 -->
      <div v-if="showWorkspace" class="workspace-panel">
        <workspace-view
          :thread-id="threadId"
          :run-id="currentRunId"
          @close="showWorkspace = false"
        />
      </div>
    </div>

    <!-- 输入区域 -->
    <div class="input-area">
      <div class="input-wrapper">
        <div class="file-preview" v-if="uploadedFiles.length > 0">
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
            <span class="file-name">{{ file.name }}</span>
            <i class="el-icon-close" @click="removeFile(index)"></i>
          </div>
        </div>
        <el-input
          v-model="inputMessage"
          type="textarea"
          :rows="3"
          placeholder="输入消息... (Shift+Enter 换行，Enter 发送)"
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
            <el-button size="small" type="text">
              <i class="el-icon-paperclip"></i>
              上传文件
            </el-button>
          </el-upload>
          <el-button
            type="primary"
            size="small"
            :loading="isStreaming"
            :disabled="!inputMessage.trim() && uploadedFiles.length === 0"
            @click="sendMessage"
          >
            {{ isStreaming ? '发送中' : '发送' }}
          </el-button>
          <el-button
            v-if="isStreaming"
            type="danger"
            size="small"
            @click="stopStreaming"
          >
            停止
          </el-button>
        </div>
      </div>
    </div>

    <!-- 配置面板 -->
    <config-panel
      v-if="showConfigPanel"
      :thread-id="threadId"
      @close="showConfigPanel = false"
      @config-changed="handleConfigChanged"
    />
  </div>
</template>

<script>
import MessageItem from './components/MessageItem.vue';
import ConfigPanel from './components/ConfigPanel.vue';
import WorkspaceView from './components/WorkspaceView.vue';
import {
  getGeneralAgentConversationDetail,
  chatGeneralAgentConversation,
} from '@/api/generalAgent';
import { SSEEventParser } from './utils/sse-parser';

export default {
  name: 'ChatView',
  components: {
    MessageItem,
    ConfigPanel,
    WorkspaceView,
  },
  props: {
    threadId: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      conversationTitle: '新对话',
      messageList: [],
      inputMessage: '',
      uploadedFiles: [],
      loadingHistory: false,
      isStreaming: false,
      showConfigPanel: false,
      showWorkspace: false,
      currentRunId: '',
      abortController: null,
      pageNo: 1,
      pageSize: 100,
    };
  },
  watch: {
    threadId: {
      immediate: true,
      handler(newVal) {
        if (newVal) {
          this.fetchHistory();
        }
      },
    },
  },
  beforeDestroy() {
    this.stopStreaming();
  },
  methods: {
    async fetchHistory() {
      this.loadingHistory = true;
      this.messageList = [];
      try {
        const res = await getGeneralAgentConversationDetail({
          threadId: this.threadId,
          pageNo: this.pageNo,
          pageSize: this.pageSize,
        });
        if (res.code === 0 && res.data?.list) {
          const messages = [];
          res.data.list.forEach(run => {
            if (run.messages && Array.isArray(run.messages)) {
              messages.push(...run.messages);
            }
            if (run.runId) {
              this.currentRunId = run.runId;
            }
          });
          this.messageList = this.flattenMessages(messages);
        }
        this.$nextTick(() => {
          this.scrollToBottom();
        });
      } catch (error) {
        console.error('获取历史消息失败:', error);
      } finally {
        this.loadingHistory = false;
      }
    },

    flattenMessages(messages) {
      const result = [];
      messages.forEach(msg => {
        result.push({
          id: msg.id || this.generateId(),
          role: msg.role,
          content: this.formatContent(msg.content),
          toolCalls: msg.toolCalls || null,
          toolCallId: msg.toolCallId || null,
        });
      });
      return result;
    },

    formatContent(content) {
      if (typeof content === 'string') {
        return content;
      }
      if (Array.isArray(content)) {
        return content
          .filter(item => item.type === 'text')
          .map(item => item.text)
          .join('\n');
      }
      if (typeof content === 'object' && content !== null) {
        if (content.text) {
          return content.text;
        }
        return JSON.stringify(content, null, 2);
      }
      return '';
    },

    handleKeyDown(e) {
      if (e.shiftKey) {
        return;
      }
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

    async sendMessage() {
      const content = this.inputMessage.trim();
      if (!content && this.uploadedFiles.length === 0) return;
      if (this.isStreaming) return;

      // 构建用户消息
      const userMessage = this.buildUserMessage(content);
      this.messageList.push({
        id: this.generateId(),
        role: 'user',
        content: content,
        files: [...this.uploadedFiles],
      });

      // 清空输入
      this.inputMessage = '';
      const files = [...this.uploadedFiles];
      this.uploadedFiles = [];

      // 开始 SSE 流式请求
      this.startStreaming(userMessage);
    },

    buildUserMessage(content) {
      const message = {
        id: this.generateId(),
        role: 'user',
      };

      if (this.uploadedFiles.length === 0) {
        message.content = content;
      } else {
        const contentArray = [];
        if (content) {
          contentArray.push({ type: 'text', text: content });
        }
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
      this.isStreaming = true;
      this.abortController = new AbortController();

      // 创建助手消息占位
      const assistantMessage = {
        id: this.generateId(),
        role: 'assistant',
        content: '',
        reasoning: '',
        toolCalls: [],
        isStreaming: true,
      };
      this.messageList.push(assistantMessage);

      const parser = new SSEEventParser();

      try {
        await chatGeneralAgentConversation({
          threadId: this.threadId,
          messages: [userMessage],
          onMessage: event => {
            this.handleSSEEvent(event, assistantMessage, parser);
          },
          onError: error => {
            console.error('SSE Error:', error);
            this.$message.error('对话请求失败: ' + error.message);
            assistantMessage.isStreaming = false;
            this.isStreaming = false;
          },
          onOpen: () => {
            // 连接建立
          },
          signal: this.abortController.signal,
        });
      } catch (error) {
        if (error.name !== 'AbortError') {
          console.error('Stream error:', error);
        }
      } finally {
        assistantMessage.isStreaming = false;
        this.isStreaming = false;
        this.abortController = null;
      }
    },

    handleSSEEvent(event, assistantMessage, parser) {
      const parsed = parser.parse(event);
      if (!parsed) return;

      switch (parsed.type) {
        case 'RUN_STARTED':
          this.currentRunId = parsed.runId;
          break;

        case 'TEXT_MESSAGE_CONTENT':
          if (
            parsed.messageId === assistantMessage.id ||
            !assistantMessage.id
          ) {
            assistantMessage.id = parsed.messageId;
            assistantMessage.content += parsed.delta || '';
          }
          break;

        case 'TEXT_MESSAGE_START':
          assistantMessage.id = parsed.messageId;
          break;

        case 'REASONING_MESSAGE_CONTENT':
          assistantMessage.reasoning += parsed.delta || '';
          break;

        case 'TOOL_CALL_START':
          assistantMessage.toolCalls.push({
            id: parsed.toolCallId,
            name: parsed.toolCallName,
            arguments: '',
            status: 'running',
          });
          break;

        case 'TOOL_CALL_ARGS':
          const tool = assistantMessage.toolCalls.find(
            t => t.id === parsed.toolCallId,
          );
          if (tool) {
            tool.arguments += parsed.delta || '';
          }
          break;

        case 'TOOL_CALL_END':
          const toolToEnd = assistantMessage.toolCalls.find(
            t => t.id === parsed.toolCallId,
          );
          if (toolToEnd) {
            toolToEnd.status = 'completed';
          }
          break;

        case 'TOOL_CALL_RESULT':
          this.messageList.push({
            id: parsed.messageId,
            role: 'tool',
            toolCallId: parsed.toolCallId,
            content: parsed.content,
          });
          break;

        case 'ACTIVITY_SNAPSHOT':
          if (parsed.activityType === 'workspace') {
            this.currentRunId = parsed.content?.runId;
          }
          break;

        case 'RUN_FINISHED':
          this.fetchHistory(); // 刷新历史以获取完整消息
          break;
      }

      this.$nextTick(() => {
        this.scrollToBottom();
      });
    },

    stopStreaming() {
      if (this.abortController) {
        this.abortController.abort();
        this.abortController = null;
      }
      this.isStreaming = false;
    },

    scrollToBottom() {
      const container = this.$refs.messageList;
      if (container) {
        container.scrollTop = container.scrollHeight;
      }
    },

    generateId() {
      return (
        'msg_' + Date.now() + '_' + Math.random().toString(36).substr(2, 9)
      );
    },

    updateTitle() {
      this.$emit('update-title', this.threadId, this.conversationTitle);
    },

    handleConfigChanged() {
      // 配置变更后可以刷新
    },
  },
};
</script>

<style lang="scss" scoped>
.chat-container {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
}

.chat-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #fff;

  .header-left {
    flex: 1;

    .title-input {
      max-width: 300px;
    }
  }

  .header-right {
    display: flex;
    gap: 8px;
  }
}

.chat-main {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.message-list {
  flex: 1;
  overflow-y: auto;
  padding: 16px;

  .loading-more {
    text-align: center;
    color: #909399;
    padding: 16px;
  }

  .streaming-indicator {
    text-align: center;
    color: #409eff;
    padding: 16px;
  }
}

.workspace-panel {
  width: 300px;
  border-left: 1px solid #e4e7ed;
  overflow-y: auto;
}

.input-area {
  border-top: 1px solid #e4e7ed;
  padding: 16px;
  background: #fafafa;

  .input-wrapper {
    background: #fff;
    border-radius: 8px;
    border: 1px solid #dcdfe6;

    &:focus-within {
      border-color: #409eff;
    }

    .file-preview {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      padding: 8px;
      border-bottom: 1px solid #ebeef5;

      .file-item {
        position: relative;
        display: flex;
        align-items: center;
        padding: 4px 8px;
        background: #f5f7fa;
        border-radius: 4px;

        .file-thumb {
          width: 32px;
          height: 32px;
          object-fit: cover;
          border-radius: 4px;
        }

        .file-icon {
          width: 32px;
          height: 32px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: #e4e7ed;
          border-radius: 4px;
        }

        .file-name {
          margin-left: 8px;
          font-size: 12px;
          max-width: 100px;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        .el-icon-close {
          margin-left: 8px;
          cursor: pointer;
          color: #909399;

          &:hover {
            color: #f56c6c;
          }
        }
      }
    }

    ::v-deep .el-textarea__inner {
      border: none;
      resize: none;
      padding: 12px;
    }

    .input-actions {
      display: flex;
      justify-content: space-between;
      align-items: center;
      padding: 8px 12px;
      border-top: 1px solid #ebeef5;
    }
  }
}
</style>
