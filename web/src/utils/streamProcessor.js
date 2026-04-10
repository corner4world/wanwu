/**
 * 流数据处理器，统一处理打字机效果中的块解析、安全截断和 HTML 渲染逻辑
 */
export default class StreamProcessor {
  constructor(options = {}) {
    this.lastIndex = options.lastIndex || 0;
    this.searchList = options.searchList || [];
    this.md = options.md;
    this.parseSub = options.parseSub;
    this.convertLatexSyntax = options.convertLatexSyntax;
    this.preProcess = options.preProcess;

    this.endStr = ''; // 原始全量文本
    this.stableChunks = []; // 已解析完成的稳定 HTML 块
    this.activeText = ''; // 当前正在生成的活跃文本缓冲区

    // 状态标志，用于安全检测点判断
    this.blockStates = {
      inCodeBlock: false,
      inThinkBlock: false,
      inPreBlock: false,
      inLatexBlock: false,
      inToolBlock: false,
    };

    this.citations = new Set(); // 存储已稳定的引文索引
  }

  /**
   * 追加新文本并处理状态机
   * @param {string} newFragment 新增的文本片段
   */
  append(newFragment) {
    this.endStr += newFragment;
    this.activeText += newFragment;

    // 转换 Latex 语法（如果提供了转换函数）仅作用于但单行闭合公式转换
    if (this.convertLatexSyntax) {
      this.activeText = this.convertLatexSyntax(this.activeText);
    }

    const lines = this.activeText.split('\n');
    if (lines.length > 1) {
      let safeFlushText = '';
      let currentScanText = '';

      // 复制当前状态用于扫描
      let scanStates = { ...this.blockStates };

      for (let i = 0; i < lines.length - 1; i++) {
        const line = lines[i];
        const lineWithNewline = line + '\n';

        // 代码块状态判断
        const codeMatches = line.match(/```/g);
        if (codeMatches && codeMatches.length % 2 !== 0) {
          scanStates.inCodeBlock = !scanStates.inCodeBlock;
        }

        // 工具结果块状态判断
        if (line.trim() === '<<<') scanStates.inToolBlock = true;
        if (line.trim() === '>>>') scanStates.inToolBlock = false;

        // 思考块状态判断
        if (line.includes('<think>')) scanStates.inThinkBlock = true;
        if (line.includes('</think>')) scanStates.inThinkBlock = false;

        // Pre 标签状态判断
        if (/<pre[\s>]/i.test(line)) scanStates.inPreBlock = true;
        if (/<\/pre>/i.test(line)) scanStates.inPreBlock = false;

        // LaTeX 块状态判断 ($$)
        const latexMatches = line.match(/\$\$/g);
        if (latexMatches && latexMatches.length % 2 !== 0) {
          scanStates.inLatexBlock = !scanStates.inLatexBlock;
        }

        // 对 \[ \] 和 \( \) 的检测，防止跨行公式在转换前被截断
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

        // 当所有块都闭合时，视为安全截断点
        const isSafe = !Object.values(scanStates).some(state => state);
        if (isSafe) {
          safeFlushText = currentScanText;
          this.blockStates = { ...scanStates };
        }
      }

      if (safeFlushText) {
        // 增量更新已稳定的引文
        this.updateCitations(safeFlushText);

        let textToRender = safeFlushText;
        if (this.preProcess) {
          textToRender = this.preProcess(textToRender);
        }

        // 如果提供了 parseSub，则在渲染前处理引用
        if (this.parseSub) {
          textToRender = this.parseSub(
            textToRender,
            this.lastIndex,
            this.searchList,
          );
        }
        this.stableChunks.push(this.md.render(textToRender));
        this.activeText = this.activeText.substring(safeFlushText.length);
      }
    }
  }

  /**
   * 获取当前渲染结果
   * @returns {Object} 包含 stableHtml, activeHtml, fullResponse 等
   */
  getRenderResult() {
    let activeHtml = '';
    // 获取当前活跃文本中的引文
    const tempCitations = this.getTempCitations(this.activeText);

    if (this.activeText) {
      let textToRender = this.activeText;

      // 活跃区代码块补全，使mdRender识别出为闭合代码块触发渲染
      const codeTicks = textToRender.match(/```/g);
      if (codeTicks && codeTicks.length % 2 !== 0) {
        textToRender += '\n```';
      }

      // 活跃区工具结果块补全
      const toolOpenCount = (textToRender.match(/^<<<$/gm) || []).length;
      const toolCloseCount = (textToRender.match(/^>>>$/gm) || []).length;
      if (toolOpenCount > toolCloseCount) {
        textToRender += '\n>>>';
      }

      if (this.preProcess) {
        textToRender = this.preProcess(textToRender);
      }

      if (this.parseSub) {
        textToRender = this.parseSub(
          textToRender,
          this.lastIndex,
          this.searchList,
        );
      }
      activeHtml = this.md.render(textToRender);
    }

    const stableHtml = this.stableChunks.join('');

    // 合并稳定引文和临时引文
    const allCitations = new Set([...this.citations, ...tempCitations]);

    return {
      response: stableHtml + activeHtml,
      stableChunks: [...this.stableChunks],
      activeResponse: activeHtml,
      oriResponse: this.endStr,
      citations: Array.from(allCitations).sort((a, b) => a - b),
    };
  }

  /**
   * 从文本中提取引文并更新到 stable citations
   */
  updateCitations(text) {
    if (!text) return;
    const pattern = /\【([0-9]{0,2})\^\】/g;
    const matches = text.matchAll(pattern);
    for (const match of matches) {
      if (match[1]) {
        this.citations.add(Number(match[1]));
      }
    }
  }

  /**
   * 从文本中提取引文但不更新 stable citations
   */
  getTempCitations(text) {
    const tempSet = new Set();
    if (!text) return tempSet;
    const pattern = /\【([0-9]{0,2})\^\】/g;
    const matches = text.matchAll(pattern);
    for (const match of matches) {
      if (match[1]) {
        tempSet.add(Number(match[1]));
      }
    }
    return tempSet;
  }

  /**
   * 更新搜索列表（用于 parseSub）
   */
  updateSearchList(list) {
    this.searchList = list || [];
  }
}
