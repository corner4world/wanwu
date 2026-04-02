<template>
  <div :class="['tool-call-block', statusClass]">
    <div class="tool-header" @click="toggleExpand">
      <div class="header-left">
        <i
          :class="isExpanded ? 'el-icon-arrow-down' : 'el-icon-arrow-right'"
        ></i>
        <div class="tool-icon-wrapper">
          <!-- 运行中状态 -->
          <div v-if="toolCall.status === 'running'" class="icon-running">
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
          <!-- 完成状态 -->
          <svg
            v-else-if="toolCall.status === 'completed'"
            class="tool-icon"
            viewBox="0 0 24 24"
            width="18"
            height="18"
          >
            <path
              fill="currentColor"
              d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"
            />
          </svg>
          <!-- 错误状态 -->
          <svg
            v-else-if="toolCall.status === 'error'"
            class="tool-icon"
            viewBox="0 0 24 24"
            width="18"
            height="18"
          >
            <path
              fill="currentColor"
              d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-2h2v2zm0-4h-2V7h2v6z"
            />
          </svg>
          <!-- 默认状态 -->
          <svg
            v-else
            class="tool-icon"
            viewBox="0 0 24 24"
            width="18"
            height="18"
          >
            <path
              fill="currentColor"
              d="M22.7 19l-9.1-9.1c.9-2.3.4-5-1.5-6.9-2-2-5-2.4-7.4-1.3L9 6 6 9 1.6 4.7C.4 7.1.9 10.1 2.9 12.1c1.9 1.9 4.6 2.4 6.9 1.5l9.1 9.1c.4.4 1 .4 1.4 0l2.3-2.3c.5-.4.5-1.1.1-1.4z"
            />
          </svg>
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
            <span>参数</span>
          </div>
          <div class="tool-arguments">
            <pre><code>{{ formattedArgs }}</code></pre>
            <button class="copy-btn" @click.stop="copyArgs" title="复制">
              <i
                :class="argsCopied ? 'el-icon-check' : 'el-icon-document-copy'"
              ></i>
            </button>
          </div>
        </div>
        <!-- 结果展示 -->
        <div v-if="result" class="tool-section">
          <div class="section-label">
            <i class="el-icon-document"></i>
            <span>结果</span>
            <span v-if="resultLength" class="result-length">
              {{ resultLength }}
            </span>
          </div>
          <div class="tool-result">
            <pre><code>{{ formattedResult }}</code></pre>
            <button class="copy-btn" @click.stop="copyResult" title="复制">
              <i
                :class="
                  resultCopied ? 'el-icon-check' : 'el-icon-document-copy'
                "
              ></i>
            </button>
          </div>
        </div>
        <!-- 运行中进度指示 -->
        <div v-if="toolCall.status === 'running'" class="tool-progress">
          <div class="progress-bar">
            <div class="progress-fill"></div>
          </div>
          <span class="progress-text">执行中...</span>
        </div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script>
export default {
  name: 'ToolCallBlock',
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
      argsCopied: false,
      resultCopied: false,
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
        running: '执行中',
        completed: '完成',
        error: '失败',
      };
      return texts[status] || '待执行';
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
      return `${len} 字符`;
    },
  },
  watch: {
    'toolCall.status': {
      immediate: true,
      handler(newStatus) {
        // 运行中时自动展开（用户可能已手动折叠，运行时强制展开）
        if (newStatus === 'running') {
          this.isExpanded = true;
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
    async copyArgs() {
      try {
        await navigator.clipboard.writeText(this.formattedArgs);
        this.argsCopied = true;
        setTimeout(() => {
          this.argsCopied = false;
        }, 2000);
      } catch (err) {
        console.error('Copy failed:', err);
      }
    },
    async copyResult() {
      try {
        await navigator.clipboard.writeText(this.formattedResult);
        this.resultCopied = true;
        setTimeout(() => {
          this.resultCopied = false;
        }, 2000);
      } catch (err) {
        console.error('Copy failed:', err);
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
$accent-color: #10a37f;
$accent-dark: #0d8a6a;
$orange-primary: #f97316;
$orange-dark: #c2410c;
$red-primary: #ef4444;
$red-dark: #dc2626;

.tool-call-block {
  margin-bottom: 14px;
  border-radius: 14px;
  border: 1px solid #e8ecf0;
  overflow: hidden;
  transition: all 0.3s ease;
  background: #fff;
  font-family: $font-sans;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

  &.status-running {
    border-color: $orange-primary;
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

    .icon-running {
      color: $orange-primary;
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
    border-color: $accent-color;
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
    border-color: $red-primary;
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
        background: linear-gradient(
          135deg,
          rgba(16, 163, 127, 0.1) 0%,
          rgba(16, 163, 127, 0.05) 100%
        );
        border-radius: 8px;
      }

      .tool-icon {
        flex-shrink: 0;
        color: $accent-color;
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
