<template>
  <div
    :class="[
      'thinking-block',
      { expanded: isExpanded, streaming: isStreaming },
    ]"
  >
    <div class="thinking-header" @click="toggleExpand">
      <div class="header-left">
        <i
          :class="isExpanded ? 'el-icon-arrow-down' : 'el-icon-arrow-right'"
        ></i>
        <div class="thinking-icon-wrapper">
          <!-- 思考图标 - 脑/灯泡 -->
          <svg
            v-if="!isStreaming"
            class="thinking-icon"
            viewBox="0 0 24 24"
            width="18"
            height="18"
          >
            <path
              fill="currentColor"
              d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm-1 17.93c-3.95-.49-7-3.85-7-7.93 0-.62.08-1.21.21-1.79L9 15v1c0 1.1.9 2 2 2v1.93zm6.9-2.54c-.26-.81-1-1.39-1.9-1.39h-1v-3c0-.55-.45-1-1-1H8v-2h2c.55 0 1-.45 1-1V7h2c1.1 0 2-.9 2-2v-.41c2.93 1.19 5 4.06 5 7.41 0 2.08-.8 3.97-2.1 5.39z"
            />
          </svg>
          <!-- 流式时的加载动画 -->
          <div v-else class="thinking-spinner">
            <svg viewBox="0 0 24 24" width="18" height="18">
              <circle
                cx="12"
                cy="12"
                r="10"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-dasharray="31.4 31.4"
                stroke-linecap="round"
              >
                <animateTransform
                  attributeName="transform"
                  type="rotate"
                  from="0 12 12"
                  to="360 12 12"
                  dur="1s"
                  repeatCount="indefinite"
                />
              </circle>
            </svg>
          </div>
        </div>
        <span class="thinking-title">
          {{ isStreaming ? '思考中...' : '思考过程' }}
        </span>
        <span v-if="isStreaming" class="thinking-timer">
          {{ formattedTimer }}
        </span>
        <span v-else-if="duration" class="thinking-duration">
          {{ duration }}
        </span>
      </div>
      <div class="header-right">
        <span v-if="lineCount > 0" class="line-count">{{ lineCount }} 行</span>
      </div>
    </div>
    <el-collapse-transition>
      <div v-show="isExpanded" class="thinking-content">
        <!-- 骨架屏加载动画 -->
        <div v-if="isStreaming && !content" class="skeleton-loader">
          <div class="skeleton-line" style="width: 90%"></div>
          <div class="skeleton-line" style="width: 75%"></div>
          <div class="skeleton-line" style="width: 85%"></div>
          <div class="skeleton-line" style="width: 60%"></div>
        </div>
        <!-- 实际内容 -->
        <div v-else class="thinking-text" v-html="formattedContent"></div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script>
export default {
  name: 'ThinkingBlock',
  props: {
    content: {
      type: String,
      default: '',
    },
    isStreaming: {
      type: Boolean,
      default: false,
    },
    duration: {
      type: String,
      default: '',
    },
    defaultExpanded: {
      type: Boolean,
      default: true,
    },
  },
  data() {
    return {
      isExpanded: this.defaultExpanded,
      timer: 0,
      timerInterval: null,
    };
  },
  computed: {
    formattedContent() {
      if (!this.content) return '';
      // 处理换行，保留空格
      return this.content
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/\n/g, '<br>')
        .replace(/\s/g, '&nbsp;');
    },
    lineCount() {
      if (!this.content) return 0;
      return this.content.split('\n').length;
    },
    formattedTimer() {
      const seconds = Math.floor(this.timer / 1000);
      const minutes = Math.floor(seconds / 60);
      const secs = seconds % 60;
      if (minutes > 0) {
        return `${minutes}:${secs.toString().padStart(2, '0')}`;
      }
      return `${secs}s`;
    },
  },
  watch: {
    isStreaming(val) {
      // 流式输出时自动展开并启动计时器
      if (val) {
        if (!this.isExpanded) {
          this.isExpanded = true;
        }
        this.startTimer();
      } else {
        this.stopTimer();
      }
    },
  },
  beforeDestroy() {
    this.stopTimer();
  },
  methods: {
    toggleExpand() {
      this.isExpanded = !this.isExpanded;
    },
    startTimer() {
      this.timer = 0;
      this.stopTimer();
      this.timerInterval = setInterval(() => {
        this.timer += 100;
      }, 100);
    },
    stopTimer() {
      if (this.timerInterval) {
        clearInterval(this.timerInterval);
        this.timerInterval = null;
      }
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
$text-muted: #6b7280;
$thinking-color: #8b5cf6;
$thinking-light: #a78bfa;
$thinking-bg: rgba(139, 92, 246, 0.08);

.thinking-block {
  margin-bottom: 16px;
  border-radius: 14px;
  background: linear-gradient(
    135deg,
    rgba(139, 92, 246, 0.04) 0%,
    #fafafa 100%
  );
  border: 1px solid rgba(139, 92, 246, 0.15);
  overflow: hidden;
  transition: all 0.3s ease;
  font-family: $font-sans;

  &.streaming {
    border-color: $thinking-color;
    background: linear-gradient(
      135deg,
      rgba(139, 92, 246, 0.06) 0%,
      #fafafa 100%
    );
    box-shadow: 0 4px 16px rgba(139, 92, 246, 0.15);

    .thinking-header {
      .thinking-title {
        color: $thinking-color;
        font-weight: 600;
      }
    }
  }

  .thinking-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    cursor: pointer;
    user-select: none;
    transition: background 0.2s ease;

    &:hover {
      background: rgba(139, 92, 246, 0.04);
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;

      i.el-icon-arrow-down,
      i.el-icon-arrow-right {
        color: $thinking-color;
        font-size: 12px;
        transition: transform 0.2s ease;
      }

      .thinking-icon-wrapper {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 28px;
        height: 28px;
        background: linear-gradient(
          135deg,
          rgba(139, 92, 246, 0.15) 0%,
          rgba(167, 139, 250, 0.1) 100%
        );
        border-radius: 8px;
      }

      .thinking-icon {
        color: $thinking-color;
        flex-shrink: 0;
      }

      .thinking-spinner {
        color: $thinking-color;

        svg {
          display: block;
        }
      }

      .thinking-title {
        font-size: 14px;
        font-weight: 600;
        color: $thinking-color;
        letter-spacing: 0.01em;
      }

      .thinking-timer {
        font-size: 13px;
        color: $thinking-color;
        background: linear-gradient(
          135deg,
          rgba(139, 92, 246, 0.15) 0%,
          rgba(139, 92, 246, 0.08) 100%
        );
        padding: 3px 10px;
        border-radius: 12px;
        font-weight: 500;
        font-variant-numeric: tabular-nums;
        border: 1px solid rgba(139, 92, 246, 0.2);
      }

      .thinking-duration {
        font-size: 13px;
        color: $thinking-light;
        margin-left: 4px;
        font-variant-numeric: tabular-nums;
      }
    }

    .header-right {
      .line-count {
        font-size: 12px;
        color: $thinking-color;
        padding: 4px 10px;
        background: rgba(139, 92, 246, 0.1);
        border-radius: 12px;
        font-weight: 500;
      }
    }
  }

  .thinking-content {
    border-top: 1px solid rgba(139, 92, 246, 0.1);
    background: #fff;

    .skeleton-loader {
      padding: 18px;

      .skeleton-line {
        height: 14px;
        background: linear-gradient(
          90deg,
          rgba(139, 92, 246, 0.05) 25%,
          rgba(139, 92, 246, 0.1) 50%,
          rgba(139, 92, 246, 0.05) 75%
        );
        background-size: 200% 100%;
        border-radius: 6px;
        margin-bottom: 12px;
        animation: shimmer 1.5s infinite;

        &:last-child {
          margin-bottom: 0;
        }
      }
    }

    .thinking-text {
      padding: 18px;
      font-size: 14px;
      line-height: 1.85;
      color: $text-secondary;
      font-family: $font-sans;
      white-space: pre-wrap;
      word-break: break-word;
      max-height: 400px;
      overflow-y: auto;
      animation: fadeIn 0.3s ease;

      &::-webkit-scrollbar {
        width: 5px;
      }

      &::-webkit-scrollbar-track {
        background: #f8f9fa;
        border-radius: 3px;
      }

      &::-webkit-scrollbar-thumb {
        background: rgba(139, 92, 246, 0.3);
        border-radius: 3px;

        &:hover {
          background: rgba(139, 92, 246, 0.5);
        }
      }
    }
  }
}

@keyframes shimmer {
  0% {
    background-position: 200% 0;
  }
  100% {
    background-position: -200% 0;
  }
}

@keyframes fadeIn {
  from {
    opacity: 0;
    transform: translateY(-4px);
  }
  to {
    opacity: 1;
    transform: translateY(0);
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
