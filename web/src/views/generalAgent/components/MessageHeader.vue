<template>
  <div class="message-header">
    <div :class="['avatar', role]">
      <img v-if="computedAvatarUrl" :src="computedAvatarUrl" :alt="roleLabel" />
      <i v-else :class="avatarIcon"></i>
    </div>
    <div class="header-info">
      <span class="role-label">{{ roleLabel }}</span>
      <span v-if="timestamp" class="timestamp">{{ formattedTime }}</span>
    </div>
    <div v-if="isStreaming" class="streaming-badge">
      <span class="pulse"></span>
      <span>生成中</span>
    </div>
  </div>
</template>

<script>
import { mapGetters } from 'vuex';
import { avatarSrc } from '@/utils/util';

const defaultAssistantAvatar = require('@/assets/imgs/robot-icon.png');

export default {
  name: 'MessageHeader',
  props: {
    role: {
      type: String,
      required: true,
      validator: val =>
        ['user', 'assistant', 'tool', 'system', 'reasoning'].includes(val),
    },
    timestamp: {
      type: [String, Number, Date],
      default: null,
    },
    isStreaming: {
      type: Boolean,
      default: false,
    },
    avatarUrl: {
      type: String,
      default: '',
    },
  },
  computed: {
    ...mapGetters('user', ['userAvatar']),
    roleLabel() {
      const labels = {
        user: 'You',
        assistant: 'Assistant',
        tool: 'Tool',
        system: 'System',
        reasoning: 'Thinking',
      };
      return labels[this.role] || this.role;
    },
    avatarIcon() {
      const icons = {
        user: 'el-icon-user',
        assistant: 'el-icon-cpu',
        tool: 'el-icon-setting',
        system: 'el-icon-info',
        reasoning: 'el-icon-cpu',
      };
      return icons[this.role] || 'el-icon-chat-dot-round';
    },
    // 助手消息使用用户头像
    computedAvatarUrl() {
      if (this.role === 'assistant') {
        if (this.userAvatar) {
          return avatarSrc(this.userAvatar);
        }
        return defaultAssistantAvatar;
      }
      return this.avatarUrl;
    },
    formattedTime() {
      if (!this.timestamp) return '';
      const date = new Date(this.timestamp);
      const now = new Date();
      const isToday = date.toDateString() === now.toDateString();

      const hours = date.getHours().toString().padStart(2, '0');
      const minutes = date.getMinutes().toString().padStart(2, '0');

      if (isToday) {
        return `${hours}:${minutes}`;
      }

      const month = (date.getMonth() + 1).toString().padStart(2, '0');
      const day = date.getDate().toString().padStart(2, '0');
      return `${month}/${day} ${hours}:${minutes}`;
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
$text-primary: #1f2937;
$text-secondary: #4b5563;
$text-muted: #6b7280;
$accent-color: #10a37f;

.message-header {
  display: flex;
  align-items: center;
  margin-bottom: 14px;
  font-family: $font-sans;

  .avatar {
    width: 38px;
    height: 38px;
    border-radius: 10px;
    display: flex;
    align-items: center;
    justify-content: center;
    margin-right: 14px;
    font-size: 16px;
    flex-shrink: 0;
    overflow: hidden;
    box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
    transition:
      transform 0.2s ease,
      box-shadow 0.2s ease;

    &:hover {
      transform: scale(1.05);
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12);
    }

    img {
      width: 100%;
      height: 100%;
      border-radius: 10px;
      object-fit: cover;
    }

    &.user {
      background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
      color: #fff;
    }

    &.assistant {
      background: transparent;
      color: #fff;
      box-shadow: none;

      img {
        border: 2px solid #e5e7eb;
        background: #fff;
      }
    }

    &.tool {
      background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
      color: #fff;
    }

    &.system {
      background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
      color: #fff;
    }

    &.reasoning {
      background: linear-gradient(135deg, #a18cd1 0%, #fbc2eb 100%);
      color: #fff;
    }
  }

  .header-info {
    display: flex;
    align-items: center;
    gap: 10px;
    flex: 1;
    min-width: 0;

    .role-label {
      font-weight: 600;
      font-size: 15px;
      color: $text-primary;
      letter-spacing: 0.01em;
    }

    .timestamp {
      font-size: 13px;
      color: $text-muted;
      font-variant-numeric: tabular-nums;
    }
  }

  .streaming-badge {
    display: flex;
    align-items: center;
    gap: 7px;
    padding: 5px 12px;
    background: linear-gradient(
      135deg,
      rgba(16, 163, 127, 0.1) 0%,
      rgba(16, 163, 127, 0.05) 100%
    );
    border-radius: 14px;
    border: 1px solid rgba(16, 163, 127, 0.15);
    font-size: 13px;
    color: $accent-color;
    font-weight: 500;

    .pulse {
      width: 6px;
      height: 6px;
      background: linear-gradient(135deg, $accent-color 0%, #0d8a6a 100%);
      border-radius: 50%;
      animation: pulse 1.5s infinite;
      box-shadow: 0 0 6px rgba(16, 163, 127, 0.5);
    }
  }
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
    transform: scale(1);
  }
  50% {
    opacity: 0.5;
    transform: scale(0.85);
  }
}
</style>
