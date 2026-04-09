<template>
  <div class="stream-markdown-container" ref="markdownRef">
    <div class="markdown-body" v-html="renderedContent"></div>
  </div>
</template>

<script>
import { md } from '../utils/markdown';

export default {
  name: 'StreamMarkdown',
  props: {
    content: {
      type: String,
      default: '',
    },
    isStreaming: {
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      stableContent: '',
      activeContent: '',
      blockStates: {
        inCodeBlock: false,
        inLatexBlock: false,
      },
      copyClickHandlers: [],
    };
  },
  computed: {
    renderedContent() {
      if (!this.content) return '';

      if (this.isStreaming) {
        return this.renderStreaming();
      }

      return md.render(this.content);
    },
  },
  watch: {
    content(newVal, oldVal) {
      if (this.isStreaming) {
        // 只在首次或内容重置时才初始化 stableContent
        if (!oldVal && newVal) {
          this.stableContent = newVal;
        } else if (
          newVal !== oldVal &&
          newVal.length >= (oldVal?.length || 0)
        ) {
          // 正常增量更新
          this.processIncremental(newVal, oldVal);
        }
        // 如果 newVal.length < oldVal.length，忽略这次更新（可能是中间状态）
      }
      this.$nextTick(() => this.bindCopyButtons());
    },
    isStreaming(val) {
      if (!val) {
        // 流式结束时，将所有 activeContent 刷新到 stableContent
        if (this.activeContent) {
          this.stableContent += this.activeContent;
          this.activeContent = '';
        }
        this.blockStates = {
          inCodeBlock: false,
          inLatexBlock: false,
        };
        this.$nextTick(() => this.bindCopyButtons());
      } else {
        // 开始流式时，初始化 stableContent
        if (!this.stableContent && this.content) {
          this.stableContent = this.content;
        }
      }
    },
  },
  mounted() {
    this.bindCopyButtons();
  },
  beforeDestroy() {
    this.unbindCopyButtons();
  },
  methods: {
    processIncremental(newContent, oldContent) {
      const incremental = newContent.slice(oldContent.length);
      this.activeContent += incremental;

      const lines = this.activeContent.split('\n');
      if (lines.length > 1) {
        let safeFlushText = '';
        let currentScanText = '';
        let scanStates = { ...this.blockStates };

        for (let i = 0; i < lines.length - 1; i++) {
          const line = lines[i];
          const lineWithNewline = line + '\n';

          const codeMatches = line.match(/```/g);
          if (codeMatches && codeMatches.length % 2 !== 0) {
            scanStates.inCodeBlock = !scanStates.inCodeBlock;
          }

          const latexMatches = line.match(/\$\$/g);
          if (latexMatches && latexMatches.length % 2 !== 0) {
            scanStates.inLatexBlock = !scanStates.inLatexBlock;
          }

          if (line.includes('\\[') && !line.includes('\\]')) {
            scanStates.inLatexBlock = true;
          }
          if (line.includes('\\]')) {
            scanStates.inLatexBlock = false;
          }
          if (line.includes('\\(') && !line.includes('\\)')) {
            scanStates.inLatexBlock = true;
          }
          if (line.includes('\\)')) {
            scanStates.inLatexBlock = false;
          }

          currentScanText += lineWithNewline;

          const isSafe = !Object.values(scanStates).some(state => state);
          if (isSafe) {
            safeFlushText = currentScanText;
            this.blockStates = { ...scanStates };
          }
        }

        if (safeFlushText) {
          this.stableContent += safeFlushText;
          this.activeContent = this.activeContent.slice(safeFlushText.length);
        }
      }
    },

    renderStreaming() {
      let result = '';

      if (this.stableContent) {
        result += md.render(this.stableContent);
      }

      if (this.activeContent) {
        let activeText = this.activeContent;

        const codeTicks = activeText.match(/```/g);
        if (codeTicks && codeTicks.length % 2 !== 0) {
          activeText += '\n```';
        }

        result += md.render(activeText);
      }

      return result;
    },

    bindCopyButtons() {
      if (!this.$refs.markdownRef) return;

      this.unbindCopyButtons();

      const copyButtons =
        this.$refs.markdownRef.querySelectorAll('.code-copy-btn');
      copyButtons.forEach(btn => {
        const handler = e => {
          e.preventDefault();
          e.stopPropagation();
          const codeBlock = e.target.closest('pre.code-block');
          const lines = codeBlock?.querySelectorAll('.code-line-content');

          let text = '';
          if (lines && lines.length > 0) {
            lines.forEach((line, i) => {
              text += line.textContent + (i < lines.length - 1 ? '\n' : '');
            });
          }

          if (text) {
            navigator.clipboard
              .writeText(text)
              .then(() => {
                const originalText = e.target.innerText;
                e.target.innerText = '已复制';
                setTimeout(() => {
                  e.target.innerText = originalText;
                }, 1500);
              })
              .catch(() => {
                this.$message?.error('复制失败');
              });
          }
        };
        btn.addEventListener('click', handler);
        this.copyClickHandlers.push({ btn, handler });
      });
    },

    unbindCopyButtons() {
      this.copyClickHandlers.forEach(({ btn, handler }) => {
        btn.removeEventListener('click', handler);
      });
      this.copyClickHandlers = [];
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/assets/showDocs/showdoc.scss';
@import '../styles/_variables.scss';

.stream-markdown-container {
  font-family: $font-sans;
}

.markdown-body {
  font-size: 16px;
  line-height: 1.85;
  color: $text-primary;
  word-wrap: break-word;
  font-family: $font-sans;

  ::v-deep {
    // 标题
    h1,
    h2,
    h3,
    h4,
    h5,
    h6 {
      margin: 24px 0 12px;
      font-weight: 600;
      line-height: 1.35;
      color: $text-primary;
      font-family: $font-sans;

      &:first-child {
        margin-top: 0;
      }
    }

    h1 {
      font-size: 1.6em;
    }

    h2 {
      font-size: 1.4em;
    }

    h3 {
      font-size: 1.2em;
    }

    h4 {
      font-size: 1.1em;
    }

    // 段落
    p {
      margin: 0 0 16px;
      line-height: 1.85;
      font-size: 16px;

      &:last-child {
        margin-bottom: 0;
      }
    }

    // 列表
    ul,
    ol {
      margin: 0 0 16px;
      padding-left: 24px;
      font-size: 16px;

      li {
        margin: 6px 0;
        line-height: 1.75;
        font-size: 16px;
      }
    }

    // 引用
    blockquote {
      margin: 16px 0;
      padding: 12px 16px;
      border-left: 4px solid #8b5cf6;
      background: linear-gradient(135deg, #faf5ff 0%, #f3e8ff 100%);
      color: $text-secondary;
      border-radius: 0 8px 8px 0;

      p:last-child {
        margin-bottom: 0;
      }
    }

    // 行内代码
    code:not([class*='hljs']):not(.line-li) {
      padding: 3px 7px;
      background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%);
      border: 1px solid #d1d5db;
      border-radius: 5px;
      font-family: $font-mono;
      font-size: 0.88em;
      color: #be185d;
    }

    // ============================================================
    // Mac Shell 风格代码块
    // 需要覆盖 showdoc.scss 中的以下冲突样式：
    // .markdown-body pre { padding: 16px; background: #f7f7f7; }
    // .markdown-body pre code { display: inline; line-height: inherit; }
    // .markdown-body code { border-radius: 3px; background: rgba(0,0,0,0.04); }
    // .markdown-body code:before/after { content: '\A0'; }
    // ============================================================
    pre.code-block {
      margin: 16px 0;
      padding: 0;
      border-radius: 10px;
      overflow: hidden;
      background: #0d0d0d;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.25);
      font-family: $font-mono;
      font-size: 14px;
      color: #c9d1d9;
      border: none;

      // 覆盖 showdoc.scss: .markdown-body pre { padding: 16px; background: #f7f7f7; }
      padding: 0 !important;
      background: #0d0d0d !important;

      code {
        // 覆盖 showdoc.scss: .markdown-body pre code { display: inline; }
        display: block !important;
        // 覆盖 showdoc.scss: .markdown-body code { border-radius: 3px; }
        border-radius: 0 !important;
        // 覆盖 showdoc.scss: .markdown-body code { background: rgba(0,0,0,0.04); }
        background: transparent !important;
        // 覆盖 showdoc.scss: .markdown-body pre code { line-height: inherit; }
        line-height: 1 !important;
        // 其他重置
        padding: 0;
        margin: 0;
        font-family: inherit;
        font-size: inherit;
        color: inherit;
        border: none;

        // 覆盖 showdoc.scss: .markdown-body code:before/after { content: '\A0'; }
        &::before,
        &::after {
          content: none;
        }
      }
    }

    // 代码块头部（红黄绿点 + 语言标签 + 复制按钮）
    .code-header {
      display: flex;
      align-items: center;
      padding: 10px 14px;
      background: #1a1a1a;
      border-bottom: 1px solid #333;
      gap: 8px;
    }

    .code-dots {
      width: 36px;
      height: 12px;
      flex-shrink: 0;

      &::before {
        content: '';
        display: block;
        width: 10px;
        height: 10px;
        border-radius: 50%;
        background: #ff5f57;
        box-shadow:
          14px 0 0 #febc2e,
          28px 0 0 #28c840;
      }
    }

    .code-lang {
      margin-left: auto;
      font-size: 12px;
      color: #888;
      font-family: $font-mono;
      text-transform: lowercase;
      padding: 2px 8px;
      background: rgba(255, 255, 255, 0.08);
      border-radius: 4px;
    }

    .code-copy-btn {
      padding: 4px 10px;
      background: rgba(255, 255, 255, 0.1);
      border: 1px solid rgba(255, 255, 255, 0.15);
      border-radius: 5px;
      color: #888;
      font-size: 12px;
      font-family: $font-sans;
      cursor: pointer;
      transition: all 0.2s ease;
      flex-shrink: 0;

      &:hover {
        background: rgba(255, 255, 255, 0.15);
        color: #bbb;
        border-color: rgba(255, 255, 255, 0.25);
      }
    }

    // 代码内容区域
    .code-content {
      display: block;
      padding: 16px 0;
      overflow-x: auto;
      overflow-y: auto;
      background: #0d0d0d;
    }

    // 代码行号和内容
    .code-lines {
      margin: 0;
      padding: 0 20px 0 56px;
      list-style: none;
      counter-reset: line-counter;
      font-family: $font-mono;
      font-size: 14px;
      line-height: 1.65;
      color: #c9d1d9;
      background: transparent;
      white-space: pre;

      .code-line {
        display: block;
        margin: 0;
        padding: 0;
        min-height: 1.65em;
        counter-increment: line-counter;
        list-style: none;

        .code-line-num {
          display: inline-block;
          width: 36px;
          margin-left: -44px;
          margin-right: 8px;
          text-align: right;
          color: #484f58;
          font-size: 12px;
          font-family: $font-mono;
          user-select: none;
          pointer-events: none;
        }

        .code-line-content {
          display: inline;
        }
      }
    }

    // 链接
    a {
      color: #8b5cf6;
      text-decoration: none;
      font-weight: 500;
      transition: color 0.2s ease;

      &:hover {
        color: #7c3aed;
        text-decoration: underline;
      }
    }

    // 图片
    img {
      max-width: 100%;
      border-radius: 10px;
      margin: 14px 0;
      box-shadow: 0 2px 12px rgba(0, 0, 0, 0.1);
    }

    // 表格
    table {
      margin: 16px 0;
      border-collapse: collapse;
      width: 100%;
      font-size: 15px;
      border-radius: 10px;
      overflow: hidden;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.08);

      th,
      td {
        padding: 12px 16px;
        border: 1px solid #e5e7eb;
        text-align: left;
      }

      th {
        background: linear-gradient(180deg, #f9fafb 0%, #f3f4f6 100%);
        font-weight: 600;
        color: $text-primary;
      }

      td {
        background: #fff;
      }

      tr:nth-child(even) td {
        background: #f9fafb;
      }
    }

    // 水平线
    hr {
      margin: 24px 0;
      border: none;
      height: 1px;
      background: linear-gradient(
        90deg,
        transparent 0%,
        #e5e7eb 20%,
        #e5e7eb 80%,
        transparent 100%
      );
    }

    // 强调
    strong {
      font-weight: 600;
      color: $text-primary;
    }

    em {
      font-style: italic;
      color: $text-secondary;
    }

    // 删除线
    del {
      color: #9ca3af;
      text-decoration: line-through;
    }

    // 任务列表
    input[type='checkbox'] {
      margin-right: 8px;
      accent-color: #8b5cf6;
      width: 16px;
      height: 16px;
    }
  }
}
</style>
