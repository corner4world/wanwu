<template>
  <div class="monaco-diff-editor-wrapper">
    <div ref="container" class="monaco-diff-editor"></div>
  </div>
</template>

<script>
import * as monaco from 'monaco-editor';

export default {
  name: 'MonacoDiffEditor',
  props: {
    original: {
      type: String,
      default: '',
    },
    modified: {
      type: String,
      default: '',
    },
    language: {
      type: String,
      default: 'plaintext',
    },
    diffMode: {
      type: String,
      default: 'side-by-side',
    },
  },
  data() {
    return {
      diffEditor: null,
    };
  },
  watch: {
    original() {
      this.updateModels();
    },
    modified() {
      this.updateModels();
    },
    language() {
      this.updateModels();
    },
    diffMode(val) {
      if (this.diffEditor) {
        this.diffEditor.updateOptions({
          renderSideBySide: val === 'side-by-side',
        });
      }
    },
  },
  mounted() {
    this.initDiffEditor();
  },
  beforeDestroy() {
    if (this.diffEditor) {
      // 先 dispose 旧的 models
      const model = this.diffEditor.getModel();
      if (model) {
        model.original?.dispose();
        model.modified?.dispose();
      }
      this.diffEditor.dispose();
      this.diffEditor = null;
    }
  },
  methods: {
    initDiffEditor() {
      this.diffEditor = monaco.editor.createDiffEditor(this.$refs.container, {
        renderSideBySide: this.diffMode === 'side-by-side',
        readOnly: true,
        theme: 'vs',
        automaticLayout: true,
        fontSize: 13,
        minimap: { enabled: false },
        scrollBeyondLastLine: false,
        renderLineHighlight: 'none',
        folding: false,
        lineNumbers: 'on',
      });
      this.updateModels();
    },
    updateModels() {
      if (!this.diffEditor) return;
      // 先 dispose 旧的 models
      const oldModel = this.diffEditor.getModel();
      if (oldModel) {
        oldModel.original?.dispose();
        oldModel.modified?.dispose();
      }
      // 创建新的 models
      const originalModel = monaco.editor.createModel(
        this.original,
        this.language,
      );
      const modifiedModel = monaco.editor.createModel(
        this.modified,
        this.language,
      );
      this.diffEditor.setModel({
        original: originalModel,
        modified: modifiedModel,
      });
    },
  },
};
</script>

<style lang="scss" scoped>
.monaco-diff-editor-wrapper {
  width: 100%;
  height: 100%;
  overflow: hidden;

  .monaco-diff-editor {
    width: 100%;
    height: 100%;
  }
}
</style>
