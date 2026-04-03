<template>
  <div :class="['message-item', `message-${message.role}`]">
    <!-- 用户消息右侧布局 -->
    <template v-if="message.role === 'user'">
      <div class="user-message-wrapper">
        <div class="user-message-content">
          <!-- 文件展示 -->
          <div
            v-if="message.files && message.files.length > 0"
            class="message-files"
          >
            <div
              v-for="(file, index) in message.files"
              :key="index"
              class="file-item"
            >
              <img
                v-if="isImageFile(file)"
                :src="file.url || file.data"
                class="file-image"
                @click="previewImage(file)"
              />
              <div v-else class="file-card">
                <i class="el-icon-document"></i>
                <span class="file-name">{{ file.name }}</span>
              </div>
            </div>
          </div>
          <!-- 文本内容 -->
          <div v-if="message.content" class="message-text">
            {{ message.content }}
          </div>
        </div>
      </div>
    </template>

    <!-- 助手消息左侧布局 -->
    <template v-else>
      <!-- 消息头部 -->
      <message-header
        :role="message.role"
        :timestamp="message.timestamp"
        :is-streaming="message.isStreaming"
      />

      <!-- 消息主体 -->
      <div class="message-body">
        <!-- 按片段顺序展示 -->
        <template v-if="hasFragments">
          <div v-for="(fragment, index) in messageFragments" :key="index">
            <!-- 思考片段 -->
            <thinking-block
              v-if="fragment.type === 'reasoning'"
              :content="fragment.content"
              :is-streaming="false"
              :duration="fragment.duration || ''"
              :default-expanded="false"
            />
            <!-- 工具调用片段 -->
            <tool-call-block
              v-else-if="fragment.type === 'tool_call'"
              :tool-call="fragment.toolCall"
              :result="fragment.toolCall.result"
              :execution-time="fragment.toolCall.executionTime || ''"
              :default-expanded="false"
            />
            <!-- Workspace活动片段 -->
            <workspace-activity
              v-else-if="fragment.type === 'workspace'"
              :workspace-info="fragment.workspaceInfo"
              :thread-id="threadId"
              :run-id="fragment.runId"
              @view-workspace="$emit('view-workspace', $event)"
              @download-all="$emit('download-all', $event)"
            />
            <!-- 文字片段 -->
            <div
              v-else-if="fragment.type === 'text' && fragment.content"
              class="message-content"
            >
              <markdown-renderer :content="fragment.content" />
            </div>
          </div>
        </template>

        <!-- 兼容旧格式：没有 fragments 时使用原来的展示方式 -->
        <template v-else>
          <!-- 阶段区域：思考过程 + 工具调用（默认折叠） -->
          <div v-if="hasStages" class="stages-container">
            <!-- 思考过程块 -->
            <thinking-block
              v-if="message.reasoning && message.reasoning.length > 0"
              :content="message.reasoning"
              :is-streaming="message.isStreaming && !message.content"
              :duration="message.reasoningDuration"
              :default-expanded="false"
            />

            <!-- 工具调用块 -->
            <tool-call-block
              v-for="toolCall in visibleToolCalls"
              :key="toolCall.id"
              :tool-call="toolCall"
              :result="getToolResult(toolCall.id)"
              :execution-time="toolCall.executionTime"
              :default-expanded="false"
            />
          </div>

          <!-- 文本内容 -->
          <div
            v-if="message.content && message.content.length > 0"
            class="message-content"
          >
            <markdown-renderer :content="message.content" />
            <typing-cursor v-if="message.isStreaming" />
          </div>
        </template>

        <!-- 流式加载指示器 -->
        <div
          v-if="message.isStreaming && !hasContent"
          class="streaming-indicator"
        >
          <div class="streaming-dots">
            <span class="dot"></span>
            <span class="dot"></span>
            <span class="dot"></span>
          </div>
          <span class="streaming-text">AI 正在响应...</span>
        </div>

        <!-- 消息操作按钮 -->
        <div v-if="!message.isStreaming && hasContent" class="message-actions">
          <el-tooltip content="复制内容" placement="top">
            <button class="action-btn" @click="copyContent">
              <i
                :class="copied ? 'el-icon-check' : 'el-icon-document-copy'"
              ></i>
            </button>
          </el-tooltip>
          <el-tooltip v-if="isLastMessage" content="重新生成" placement="top">
            <button class="action-btn" @click="regenerate">
              <i class="el-icon-refresh-right"></i>
            </button>
          </el-tooltip>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import MessageHeader from './MessageHeader.vue';
import ThinkingBlock from './ThinkingBlock.vue';
import ToolCallBlock from './ToolCallBlock.vue';
import MarkdownRenderer from './MarkdownRenderer.vue';
import TypingCursor from './TypingCursor.vue';
import WorkspaceActivity from './WorkspaceActivity.vue';

export default {
  name: 'MessageItem',
  components: {
    MessageHeader,
    ThinkingBlock,
    ToolCallBlock,
    MarkdownRenderer,
    TypingCursor,
    WorkspaceActivity,
  },
  props: {
    message: {
      type: Object,
      required: true,
    },
    toolResults: {
      type: Array,
      default: () => [],
    },
    isLastMessage: {
      type: Boolean,
      default: false,
    },
    threadId: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      copied: false,
    };
  },
  computed: {
    hasFragments() {
      return this.message.fragments && this.message.fragments.length > 0;
    },
    messageFragments() {
      if (!this.message.fragments) return [];
      return this.message.fragments;
    },
    hasStages() {
      const hasReasoning =
        this.message.reasoning && this.message.reasoning.length > 0;
      const hasToolCalls =
        this.message.toolCalls && this.message.toolCalls.length > 0;
      return hasReasoning || hasToolCalls;
    },
    hasContent() {
      const hasText = this.message.content && this.message.content.length > 0;
      const hasReasoning =
        this.message.reasoning && this.message.reasoning.length > 0;
      const hasToolCalls =
        this.message.toolCalls && this.message.toolCalls.length > 0;
      const hasFragmentsContent =
        this.message.fragments &&
        this.message.fragments.some(
          f =>
            f.content || (f.toolCall && f.toolCall.result) || f.workspaceInfo,
        );
      return hasText || hasReasoning || hasToolCalls || hasFragmentsContent;
    },
    visibleToolCalls() {
      if (!this.message.toolCalls || !Array.isArray(this.message.toolCalls)) {
        return [];
      }
      return this.message.toolCalls;
    },
    // 获取完整的可复制内容
    fullContent() {
      const parts = [];

      // 如果有 fragments，按片段顺序提取
      if (this.message.fragments && this.message.fragments.length > 0) {
        this.message.fragments.forEach(fragment => {
          if (fragment.type === 'text' && fragment.content) {
            parts.push(fragment.content);
          } else if (fragment.type === 'reasoning' && fragment.content) {
            parts.push(`【思考过程】\n${fragment.content}`);
          } else if (fragment.type === 'tool_call' && fragment.toolCall) {
            const tc = fragment.toolCall;
            parts.push(
              `【工具调用】${tc.name}\n参数: ${tc.arguments || '{}'}\n结果: ${tc.result || '无'}`,
            );
          }
        });
      } else {
        // 兼容旧格式
        if (this.message.reasoning) {
          parts.push(`【思考过程】\n${this.message.reasoning}`);
        }
        if (this.message.toolCalls && this.message.toolCalls.length > 0) {
          this.message.toolCalls.forEach(tc => {
            parts.push(
              `【工具调用】${tc.name}\n参数: ${tc.arguments || '{}'}\n结果: ${tc.result || '无'}`,
            );
          });
        }
        if (this.message.content) {
          parts.push(this.message.content);
        }
      }

      return parts.join('\n\n');
    },
  },
  methods: {
    isImageFile(file) {
      const imageTypes = [
        'image/jpeg',
        'image/png',
        'image/gif',
        'image/webp',
        'image/bmp',
      ];
      const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'webp', 'bmp'];

      if (file.type && imageTypes.includes(file.type)) {
        return true;
      }

      if (file.name) {
        const ext = file.name.split('.').pop().toLowerCase();
        return imageExts.includes(ext);
      }

      return false;
    },

    getToolResult(toolCallId) {
      // 优先从 toolCall 本身获取 result
      const toolCall = this.message.toolCalls?.find(t => t.id === toolCallId);
      if (toolCall && toolCall.result) {
        return toolCall.result;
      }
      // 从 toolResults 数组获取
      if (this.message.toolResults && this.message.toolResults.length > 0) {
        const result = this.message.toolResults.find(
          r => r.toolCallId === toolCallId,
        );
        if (result) return result.content;
      }
      // 从 props 传入的 toolResults 获取
      if (this.toolResults && this.toolResults.length > 0) {
        const result = this.toolResults.find(r => r.toolCallId === toolCallId);
        if (result) return result.content;
      }
      return '';
    },

    previewImage(file) {
      const url = file.url || file.data;
      if (url) {
        const div = document.createElement('div');
        div.style.cssText = `
          position: fixed;
          top: 0;
          left: 0;
          right: 0;
          bottom: 0;
          background: rgba(0,0,0,0.9);
          display: flex;
          align-items: center;
          justify-content: center;
          z-index: 9999;
          cursor: zoom-out;
        `;
        div.onclick = () => document.body.removeChild(div);

        const img = document.createElement('img');
        img.src = url;
        img.style.cssText = `
          max-width: 90%;
          max-height: 90%;
          object-fit: contain;
        `;

        div.appendChild(img);
        document.body.appendChild(div);
      }
    },

    async copyContent() {
      try {
        await navigator.clipboard.writeText(this.fullContent);
        this.copied = true;
        setTimeout(() => {
          this.copied = false;
        }, 2000);
      } catch (err) {
        console.error('Copy failed:', err);
        // 降级方案
        const textarea = document.createElement('textarea');
        textarea.value = this.fullContent;
        textarea.style.position = 'fixed';
        textarea.style.opacity = '0';
        document.body.appendChild(textarea);
        textarea.select();
        try {
          document.execCommand('copy');
          this.copied = true;
          setTimeout(() => {
            this.copied = false;
          }, 2000);
        } catch (e) {
          console.error('Fallback copy failed:', e);
        }
        document.body.removeChild(textarea);
      }
    },

    regenerate() {
      this.$emit('regenerate', this.message);
    },
  },
};
</script>

<style lang="scss" scoped>
// 字体变量
$font-sans:
  -apple-system, BlinkMacSystemFont, 'Segoe UI', 'PingFang SC',
  'Hiragino Sans GB', 'Microsoft YaHei', 'Helvetica Neue', Helvetica, Arial,
  sans-serif;
$font-mono:
  'JetBrains Mono', 'SF Mono', 'Fira Code', Monaco, Consolas, 'Liberation Mono',
  monospace;

// 颜色变量
$text-primary: #1f2937;
$text-secondary: #4b5563;
$accent-color: #10a37f;
$accent-dark: #0d8a6a;
$user-gradient-start: #10a37f;
$user-gradient-end: #0d8a6a;

.message-item {
  padding: 20px 0;
  border-bottom: 1px solid #f0f0f0;
  font-family: $font-sans;

  &:last-child {
    border-bottom: none;
  }

  // 用户消息 - 右侧显示
  &.message-user {
    display: flex;
    justify-content: flex-end;
    padding-left: 48px;

    .user-message-wrapper {
      display: flex;
      flex-direction: column;
      align-items: flex-end;
      max-width: 70%;
    }

    .user-message-content {
      background: linear-gradient(
        135deg,
        $user-gradient-start 0%,
        $user-gradient-end 100%
      );
      color: #fff;
      padding: 14px 20px;
      border-radius: 20px 20px 6px 20px;
      box-shadow: 0 3px 12px rgba(16, 163, 127, 0.25);
      position: relative;

      // 添加微光效果
      &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        bottom: 0;
        border-radius: inherit;
        background: linear-gradient(
          135deg,
          rgba(255, 255, 255, 0.15) 0%,
          transparent 50%
        );
        pointer-events: none;
      }
    }

    .message-text {
      font-size: 16px;
      line-height: 1.85;
      word-break: break-word;
      white-space: pre-wrap;
      letter-spacing: 0.02em;
    }

    .message-files {
      display: flex;
      flex-wrap: wrap;
      gap: 10px;
      margin-bottom: 10px;
      justify-content: flex-end;

      .file-item {
        .file-image {
          max-width: 240px;
          max-height: 180px;
          border-radius: 14px;
          cursor: pointer;
          transition: all 0.25s ease;
          box-shadow: 0 3px 12px rgba(0, 0, 0, 0.15);

          &:hover {
            transform: scale(1.02) translateY(-2px);
            box-shadow: 0 6px 20px rgba(0, 0, 0, 0.2);
          }
        }

        .file-card {
          display: flex;
          align-items: center;
          gap: 10px;
          padding: 12px 16px;
          background: rgba(255, 255, 255, 0.18);
          border-radius: 12px;
          backdrop-filter: blur(10px);
          border: 1px solid rgba(255, 255, 255, 0.1);
          transition: all 0.2s ease;

          &:hover {
            background: rgba(255, 255, 255, 0.25);
            transform: translateY(-1px);
          }

          i {
            font-size: 20px;
            color: #fff;
          }

          .file-name {
            font-size: 14px;
            color: #fff;
            max-width: 180px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
            font-weight: 500;
          }
        }
      }
    }
  }

  // 助手消息 - 左侧显示
  &.message-assistant {
    padding-right: 48px;
  }
}

.message-body {
  padding-left: 44px;
}

.stages-container {
  margin-bottom: 16px;
  padding: 16px;
  background: linear-gradient(135deg, #fafbfc 0%, #f5f7f9 100%);
  border-radius: 16px;
  border: 1px solid #e8ecf0;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
}

.message-content {
  min-height: 20px;
  // 行高由 MarkdownRenderer 组件控制
}

.message-actions {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px solid #f0f0f0;

  .action-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    background: #f9fafb;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    color: #6b7280;
    font-size: 14px;
    cursor: pointer;
    transition: all 0.2s ease;

    i {
      font-size: 15px;
    }

    &:hover {
      background: #fff;
      border-color: #d1d5db;
      color: #374151;
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.05);
    }

    &:active {
      transform: scale(0.95);
    }

    i.el-icon-check {
      color: $accent-color;
    }
  }
}

.streaming-indicator {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 20px;
  background: linear-gradient(
    135deg,
    rgba(16, 163, 127, 0.06) 0%,
    #fafafa 100%
  );
  border-radius: 12px;
  border: 1px solid rgba(16, 163, 127, 0.12);

  .streaming-dots {
    display: flex;
    align-items: center;
    gap: 5px;

    .dot {
      width: 8px;
      height: 8px;
      background: linear-gradient(135deg, $accent-color 0%, $accent-dark 100%);
      border-radius: 50%;
      animation: bounce 1.4s infinite ease-in-out;
      box-shadow: 0 0 8px rgba(16, 163, 127, 0.4);

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
  }

  .streaming-text {
    font-size: 14px;
    color: $accent-color;
    font-weight: 500;
    letter-spacing: 0.02em;
  }
}

@keyframes bounce {
  0%,
  60%,
  100% {
    transform: translateY(0);
    opacity: 0.5;
  }
  30% {
    transform: translateY(-8px);
    opacity: 1;
  }
}
</style>
