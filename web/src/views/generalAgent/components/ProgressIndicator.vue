<template>
  <div class="progress-indicator">
    <div class="progress-steps">
      <div
        v-for="(step, index) in steps"
        :key="step.key"
        :class="['progress-step', step.status]"
      >
        <div class="step-icon">
          <i v-if="step.status === 'completed'" class="el-icon-check"></i>
          <i v-else-if="step.status === 'running'" class="el-icon-loading"></i>
          <span v-else class="step-dot"></span>
        </div>
        <div class="step-content">
          <span class="step-label">{{ step.label }}</span>
          <span
            v-if="step.duration && step.status === 'completed'"
            class="step-duration"
          >
            {{ step.duration }}
          </span>
        </div>
        <div v-if="index < steps.length - 1" class="step-connector">
          <div
            :class="[
              'connector-line',
              {
                active:
                  step.status === 'completed' || step.status === 'running',
              },
            ]"
          ></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ProgressIndicator',
  props: {
    // 当前阶段: 'thinking' | 'tool_calling' | 'generating'
    currentStage: {
      type: String,
      default: '',
    },
    // 思考耗时
    thinkingDuration: {
      type: String,
      default: '',
    },
    // 工具调用耗时
    toolDuration: {
      type: String,
      default: '',
    },
    // 是否正在流式输出
    isStreaming: {
      type: Boolean,
      default: false,
    },
  },
  computed: {
    steps() {
      return [
        {
          key: 'understanding',
          label: '理解问题',
          status: this.getStepStatus('understanding'),
        },
        {
          key: 'thinking',
          label: '思考中',
          status: this.getStepStatus('thinking'),
          duration: this.thinkingDuration,
        },
        {
          key: 'tool_calling',
          label: '调用工具',
          status: this.getStepStatus('tool_calling'),
          duration: this.toolDuration,
        },
        {
          key: 'generating',
          label: '生成回答',
          status: this.getStepStatus('generating'),
        },
      ];
    },
  },
  methods: {
    getStepStatus(stepKey) {
      const stageOrder = [
        'understanding',
        'thinking',
        'tool_calling',
        'generating',
      ];
      const currentIndex = stageOrder.indexOf(this.currentStage);
      const stepIndex = stageOrder.indexOf(stepKey);

      if (!this.isStreaming) {
        return 'pending';
      }

      if (stepIndex < currentIndex) {
        return 'completed';
      } else if (stepIndex === currentIndex) {
        return 'running';
      } else {
        return 'pending';
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

// 颜色变量
$thinking-color: #8b5cf6;
$tool-color: #f97316;
$success-color: #10a37f;
$text-muted: #6b7280;

.progress-indicator {
  padding: 14px 20px;
  background: linear-gradient(135deg, #fafbfc 0%, #ffffff 100%);
  border-radius: 14px;
  margin-bottom: 18px;
  border: 1px solid #e8ecf0;
  font-family: $font-sans;
}

.progress-steps {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.progress-step {
  display: flex;
  align-items: center;
  flex: 1;

  .step-icon {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 10px;
    transition: all 0.3s ease;
    flex-shrink: 0;

    i {
      font-size: 13px;
    }

    .step-dot {
      width: 8px;
      height: 8px;
      border-radius: 50%;
      background: #d1d5db;
    }
  }

  .step-content {
    display: flex;
    flex-direction: column;
    min-width: 0;

    .step-label {
      font-size: 14px;
      color: $text-muted;
      font-weight: 500;
      white-space: nowrap;
      letter-spacing: 0.01em;
    }

    .step-duration {
      font-size: 12px;
      color: #9ca3af;
      margin-top: 2px;
      font-variant-numeric: tabular-nums;
    }
  }

  .step-connector {
    flex: 0 0 50px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin: 0 12px;

    .connector-line {
      width: 100%;
      height: 3px;
      background: #e5e7eb;
      border-radius: 2px;
      transition: background 0.3s ease;

      &.active {
        background: linear-gradient(
          90deg,
          $success-color 0%,
          $success-color 50%,
          #e5e7eb 50%
        );
        background-size: 200% 100%;
        animation: connectorPulse 1.5s ease-in-out infinite;
      }
    }
  }

  // 状态样式
  &.pending {
    .step-icon {
      background: #f9fafb;
      border: 2px solid #e5e7eb;
    }

    .step-label {
      color: #9ca3af;
    }
  }

  &.running {
    .step-icon {
      background: linear-gradient(
        135deg,
        rgba($thinking-color, 0.15) 0%,
        rgba($thinking-color, 0.05) 100%
      );
      border: 2px solid $thinking-color;
      box-shadow: 0 0 12px rgba($thinking-color, 0.3);

      i {
        color: $thinking-color;
        animation: spin 1s linear infinite;
      }
    }

    .step-label {
      color: $thinking-color;
      font-weight: 600;
    }
  }

  &.completed {
    .step-icon {
      background: linear-gradient(
        135deg,
        rgba($success-color, 0.15) 0%,
        rgba($success-color, 0.05) 100%
      );
      border: 2px solid $success-color;

      i {
        color: $success-color;
      }
    }

    .step-label {
      color: $success-color;
      font-weight: 500;
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

@keyframes connectorPulse {
  0% {
    background-position: 100% 0;
  }
  100% {
    background-position: 0 0;
  }
}
</style>
