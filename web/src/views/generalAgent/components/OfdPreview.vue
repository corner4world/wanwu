<template>
  <div class="ofd-preview-container" @keydown="handleKeydown">
    <!-- 工具栏（加载成功后显示） -->
    <div v-if="pageCount > 0" class="ofd-toolbar">
      <button
        :disabled="pageIndex <= 1"
        class="nav-btn"
        title="上一页"
        @click="prePage"
      >
        <i class="el-icon-arrow-left"></i>
      </button>
      <span class="page-info">{{ pageIndex }} / {{ pageCount }}</span>
      <button
        :disabled="pageIndex >= pageCount"
        class="nav-btn"
        title="下一页"
        @click="nextPage"
      >
        <i class="el-icon-arrow-right"></i>
      </button>
      <button class="nav-btn" title="放大" @click="zoomIn">
        <i class="el-icon-zoom-in"></i>
      </button>
      <button class="nav-btn" title="缩小" @click="zoomOut">
        <i class="el-icon-zoom-out"></i>
      </button>
      <span class="file-name">{{ fileName }}</span>
    </div>

    <!-- OFD 渲染区域（始终渲染，cafe-ofd 有自己的加载进度条） -->
    <div v-show="!error" ref="viewerWrapper" class="ofd-viewer-wrapper">
      <cafe-ofd
        v-if="filePath"
        ref="ofdViewer"
        :file-path="filePath"
        :width="viewerWidth"
        @on-success="onSuccess"
        @on-error="onError"
        @on-scroll="onScroll"
      />
    </div>

    <!-- 错误状态覆盖层 -->
    <div v-if="error" class="ofd-error">
      <i class="el-icon-warning-outline"></i>
      <span>{{ error }}</span>
      <el-button size="small" @click="retryLoad">
        {{ $t('common.button.retry') }}
      </el-button>
    </div>
  </div>
</template>

<script>
import cafeOfdModule from 'cafe-ofd';
import 'cafe-ofd/package/index.css';

export default {
  name: 'OfdPreview',
  components: {
    // cafe-ofd 导出的是 { install, cafeOfd }，实际组件是 .cafeOfd
    'cafe-ofd': cafeOfdModule.cafeOfd,
  },
  props: {
    src: {
      type: Blob,
      default: null,
    },
    fileName: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      error: null,
      filePath: '',
      pageIndex: 1,
      pageCount: 0,
      viewerWidth: 900,
    };
  },
  watch: {
    src: {
      immediate: true,
      handler(newSrc) {
        if (newSrc) {
          this.$nextTick(() => this.loadOfd());
        }
      },
    },
  },
  mounted() {
    addEventListener('keydown', this.handleKeydown);
    this.updateViewerWidth();
    this._resizeObserver = new ResizeObserver(() => {
      this.updateViewerWidth();
    });
    if (this.$refs.viewerWrapper) {
      this._resizeObserver.observe(this.$refs.viewerWrapper);
    }
  },
  beforeDestroy() {
    removeEventListener('keydown', this.handleKeydown);
    if (this._resizeObserver) {
      this._resizeObserver.disconnect();
      this._resizeObserver = null;
    }
    this.cleanupBlobUrl();
  },
  methods: {
    loadOfd() {
      if (!this.src) return;

      this.error = null;
      this.pageIndex = 1;
      this.pageCount = 0;
      this.cleanupBlobUrl();

      try {
        // cafe-ofd 的 filePath 支持 URL 字符串和 File 对象
        // 将 Blob 转为 Object URL 供 cafe-ofd 使用

        this.filePath = URL.createObjectURL(this.src);
      } catch (err) {
        console.error('[OfdPreview] 创建 Blob URL 失败:', err);
        this.error = this.$t('generalAgent.filePreview.ofdPreviewFailed');
      }
    },

    retryLoad() {
      this.loadOfd();
    },

    cleanupBlobUrl() {
      if (this.filePath && this.filePath.startsWith('blob:')) {
        URL.revokeObjectURL(this.filePath);
      }
      this.filePath = '';
    },

    onSuccess(pageCount, ofdObj) {
      this.pageCount = pageCount;
      this.error = null;
      this.$nextTick(() => {
        this.updateViewerWidth();
      });
    },

    onError(err) {
      console.error('[OfdPreview] OFD 渲染失败:', err);
      this.error = this.$t('generalAgent.filePreview.ofdPreviewFailed');
    },

    onScroll(pageIndex, ofdObj, scrolled) {
      this.pageIndex = pageIndex;
    },

    prePage() {
      if (this.$refs.ofdViewer) {
        this.$refs.ofdViewer.prePage();
      }
    },

    nextPage() {
      if (this.$refs.ofdViewer) {
        this.$refs.ofdViewer.nextPage();
      }
    },

    zoomIn() {
      if (this.$refs.ofdViewer) {
        this.$refs.ofdViewer.scale(50);
      }
    },

    zoomOut() {
      if (this.$refs.ofdViewer) {
        this.$refs.ofdViewer.scale(-50);
      }
    },

    updateViewerWidth() {
      if (this.$refs.viewerWrapper) {
        this.viewerWidth = this.$refs.viewerWrapper.clientWidth || 900;
      }
    },

    handleKeydown(event) {
      if (this.error) return;

      const activeElement = document.activeElement;
      const editableTags = ['INPUT', 'TEXTAREA', 'SELECT'];
      const isEditable =
        activeElement &&
        (editableTags.includes(activeElement.tagName) ||
          activeElement.isContentEditable ||
          activeElement.getAttribute('contenteditable') === 'true');
      if (isEditable) return;

      if (
        event.key === 'ArrowLeft' ||
        event.key === 'ArrowUp' ||
        event.key === 'PageUp'
      ) {
        this.prePage();
        event.preventDefault();
      } else if (
        event.key === 'ArrowRight' ||
        event.key === 'ArrowDown' ||
        event.key === 'PageDown' ||
        event.key === ' '
      ) {
        this.nextPage();
        event.preventDefault();
      } else if (event.key === 'Escape') {
        this.$emit('close');
        event.preventDefault();
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.ofd-preview-container {
  width: 100%;
  height: 100%;
  min-height: 600px;
  background: #fff;
  display: flex;
  flex-direction: column;
  outline: none;

  &:focus {
    outline: none;
  }
}

.ofd-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 400px;
  color: #f56c6c;

  i {
    font-size: 48px;
    margin-bottom: 16px;
  }

  span {
    margin-bottom: 16px;
  }

  .el-button {
    margin: 8px;
  }
}

.ofd-toolbar {
  display: flex;
  align-items: center;
  padding: 8px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  gap: 8px;
  flex-shrink: 0;

  .nav-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid #dcdfe6;
    background: #fff;
    border-radius: 4px;
    cursor: pointer;
    color: #606266;
    transition: all 0.2s;

    &:hover:not(:disabled) {
      border-color: #10a37f;
      color: #10a37f;
    }

    &:disabled {
      cursor: not-allowed;
      opacity: 0.5;
    }
  }

  .page-info {
    font-size: 14px;
    color: #303133;
    min-width: 60px;
    text-align: center;
  }

  .file-name {
    margin-left: auto;
    font-size: 13px;
    color: #909399;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}

.ofd-viewer-wrapper {
  flex: 1;
  overflow: auto;
  min-height: 0;

  // cafe-ofd 组件内部样式微调
  ::v-deep .ofd-container {
    width: 100%;
    height: 100%;
  }

  ::v-deep .ofd-body {
    flex: 1;
  }

  ::v-deep .ofd-item {
    margin-bottom: 10px;
  }
}
</style>
