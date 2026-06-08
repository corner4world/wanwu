<template>
  <div class="monaco-editor-wrapper">
    <monaco-editor
      v-model="code"
      :language="language"
      :theme="theme"
      :options="editorOptions"
      :editorMounted="onEditorMount"
      class="monaco-editor"
    />
  </div>
</template>

<script>
import MonacoEditor from 'monaco-editor-vue';
import * as monaco from 'monaco-editor';

const CONTENT_READY_TIMEOUT_MS = 3000;

export default {
  name: 'MonacoEditorWrapper',
  components: {
    MonacoEditor,
  },
  props: {
    value: {
      type: String,
      default: '',
    },
    language: {
      type: String,
      default: 'plaintext',
    },
    theme: {
      type: String,
      default: 'vs-dark',
    },
    readOnly: {
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      code: this.value,
      editor: null,
      highlightDecorations: [], // Decoration ids for active search highlights.
      contentReadyTimer: null, // Timer for waitForContentReady
      editorOptions: {
        automaticLayout: true,
        readOnly: this.readOnly,
        minimap: {
          enabled: true,
        },
        fontSize: 14,
        lineNumbers: 'on',
        scrollBeyondLastLine: false,
        wordWrap: 'on',
        tabSize: 2,
      },
    };
  },
  watch: {
    value(newVal) {
      if (this.code !== newVal) {
        this.code = newVal;
      }
    },
    code(newVal) {
      this.$emit('input', newVal);
      this.$emit('change', newVal);
    },
    readOnly(newVal) {
      if (this.editor) {
        this.editor.updateOptions({ readOnly: newVal });
      }
    },
  },
  beforeDestroy() {
    // 清理 pending timer
    if (this.contentReadyTimer) {
      clearTimeout(this.contentReadyTimer);
      this.contentReadyTimer = null;
    }
    // 清理编辑器
    if (this.editor) {
      this.clearHighlight();
      const model = this.editor.getModel();
      if (model) {
        model.dispose();
      }
      this.editor.dispose();
      this.editor = null;
    }
  },
  methods: {
    onEditorMount(editor) {
      this.editor = editor;
      this.$emit('editorMounted', editor);
    },
    getValue() {
      return this.code;
    },
    setValue(value) {
      this.code = value;
    },
    focus() {
      if (this.editor) {
        this.editor.focus();
      }
    },
    // Wait until the editor model has visible content.
    waitForContentReady() {
      return new Promise(resolve => {
        if (!this.editor) {
          resolve(false);
          return;
        }

        const model = this.editor.getModel();
        if (!model) {
          resolve(false);
          return;
        }

        if (model.getLineCount() > 1 || model.getValue().length > 0) {
          resolve(true);
          return;
        }

        const disposable = model.onDidChangeContent(() => {
          disposable.dispose();
          if (this.contentReadyTimer) {
            clearTimeout(this.contentReadyTimer);
            this.contentReadyTimer = null;
          }
          resolve(true);
        });

        this.contentReadyTimer = setTimeout(() => {
          disposable.dispose();
          this.contentReadyTimer = null;
          resolve(true);
        }, CONTENT_READY_TIMEOUT_MS);
      });
    },
    // Scroll to a line and optionally highlight a keyword on that line.
    async scrollToLine(lineNumber, keyword) {
      if (!this.editor) {
        return false;
      }

      const model = this.editor.getModel();
      if (!model) {
        return false;
      }

      // Wait for content before trying to reveal a line.
      await this.waitForContentReady();

      // Validate the requested line number.
      const lineCount = model.getLineCount();
      if (lineNumber < 1 || lineNumber > lineCount) {
        return false;
      }

      // Delay the reveal until the next paint so Monaco layout is ready.
      return new Promise(resolve => {
        requestAnimationFrame(() => {
          try {
            // Reveal the target line in the center of the viewport.
            this.editor.revealLineInCenter(lineNumber);

            // Apply inline highlight decorations when a keyword is provided.
            if (keyword) {
              const lineContent = model.getLineContent(lineNumber);

              const matches = [];
              const lowerLine = lineContent.toLowerCase();
              const lowerKeyword = keyword.toLowerCase();
              let startIndex = 0;
              let pos = lowerLine.indexOf(lowerKeyword, startIndex);

              while (pos !== -1) {
                matches.push({
                  startLineNumber: lineNumber,
                  startColumn: pos + 1,
                  endLineNumber: lineNumber,
                  endColumn: pos + keyword.length + 1,
                });
                startIndex = pos + keyword.length;
                pos = lowerLine.indexOf(lowerKeyword, startIndex);
              }

              if (matches.length > 0) {
                this.highlightDecorations = this.editor.deltaDecorations(
                  this.highlightDecorations,
                  matches.map(range => ({
                    range: new monaco.Range(
                      range.startLineNumber,
                      range.startColumn,
                      range.endLineNumber,
                      range.endColumn,
                    ),
                    options: {
                      isWholeLine: false,
                      className: 'search-highlight-line',
                      inlineClassName: 'search-highlight-inline',
                    },
                  })),
                );

                this.editor.setPosition({
                  lineNumber: matches[0].startLineNumber,
                  column: matches[0].startColumn,
                });
                this.editor.focus();

                resolve(true);
              } else {
                resolve(false);
              }
            } else {
              // No keyword to highlight; just focus the editor after scrolling.
              this.editor.focus();
              resolve(true);
            }
          } catch (error) {
            resolve(false);
          }
        });
      });
    },
    // Clear existing highlight decorations.
    clearHighlight() {
      if (this.editor && this.highlightDecorations.length > 0) {
        this.highlightDecorations = this.editor.deltaDecorations(
          this.highlightDecorations,
          [],
        );
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.monaco-editor-wrapper {
  width: 100%;
  height: 100%;
  overflow: hidden;

  .monaco-editor {
    width: 100%;
    height: 100%;
  }
}
</style>

<style lang="scss">
// Guard Monaco canvas from global `canvas { width: 100% !important; }`.
// That global rule can stretch Monaco's overview ruler canvas across the editor.

.monaco-editor canvas {
  width: auto !important;
}

// Keep the overview ruler as a narrow strip on the right.
.monaco-editor .decorationsOverviewRuler {
  width: 14px !important;
}

// The minimap canvas still needs to fill its own container.
.monaco-editor .minimap canvas {
  width: 100% !important;
}

// Search highlight decoration styles must stay global for Monaco decorations.
.search-highlight-inline {
  background-color: rgba(255, 215, 0, 0.5) !important;
  border: 1px solid rgba(255, 180, 0, 0.8);
  border-radius: 2px;
}
</style>
