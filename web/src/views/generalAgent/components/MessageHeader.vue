<template>
  <div
    class="message-header"
    :class="{ 'assistant-only': role === 'assistant' }"
  >
    <div :class="['avatar', role]">
      <img v-if="computedAvatarUrl" :src="computedAvatarUrl" :alt="roleLabel" />
      <i v-else :class="avatarIcon"></i>
    </div>
    <template v-if="role !== 'assistant'">
      <div class="header-info">
        <span class="role-label">{{ roleLabel }}</span>
        <span v-if="timestamp" class="timestamp">{{ formattedTime }}</span>
      </div>
      <div v-if="isStreaming" class="streaming-badge">
        <span class="pulse"></span>
        <span>{{ $t('app.generate.generating') }}</span>
      </div>
    </template>
  </div>
</template>

<script>
import { mapGetters } from 'vuex';
import { avatarSrc } from '@/utils/util';

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
    ...mapGetters('user', ['userAvatar', 'commonInfo']),
    roleLabel() {
      const labels = {
        user: this.$t('generalAgent.messageHeader.user'),
        assistant: this.$t('generalAgent.messageHeader.assistant'),
        tool: this.$t('generalAgent.messageHeader.tool'),
        system: this.$t('generalAgent.messageHeader.system'),
        reasoning: this.$t('generalAgent.messageHeader.reasoning'),
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
    platformLogo() {
      const tab = this.commonInfo?.data?.tab || {};
      return tab.logo?.path || null;
    },
    computedAvatarUrl() {
      if (this.role === 'user') {
        if (this.userAvatar) {
          return avatarSrc(this.userAvatar);
        }
        return null;
      }
      if (this.role === 'assistant') {
        if (this.platformLogo) {
          return avatarSrc(this.platformLogo);
        }
        return null;
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
@import '../styles/_variables.scss';

.message-header {
  display: flex;
  align-items: center;
  margin-bottom: 14px;
  font-family: $font-sans;

  &.assistant-only {
    margin-bottom: 0;
  }

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
      background: linear-gradient(135deg, #10a37f 0%, #0d8a6a 100%);
      color: #fff;

      img {
        border: 2px solid #fff;
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
