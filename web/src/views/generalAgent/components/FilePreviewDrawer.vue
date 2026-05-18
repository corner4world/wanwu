<template>
  <transition name="preview-slide">
    <div v-if="visible" class="preview-panel" :style="mergedPanelStyle">
      <!-- 左侧拖拽手柄 -->
      <div class="resize-handle" @mousedown="startResize"></div>
      <!-- 拖拽时的透明遮罩，防止 iframe 捕获鼠标事件 -->
      <div v-if="isResizing" class="resize-overlay"></div>
      <div class="preview-header">
        <div class="preview-title" :title="filePath || (file ? file.name : '')">
          {{ filePath || (file ? file.name : '') }}
        </div>
        <div class="preview-actions">
          <el-button size="small" @click="handleDownload">
            <i class="el-icon-download"></i>
            {{ $t('common.button.download') }}
          </el-button>
          <el-button
            size="small"
            v-if="
              previewUrl &&
              ['image', 'video', 'audio', 'pdf', 'html'].includes(previewType)
            "
            @click="openInNewTab"
          >
            <i class="el-icon-link"></i>
            {{ $t('generalAgent.filePreview.newWindow') }}
          </el-button>
          <button class="close-btn" @click="handleClose">
            <i class="el-icon-close"></i>
          </button>
        </div>
      </div>
      <div class="preview-body" ref="previewBody">
        <!-- 加载中 -->
        <div v-if="loading" class="preview-loading">
          <i class="el-icon-loading"></i>
          <span>{{ $t('generalAgent.filePreview.loading') }}</span>
        </div>

        <!-- 预览内容 -->
        <template v-else>
          <!-- 图片预览 -->
          <div v-if="previewType === 'image'" class="preview-image-wrapper">
            <img :src="previewUrl" class="preview-image" @error="handleError" />
          </div>

          <!-- 视频预览 -->
          <div
            v-else-if="previewType === 'video'"
            class="preview-video-wrapper"
          >
            <video :src="previewUrl" controls class="preview-video">
              {{ $t('common.fileUpload.videoTips') }}
            </video>
          </div>

          <!-- 音频预览 -->
          <div
            v-else-if="previewType === 'audio'"
            class="preview-audio-wrapper"
          >
            <div class="audio-cover">
              <i class="el-icon-headset"></i>
            </div>
            <audio :src="previewUrl" controls class="preview-audio">
              {{ $t('common.fileUpload.audioTips') }}
            </audio>
          </div>

          <!-- PDF 预览 -->
          <div v-else-if="previewType === 'pdf'" class="preview-pdf-wrapper">
            <iframe :src="previewUrl" class="preview-pdf"></iframe>
          </div>

          <!-- PPT 预览 -->
          <div v-else-if="previewType === 'ppt'" class="preview-ppt-wrapper">
            <ppt-preview
              :src="blob"
              :file-name="file ? file.name : ''"
              @close="handleClose"
            />
          </div>

          <!-- HTML 预览 -->
          <div v-else-if="previewType === 'html'" class="preview-html-wrapper">
            <iframe
              :src="previewUrl"
              class="preview-html-frame"
              sandbox="allow-scripts allow-same-origin"
            ></iframe>
          </div>

          <!-- Excel 预览 -->
          <div
            v-else-if="previewType === 'excel' && previewExcelData"
            class="preview-excel-wrapper"
          >
            <div class="excel-tabs">
              <button
                v-for="(sheet, index) in previewExcelData"
                :key="sheet.name"
                :class="['excel-tab', { active: activeSheetIndex === index }]"
                @click="activeSheetIndex = index"
              >
                {{ sheet.name }}
              </button>
            </div>
            <div class="excel-table-container">
              <table
                class="excel-table"
                v-if="previewExcelData[activeSheetIndex]"
              >
                <tbody>
                  <tr
                    v-for="(row, rowIndex) in previewExcelData[activeSheetIndex]
                      .data"
                    :key="rowIndex"
                  >
                    <td
                      v-for="(cell, colIndex) in row"
                      :key="colIndex"
                      :class="{
                        'excel-header': rowIndex === 0,
                        'merged-cell': isMergedCell(
                          activeSheetIndex,
                          rowIndex,
                          colIndex,
                        ),
                      }"
                      :rowspan="
                        getRowspan(activeSheetIndex, rowIndex, colIndex)
                      "
                      :colspan="
                        getColspan(activeSheetIndex, rowIndex, colIndex)
                      "
                      v-show="
                        !isHiddenByMerge(activeSheetIndex, rowIndex, colIndex)
                      "
                    >
                      {{ cell }}
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Markdown 预览 -->
          <div
            v-else-if="previewType === 'markdown'"
            class="preview-markdown-wrapper"
            ref="markdownRef"
          >
            <div class="markdown-body" v-html="renderedMarkdown"></div>
          </div>

          <!-- Word 预览 -->
          <div
            v-else-if="previewType === 'word'"
            class="preview-office-wrapper"
          >
            <vue-office-docx :src="blob" @error="handleWordError" />
          </div>

          <!-- 文本/代码预览 -->
          <div
            v-else-if="previewType === 'text'"
            class="preview-text-wrapper"
            ref="codeRef"
          >
            <div class="markdown-body" v-html="renderedCode"></div>
          </div>

          <!-- 不支持的格式 -->
          <div v-else class="preview-unsupported">
            <i class="el-icon-document"></i>
            <p class="file-name">{{ file ? file.name : '' }}</p>
            <p class="notice-text">
              {{ $t('generalAgent.filePreview.unsupportedType') }}
            </p>
          </div>
        </template>
      </div>
    </div>
  </transition>
</template>

<script>
import PptPreview from './PptPreview.vue';
import { md, highlightCode } from '../utils/markdown';
import VueOfficeDocx from '@vue-office/docx';
import '@vue-office/docx/lib/index.css';
import { resDownloadFile, getFileType } from '@/utils/util';
import * as XLSX from 'xlsx';

export default {
  name: 'FilePreviewDrawer',
  components: {
    PptPreview,
    VueOfficeDocx,
  },
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    file: {
      type: Object,
      default: null,
    },
    filePath: {
      type: String,
      default: '',
    },
    // 只接收 blob 数据
    blob: {
      type: Blob,
      default: null,
    },
    loading: {
      type: Boolean,
      default: false,
    },
    panelStyle: {
      type: Object,
      default: () => ({}),
    },
  },
  data() {
    return {
      copyClickHandlers: [],
      isResizing: false,
      panelWidth: null,
      activeSheetIndex: 0,
      // 内部预览状态
      previewType: '',
      previewUrl: '',
      previewContent: '',
      previewBlobUrl: '',
      previewExcelData: null,
    };
  },
  computed: {
    fileExt() {
      if (!this.file || !this.file.name) return '';
      return this.file.name.split('.').pop().toLowerCase();
    },
    renderedMarkdown() {
      if (!this.previewContent) return '';
      return md.render(this.previewContent);
    },
    renderedCode() {
      if (!this.previewContent) return '';
      return highlightCode(this.previewContent, this.fileExt);
    },
    mergedPanelStyle() {
      const style = { ...this.panelStyle };
      if (this.panelWidth) {
        style.width = `${this.panelWidth}px`;
      }
      return style;
    },
  },
  watch: {
    visible(val) {
      if (val && this.blob) {
        // 打开且有 blob 数据时，处理预览
        this.processBlob();
      } else if (!val) {
        // 关闭时清理资源
        this.cleanupPreview();
      }
    },
    blob(newVal) {
      // blob 变化时重新处理
      if (newVal && this.visible) {
        this.processBlob();
      }
    },
  },
  mounted() {
    if (this.visible) {
      this.$nextTick(() => {
        this.bindCopyButtons();
        this.bindCodeCopyButtons();
      });
    }
  },
  beforeDestroy() {
    this.unbindCopyButtons();
    this.stopResize();
  },
  methods: {
    // 处理 blob 数据
    async processBlob() {
      if (!this.blob || !this.file) {
        return;
      }

      // 重置状态
      this.previewType = '';
      this.previewUrl = '';
      this.previewContent = '';
      this.previewBlobUrl = '';
      this.previewExcelData = null;
      this.activeSheetIndex = 0;

      try {
        this.previewType = getFileType(this.file.name);

        if (
          ['image', 'video', 'audio', 'pdf', 'html'].includes(this.previewType)
        ) {
          // 对于图片、视频、音频、PDF、HTML，创建 Object URL
          this.previewBlobUrl = URL.createObjectURL(this.blob);
          this.previewUrl = this.previewBlobUrl;
        } else if (['markdown', 'text'].includes(this.previewType)) {
          this.previewContent = await this.blob.text();
        } else if (this.previewType === 'excel') {
          const arrayBuffer = await this.blob.arrayBuffer();
          const workbook = XLSX.read(arrayBuffer, { type: 'array' });
          const excelData = workbook.SheetNames.map(sheetName => {
            const sheet = workbook.Sheets[sheetName];
            const jsonData = XLSX.utils.sheet_to_json(sheet, {
              header: 1,
              defval: '',
            });
            const merges = sheet['!merges'] || [];
            return {
              name: sheetName,
              data: jsonData,
              merges: merges.map(m => ({
                sr: m.s.r,
                sc: m.s.c,
                er: m.e.r,
                ec: m.e.c,
              })),
              colCount: Math.max(
                1,
                sheet['!ref']
                  ? XLSX.utils.decode_range(sheet['!ref']).e.c + 1
                  : 1,
              ),
            };
          });
          this.previewExcelData = excelData;
        }
      } catch (error) {
        console.error('处理文件失败:', error);
        this.$message.error(this.$t('generalAgent.filePreview.processFailed'));
        this.previewType = 'unsupported';
      } finally {
        this.$nextTick(() => {
          this.bindCopyButtons();
          this.bindCodeCopyButtons();
        });
      }
    },

    // 清理预览资源
    cleanupPreview() {
      if (this.previewBlobUrl && typeof this.previewBlobUrl === 'string') {
        URL.revokeObjectURL(this.previewBlobUrl);
        this.previewBlobUrl = '';
      }
      this.unbindCopyButtons();
      this.stopResize();
    },

    handleClose() {
      this.$emit('update:visible', false);
      this.$emit('close');
    },

    startResize(e) {
      e.preventDefault();
      this.isResizing = true;
      document.addEventListener('mousemove', this.doResize);
      document.addEventListener('mouseup', this.stopResize);
      document.body.style.cursor = 'ew-resize';
      document.body.style.userSelect = 'none';
    },

    doResize(e) {
      if (!this.isResizing) return;
      const panelEl = this.$el;
      if (!panelEl) return;
      const rect = panelEl.getBoundingClientRect();
      // 面板右边缘固定，宽度 = 右边缘 - 鼠标位置
      const newWidth = rect.right - e.clientX;
      const minWidth = 500;
      // 最大宽度不超过视口减去边距
      const maxWidth = window.innerWidth - 100;
      this.panelWidth = Math.max(minWidth, Math.min(maxWidth, newWidth));
    },

    stopResize() {
      this.isResizing = false;
      document.removeEventListener('mousemove', this.doResize);
      document.removeEventListener('mouseup', this.stopResize);
      document.body.style.cursor = '';
      document.body.style.userSelect = '';
    },

    handleError() {
      this.$message.error(this.$t('generalAgent.filePreview.previewFailed'));
    },

    openInNewTab() {
      if (this.previewUrl) {
        window.open(this.previewUrl, '_blank');
      }
    },

    isMergedCell(sheetIndex, row, col) {
      if (!this.previewExcelData || !this.previewExcelData[sheetIndex])
        return false;
      const merges = this.previewExcelData[sheetIndex].merges || [];
      return merges.some(m => row === m.sr && col === m.sc);
    },

    isHiddenByMerge(sheetIndex, row, col) {
      if (!this.previewExcelData || !this.previewExcelData[sheetIndex])
        return false;
      const merges = this.previewExcelData[sheetIndex].merges || [];
      return merges.some(
        m =>
          row >= m.sr &&
          row <= m.er &&
          col >= m.sc &&
          col <= m.ec &&
          !(row === m.sr && col === m.sc),
      );
    },

    getRowspan(sheetIndex, row, col) {
      if (!this.previewExcelData || !this.previewExcelData[sheetIndex])
        return 1;
      const merges = this.previewExcelData[sheetIndex].merges || [];
      const merge = merges.find(m => row === m.sr && col === m.sc);
      return merge ? merge.er - merge.sr + 1 : 1;
    },

    getColspan(sheetIndex, row, col) {
      if (!this.previewExcelData || !this.previewExcelData[sheetIndex])
        return 1;
      const merges = this.previewExcelData[sheetIndex].merges || [];
      const merge = merges.find(m => row === m.sr && col === m.sc);
      return merge ? merge.ec - merge.sc + 1 : 1;
    },

    // 下载文件
    async handleDownload() {
      if (!this.file || !this.blob) {
        return;
      }

      try {
        resDownloadFile(this.blob, this.file.name);
        this.$message.success(
          this.$t('generalAgent.workspace.downloadSuccess'),
        );
      } catch (error) {
        console.error('下载文件失败:', error);
        this.$message.error(this.$t('generalAgent.workspace.downloadFailed'));
      }
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
                e.target.innerText = this.$t('common.copy.copySuccess');
                setTimeout(() => {
                  e.target.innerText = originalText;
                }, 1500);
              })
              .catch(() => {
                this.$message.error(this.$t('tempSquare.copyFailed'));
              });
          }
        };
        btn.addEventListener('click', handler);
        this.copyClickHandlers.push({ btn, handler });
      });
    },

    bindCodeCopyButtons() {
      if (!this.$refs.codeRef) return;

      const copyButtons = this.$refs.codeRef.querySelectorAll('.code-copy-btn');
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
                e.target.innerText = this.$t('common.copy.copySuccess');
                setTimeout(() => {
                  e.target.innerText = originalText;
                }, 1500);
              })
              .catch(() => {
                this.$message.error(this.$t('tempSquare.copyFailed'));
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

    handleWordError(error) {
      console.error('Word 文档渲染失败:', error);
      this.$message.error(
        this.$t('generalAgent.filePreview.wordPreviewFailed'),
      );
    },
  },
};
</script>

<style scoped>
.preview-panel {
  position: relative;
  height: 100%;
  min-width: 500px;
  background: #fff;
  border-left: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
  z-index: 10;
}

.resize-handle {
  position: absolute;
  left: -3px; /* 拖拽手柄稍微偏移，使其覆盖在边框上 */
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: ew-resize;
  background: transparent;
  border-radius: 12px 0 0 12px;
  transition: background 0.2s;
  z-index: 10;
}

.resize-handle:hover {
  background: rgba(16, 163, 127, 0.3);
}

.resize-handle:active {
  background: rgba(16, 163, 127, 0.5);
}

.resize-overlay {
  position: absolute;
  left: 0;
  top: 0;
  right: 0;
  bottom: 0;
  z-index: 5;
  cursor: ew-resize;
}

.preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #fafafa;
  flex-shrink: 0;
}

.preview-title {
  font-size: 14px;
  font-weight: 500;
  color: #1a1a1a;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
  margin-right: 12px;
}

.preview-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.close-btn {
  width: 28px;
  height: 28px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: transparent;
  border: none;
  cursor: pointer;
  color: #606266;
  border-radius: 4px;
  transition: all 0.2s;
}

.close-btn:hover {
  background: #f0f2f5;
  color: #10a37f;
}

.preview-body {
  flex: 1;
  overflow: auto;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

/* 覆盖 markdown.scss 中影响布局的伪元素样式 */
.preview-body .markdown-body::before,
.preview-body .markdown-body::after {
  display: none !important;
}

/* 所有预览容器统一设置 */
.preview-body > div {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
}

/* 滑入滑出动画 - 从左向右收起，从右向左展开 */
.preview-slide-enter-active,
.preview-slide-leave-active {
  transition: all 0.3s ease;
}

/* 进入时：从右边滑入（初始位置在 workspace 下方，不可见） */
.preview-slide-enter {
  transform: translateX(100%);
  opacity: 0;
}

/* 离开时：滑出到右边 */
.preview-slide-leave-to {
  transform: translateX(100%);
  opacity: 0;
}

.preview-container {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.preview-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 0;
  color: #909399;
  font-size: 14px;
}

.preview-loading i {
  font-size: 32px;
  margin-bottom: 12px;
  color: #10a37f;
}

.preview-image-wrapper {
  background: #f5f7fa;
  padding: 20px;
}

.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
}

.preview-video-wrapper {
  background: #000;
  padding: 20px;
}

.preview-video {
  max-width: 100%;
  max-height: 100%;
}

.preview-audio-wrapper {
  gap: 20px;
}

.preview-excel-wrapper {
  display: flex;
  flex-direction: column;
  background: #fff;
}

.excel-tabs {
  display: flex;
  gap: 4px;
  padding: 8px 12px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  flex-shrink: 0;
  overflow-x: auto;
}

.excel-tab {
  padding: 6px 16px;
  background: #fff;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  font-size: 13px;
  color: #606266;
  cursor: pointer;
  white-space: nowrap;
  transition: all 0.2s;
}

.excel-tab:hover {
  border-color: #10a37f;
  color: #10a37f;
}

.excel-tab.active {
  background: #10a37f;
  border-color: #10a37f;
  color: #fff;
}

.excel-table-container {
  flex: 1;
  overflow: auto;
  min-height: 0;
}

.excel-table {
  border-collapse: collapse;
  width: max-content;
  min-width: 100%;
  font-size: 13px;
}

.excel-table td {
  border: 1px solid #e4e7ed;
  padding: 6px 10px;
  text-align: left;
  white-space: nowrap;
  max-width: 300px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.excel-table td.excel-header {
  background: #f5f7fa;
  font-weight: 600;
  color: #303133;
  position: sticky;
  top: 0;
  z-index: 1;
}

.excel-table td.merged-cell {
  text-align: center;
  vertical-align: middle;
}

.audio-cover {
  width: 120px;
  height: 120px;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  border-radius: 12px;
  color: white;
}

.audio-cover i {
  font-size: 48px;
}

.preview-audio {
  width: 100%;
  max-width: 500px;
}

.preview-pdf-wrapper,
.preview-ppt-wrapper,
.preview-html-wrapper {
  overflow: hidden;
}

.preview-pdf,
.preview-html-frame {
  width: 100%;
  height: 100%;
  border: none;
  flex: 1;
}

.preview-markdown-wrapper {
  overflow: auto;
  padding: 20px;
  background: #fff;
}

.preview-office-wrapper {
  flex: 1;
  min-height: 0;
  overflow: auto;
  background: #fff;
}

.preview-unsupported {
  align-items: center;
  justify-content: center;
  color: #909399;
}

.preview-unsupported i {
  font-size: 48px;
  margin-bottom: 16px;
  color: #c0c4cc;
}

.preview-unsupported .file-name {
  font-size: 16px;
  color: #606266;
  margin-bottom: 8px;
}

.preview-unsupported .notice-text {
  font-size: 14px;
  margin-bottom: 16px;
}

.preview-toolbar {
  padding: 12px 20px;
  border-top: 1px solid #e4e7ed;
  background: #fafafa;
  display: flex;
  gap: 8px;
}
</style>

<style lang="scss" scoped>
@import '@/assets/showDocs/showdoc.scss';
@import '../styles/_markdown-common.scss';

// 代码文件预览样式
.preview-text-wrapper {
  background: #0d0d0d;
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;

  .markdown-body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;

    // 覆盖 markdown.scss 中的 display: table，避免影响 flex 布局
    &::before,
    &::after {
      display: none !important;
    }

    ::v-deep pre.code-block {
      flex: 1;
      min-height: 0;
      margin: 0;
      display: flex;
      flex-direction: column;

      code {
        flex: 1;
        min-height: 0;
        display: flex;
        flex-direction: column;
      }

      .code-content {
        flex: 1;
        min-height: 0;
        overflow: auto;
      }
    }
  }
}

.markdown-body {
  ::v-deep {
    @include markdown-content-base;
    @include complete-code-block-fullscreen;
  }
}
</style>
