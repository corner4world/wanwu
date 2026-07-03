import MarkdownIt from 'markdown-it';
import mk from '@ruanyf/markdown-it-katex';
import { i18n } from '@/lang';
import { sanitizeHtml } from '@/utils/sanitize';

let hljs = require('highlight.js');
hljs.configure({
  lineNumbers: true,
});
import 'highlight.js/styles/atom-one-dark.css';

export const md = MarkdownIt({
  // 在源码中启用 HTML 标签
  html: true,
  // 如果结果以 <pre ... 开头，内部包装器则会跳过。
  highlight: function (str, lang) {
    // 经过highlight.js处理后的html
    let preCode = '';
    try {
      if (lang && hljs.getLanguage(lang)) {
        preCode = hljs.highlight(str, { language: lang }).value;
      } else {
        preCode = md.utils.escapeHtml(str);
      }
    } catch (err) {
      preCode = md.utils.escapeHtml(str);
    }

    const lines = preCode.split(/\n/).slice(0, -1);
    let _lines = lines.filter((it, i) => it !== '');

    // 添加自定义行号
    let html = _lines
      .map((item, index) => {
        return (
          '<li class="line-li"><span class="line-numbers-rows"></span>' +
          item +
          '</li>'
        );
      })
      .join('');
    html = '<ol style="padding: 0px 30px;">' + html + '</ol>';

    // 代码复制功能
    let htmlCode = `<div style="color: #888;border-radius: 0 0 5px 5px;">`;

    htmlCode += `<div class="code-header">`;
    htmlCode += `${lang}<a class="copy-btn mk-copy-btn" style="cursor: pointer;">${i18n.t('common.button.copy')} </a>`;
    htmlCode += `</div>`;

    htmlCode += `<pre class="hljs" style="padding:0 10px!important;margin-bottom:5px;overflow: auto;display: block;border-radius: 5px;"><code>${html}</code></pre>`;
    htmlCode += '</div>';
    return htmlCode;
  },
});

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
// 禁用缩进代码块（Indented Code Block）规则
// 解决流式输出中因格式化产生的行首空格导致 Markdown 语法（如加粗、标题等）失效的问题
md.disable('code');

// 对 md.render 输出进行 DOMPurify 净化，防止 XSS 攻击
// html:true 允许原始 HTML 标签通过，需在输出层统一过滤恶意内容
const _originalRender = md.render.bind(md);
md.render = function (text) {
  return sanitizeHtml(_originalRender(text));
};
