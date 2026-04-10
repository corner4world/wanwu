import MarkdownIt from 'markdown-it';
import mk from '@ruanyf/markdown-it-katex';
import { i18n } from '@/lang';
import hljs from 'highlight.js';
import 'highlight.js/styles/atom-one-dark.css';

hljs.configure({
  lineNumbers: true,
});

/**
 * 转义 HTML 特殊字符
 */
function escapeHtml(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/'/g, '&#39;');
}

/**
 * 生成 Mac Shell 风格代码块 HTML
 * @param {string} code - 代码内容
 * @param {string} lang - 语言标识
 * @returns {string} HTML 字符串
 */
export function highlightCode(code, lang) {
  let preCode = '';
  try {
    if (lang && hljs && hljs.getLanguage(lang)) {
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
  if (lines[lines.length - 1] === '') lines.pop();

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
  highlight: function (str, lang) {
    return highlightCode(str, lang);
  },
});

md.use(mk, { throwOnError: false, errorColor: '#000000', output: 'mathml' });

md.disable('code');
