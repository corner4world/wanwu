<template>
  <div class="skill-file-editor">
    <div v-if="loading" class="editor-loading">
      <i class="el-icon-loading"></i>
      <span>
        {{ $t('generalAgent.skill.skillWorkBench.editor.loadingFile') }}
      </span>
    </div>
    <template v-else>
      <div class="editor-breadcrumb">
        <span class="file-info">{{ fileInfo }}</span>
        <div class="breadcrumb-actions">
          <i
            v-if="isMarkdownFile"
            :class="['action-icon', { active: showPreview }]"
            class="el-icon-view"
            :title="
              showPreview
                ? $t('generalAgent.skill.skillWorkBench.editor.closePreview')
                : $t('generalAgent.skill.skillWorkBench.common.preview')
            "
            @click="togglePreview"
          ></i>
          <i
            :class="['action-icon', { disabled: !isModified, saving: saving }]"
            class="el-icon-check"
            :title="$t('generalAgent.skill.skillWorkBench.editor.saveShortcut')"
            @click="saveFile"
          ></i>
        </div>
      </div>
      <div class="editor-wrapper">
        <div
          class="editor-pane"
          :class="{ 'with-preview': showPreview && isMarkdownFile }"
        >
          <MonacoEditor
            ref="editor"
            v-model="currentContent"
            :language="currentLanguage"
            theme="vs"
            @change="handleContentChange"
            @editorMounted="onEditorMounted"
          />
        </div>
        <div v-if="showPreview && isMarkdownFile" class="preview-pane">
          <div class="preview-content" v-html="renderedMarkdown"></div>
        </div>
      </div>
    </template>
  </div>
</template>

<script>
import MonacoEditor from '@/components/MonacoEditor/index.vue';
import {
  getSkillWorkspaceFile,
  updateSkillWorkspaceFile,
} from '@/api/skillResource/skillWorkSpace';
import { marked } from 'marked';
import DOMPurify from 'dompurify';
import { getLanguageByPath } from './workspaceConstants';

export default {
  name: 'SkillFileEditor',
  components: {
    MonacoEditor,
  },
  props: {
    customSkillId: {
      type: String,
      required: true,
    },
    file: {
      type: Object,
      required: true,
    },
    active: {
      type: Boolean,
      default: false,
    },
    highlightRequest: {
      type: Object,
      default: null,
    },
  },
  data() {
    return {
      loading: false,
      saving: false,
      originalContent: '',
      currentContent: '',
      showPreview: false,
      pendingHighlight: null,
      highlightFrame: null,
    };
  },
  computed: {
    currentLanguage() {
      return getLanguageByPath(this.file.path);
    },
    isMarkdownFile() {
      return (this.file.path || '').toLowerCase().endsWith('.md');
    },
    isModified() {
      return this.currentContent !== this.originalContent;
    },
    fileInfo() {
      const size = Number(this.file.size || 0);
      return `${this.file.path} · ${(size / 1024).toFixed(2)} KB · ${this.currentLanguage}`;
    },
    renderedMarkdown() {
      if (!this.currentContent) return '';
      return DOMPurify.sanitize(marked(this.currentContent, { breaks: true }));
    },
  },
  watch: {
    isModified: {
      handler(val) {
        this.$emit('modified-change', {
          path: this.file.path,
          modified: val,
        });
      },
      immediate: true,
    },
    highlightRequest: {
      handler(val) {
        this.queueHighlight(val);
      },
      deep: true,
      immediate: true,
    },
  },
  mounted() {
    document.addEventListener('keydown', this.handleKeyDown);
    this.loadFile();
  },
  beforeDestroy() {
    document.removeEventListener('keydown', this.handleKeyDown);
    if (this.highlightFrame) {
      cancelAnimationFrame(this.highlightFrame);
      this.highlightFrame = null;
    }
  },
  methods: {
    async loadFile(options = {}) {
      const { force = false, silent = false } = options;
      if (!this.customSkillId || !this.file.path) return;
      if (this.isModified && !force) {
        return { skipped: true, reason: 'modified', path: this.file.path };
      }
      this.loading = true;
      try {
        const res = await getSkillWorkspaceFile(
          this.customSkillId,
          this.file.path,
        );
        if (res.code === 0 && res.data) {
          this.originalContent = res.data.content || '';
          this.currentContent = this.originalContent;
          if (res.data.size !== undefined) {
            this.$emit('meta-change', {
              path: this.file.path,
              size: res.data.size,
            });
          }
          return { success: true, path: this.file.path };
        }
        return { failed: true, path: this.file.path };
      } catch (e) {
        if (!silent) {
          this.$message.error(
            this.$t('generalAgent.skill.skillWorkBench.editor.loadFailed'),
          );
        }
        return { failed: true, path: this.file.path, error: e };
      } finally {
        this.loading = false;
        this.scheduleHighlight();
      }
    },
    togglePreview() {
      this.showPreview = !this.showPreview;
    },
    onEditorMounted() {
      this.scheduleHighlight();
    },
    handleContentChange(content) {
      this.currentContent = content;
    },
    queueHighlight(request) {
      if (!request) return;
      this.pendingHighlight = {
        line: request.line,
        keyword: request.keyword,
        seq: request.seq,
      };
      this.scheduleHighlight();
    },
    scheduleHighlight() {
      if (!this.pendingHighlight || this.highlightFrame) return;
      this.$nextTick(() => {
        if (!this.pendingHighlight || this.highlightFrame) return;
        this.highlightFrame = requestAnimationFrame(() => {
          this.highlightFrame = null;
          this.executeHighlight();
        });
      });
    },
    async executeHighlight() {
      if (!this.pendingHighlight) return;
      if (
        this.loading ||
        !this.$refs.editor ||
        !this.$refs.editor.scrollToLine
      ) {
        return;
      }

      const highlight = this.pendingHighlight;
      const { line, keyword } = highlight;

      if (this.$refs.editor && this.$refs.editor.clearHighlight) {
        this.$refs.editor.clearHighlight();
      }

      try {
        await this.$refs.editor.scrollToLine(line, keyword);
        if (this.pendingHighlight === highlight) {
          this.pendingHighlight = null;
        }
      } catch (error) {
        console.error('SkillFileEditor highlight error:', error);
      }
    },
    async saveFile() {
      if (!this.file.path || !this.isModified || this.saving) return;
      try {
        this.saving = true;
        const res = await updateSkillWorkspaceFile(this.customSkillId, {
          path: this.file.path,
          content: this.currentContent,
        });
        if (res.code === 0) {
          this.$message.success(
            this.$t('generalAgent.skill.skillWorkBench.editor.saveSuccess'),
          );
          this.originalContent = this.currentContent;
          this.$emit('saved', this.file.path);
        }
      } catch (e) {
        this.$message.error(
          this.$t('generalAgent.skill.skillWorkBench.editor.saveFailed'),
        );
      } finally {
        this.saving = false;
      }
    },
    handleKeyDown(e) {
      if (!this.active) return;
      if ((e.ctrlKey || e.metaKey) && e.key === 's') {
        e.preventDefault();
        this.saveFile();
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.skill-file-editor {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;

  .editor-loading {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    color: #888;
    font-size: 13px;
  }

  .editor-breadcrumb {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 3px 12px;
    background: #f8f8f8;
    border-bottom: 1px solid #e8e8e8;
    flex-shrink: 0;

    .file-info {
      font-size: 12px;
      color: #888;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .breadcrumb-actions {
      display: flex;
      align-items: center;
      gap: 8px;
      flex-shrink: 0;

      .action-icon {
        font-size: 14px;
        cursor: pointer;
        color: #666;
        padding: 4px;
        border-radius: 3px;

        &:hover {
          color: #333;
          background: #e0e0e0;
        }
        &.active {
          color: #5983ff;
          background: rgba(89, 131, 255, 0.12);
        }
        &.disabled {
          color: #bbb;
          cursor: default;
          &:hover {
            background: none;
          }
        }
        &.saving {
          opacity: 0.5;
          cursor: wait;
        }
        &.el-icon-check {
          font-weight: bold;
          &:not(.disabled):not(.saving) {
            color: #67c23a;
            &:hover {
              color: #85ce61;
              background: #e1f3d8;
            }
          }
        }
      }
    }
  }

  .editor-wrapper {
    flex: 1;
    display: flex;
    overflow: hidden;

    .editor-pane {
      flex: 1;
      display: flex;
      flex-direction: column;
      overflow: hidden;

      &.with-preview {
        min-width: 0;
      }
    }

    .preview-pane {
      width: 50%;
      border-left: 1px solid #e0e0e0;
      overflow-y: auto;
      background: #fff;

      .preview-content {
        padding: 16px;
        font-size: 14px;
        line-height: 1.6;
        color: #333;

        ::v-deep {
          h1,
          h2,
          h3,
          h4,
          h5,
          h6 {
            margin: 16px 0 8px;
            color: #333;
          }
          h1 {
            font-size: 24px;
            border-bottom: 1px solid #e0e0e0;
            padding-bottom: 8px;
          }
          h2 {
            font-size: 20px;
          }
          h3 {
            font-size: 16px;
          }
          p {
            margin: 8px 0;
          }
          code {
            background: #f5f5f5;
            padding: 2px 6px;
            border-radius: 3px;
            font-family: monospace;
            font-size: 13px;
          }
          pre {
            background: #f6f8fa;
            padding: 12px;
            border-radius: 6px;
            overflow-x: auto;
            code {
              background: none;
              padding: 0;
            }
          }
          ul,
          ol {
            padding-left: 24px;
            margin: 8px 0;
          }
          li {
            margin: 4px 0;
          }
          blockquote {
            border: 1px solid #ddd;
            margin: 8px 0;
            padding: 8px 16px;
            color: #666;
            background: #f9f9f9;
          }
          table {
            border-collapse: collapse;
            width: 100%;
            margin: 8px 0;
            th,
            td {
              border: 1px solid #ddd;
              padding: 8px;
              text-align: left;
            }
            th {
              background: #f5f5f5;
            }
          }
          a {
            color: #5983ff;
            text-decoration: none;
          }
          a:hover {
            text-decoration: underline;
          }
          img {
            max-width: 100%;
          }
          hr {
            border: none;
            border-top: 1px solid #e0e0e0;
            margin: 16px 0;
          }
        }
      }
    }
  }
}
</style>
