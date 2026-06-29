import MarkdownIt from 'markdown-it';
import mk from '@ruanyf/markdown-it-katex';
import { i18n } from '@/lang';
import hljs from 'highlight.js';
import 'highlight.js/styles/atom-one-dark.css';
import { sanitizeHtml, escapeHtml } from '@/utils/sanitize';

hljs.configure({
  lineNumbers: true,
});

/**
 * 生成 Mac Shell 风格代码块 HTML
 * @param {string} code - 代码内容
 * @param {string} lang - 语言标识
 * @returns {string} HTML 字符串
 */
export function highlightCode(code, lang) {
  let preCode = '';
  try {
    if (lang && hljs?.getLanguage(lang)) {
      preCode = hljs.highlight(code, { language: lang }).value;
    } else if (hljs) {
      preCode = hljs.highlightAuto(code).value;
    } else {
      preCode = escapeHtml(code);
    }
  } catch (err) {
    preCode = escapeHtml(code);
  }

  const lines = preCode.split(/\n/);
  if (lines.at(-1) === '') lines.pop();

  let html = lines
    .map((item, index) => {
      return (
        '<li class="code-line">' +
        '<span class="code-line-num">' +
        (index + 1) +
        '</span>' +
        '<span class="code-line-content">' +
        item +
        '</span>' +
        '</li>'
      );
    })
    .join('');

  const langLabel = lang || 'text';
  let htmlCode = '<pre class="code-block"><code>';

  htmlCode += '<span class="code-header">';
  htmlCode += '<span class="code-dots"></span>';
  htmlCode += '<span class="code-lang">' + langLabel + '</span>';
  htmlCode +=
    '<span class="code-copy-btn">' + i18n.t('common.button.copy') + '</span>';
  htmlCode += '</span>';

  htmlCode +=
    '<span class="code-content"><ol class="code-lines">' +
    html +
    '</ol></span>';

  htmlCode += '</code></pre>';
  return htmlCode;
}

/**
 * 创建配置好的 MarkdownIt 实例
 */
export const md = MarkdownIt({
  html: true,
  linkify: true, // 启用自动链接识别,将纯文本URL转换为可点击链接
  highlight: function (str, lang) {
    return highlightCode(str, lang);
  },
});

// 禁用模糊链接匹配，避免文件名被误识别为链接
md.linkify.set({ fuzzyLink: false });

md.use(mk, { throwOnError: false, errorColor: '#000000', output: 'mathml' });

function applyTableWrapper(markdown) {
  const defaultTableOpen =
    markdown.renderer.rules.table_open ||
    function (tokens, idx, options, env, self) {
      return self.renderToken(tokens, idx, options);
    };
  const defaultTableClose =
    markdown.renderer.rules.table_close ||
    function (tokens, idx, options, env, self) {
      return self.renderToken(tokens, idx, options);
    };

  markdown.renderer.rules.table_open = function (
    tokens,
    idx,
    options,
    env,
    self,
  ) {
    tokens[idx].attrJoin('class', 'md-table');
    tokens[idx].attrJoin(
      'style',
      'width:max-content;min-width:100%;max-width:none;margin:0;white-space:nowrap;',
    );
    return (
      '<div class="md-table-wrapper" style="max-width:100%;max-height:min(60vh,520px);overflow:auto;margin:12px 0;">' +
      defaultTableOpen(tokens, idx, options, env, self)
    );
  };

  markdown.renderer.rules.table_close = function (
    tokens,
    idx,
    options,
    env,
    self,
  ) {
    return defaultTableClose(tokens, idx, options, env, self) + '</div>';
  };
}

applyTableWrapper(md);
md.disable('code');

// 对 md.render 输出进行 DOMPurify 净化，防止 XSS 攻击
// html:true 允许原始 HTML 标签通过，需在输出层统一过滤恶意内容
const _originalRender = md.render.bind(md);
md.render = function (text) {
  return sanitizeHtml(_originalRender(text));
};
