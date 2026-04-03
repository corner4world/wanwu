<template>
  <div class="markdown-renderer">
    <!-- 渲染的 HTML 内容 -->
    <div v-html="renderedContent"></div>

    <!-- PPT 预览弹窗 -->
    <el-dialog
      :visible.sync="pptDialogVisible"
      :title="pptDialogTitle"
      width="90%"
      top="5vh"
      custom-class="ppt-preview-dialog"
      @close="closePptDialog"
    >
      <ppt-preview v-if="pptDialogVisible" :src="currentPptUrl" />
    </el-dialog>
  </div>
</template>

<script>
import hljs from 'highlight.js/lib/core';
// 按需加载常用语言
import javascript from 'highlight.js/lib/languages/javascript';
import typescript from 'highlight.js/lib/languages/typescript';
import python from 'highlight.js/lib/languages/python';
import java from 'highlight.js/lib/languages/java';
import go from 'highlight.js/lib/languages/go';
import rust from 'highlight.js/lib/languages/rust';
import cpp from 'highlight.js/lib/languages/cpp';
import csharp from 'highlight.js/lib/languages/csharp';
import php from 'highlight.js/lib/languages/php';
import ruby from 'highlight.js/lib/languages/ruby';
import swift from 'highlight.js/lib/languages/swift';
import kotlin from 'highlight.js/lib/languages/kotlin';
import sql from 'highlight.js/lib/languages/sql';
import bash from 'highlight.js/lib/languages/bash';
import json from 'highlight.js/lib/languages/json';
import yaml from 'highlight.js/lib/languages/yaml';
import xml from 'highlight.js/lib/languages/xml';
import css from 'highlight.js/lib/languages/css';
import scss from 'highlight.js/lib/languages/scss';
import markdown from 'highlight.js/lib/languages/markdown';
import PptPreview from './PptPreview.vue';

// 注册语言
hljs.registerLanguage('javascript', javascript);
hljs.registerLanguage('typescript', typescript);
hljs.registerLanguage('python', python);
hljs.registerLanguage('java', java);
hljs.registerLanguage('go', go);
hljs.registerLanguage('rust', rust);
hljs.registerLanguage('cpp', cpp);
hljs.registerLanguage('csharp', csharp);
hljs.registerLanguage('php', php);
hljs.registerLanguage('ruby', ruby);
hljs.registerLanguage('swift', swift);
hljs.registerLanguage('kotlin', kotlin);
hljs.registerLanguage('sql', sql);
hljs.registerLanguage('bash', bash);
hljs.registerLanguage('json', json);
hljs.registerLanguage('yaml', yaml);
hljs.registerLanguage('xml', xml);
hljs.registerLanguage('html', xml);
hljs.registerLanguage('css', css);
hljs.registerLanguage('scss', scss);
hljs.registerLanguage('markdown', markdown);

// 语言别名映射
const languageAliases = {
  js: 'javascript',
  ts: 'typescript',
  py: 'python',
  sh: 'bash',
  shell: 'bash',
  yml: 'yaml',
};

// 生成唯一ID
let pptIdCounter = 0;
const generatePptId = () => `ppt-preview-${++pptIdCounter}`;

export default {
  name: 'MarkdownRenderer',
  components: {
    PptPreview,
  },
  props: {
    content: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      pptDialogVisible: false,
      currentPptUrl: '',
      pptDialogTitle: 'PPT 预览',
      pptLinks: [], // 存储 PPT 链接信息
    };
  },
  computed: {
    renderedContent() {
      if (!this.content) return '';
      return this.parseMarkdown(this.content);
    },
  },
  mounted() {
    // 绑定 PPT 预览点击事件
    this.bindPptClickEvents();
  },
  updated() {
    // 更新后重新绑定事件
    this.bindPptClickEvents();
  },
  methods: {
    // 绑定 PPT 预览点击事件
    bindPptClickEvents() {
      this.$nextTick(() => {
        const pptElements = this.$el.querySelectorAll('.ppt-preview-card');
        pptElements.forEach(el => {
          el.removeEventListener('click', this.handlePptClick);
          el.addEventListener('click', this.handlePptClick);
        });
      });
    },

    // 处理 PPT 点击
    handlePptClick(event) {
      const url = event.currentTarget.dataset.url;
      const title = event.currentTarget.dataset.title || 'PPT 预览';
      if (url) {
        this.currentPptUrl = url;
        this.pptDialogTitle = title;
        this.pptDialogVisible = true;
      }
    },

    // 关闭 PPT 弹窗
    closePptDialog() {
      this.pptDialogVisible = false;
      this.currentPptUrl = '';
    },

    // 生成 PPT 预览卡片 HTML
    generatePptCard(url, title = 'PPT 文档') {
      const id = generatePptId();
      this.pptLinks.push({ id, url, title });
      return `<div class="ppt-preview-card" data-url="${url}" data-title="${title}">
        <div class="ppt-card-icon">
          <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor">
            <path d="M19 3H5c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h14c1.1 0 2-.9 2-2V5c0-1.1-.9-2-2-2zm0 16H5V5h14v14zM7 10h2v7H7zm4-3h2v10h-2zm4 6h2v4h-2z"/>
          </svg>
        </div>
        <div class="ppt-card-info">
          <div class="ppt-card-title">${title}</div>
          <div class="ppt-card-hint">点击预览</div>
        </div>
      </div>`;
    },

    // 解码URL中的转义字符
    decodeUrl(url) {
      try {
        // 先解码 HTML 实体 (如 &amp; -> &)
        let decoded = url
          .replace(/&amp;/g, '&')
          .replace(/&lt;/g, '<')
          .replace(/&gt;/g, '>')
          .replace(/&quot;/g, '"')
          .replace(/&#39;/g, "'");

        // 再解码 \u0026 这类 Unicode 转义
        decoded = decoded.replace(/\\u([0-9a-fA-F]{4})/g, (match, hex) => {
          return String.fromCharCode(parseInt(hex, 16));
        });

        return decoded;
      } catch (e) {
        return url;
      }
    },

    parseMarkdown(text) {
      let html = text;

      // 处理代码块（必须先处理，避免内部内容被其他规则影响）
      html = this.parseCodeBlocks(html);

      // 处理标题
      html = html.replace(/^### (.+)$/gm, '<h3>$1</h3>');
      html = html.replace(/^## (.+)$/gm, '<h2>$1</h2>');
      html = html.replace(/^# (.+)$/gm, '<h1>$1</h1>');

      // 处理粗体和斜体
      html = html.replace(/\*\*\*(.+?)\*\*\*/g, '<strong><em>$1</em></strong>');
      html = html.replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>');
      html = html.replace(/\*(.+?)\*/g, '<em>$1</em>');
      html = html.replace(/___(.+?)___/g, '<strong><em>$1</em></strong>');
      html = html.replace(/__(.+?)__/g, '<strong>$1</strong>');
      html = html.replace(/_(.+?)_/g, '<em>$1</em>');

      // 处理行内代码（排除代码块内的）
      html = html.replace(
        /`([^`\n]+)`/g,
        '<code class="inline-code">$1</code>',
      );

      // 处理图片（markdown 图片语法 ![alt](url)）- 必须在链接之前处理
      html = html.replace(/!\[([^\]]*)\]\(([^)]+)\)/g, (match, alt, url) => {
        const decodedUrl = this.decodeUrl(url);
        return `<img src="${decodedUrl}" alt="${alt}" class="markdown-image" />`;
      });

      // 处理链接（根据URL类型渲染不同媒体）
      html = html.replace(/\[([^\]]+)\]\(([^)]+)\)/g, (match, text, url) => {
        const decodedUrl = this.decodeUrl(url);
        const lowerUrl = decodedUrl.toLowerCase();

        // 图片
        if (
          /\.(png|jpe?g|gif|webp|svg|bmp|ico|avif)(\?.*)?$/i.test(decodedUrl)
        ) {
          return `<img src="${decodedUrl}" alt="${text}" class="markdown-image" />`;
        }

        // 视频
        if (/\.(mp4|webm|ogg|mov|m4v)(\?.*)?$/i.test(decodedUrl)) {
          return `<video src="${decodedUrl}" controls class="markdown-video">${text}</video>`;
        }

        // 音频
        if (/\.(mp3|wav|ogg|m4a|flac|aac)(\?.*)?$/i.test(decodedUrl)) {
          return `<audio src="${decodedUrl}" controls class="markdown-audio">${text}</audio>`;
        }

        // PDF
        if (/\.(pdf)(\?.*)?$/i.test(decodedUrl)) {
          return `<a href="${decodedUrl}" target="_blank" rel="noopener" class="markdown-pdf-link">📄 ${text}</a>`;
        }

        // PPT
        if (/\.(pptx?)(\?.*)?$/i.test(decodedUrl)) {
          return this.generatePptCard(decodedUrl, text);
        }

        // 普通链接
        return `<a href="${decodedUrl}" target="_blank" rel="noopener">${text}</a>`;
      });

      // 处理纯文本 URL（自动识别视频、音频、图片链接）
      // 只匹配不在 HTML 标签属性中的 URL
      html = html.replace(
        /(^|[^"'>])(https?:\/\/[^\s<>"']+)/g,
        (match, prefix, url) => {
          const decodedUrl = this.decodeUrl(url);
          const cleanUrl = decodedUrl.replace(/[.,;:!?]+$/, ''); // 移除末尾标点

          // 视频
          if (/\.(mp4|webm|ogg|mov|m4v)(\?.*)?$/i.test(cleanUrl)) {
            return `${prefix}<video src="${cleanUrl}" controls class="markdown-video"></video>`;
          }

          // 音频
          if (/\.(mp3|wav|ogg|m4a|flac|aac)(\?.*)?$/i.test(cleanUrl)) {
            return `${prefix}<audio src="${cleanUrl}" controls class="markdown-audio"></audio>`;
          }

          // 图片
          if (
            /\.(png|jpe?g|gif|webp|svg|bmp|ico|avif)(\?.*)?$/i.test(cleanUrl)
          ) {
            return `${prefix}<img src="${cleanUrl}" alt="" class="markdown-image" />`;
          }

          // PDF
          if (/\.(pdf)(\?.*)?$/i.test(cleanUrl)) {
            return `${prefix}<a href="${cleanUrl}" target="_blank" rel="noopener" class="markdown-pdf-link">📄 查看 PDF</a>`;
          }

          // PPT
          if (/\.(pptx?)(\?.*)?$/i.test(cleanUrl)) {
            return `${prefix}${this.generatePptCard(cleanUrl, 'PPT 文档')}`;
          }

          // 普通链接
          return `${prefix}<a href="${cleanUrl}" target="_blank" rel="noopener">${decodedUrl}</a>`;
        },
      );

      // 处理引用
      html = html.replace(/^&gt; (.+)$/gm, '<blockquote>$1</blockquote>');
      html = html.replace(/^> (.+)$/gm, '<blockquote>$1</blockquote>');

      // 处理无序列表
      html = html.replace(/^[\-\*] (.+)$/gm, '<li>$1</li>');
      html = html.replace(/(<li>.*<\/li>\n?)+/g, '<ul>$&</ul>');

      // 处理有序列表
      html = html.replace(/^\d+\. (.+)$/gm, '<li>$1</li>');

      // 处理水平线
      html = html.replace(/^---$/gm, '<hr />');
      html = html.replace(/^\*\*\*$/gm, '<hr />');

      // 处理段落（双换行）
      html = html.replace(/\n\n/g, '</p><p>');

      // 处理单换行
      html = html.replace(/\n/g, '<br>');

      // 包裹段落
      if (!html.startsWith('<')) {
        html = '<p>' + html + '</p>';
      }

      // 清理空段落
      html = html.replace(/<p><\/p>/g, '');
      html = html.replace(/<p><br><\/p>/g, '');
      html = html.replace(/<p>(<h[1-6]>)/g, '$1');
      html = html.replace(/(<\/h[1-6]>)<\/p>/g, '$1');
      html = html.replace(/<p>(<ul>)/g, '$1');
      html = html.replace(/(<\/ul>)<\/p>/g, '$1');
      html = html.replace(/<p>(<blockquote>)/g, '$1');
      html = html.replace(/(<\/blockquote>)<\/p>/g, '$1');
      html = html.replace(/<p>(<div class="code-block)/g, '$1');
      html = html.replace(/(<\/div>\s*<\/div>)<\/p>/g, '$1');

      // 清理块级元素后多余的br标签
      html = html.replace(/<\/li><br>/g, '</li>');
      html = html.replace(/<\/ul><br>/g, '</ul>');
      html = html.replace(/<\/ol><br>/g, '</ol>');
      html = html.replace(/<\/h1><br>/g, '</h1>');
      html = html.replace(/<\/h2><br>/g, '</h2>');
      html = html.replace(/<\/h3><br>/g, '</h3>');
      html = html.replace(/<\/h4><br>/g, '</h4>');
      html = html.replace(/<\/h5><br>/g, '</h5>');
      html = html.replace(/<\/h6><br>/g, '</h6>');
      html = html.replace(/<\/blockquote><br>/g, '</blockquote>');
      html = html.replace(/<\/pre><br>/g, '</pre>');
      html = html.replace(/<hr \/><br>/g, '<hr />');

      return html;
    },

    parseCodeBlocks(text) {
      // 处理围栏代码块
      return text.replace(/```(\w*)\n?([\s\S]*?)```/g, (match, lang, code) => {
        const language = this.normalizeLanguage(lang);
        const trimmedCode = code.replace(/\n$/, '');
        const highlighted = this.highlightCode(trimmedCode, language);
        const escapedCode = this.escapeHtml(trimmedCode);

        return `<div class="code-block-wrapper" data-language="${language || 'text'}">
          <div class="code-header">
            <span class="language-tag">${language || 'text'}</span>
            <button class="copy-btn" onclick="window.copyCodeBlock(this)">
              <svg viewBox="0 0 24 24" width="14" height="14" fill="currentColor">
                <path d="M16 1H4c-1.1 0-2 .9-2 2v14h2V3h12V1zm3 4H8c-1.1 0-2 .9-2 2v14c0 1.1.9 2 2 2h11c1.1 0 2-.9 2-2V7c0-1.1-.9-2-2-2zm0 16H8V7h11v14z"/>
              </svg>
              <span>Copy</span>
            </button>
          </div>
          <div class="code-content">
            <pre><code class="language-${language || 'text'}">${highlighted}</code></pre>
            <textarea class="code-source" style="display:none">${escapedCode}</textarea>
          </div>
        </div>`;
      });
    },

    normalizeLanguage(lang) {
      if (!lang) return '';
      const lower = lang.toLowerCase();
      return languageAliases[lower] || lower;
    },

    highlightCode(code, language) {
      try {
        if (language && hljs.getLanguage(language)) {
          return hljs.highlight(code, { language }).value;
        }
        return hljs.highlightAuto(code).value;
      } catch (e) {
        return this.escapeHtml(code);
      }
    },

    escapeHtml(text) {
      const div = document.createElement('div');
      div.textContent = text;
      return div.innerHTML;
    },
  },
};

// 全局复制函数
if (typeof window !== 'undefined') {
  window.copyCodeBlock = function (btn) {
    const wrapper = btn.closest('.code-block-wrapper');
    const textarea = wrapper.querySelector('.code-source');
    const code = textarea
      ? textarea.value
      : wrapper.querySelector('code').textContent;

    navigator.clipboard
      .writeText(code)
      .then(() => {
        const span = btn.querySelector('span');
        const originalText = span.textContent;
        span.textContent = 'Copied!';
        btn.classList.add('copied');

        setTimeout(() => {
          span.textContent = originalText;
          btn.classList.remove('copied');
        }, 2000);
      })
      .catch(err => {
        console.error('Copy failed:', err);
      });
  };
}
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

// 颜色变量 - 更柔和的配色
$text-primary: #374151;
$text-secondary: #4b5563;
$text-muted: #6b7280;
$text-body: #3f3f46;
$accent-color: #10a37f;
$accent-light: rgba(16, 163, 127, 0.1);
$code-bg: #f8f9fa;
$code-border: #e5e7eb;
$blockquote-bg: #f0fdf4;
$blockquote-border: #10a37f;

.markdown-renderer {
  font-family: $font-sans;
  font-size: 16px !important;
  line-height: 2;
  color: $text-body;
  word-wrap: break-word;
  letter-spacing: 0.02em;
  text-rendering: optimizeLegibility;
  -webkit-font-smoothing: antialiased;

  ::v-deep {
    // 标题样式
    h1,
    h2,
    h3,
    h4,
    h5,
    h6 {
      margin: 28px 0 14px;
      font-weight: 600;
      line-height: 1.35;
      color: $text-primary;
      letter-spacing: -0.01em;

      &:first-child {
        margin-top: 0;
      }
    }

    h1 {
      font-size: 1.75em;
      padding-bottom: 10px;
      border-bottom: 2px solid #e5e7eb;
    }
    h2 {
      font-size: 1.5em;
      padding-bottom: 8px;
      border-bottom: 1px solid #f0f0f0;
    }
    h3 {
      font-size: 1.3em;
    }
    h4 {
      font-size: 1.15em;
      font-weight: 600;
    }
    h5 {
      font-size: 1.05em;
      font-weight: 600;
    }
    h6 {
      font-size: 1em;
      color: $text-secondary;
    }

    // 段落 - 增加段落间距
    p {
      margin: 0 0 20px;
      font-size: 16px !important;
      line-height: 2;

      // 确保p内的所有内联元素继承字体大小
      strong,
      em,
      span,
      a,
      code,
      del,
      mark {
        font-size: inherit;
      }

      &:last-child {
        margin-bottom: 0;
      }
    }

    // 连续段落优化
    p + p {
      text-indent: 0;
    }

    // 列表样式优化
    ul,
    ol {
      margin: 0 0 20px;
      padding-left: 28px;
      font-size: 16px !important;

      li {
        margin: 10px 0;
        line-height: 1.85;
        padding-left: 4px;
        font-size: 16px !important;

        // 确保li内的所有内联元素继承字体大小
        strong,
        em,
        span,
        a,
        code,
        del,
        mark {
          font-size: inherit;
        }

        &::marker {
          color: $accent-color;
          font-weight: 500;
        }
      }

      ul,
      ol {
        margin: 10px 0;
      }
    }

    ul {
      list-style-type: disc;

      ul {
        list-style-type: circle;

        ul {
          list-style-type: square;
        }
      }
    }

    // 引用块 - 现代化样式
    blockquote {
      margin: 22px 0;
      padding: 18px 24px;
      border-left: 4px solid $blockquote-border;
      background: linear-gradient(
        135deg,
        $blockquote-bg 0%,
        rgba(240, 253, 244, 0.5) 100%
      );
      color: $text-secondary;
      border-radius: 0 14px 14px 0;
      font-style: italic;
      line-height: 1.9;

      p:last-child {
        margin-bottom: 0;
      }

      blockquote {
        margin: 14px 0;
        border-left-color: #86efac;
      }
    }

    // 行内代码 - 更柔和的样式
    .inline-code,
    code:not([class*='language-']) {
      padding: 3px 7px;
      background: linear-gradient(135deg, #f3f4f6 0%, #e5e7eb 100%);
      border: 1px solid #d1d5db;
      border-radius: 5px;
      font-family: $font-mono;
      font-size: 0.88em;
      color: #be185d;
      font-weight: 400;
    }

    // 代码块容器 - GitHub 风格
    .code-block-wrapper {
      margin: 20px 0;
      border-radius: 12px;
      overflow: hidden;
      border: 1px solid #30363d;
      background: #0d1117;
      box-shadow: 0 4px 20px rgba(0, 0, 0, 0.15);

      .code-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        padding: 12px 18px;
        background: linear-gradient(180deg, #161b22 0%, #0d1117 100%);
        border-bottom: 1px solid #30363d;

        .language-tag {
          display: flex;
          align-items: center;
          gap: 8px;
          font-size: 13px;
          color: #8b949e;
          font-family: $font-mono;
          text-transform: lowercase;
          font-weight: 500;

          &::before {
            content: '';
            width: 10px;
            height: 10px;
            border-radius: 50%;
            background: linear-gradient(135deg, #f97316 0%, #f59e0b 100%);
          }
        }

        .copy-btn {
          display: flex;
          align-items: center;
          gap: 6px;
          padding: 6px 14px;
          background: transparent;
          border: 1px solid #30363d;
          border-radius: 8px;
          color: #8b949e;
          font-size: 13px;
          font-family: $font-sans;
          cursor: pointer;
          transition: all 0.2s ease;

          svg {
            flex-shrink: 0;
          }

          &:hover {
            background: #21262d;
            border-color: #8b949e;
            color: #c9d1d9;
            transform: translateY(-1px);
          }

          &.copied {
            color: #3fb950;
            border-color: #3fb950;
            background: rgba(63, 185, 80, 0.1);

            svg {
              color: #3fb950;
            }
          }
        }
      }

      .code-content {
        padding: 18px 20px;
        overflow-x: auto;
        -webkit-overflow-scrolling: touch;

        &::-webkit-scrollbar {
          height: 8px;
        }

        &::-webkit-scrollbar-track {
          background: #0d1117;
        }

        &::-webkit-scrollbar-thumb {
          background: #30363d;
          border-radius: 4px;

          &:hover {
            background: #484f58;
          }
        }

        pre {
          margin: 0;
          min-width: fit-content;

          code {
            font-family: $font-mono;
            font-size: 14px;
            line-height: 1.7;
            color: #c9d1d9;
            background: transparent;
            font-variant-ligatures: common-ligatures;
          }
        }
      }
    }

    // 链接样式
    a {
      color: $accent-color;
      text-decoration: none;
      font-weight: 500;
      border-bottom: 1px solid transparent;
      transition: all 0.2s ease;

      &:hover {
        border-bottom-color: $accent-color;
      }
    }

    // 图片样式
    .markdown-image {
      max-width: 100%;
      border-radius: 12px;
      margin: 18px 0;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.1);
      transition:
        transform 0.2s ease,
        box-shadow 0.2s ease;

      &:hover {
        transform: scale(1.01);
        box-shadow: 0 6px 20px rgba(0, 0, 0, 0.15);
      }
    }

    // 视频样式
    .markdown-video {
      max-width: 100%;
      width: 100%;
      max-height: 480px;
      border-radius: 12px;
      margin: 18px 0;
      box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
      background: #000;
      display: block;
    }

    // 音频样式
    .markdown-audio {
      width: 100%;
      max-width: 400px;
      margin: 18px 0;
      border-radius: 8px;
    }

    // PDF链接样式
    .markdown-pdf-link {
      display: inline-flex;
      align-items: center;
      gap: 8px;
      padding: 12px 18px;
      background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
      border: 1px solid #f59e0b;
      border-radius: 10px;
      color: #92400e;
      font-weight: 500;
      text-decoration: none;
      margin: 10px 0;
      transition: all 0.2s ease;

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 4px 12px rgba(245, 158, 11, 0.3);
      }
    }

    // 表格样式
    table {
      margin: 20px 0;
      border-collapse: collapse;
      width: 100%;
      font-size: 15px;
      border-radius: 12px;
      overflow: hidden;
      box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);

      th,
      td {
        padding: 14px 18px;
        border: 1px solid #e5e7eb;
        text-align: left;
        font-size: 14px;
      }

      th {
        background: linear-gradient(180deg, #f9fafb 0%, #f3f4f6 100%);
        font-weight: 600;
        color: $text-primary;
        text-transform: uppercase;
        font-size: 13px;
        letter-spacing: 0.05em;
      }

      td {
        background: #fff;
        font-size: 15px;
      }

      tr:nth-child(even) td {
        background: #f9fafb;
      }

      tr:hover td {
        background: #f0fdf4;
      }
    }

    // 水平线
    hr {
      margin: 32px 0;
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

    // 强调样式
    strong {
      font-weight: 600;
      color: $text-primary;
      letter-spacing: -0.01em;
      font-size: inherit;
    }

    em {
      font-style: italic;
      color: $text-secondary;
      font-size: inherit;
    }

    // 删除线
    del {
      color: $text-muted;
      text-decoration: line-through;
      font-size: inherit;
    }

    // 任务列表
    input[type='checkbox'] {
      margin-right: 8px;
      accent-color: $accent-color;
      width: 16px;
      height: 16px;
    }

    // 脚注引用
    sup {
      font-size: 0.75em;
      vertical-align: super;
      color: $accent-color;
      cursor: pointer;

      &:hover {
        text-decoration: underline;
      }
    }

    // 键盘按键样式
    kbd {
      display: inline-block;
      padding: 4px 8px;
      font-family: $font-mono;
      font-size: 13px;
      line-height: 1.4;
      color: #1f2937;
      background: linear-gradient(180deg, #fff 0%, #f3f4f6 100%);
      border: 1px solid #d1d5db;
      border-radius: 6px;
      box-shadow:
        0 1px 2px rgba(0, 0, 0, 0.1),
        inset 0 1px 0 #fff;
    }

    // 高亮文本
    mark {
      background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
      color: #92400e;
      padding: 2px 6px;
      border-radius: 4px;
    }

    // 定义列表
    dl {
      margin: 18px 0;

      dt {
        font-weight: 600;
        color: $text-primary;
        margin-top: 14px;
      }

      dd {
        margin-left: 28px;
        color: $text-secondary;
      }
    }

    // 缩写
    abbr {
      border-bottom: 1px dotted $text-muted;
      cursor: help;
    }

    // PPT 预览卡片
    .ppt-preview-card {
      display: flex;
      align-items: center;
      gap: 16px;
      padding: 16px 20px;
      margin: 16px 0;
      background: linear-gradient(135deg, #fef3c7 0%, #fde68a 100%);
      border: 1px solid #f59e0b;
      border-radius: 12px;
      cursor: pointer;
      transition: all 0.2s ease;

      &:hover {
        transform: translateY(-2px);
        box-shadow: 0 6px 20px rgba(245, 158, 11, 0.3);
      }

      .ppt-card-icon {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 56px;
        height: 56px;
        background: #fff;
        border-radius: 10px;
        color: #f59e0b;
        flex-shrink: 0;
      }

      .ppt-card-info {
        flex: 1;
        min-width: 0;
      }

      .ppt-card-title {
        font-size: 15px;
        font-weight: 600;
        color: #92400e;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .ppt-card-hint {
        font-size: 13px;
        color: #b45309;
        margin-top: 4px;
      }
    }
  }
}

// PPT 预览弹窗样式（非 scoped）
::v-deep .ppt-preview-dialog {
  .el-dialog__body {
    padding: 0;
    max-height: 80vh;
    overflow: auto;
  }
}
</style>
