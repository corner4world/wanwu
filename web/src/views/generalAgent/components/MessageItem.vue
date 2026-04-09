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
                :src="file.displayUrl || file.url || file.data"
                class="file-image"
                @click="previewImage(file)"
              />
              <div v-else class="file-card">
                <i class="el-icon-document"></i>
                <span class="file-name">{{ file.fileName }}</span>
              </div>
            </div>
          </div>
          <!-- 文本内容 -->
          <div v-if="message.content" class="message-text">
            {{ message.content }}
          </div>
        </div>
        <div class="user-avatar">
          <img v-if="userAvatarUrl" :src="userAvatarUrl" alt="You" />
          <i v-else class="el-icon-user"></i>
        </div>
      </div>
    </template>

    <!-- 助手消息左侧布局 -->
    <template v-else>
      <div class="assistant-message-wrapper">
        <!-- 头像 -->
        <div class="assistant-avatar">
          <message-header
            :role="message.role"
            :timestamp="message.timestamp"
            :is-streaming="message.isStreaming"
          />
        </div>
        <!-- 消息主体 -->
        <div class="message-body">
          <!-- 按片段顺序展示 -->
          <template v-if="hasFragments">
            <template v-for="(fragment, index) in messageFragments">
              <!-- 思考片段 -->
              <thinking-block
                v-if="fragment.type === 'reasoning'"
                :key="'reasoning-' + index"
                :content="fragment.content"
                :is-streaming="fragment.isStreaming || false"
                :duration="fragment.duration || ''"
                :default-expanded="false"
              />
              <!-- 工具调用片段 -->
              <tool-call-block
                v-else-if="fragment.type === 'tool_call' && fragment.toolCall"
                :key="'tool-' + index"
                :tool-call="fragment.toolCall"
                :result="fragment.toolCall.result || ''"
                :execution-time="fragment.toolCall.executionTime || ''"
                :default-expanded="false"
              />
              <!-- Workspace活动片段 -->
              <workspace-activity
                v-else-if="fragment.type === 'workspace'"
                :key="'workspace-' + index"
                :workspace-info="fragment.workspaceInfo"
                :thread-id="threadId"
                :run-id="fragment.runId"
                @view-workspace="$emit('view-workspace', $event)"
                @download-all="$emit('download-all')"
              />
              <!-- Activity 片段（子智能体） -->
              <activity-block
                v-else-if="fragment.type === 'activity'"
                :key="'activity-' + index"
                :activity-type="fragment.activityType"
                :activity-name="fragment.agentName"
                :fragments="fragment.fragments"
                :is-streaming="fragment.isStreaming || false"
                :duration="fragment.duration || ''"
                :default-expanded="false"
              >
                <template v-for="(subFragment, subIndex) in fragment.fragments">
                  <thinking-block
                    v-if="subFragment.type === 'reasoning'"
                    :key="'sub-reasoning-' + index + '-' + subIndex"
                    :content="subFragment.content"
                    :is-streaming="subFragment.isStreaming || false"
                    :duration="subFragment.duration || ''"
                    :default-expanded="false"
                  />
                  <tool-call-block
                    v-else-if="
                      subFragment.type === 'tool_call' && subFragment.toolCall
                    "
                    :key="'sub-tool-' + index + '-' + subIndex"
                    :tool-call="subFragment.toolCall"
                    :result="subFragment.toolCall.result || ''"
                    :execution-time="subFragment.toolCall.executionTime || ''"
                    :default-expanded="false"
                  />
                  <div
                    v-else-if="
                      subFragment.type === 'text' && subFragment.content
                    "
                    :key="'sub-text-' + index + '-' + subIndex"
                    class="message-content"
                  >
                    <stream-markdown
                      :content="subFragment.content"
                      :is-streaming="subFragment.isStreaming || false"
                    />
                  </div>
                </template>
              </activity-block>
              <!-- 文字片段 -->
              <div
                v-else-if="fragment.type === 'text' && fragment.content"
                :key="'text-' + index"
                class="message-content"
              >
                <stream-markdown
                  :content="fragment.content"
                  :is-streaming="fragment.isStreaming || false"
                />
              </div>
            </template>
          </template>

          <!-- 流式加载指示器 -->
          <div v-if="message.isStreaming" class="streaming-indicator">
            <div class="streaming-dots">
              <span class="dot"></span>
              <span class="dot"></span>
              <span class="dot"></span>
            </div>
          </div>

          <!-- 消息操作按钮 -->
          <div
            v-if="!message.isStreaming && hasContent"
            class="message-actions"
          >
            <CopyIcon class="action-btn" :text="fullContent" type="button" />
            <button class="action-btn" @click="regenerate">
              <i class="el-icon-refresh-right"></i>
            </button>
          </div>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import { mapGetters } from 'vuex';
import { avatarSrc, isImageFile } from '@/utils/util';
import MessageHeader from './MessageHeader.vue';
import ThinkingBlock from './ThinkingBlock.vue';
import ToolCallBlock from './ToolCallBlock.vue';
import StreamMarkdown from './StreamMarkdown.vue';
import TypingCursor from './TypingCursor.vue';
import WorkspaceActivity from './WorkspaceActivity.vue';
import ActivityBlock from './ActivityBlock.vue';
import CopyIcon from '@/components/copyIcon.vue';

export default {
  name: 'MessageItem',
  components: {
    MessageHeader,
    ThinkingBlock,
    ToolCallBlock,
    StreamMarkdown,
    TypingCursor,
    WorkspaceActivity,
    ActivityBlock,
    CopyIcon,
  },
  props: {
    message: {
      type: Object,
      required: true,
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
    return {};
  },
  computed: {
    ...mapGetters('user', ['userAvatar']),
    userAvatarUrl() {
      if (this.userAvatar) {
        return avatarSrc(this.userAvatar);
      }
      return null;
    },
    hasFragments() {
      return this.message.fragments && this.message.fragments.length > 0;
    },
    messageFragments() {
      if (!this.message.fragments) return [];
      return this.message.fragments;
    },
    hasContent() {
      const checkFragmentContent = fragments => {
        if (!fragments) return false;
        return fragments.some(
          f =>
            (f.type === 'text' && (f.content || f.isStreaming)) ||
            (f.type === 'reasoning' && (f.content || f.isStreaming)) ||
            (f.type === 'tool_call' && f.toolCall) ||
            (f.type === 'workspace' && f.workspaceInfo) ||
            (f.type === 'activity' &&
              f.fragments &&
              checkFragmentContent(f.fragments)),
        );
      };

      return checkFragmentContent(this.message.fragments);
    },
    // 获取完整的可复制内容
    fullContent() {
      const parts = [];

      const processFragments = fragments => {
        if (!fragments) return;
        fragments.forEach(fragment => {
          if (fragment.type === 'text' && fragment.content) {
            parts.push(fragment.content);
          } else if (fragment.type === 'reasoning' && fragment.content) {
            parts.push(
              this.$t('generalAgent.message.reasoning') + fragment.content,
            );
          } else if (fragment.type === 'tool_call' && fragment.toolCall) {
            const tc = fragment.toolCall;
            parts.push(
              `${this.$t('generalAgent.message.toolCall')}${tc.name}\n${this.$t('generalAgent.message.args')}${tc.arguments || '{}'}\n${this.$t('generalAgent.message.result')}${tc.result || this.$t('generalAgent.message.none')}`,
            );
          } else if (fragment.type === 'activity' && fragment.fragments) {
            parts.push(
              `${this.$t('generalAgent.message.subAgent')}${fragment.agentName || ''}`,
            );
            processFragments(fragment.fragments);
          }
        });
      };

      // 如果有 fragments，按片段顺序提取
      if (this.message.fragments && this.message.fragments.length > 0) {
        processFragments(this.message.fragments);
      }

      return parts.join('\n\n');
    },
  },
  methods: {
    isImageFile,

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

    regenerate() {
      this.$emit('regenerate', this.message);
    },
  },
};
</script>

<style lang="scss" scoped>
@import '../styles/_variables.scss';

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
      flex-direction: row;
      align-items: flex-end;
      gap: 12px;
      max-width: 70%;
    }

    .user-avatar {
      width: 38px;
      height: 38px;
      border-radius: 10px;
      display: flex;
      align-items: center;
      justify-content: center;
      flex-shrink: 0;
      overflow: hidden;
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);

      img {
        width: 100%;
        height: 100%;
        border-radius: 10px;
        object-fit: cover;
      }

      i {
        color: #fff;
        font-size: 18px;
      }
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

.assistant-message-wrapper {
  display: flex;
  align-items: flex-start;
  gap: 14px;
}

.assistant-avatar {
  flex-shrink: 0;

  .message-header {
    margin-bottom: 0;
  }
}

.message-body {
  flex: 1;
  min-width: 0;
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
  margin-bottom: 16px;
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
