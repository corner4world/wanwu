<template>
  <div class="md-outline" v-if="outlineList && outlineList.length">
    <!--<div class="outline-title">{{ $t('common.catalogue') }}</div>-->
    <ul class="outline-list">
      <li
        v-for="(item, index) in outlineList"
        :key="index"
        :class="[
          'outline-item',
          `outline-level-${item.level}`,
          { active: activeId === item.id },
        ]"
        :style="{ paddingLeft: `${(item.level - 1) * 12 + 12}px` }"
        @click="scrollToHeading(item)"
      >
        <span class="outline-text">{{ item.text }}</span>
      </li>
    </ul>
  </div>
</template>

<script>
import { marked } from 'marked';

export default {
  name: 'MdOutline',
  props: {
    content: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      outlineList: [],
      activeId: '',
      isScrolling: false,
      scrollTimer: null,
    };
  },
  watch: {
    content: {
      handler(val) {
        this.activeId = '';
        if (val) {
          this.parseOutline(val);
        }
      },
      immediate: true,
    },
  },
  mounted() {
    window.addEventListener('scroll', this.handleScroll);
  },
  beforeDestroy() {
    window.removeEventListener('scroll', this.handleScroll);
    if (this.scrollTimer) {
      clearTimeout(this.scrollTimer);
    }
  },
  methods: {
    // 去除 Markdown 内联标记（**, *, `, _ 等），用于目录文本展示
    stripMarkdown(text) {
      return text
        .replace(/\*\*(.*?)\*\*/g, '$1') // **加粗**
        .replace(/\*(.*?)\*/g, '$1') // *斜体*
        .replace(/__(.*?)__/g, '$1') // __下划线__
        .replace(/_(.*?)_/g, '$1') // _斜体_
        .replace(/`([^`]*)`/g, '$1') // `行内代码`
        .replace(/~~(.*?)~~/g, '$1') // ~~删除线~~
        .replace(/!\[.*?\]\(.*?\)/g, '') // 图片 ![alt](url)
        .replace(/\[([^\]]*)\]\([^)]*\)/g, '$1'); // 链接 [text](url)
    },
    parseOutline(markdown) {
      const list = [];
      let index = 0;

      // 使用 marked.lexer 正确解析 markdown，自动排除代码块中的内容
      const tokens = marked.lexer(markdown);
      tokens.forEach(token => {
        if (token.type === 'heading') {
          list.push({
            level: token.depth,
            text: this.stripMarkdown(token.text),
            id: `heading-${index}`,
          });
          index++;
        }
      });

      this.outlineList = list;
      // 默认选中第一个目录项
      if (list.length > 0) {
        this.activeId = list[0].id;
      }
    },
    scrollToHeading(item) {
      // 通过 id 直接定位，不依赖索引匹配，避免标题文本重复或代码块导致索引错位
      const el = document.getElementById(item.id);
      if (el) {
        el.scrollIntoView({
          behavior: 'smooth',
          block: 'start',
        });
        this.activeId = item.id;
      }
    },
    handleScroll() {
      if (this.scrollTimer) {
        clearTimeout(this.scrollTimer);
      }
      this.isScrolling = true;
      this.scrollTimer = setTimeout(() => {
        this.isScrolling = false;
        this.updateActiveHeading();
      }, 50);
    },
    updateActiveHeading() {
      const markdownEl = document.querySelector(
        '.docs-page-content .mark__content',
      );
      if (!markdownEl || !this.outlineList.length) return;

      const headings = markdownEl.querySelectorAll('h1, h2, h3, h4, h5, h6');
      if (!headings.length) return;

      let currentId = '';

      headings.forEach(heading => {
        const rect = heading.getBoundingClientRect();
        if (rect.top <= 100) {
          currentId = heading.id;
        }
      });

      // 通过 id 匹配，不依赖索引
      if (currentId && this.outlineList.find(item => item.id === currentId)) {
        this.activeId = currentId;
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.md-outline {
  position: sticky;
  top: 10px;
  width: 220px;
  max-height: calc(100vh - 170px);
  overflow-y: auto;
  padding: 16px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e8e8e8;

  .outline-title {
    font-size: 14px;
    font-weight: 600;
    color: #333;
    margin-bottom: 12px;
    padding-bottom: 8px;
    border-bottom: 1px solid #eee;
    text-align: center;
  }

  .outline-list {
    list-style: none;
    padding: 0;
    margin: 0;

    .outline-item {
      padding: 6px 12px;
      cursor: pointer;
      font-size: 13px;
      color: #666;
      border-radius: 4px;
      transition: all 0.2s;
      white-space: nowrap;
      overflow: hidden;
      text-overflow: ellipsis;
      margin-bottom: 2px;

      &:hover {
        background: #f5f7fa;
      }

      &.active {
        color: $color !important;
        background: $color_opacity;
        font-weight: 500;
      }

      &.outline-level-1 {
        font-weight: 600;
        font-size: 14px;
      }

      &.outline-level-2 {
        font-size: 13px;
      }

      &.outline-level-3,
      &.outline-level-4,
      &.outline-level-5,
      &.outline-level-6 {
        font-size: 12px;
        color: #888;
      }

      .outline-text {
        display: block;
      }
    }
  }

  &::-webkit-scrollbar {
    width: 4px;
  }

  &::-webkit-scrollbar-thumb {
    background: #ddd;
    border-radius: 2px;
  }
}
</style>
