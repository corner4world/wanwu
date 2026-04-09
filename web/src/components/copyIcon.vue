<template>
  <!-- 纯图标模式（无边框） -->
  <i
    v-if="type === 'icon'"
    :class="copied ? 'el-icon-check' : 'el-icon-document-copy'"
    class="copy-icon-only"
    style="cursor: pointer"
    @click="handleCopy"
  ></i>
  <!-- 图标按钮模式（带边框） -->
  <button
    v-else-if="type === 'button'"
    class="copy-icon-btn"
    :class="{ 'icon-copied': copied }"
    :title="title"
    @click.stop="handleCopy"
  >
    <i :class="copied ? 'el-icon-check' : 'el-icon-document-copy'"></i>
  </button>
  <!-- 默认按钮模式 -->
  <el-button v-else v-bind="$attrs" @click="handleCopy" class="copy-icon">
    <i v-if="showIcon" class="el-icon-document-copy"></i>
    {{ $t('common.button.copy') }}
  </el-button>
</template>

<script>
export default {
  name: 'CopyIcon',
  inheritAttrs: false,
  props: {
    // 需要复制的文本内容
    text: {
      type: String,
      required: true,
    },
    // 显示类型：'default' 默认按钮 | 'icon' 纯图标 | 'button' 图标按钮
    type: {
      type: String,
      default: 'default',
      validator: value => ['default', 'icon', 'button'].includes(value),
    },
    // 是否显示图标（仅在 type='default' 时有效）
    showIcon: {
      type: Boolean,
      default: true,
    },
    // 按钮标题提示（仅在 type='button' 时有效）
    title: {
      type: String,
      default: '复制',
    },
  },
  data() {
    return {
      copied: false,
    };
  },
  methods: {
    handleCopy() {
      const res = this.$copy(this.text);
      if (res) {
        this.copied = true;
        this.$message.success(this.$t('common.copy.success'));
        setTimeout(() => {
          this.copied = false;
        }, 2000);
      } else {
        this.$message.error(this.$t('common.copy.error'));
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.copy-icon-only {
  transition: color 0.2s ease;

  &.el-icon-check {
    color: #10a37f;
  }
}

.copy-icon-btn {
  padding: 7px 14px;
  background: #fff;
  border: 1px solid #d1d5db;
  border-radius: 8px;
  color: #6b7280;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s ease;
  display: flex;
  align-items: center;
  gap: 6px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);

  &:hover {
    background: #f9fafb;
    border-color: #9ca3af;
    color: #111827;
    transform: translateY(-1px);
    box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
  }

  i {
    font-size: 14px;
  }

  &.icon-copied {
    i.el-icon-check {
      color: #10a37f;
    }
  }
}

.icon-copied {
  color: #10a37f;
}
</style>
