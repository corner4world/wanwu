<template>
  <div :class="['thinking-block', { streaming: isStreaming }]">
    <div class="thinking-header" @click="toggleExpand">
      <div class="header-left">
        <i :class="isExpanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'"></i>
        <div class="thinking-icon-wrapper">
          <img
            :src="require('@/assets/imgs/think-icon.png')"
            class="think-icon"
          />
        </div>
        <span class="thinking-title">
          {{
            isStreaming
              ? $t('generalAgent.thinking.thinking')
              : $t('generalAgent.thinking.title')
          }}
        </span>
      </div>
      <div class="header-right">
        <span v-if="isStreaming" class="thinking-time">
          {{ formattedTimer }}
        </span>
        <span v-else-if="duration" class="thinking-time">
          {{ duration }}
        </span>
        <div v-if="isStreaming" class="status-badge running">
          <span class="status-dot"></span>
          <span class="status-text">
            {{ $t('generalAgent.thinking.running') }}
          </span>
        </div>
        <div v-else class="status-badge completed">
          <span class="status-dot"></span>
          <span class="status-text">
            {{ $t('generalAgent.thinking.completed') }}
          </span>
        </div>
      </div>
    </div>
    <el-collapse-transition>
      <div v-show="isExpanded" class="thinking-body">
        <div v-if="isStreaming && !content" class="skeleton-loader">
          <div class="skeleton-line" style="width: 90%"></div>
          <div class="skeleton-line" style="width: 75%"></div>
          <div class="skeleton-line" style="width: 85%"></div>
          <div class="skeleton-line" style="width: 60%"></div>
        </div>
        <div v-else-if="content" class="thinking-content">
          <pre><code>{{ content }}</code></pre>
          <CopyIcon :text="content" type="button" class="copy-btn" />
        </div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script>
import CopyIcon from '@/components/copyIcon.vue';

export default {
  name: 'ThinkingBlock',
  components: {
    CopyIcon,
  },
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
      default: false,
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
    isStreaming: {
      immediate: true,
      handler(val) {
        if (val) {
          this.isExpanded = true;
          this.startTimer();
        } else {
          this.stopTimer();
          if (this.content) {
            this.isExpanded = false;
          }
        }
      },
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
@import '../styles/_variables.scss';

.thinking-block {
  margin-bottom: 14px;
  border-radius: 14px;
  border: 1px solid rgba(139, 92, 246, 0.2);
  overflow: hidden;
  transition: all 0.3s ease;
  background: #fff;
  font-family: $font-sans;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

  &.streaming {
    border-color: rgba(139, 92, 246, 0.4);
    box-shadow: 0 4px 16px rgba(139, 92, 246, 0.15);

    .thinking-header {
      background: linear-gradient(
        135deg,
        rgba(139, 92, 246, 0.06) 0%,
        #fafafa 100%
      );

      .thinking-title {
        color: $thinking-dark;
        font-weight: 600;
      }
    }

    .think-icon {
      animation: spin 1s linear infinite;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(139, 92, 246, 0.12) 0%,
        rgba(139, 92, 246, 0.08) 100%
      );
      border: 1px solid rgba(139, 92, 246, 0.2);

      .status-dot {
        background: $thinking-color;
        animation: pulse 1.5s infinite;
        box-shadow: 0 0 6px rgba(139, 92, 246, 0.5);
      }

      .status-text {
        color: $thinking-dark;
      }
    }
  }

  .thinking-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    background: linear-gradient(
      135deg,
      rgba(139, 92, 246, 0.06) 0%,
      #fafafa 100%
    );
    cursor: pointer;
    user-select: none;
    transition: background 0.2s ease;

    &:hover {
      background: linear-gradient(180deg, #f5f7f9 0%, #f3f4f6 100%);
    }

    .header-left {
      display: flex;
      align-items: center;
      gap: 12px;
      flex: 1;
      min-width: 0;

      i.el-icon-arrow-down,
      i.el-icon-arrow-right {
        color: $text-muted;
        font-size: 12px;
        transition: transform 0.2s ease;
      }

      .thinking-icon-wrapper {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;

        .think-icon {
          width: 18px;
          height: 18px;
          object-fit: contain;
        }
      }

      .thinking-title {
        font-size: 14px;
        font-weight: 600;
        color: $text-primary;
        letter-spacing: 0.01em;
      }
    }

    .header-right {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-shrink: 0;

      .thinking-time {
        font-size: 13px;
        color: $text-muted;
        font-variant-numeric: tabular-nums;
        background: rgba(0, 0, 0, 0.04);
        padding: 3px 8px;
        border-radius: 8px;
      }

      .status-badge {
        display: flex;
        align-items: center;
        gap: 6px;
        padding: 4px 12px;
        border-radius: 14px;
        font-size: 12px;
        font-weight: 500;
        letter-spacing: 0.02em;

        .status-dot {
          width: 6px;
          height: 6px;
          border-radius: 50%;
        }

        .status-text {
          line-height: 1;
        }

        &.running {
          background: linear-gradient(
            135deg,
            rgba(139, 92, 246, 0.12) 0%,
            rgba(139, 92, 246, 0.08) 100%
          );
          border: 1px solid rgba(139, 92, 246, 0.2);

          .status-dot {
            background: $thinking-color;
            animation: pulse 1.5s infinite;
            box-shadow: 0 0 6px rgba(139, 92, 246, 0.5);
          }

          .status-text {
            color: $thinking-dark;
          }
        }

        &.completed {
          background: linear-gradient(
            135deg,
            rgba(16, 163, 127, 0.12) 0%,
            rgba(16, 163, 127, 0.08) 100%
          );
          border: 1px solid rgba(16, 163, 127, 0.2);

          .status-dot {
            background: #10a37f;
            box-shadow: 0 0 4px rgba(16, 163, 127, 0.4);
          }

          .status-text {
            color: #0d8a6a;
          }
        }
      }
    }
  }

  .thinking-body {
    padding: 16px 18px;
    background: #fff;
    border-top: 1px solid #e8ecf0;

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

    .thinking-content {
      position: relative;
      background: linear-gradient(135deg, #fafbfc 0%, #f8f9fa 100%);
      border-radius: 12px;
      overflow: hidden;
      border: 1px solid #e8ecf0;

      pre {
        margin: 0;
        padding: 16px 50px 16px 16px;
        overflow-x: auto;

        code {
          font-family: $font-mono;
          font-size: 14px;
          line-height: 1.7;
          color: #1f2937;
          white-space: pre-wrap;
          word-break: break-word;
        }
      }

      .copy-btn {
        position: absolute;
        top: 10px;
        right: 10px;
      }
    }
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

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
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
</style>
