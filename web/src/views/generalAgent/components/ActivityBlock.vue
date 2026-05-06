<template>
  <div class="activity-block">
    <div
      :class="['activity-header', { streaming: isStreaming }]"
      @click="toggleExpand"
    >
      <div class="header-left">
        <i :class="isExpanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'"></i>
        <div class="activity-icon-wrapper">
          <img
            :src="require('@/assets/imgs/intelligent.png')"
            class="activity-icon-img"
          />
        </div>
        <span class="activity-title">{{ activityTitle }}</span>
        <span v-if="fragmentCount > 0" class="fragment-count">
          {{ fragmentCount }} {{ $t('generalAgent.activityBlock.steps') }}
        </span>
      </div>
      <div class="header-right">
        <span v-if="isStreaming && formattedTimer" class="activity-time">
          {{ formattedTimer }}
        </span>
        <span v-else-if="!isStreaming && duration" class="activity-time">
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
      <div v-show="isExpanded" class="activity-body">
        <div class="activity-content">
          <slot></slot>
        </div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script>
export default {
  name: 'ActivityBlock',
  props: {
    activityType: {
      type: String,
      default: 'sub_agent',
    },
    activityName: {
      type: String,
      default: '',
    },
    fragments: {
      type: Array,
      default: () => [],
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
    activityTitle() {
      if (this.activityName) {
        return this.activityName;
      }
      return this.activityType === 'sub_agent'
        ? this.$t('generalAgent.message.subAgentExecution')
        : this.$t('generalAgent.activityBlock.defaultTitle');
    },
    fragmentCount() {
      if (!this.fragments) return 0;
      return this.fragments.filter(
        f => f.type === 'reasoning' || f.type === 'tool_call',
      ).length;
    },
    hasPendingQuestion() {
      if (!this.fragments) return false;
      return this.fragments.some(
        f => f.type === 'question' && f.status === 'pending',
      );
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
    isStreaming: {
      immediate: true,
      handler(val, oldVal) {
        if (val) {
          this.isExpanded = true;
          this.startTimer();
        } else {
          this.stopTimer();
          // 从 streaming 变为非 streaming 时，如果有 pending question 保持展开
          if (oldVal === true && !this.hasPendingQuestion) {
            this.isExpanded = false;
          }
        }
      },
    },
    hasPendingQuestion: {
      immediate: true,
      handler(val) {
        if (val) {
          this.isExpanded = true;
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

.activity-block {
  margin-bottom: 14px;
  font-family: $font-sans;
}

.activity-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-radius: 14px;
  border: 1px solid rgba(59, 130, 246, 0.2);
  background: linear-gradient(
    135deg,
    rgba(59, 130, 246, 0.06) 0%,
    #fafafa 100%
  );
  cursor: pointer;
  user-select: none;
  transition: all 0.3s ease;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

  &:hover {
    background: linear-gradient(180deg, #f5f7f9 0%, #f3f4f6 100%);
  }

  &.streaming {
    border-color: rgba(59, 130, 246, 0.4);
    box-shadow: 0 4px 16px rgba(59, 130, 246, 0.15);

    .activity-title {
      color: $accent-dark;
      font-weight: 600;
    }

    .activity-icon-img {
      animation: spin 1s linear infinite;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(59, 130, 246, 0.12) 0%,
        rgba(59, 130, 246, 0.08) 100%
      );
      border: 1px solid rgba(59, 130, 246, 0.2);

      .status-dot {
        background: $accent-color;
        animation: pulse 1.5s infinite;
        box-shadow: 0 0 6px rgba(59, 130, 246, 0.5);
      }

      .status-text {
        color: $accent-dark;
      }
    }
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

    .activity-icon-wrapper {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 24px;
      height: 24px;

      .activity-icon-img {
        width: 18px;
        height: 18px;
        object-fit: contain;
      }
    }

    .activity-title {
      font-size: 14px;
      font-weight: 600;
      color: $text-primary;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      letter-spacing: 0.01em;
    }

    .fragment-count {
      font-size: 12px;
      color: $text-muted;
      background: rgba(0, 0, 0, 0.04);
      padding: 3px 8px;
      border-radius: 8px;
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;

    .activity-time {
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
          rgba(99, 102, 241, 0.12) 0%,
          rgba(99, 102, 241, 0.08) 100%
        );
        border: 1px solid rgba(99, 102, 241, 0.2);

        .status-dot {
          background: $accent-color;
          animation: pulse 1.5s infinite;
          box-shadow: 0 0 6px rgba(99, 102, 241, 0.5);
        }

        .status-text {
          color: $accent-dark;
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

.activity-body {
  position: relative;
  padding-left: 20px;
  margin-top: 12px;

  &::before {
    content: '';
    position: absolute;
    left: 6px;
    top: 0;
    bottom: 0;
    width: 2px;
    background: linear-gradient(
      180deg,
      rgba(59, 130, 246, 0.3) 0%,
      rgba(59, 130, 246, 0.1) 100%
    );
    border-radius: 1px;
  }
}

.activity-content {
  padding-top: 4px;
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
</style>
