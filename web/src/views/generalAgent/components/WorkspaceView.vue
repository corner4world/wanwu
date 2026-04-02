<template>
  <div class="workspace-view">
    <div class="workspace-header">
      <span>工作空间</span>
      <i class="el-icon-close" @click="$emit('close')"></i>
    </div>
    <div class="workspace-body">
      <div v-if="loading" class="loading-state">
        <i class="el-icon-loading"></i>
        <span>加载中...</span>
      </div>
      <div v-else-if="!workspaceInfo.fileCount" class="empty-state">
        <i class="el-icon-folder-opened"></i>
        <span>暂无文件</span>
      </div>
      <div v-else class="file-tree">
        <div class="workspace-info">
          <span>共 {{ workspaceInfo.fileCount }} 个文件</span>
          <span>大小: {{ formatSize(workspaceInfo.totalSize) }}</span>
        </div>
        <div class="file-list">
          <div
            v-for="file in files"
            :key="file.name"
            :class="[
              'file-item',
              { 'is-directory': file.type === 'directory' },
            ]"
            @click="handleFileClick(file)"
          >
            <i :class="getFileIcon(file)"></i>
            <span class="file-name">{{ file.name }}</span>
            <span v-if="file.type !== 'directory'" class="file-size">
              {{ formatSize(file.size) }}
            </span>
          </div>
        </div>
      </div>
    </div>

    <!-- 文件预览对话框 -->
    <el-dialog
      :visible.sync="previewVisible"
      :title="previewFile.name"
      width="80%"
      top="5vh"
      append-to-body
    >
      <div class="preview-content">
        <img
          v-if="isImage(previewFile)"
          :src="previewUrl"
          class="preview-image"
        />
        <pre v-else-if="isText(previewFile)" class="preview-text">{{
          previewContent
        }}</pre>
        <div v-else class="preview-unsupported">
          <i class="el-icon-document"></i>
          <p>不支持预览此类型文件</p>
          <el-button
            type="primary"
            size="small"
            @click="downloadFile(previewFile)"
          >
            下载文件
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import {
  getGeneralAgentWorkspace,
  previewGeneralAgentWorkspace,
  downloadGeneralAgentWorkspace,
} from '@/api/generalAgent';

export default {
  name: 'WorkspaceView',
  props: {
    threadId: {
      type: String,
      required: true,
    },
    runId: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      loading: false,
      workspaceInfo: {
        fileCount: 0,
        totalSize: 0,
        isDisplay: false,
      },
      files: [],
      previewVisible: false,
      previewFile: {},
      previewUrl: '',
      previewContent: '',
    };
  },
  watch: {
    runId: {
      immediate: true,
      handler(newVal) {
        if (newVal) {
          this.loadWorkspace();
        }
      },
    },
  },
  methods: {
    async loadWorkspace() {
      if (!this.runId) return;

      this.loading = true;
      try {
        const res = await getGeneralAgentWorkspace({
          threadId: this.threadId,
          runId: this.runId,
        });
        if (res.code === 0 && res.data) {
          this.workspaceInfo = {
            fileCount: res.data.fileCount || 0,
            totalSize: res.data.totalSize || 0,
            isDisplay: res.data.isDisplay || false,
          };
          this.files = res.data.files || [];
        }
      } catch (error) {
        console.error('加载工作空间失败:', error);
      } finally {
        this.loading = false;
      }
    },

    async handleFileClick(file) {
      if (file.type === 'directory') {
        // TODO: 支持目录导航
        return;
      }

      this.previewFile = file;

      if (this.isImage(file)) {
        try {
          const blob = await previewGeneralAgentWorkspace({
            threadId: this.threadId,
            runId: this.runId,
            path: file.name,
          });
          this.previewUrl = URL.createObjectURL(blob);
          this.previewVisible = true;
        } catch (error) {
          console.error('预览文件失败:', error);
        }
      } else if (this.isText(file)) {
        try {
          const blob = await previewGeneralAgentWorkspace({
            threadId: this.threadId,
            runId: this.runId,
            path: file.name,
          });
          this.previewContent = await blob.text();
          this.previewVisible = true;
        } catch (error) {
          console.error('预览文件失败:', error);
        }
      } else {
        this.previewVisible = true;
      }
    },

    async downloadFile(file) {
      try {
        const blob = await downloadGeneralAgentWorkspace({
          threadId: this.threadId,
          runId: this.runId,
          path: file.name,
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = file.name;
        a.click();
        URL.revokeObjectURL(url);
      } catch (error) {
        console.error('下载文件失败:', error);
        this.$message.error('下载失败');
      }
    },

    getFileIcon(file) {
      if (file.type === 'directory') {
        return 'el-icon-folder';
      }
      const ext = file.name.split('.').pop().toLowerCase();
      const iconMap = {
        pdf: 'el-icon-document',
        doc: 'el-icon-document',
        docx: 'el-icon-document',
        xls: 'el-icon-document',
        xlsx: 'el-icon-document',
        ppt: 'el-icon-document',
        pptx: 'el-icon-document',
        txt: 'el-icon-document',
        md: 'el-icon-document',
        json: 'el-icon-document',
        js: 'el-icon-document',
        ts: 'el-icon-document',
        py: 'el-icon-document',
        java: 'el-icon-document',
        go: 'el-icon-document',
        html: 'el-icon-document',
        css: 'el-icon-document',
        png: 'el-icon-picture',
        jpg: 'el-icon-picture',
        jpeg: 'el-icon-picture',
        gif: 'el-icon-picture',
        svg: 'el-icon-picture',
        mp4: 'el-icon-video-camera',
        mp3: 'el-icon-headset',
        wav: 'el-icon-headset',
        zip: 'el-icon-files',
        rar: 'el-icon-files',
        tar: 'el-icon-files',
        gz: 'el-icon-files',
      };
      return iconMap[ext] || 'el-icon-document';
    },

    isImage(file) {
      if (!file.name) return false;
      const ext = file.name.split('.').pop().toLowerCase();
      return ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp'].includes(ext);
    },

    isText(file) {
      if (!file.name) return false;
      const ext = file.name.split('.').pop().toLowerCase();
      return [
        'txt',
        'md',
        'json',
        'js',
        'ts',
        'py',
        'java',
        'go',
        'html',
        'css',
        'xml',
        'yaml',
        'yml',
        'sh',
        'sql',
      ].includes(ext);
    },

    formatSize(bytes) {
      if (!bytes) return '0 B';
      const units = ['B', 'KB', 'MB', 'GB'];
      let size = bytes;
      let unitIndex = 0;
      while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024;
        unitIndex++;
      }
      return `${size.toFixed(1)} ${units[unitIndex]}`;
    },
  },
};
</script>

<style lang="scss" scoped>
.workspace-view {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.workspace-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  font-size: 14px;
  font-weight: 500;

  .el-icon-close {
    cursor: pointer;
    color: #909399;

    &:hover {
      color: #409eff;
    }
  }
}

.workspace-body {
  flex: 1;
  overflow-y: auto;
  padding: 12px;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #909399;

  i {
    font-size: 32px;
    margin-bottom: 8px;
  }
}

.workspace-info {
  display: flex;
  justify-content: space-between;
  padding: 8px 12px;
  background: #f5f7fa;
  border-radius: 4px;
  margin-bottom: 12px;
  font-size: 12px;
  color: #606266;
}

.file-list {
  .file-item {
    display: flex;
    align-items: center;
    padding: 10px 12px;
    border-radius: 4px;
    cursor: pointer;
    transition: background 0.2s;

    &:hover {
      background: #f5f7fa;
    }

    i {
      font-size: 18px;
      margin-right: 10px;
      color: #909399;
    }

    &.is-directory i {
      color: #e6a23c;
    }

    .file-name {
      flex: 1;
      font-size: 14px;
      color: #303133;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
    }

    .file-size {
      font-size: 12px;
      color: #909399;
      margin-left: 12px;
    }
  }
}

.preview-content {
  min-height: 300px;
  display: flex;
  justify-content: center;
  align-items: center;

  .preview-image {
    max-width: 100%;
    max-height: 70vh;
  }

  .preview-text {
    width: 100%;
    max-height: 70vh;
    overflow: auto;
    background: #f5f7fa;
    padding: 16px;
    border-radius: 4px;
    font-size: 13px;
    white-space: pre-wrap;
    word-break: break-all;
  }

  .preview-unsupported {
    text-align: center;
    color: #909399;

    i {
      font-size: 48px;
      margin-bottom: 16px;
    }

    p {
      margin-bottom: 16px;
    }
  }
}
</style>
