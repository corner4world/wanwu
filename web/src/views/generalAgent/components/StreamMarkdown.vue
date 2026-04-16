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
            this.$copy(text).then(() => {
              const originalText = e.target.innerText;
              e.target.innerText = '已复制';
              setTimeout(() => {
                e.target.innerText = originalText;
              }, 1500);
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
@import '../styles/_markdown-common.scss';

.stream-markdown-container {
  font-family: $font-sans;
}

.markdown-body {
  ::v-deep {
    @include markdown-content-base;
    @include complete-code-block-stream;
  }
}
</style>
