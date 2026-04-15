<!--
  OverflowTags 组件
 自动根据容器宽度截断标签，溢出部分显示在 Tooltip 中。
  -->
<template>
  <div class="overflow-tags-container" ref="container">
    <label class="tag-item" v-for="tag in visibleTags" :key="tag">
      {{ tag }}
    </label>
    <el-tooltip
      effect="light"
      placement="bottom"
      v-if="hiddenTags.length"
      popper-class="custom-tooltip"
    >
      <div slot="content" class="overflow-tags-tooltip">
        <label class="tag-item" v-for="tag in hiddenTags" :key="tag">
          {{ tag }}
        </label>
      </div>
      <label class="tag-item ellipsis">...</label>
    </el-tooltip>
  </div>
</template>

<script>
export default {
  name: 'OverflowTags',
  props: {
    tags: {
      type: Array,
      default: () => [],
    },
    minWidth: {
      type: Number,
      default: 170.5,
    },
    fontSize: {
      type: String,
      default: '11px',
    },
  },
  data() {
    return {
      visibleTags: [],
      hiddenTags: [],
      resizeObserver: null,
      canvas: null,
    };
  },
  watch: {
    tags: {
      handler() {
        this.$nextTick(() => {
          this.calculateLayout();
        });
      },
      immediate: true,
    },
  },
  mounted() {
    this.initResizeObserver();
  },
  beforeDestroy() {
    if (this.resizeObserver) {
      this.resizeObserver.disconnect();
    }
  },
  methods: {
    initResizeObserver() {
      this.resizeObserver = new ResizeObserver(() => {
        this.calculateLayout();
      });
      if (this.$refs.container) {
        this.resizeObserver.observe(this.$refs.container);
      }
    },
    calculateLayout() {
      if (!this.$refs.container || !this.tags.length) {
        this.visibleTags = this.tags;
        this.hiddenTags = [];
        return;
      }

      const containerWidth = Math.max(
        this.$refs.container.clientWidth,
        this.minWidth,
      );
      const GAP = 5; // 间距
      const ELLIPSIS_WIDTH = 20; // "..." 宽度预留
      const PADDING = 14; // label padding 左右各 7px

      let currentWidth = 0;
      const visible = [];
      const hidden = [];

      for (let i = 0; i < this.tags.length; i++) {
        const tag = this.tags[i];
        const tagWidth = this.measureTextWidth(tag, this.fontSize) + PADDING;

        const isLast = i === this.tags.length - 1;
        const limit = isLast ? containerWidth : containerWidth - ELLIPSIS_WIDTH;

        if (currentWidth + tagWidth <= limit) {
          visible.push(tag);
          currentWidth += tagWidth + GAP;
        } else {
          hidden.push(...this.tags.slice(i));
          break;
        }
      }

      this.visibleTags = visible;
      this.hiddenTags = hidden;
    },
    measureTextWidth(text, fontSize) {
      if (!this.canvas) {
        this.canvas = document.createElement('canvas');
      }
      const context = this.canvas.getContext('2d');
      // 这里的字体定义应与 CSS 保持一致
      context.font = `${fontSize} PingFang SC, Microsoft YaHei, Arial, sans-serif`;
      return context.measureText(text).width;
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/customTooltip.scss';

.overflow-tags-container {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  width: 100%;
  height: 22px;
  overflow: hidden;

  .tag-item {
    font-size: 11px;
    padding: 3px 7px;
    border-radius: 3px;
    margin-right: 5px;
    white-space: nowrap;
    display: inline-block;
    height: 22px;
    box-sizing: border-box;
    color: $tag_color;
    background: $tag_bg;

    &.ellipsis {
      cursor: pointer;
      padding: 0 4px;
      margin-right: 0;
    }
  }
}
</style>

<style lang="scss">
.overflow-tags-tooltip {
  display: flex;
  max-width: 300px;
  gap: 5px;
  padding: 5px 0;

  .tag-item {
    display: inline-block;
    font-size: 11px;
    padding: 3px 7px;
    border-radius: 3px;
    white-space: nowrap;
    color: $tag_color;
    background: $tag_bg;
  }
}
</style>
