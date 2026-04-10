import { i18n } from '@/lang';

/**
 * 预处理工具结果块
 * 将 <<< ... >>> 包裹的内容转换为自定义 HTML，
 * 自动提取标题、区分 JSON 数据与纯文本描述，应用不同样式并注入基于数据属性的原生复制能力。
 */

function escapeHtml(str) {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;');
}

/**
 * @param {string} text 原始 markdown 文本
 * @returns {string} 替换后的文本（包含 HTML）
 */
export function processToolResultBlocks(text) {
  // 匹配 <<< 和 >>>，并提取中间的首行标题及剩余内容部分
  return text.replace(
    /^<<<\r?\n(.*?)\r?\n([\s\S]*?)\r?\n>>>$/gm,
    (match, title, content) => {
      const rawTitle = title ? title.trim() : '';
      const trimmedContent = content.trim();
      let isJson = false;
      let displayContent = trimmedContent;

      try {
        const parsed = JSON.parse(trimmedContent);
        isJson = true;
        displayContent = JSON.stringify(parsed, null, 2);
      } catch (e) {
        isJson = false;
      }

      const encodedRawContent = encodeURIComponent(trimmedContent);
      const boxClass = isJson ? 'tool-result-json' : 'tool-result-text';
      const contentHtml = isJson
        ? `<pre>${escapeHtml(displayContent)}</pre>`
        : `${escapeHtml(trimmedContent)}`;

      return `<div class="tool-result-container">
      <div class="code-header tool-result-header">
        <span class="tool-result-title">${escapeHtml(rawTitle)}</span>
        <a class="copy-btn mk-copy-btn" data-clipboard-text="${encodedRawContent}" style="cursor: pointer;">${i18n.t('common.button.copy')}</a>
      </div>
      <div class="tool-result-content tool-result-box ${boxClass}">${contentHtml}</div>
    </div>`;
    },
  );
}
