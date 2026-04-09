<template>
  <div :class="['tool-call-block', statusClass]">
    <div class="tool-header" @click="toggleExpand">
      <div class="header-left">
        <i :class="isExpanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'"></i>
        <div class="tool-icon-wrapper">
          <img
            :src="require('@/assets/imgs/tool-icon.png')"
            class="tool-icon-img"
          />
        </div>
        <span class="tool-name">{{ formatToolName(toolCall.name) }}</span>
      </div>
      <div class="header-right">
        <span v-if="executionTime" class="execution-time">
          {{ executionTime }}
        </span>
        <div :class="['status-badge', toolCall.status]">
          <span class="status-dot"></span>
          <span class="status-text">{{ statusText }}</span>
        </div>
      </div>
    </div>
    <el-collapse-transition>
      <div v-show="isExpanded" class="tool-body">
        <!-- 参数展示 -->
        <div v-if="hasArgs" class="tool-section">
          <div class="section-label">
            <i class="el-icon-setting"></i>
            <span>{{ $t('generalAgent.toolCall.parameters') }}</span>
          </div>
          <div class="tool-arguments">
            <pre><code>{{ formattedArgs }}</code></pre>
            <CopyIcon :text="formattedArgs" type="button" class="copy-btn" />
          </div>
        </div>
        <!-- 结果展示 -->
        <div v-if="result" class="tool-section">
          <div class="section-label">
            <i class="el-icon-document"></i>
            <span>{{ $t('generalAgent.toolCall.result') }}</span>
            <span v-if="resultLength" class="result-length">
              {{ resultLength }}
            </span>
          </div>
          <div class="tool-result">
            <pre><code>{{ formattedResult }}</code></pre>
            <CopyIcon :text="formattedResult" type="button" class="copy-btn" />
          </div>
        </div>
        <!-- 运行中进度指示 -->
        <div v-if="toolCall.status === 'running'" class="tool-progress">
          <div class="progress-bar">
            <div class="progress-fill"></div>
          </div>
        </div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script>
import CopyIcon from '@/components/copyIcon.vue';

export default {
  name: 'ToolCallBlock',
  components: {
    CopyIcon,
  },
  props: {
    toolCall: {
      type: Object,
      required: true,
      default: () => ({
        id: '',
        name: '',
        arguments: '',
        status: 'running',
      }),
    },
    result: {
      type: String,
      default: '',
    },
    executionTime: {
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
    };
  },
  computed: {
    statusClass() {
      const status = this.toolCall.status;
      return (
        {
          running: 'status-running',
          completed: 'status-completed',
          error: 'status-error',
        }[status] || ''
      );
    },
    statusText() {
      const status = this.toolCall.status;
      const texts = {
        running: this.$t('generalAgent.toolCall.running'),
        completed: this.$t('generalAgent.toolCall.completed'),
        error: this.$t('generalAgent.toolCall.failed'),
      };
      return texts[status] || this.$t('generalAgent.toolCall.pending');
    },
    hasArgs() {
      if (!this.toolCall.arguments) return false;
      try {
        const parsed = JSON.parse(this.toolCall.arguments);
        return Object.keys(parsed).length > 0;
      } catch {
        return this.toolCall.arguments.length > 0;
      }
    },
    parsedArgs() {
      try {
        return JSON.parse(this.toolCall.arguments || '{}');
      } catch {
        return null;
      }
    },
    formattedArgs() {
      if (!this.toolCall.arguments) return '';
      try {
        const parsed = JSON.parse(this.toolCall.arguments);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return this.toolCall.arguments;
      }
    },
    formattedResult() {
      if (!this.result) return '';
      try {
        const parsed = JSON.parse(this.result);
        return JSON.stringify(parsed, null, 2);
      } catch {
        return this.result;
      }
    },
    resultLength() {
      if (!this.result) return '';
      const len = this.result.length;
      if (len > 1024) {
        return `${(len / 1024).toFixed(1)} KB`;
      }
      return this.$t('generalAgent.toolCall.chars', { len });
    },
  },
  watch: {
    'toolCall.status': {
      immediate: true,
      handler(newStatus, oldStatus) {
        // 运行中时自动展开
        if (newStatus === 'running') {
          this.isExpanded = true;
        }
        // 完成或失败时自动收起（从 running 变化时）
        if (
          oldStatus === 'running' &&
          (newStatus === 'completed' || newStatus === 'error')
        ) {
          this.isExpanded = false;
        }
      },
    },
  },
  methods: {
    toggleExpand() {
      this.isExpanded = !this.isExpanded;
    },
    formatToolName(name) {
      if (!name) return 'Unknown Tool';
      // 将下划线或连字符转为空格，首字母大写
      return name.replace(/[_-]/g, ' ').replace(/\b\w/g, c => c.toUpperCase());
    },
  },
};
</script>

<style lang="scss" scoped>
@import '../styles/_variables.scss';

.tool-call-block {
  margin-bottom: 14px;
  border-radius: 14px;
  border: 1px solid rgba(249, 115, 22, 0.2);
  overflow: hidden;
  transition: all 0.3s ease;
  background: #fff;
  font-family: $font-sans;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

  &.status-running {
    border-color: rgba(249, 115, 22, 0.4);
    box-shadow: 0 4px 16px rgba(249, 115, 22, 0.15);

    .tool-header {
      background: linear-gradient(
        135deg,
        rgba(249, 115, 22, 0.06) 0%,
        #fafafa 100%
      );

      .tool-name {
        color: $orange-dark;
      }
    }

    .tool-icon-img {
      animation: spin 1s linear infinite;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(249, 115, 22, 0.12) 0%,
        rgba(249, 115, 22, 0.08) 100%
      );
      border: 1px solid rgba(249, 115, 22, 0.2);

      .status-dot {
        background: $orange-primary;
        animation: pulse 1.5s infinite;
        box-shadow: 0 0 6px rgba(249, 115, 22, 0.5);
      }

      .status-text {
        color: $orange-dark;
      }
    }
  }

  &.status-completed {
    border-color: rgba(16, 163, 127, 0.3);
    box-shadow: 0 2px 8px rgba(16, 163, 127, 0.08);

    .tool-header {
      background: linear-gradient(
        135deg,
        rgba(16, 163, 127, 0.06) 0%,
        #fafafa 100%
      );
    }

    .tool-icon {
      color: $accent-color;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(16, 163, 127, 0.12) 0%,
        rgba(16, 163, 127, 0.08) 100%
      );
      border: 1px solid rgba(16, 163, 127, 0.2);

      .status-dot {
        background: $accent-color;
        box-shadow: 0 0 4px rgba(16, 163, 127, 0.4);
      }

      .status-text {
        color: $accent-dark;
      }
    }
  }

  &.status-error {
    border-color: rgba(239, 68, 68, 0.3);
    box-shadow: 0 2px 8px rgba(239, 68, 68, 0.08);

    .tool-header {
      background: linear-gradient(
        135deg,
        rgba(239, 68, 68, 0.06) 0%,
        #fafafa 100%
      );
    }

    .tool-icon {
      color: $red-primary;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(239, 68, 68, 0.12) 0%,
        rgba(239, 68, 68, 0.08) 100%
      );
      border: 1px solid rgba(239, 68, 68, 0.2);

      .status-dot {
        background: $red-primary;
      }

      .status-text {
        color: $red-dark;
      }
    }
  }

  .tool-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 18px;
    background: linear-gradient(180deg, #fafbfc 0%, #f8f9fa 100%);
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

      .tool-icon-wrapper {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 24px;
        height: 24px;

        .tool-icon-img {
          width: 18px;
          height: 18px;
          object-fit: contain;
        }
      }

      .tool-name {
        font-size: 14px;
        font-weight: 600;
        color: $text-primary;
        font-family: $font-mono;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
        letter-spacing: 0.01em;
      }
    }

    .header-right {
      display: flex;
      align-items: center;
      gap: 12px;
      flex-shrink: 0;

      .execution-time {
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
      }
    }
  }

  .tool-body {
    padding: 16px 18px;
    background: #fff;
    border-top: 1px solid #e8ecf0;

    .tool-section {
      margin-bottom: 18px;

      &:last-child {
        margin-bottom: 0;
      }

      .section-label {
        display: flex;
        align-items: center;
        gap: 8px;
        font-size: 11px;
        font-weight: 600;
        color: $text-secondary;
        text-transform: uppercase;
        letter-spacing: 0.08em;
        margin-bottom: 12px;

        i {
          font-size: 13px;
          color: $text-muted;
        }

        .result-length {
          margin-left: auto;
          font-weight: 400;
          color: $text-muted;
          text-transform: none;
          font-size: 10px;
          background: rgba(0, 0, 0, 0.04);
          padding: 2px 8px;
          border-radius: 8px;
        }
      }

      .tool-arguments,
      .tool-result {
        position: relative;
        background: linear-gradient(135deg, #fafbfc 0%, #f8f9fa 100%);
        border-radius: 12px;
        overflow: hidden;
        border: 1px solid #e8ecf0;

        pre {
          margin: 0;
          padding: 16px 50px 16px 16px;
          overflow-x: auto;
          max-height: 320px;

          &::-webkit-scrollbar {
            height: 6px;
            width: 6px;
          }

          &::-webkit-scrollbar-track {
            background: transparent;
          }

          &::-webkit-scrollbar-thumb {
            background: #d1d5db;
            border-radius: 3px;

            &:hover {
              background: #9ca3af;
            }
          }

          code {
            font-family: $font-mono;
            font-size: 14px;
            line-height: 1.7;
            color: #1f2937;
          }
        }

        .copy-btn {
          position: absolute;
          top: 10px;
          right: 10px;
          padding: 7px 14px;
          background: #fff;
          border: 1px solid #d1d5db;
          border-radius: 8px;
          color: $text-secondary;
          font-size: 13px;
          font-family: $font-sans;
          cursor: pointer;
          transition: all 0.2s ease;
          display: flex;
          align-items: center;
          gap: 6px;
          box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);

          &:hover {
            background: #f9fafb;
            border-color: #9ca3af;
            color: $text-primary;
            transform: translateY(-1px);
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
          }

          i.el-icon-check {
            color: $accent-color;
          }
        }
      }
    }

    .tool-progress {
      display: flex;
      align-items: center;
      gap: 14px;
      padding: 12px 0;

      .progress-bar {
        flex: 1;
        height: 5px;
        background: #f0f0f0;
        border-radius: 3px;
        overflow: hidden;

        .progress-fill {
          height: 100%;
          background: linear-gradient(90deg, $orange-primary, #fbbf24);
          border-radius: 3px;
          animation: progressPulse 1.5s ease-in-out infinite;
        }
      }

      .progress-text {
        font-size: 12px;
        color: $orange-primary;
        white-space: nowrap;
        font-weight: 500;
      }
    }
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

@keyframes progressPulse {
  0% {
    width: 20%;
    margin-left: 0;
  }
  50% {
    width: 60%;
    margin-left: 20%;
  }
  100% {
    width: 20%;
    margin-left: 80%;
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
