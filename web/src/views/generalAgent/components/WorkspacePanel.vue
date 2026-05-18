<template>
  <div class="workspace-panel">
    <div class="panel-header">
      <div class="header-left">
        <svg
          viewBox="0 0 24 24"
          width="18"
          height="18"
          fill="currentColor"
          class="header-icon"
        >
          <path
            d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"
          />
        </svg>
        <span class="header-title">
          {{ $t('generalAgent.workspace.title') }}
        </span>
      </div>
      <div class="header-actions">
        <el-tooltip
          :content="$t('generalAgent.workspace.refresh')"
          placement="bottom"
        >
          <button
            class="header-btn"
            @click="refreshCurrent"
            :disabled="loading"
          >
            <i :class="loading ? 'el-icon-loading' : 'el-icon-refresh'"></i>
          </button>
        </el-tooltip>
        <el-tooltip
          :content="$t('generalAgent.workspace.close')"
          placement="bottom"
        >
          <button class="header-btn" @click="$emit('close')">
            <i class="el-icon-close"></i>
          </button>
        </el-tooltip>
      </div>
    </div>

    <div class="panel-body">
      <!-- 加载状态 -->
      <div v-if="loading" class="loading-state">
        <i class="el-icon-loading"></i>
        <span>{{ $t('generalAgent.workspace.loading') }}</span>
      </div>

      <!-- 空状态 -->
      <div v-else-if="!workspaceInfo.fileCount" class="empty-state">
        <svg
          viewBox="0 0 24 24"
          width="48"
          height="48"
          fill="currentColor"
          class="empty-icon"
        >
          <path
            d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"
            opacity="0.3"
          />
        </svg>
        <span class="empty-text">{{ $t('generalAgent.workspace.empty') }}</span>
        <span class="empty-hint">
          {{ $t('generalAgent.workspace.emptyHint') }}
        </span>
      </div>

      <!-- 文件树 -->
      <file-tree
        v-else
        :files="files"
        :current-path="currentPath"
        :workspace-info="currentDirInfo"
        @navigate="navigateTo"
        @preview="handleFileClick"
        @download="downloadFile"
      />
    </div>
  </div>
</template>

<script>
import {
  getGeneralAgentWorkspace,
  downloadGeneralAgentWorkspace,
} from '@/api/generalAgent';
import FileTree from './FileTree.vue';
import { resDownloadFile } from '@/utils/util';

export default {
  name: 'WorkspacePanel',
  components: {
    FileTree,
  },
  props: {
    threadId: {
      type: String,
      required: true,
    },
    runId: {
      type: String,
      default: '',
    },
    initialData: {
      type: Object,
      default: null,
    },
  },
  data() {
    return {
      loading: false,
      loadRequestId: 0,
      currentPath: '',
      rootFiles: [],
      files: [],
      workspaceInfo: {
        fileCount: 0,
        totalSize: 0,
        isDisplay: false,
      },
    };
  },
  computed: {
    currentDirInfo() {
      const fileCount = this.files.filter(f => !this.isDirectory(f)).length;
      const totalSize = this.files
        .filter(f => !this.isDirectory(f))
        .reduce((sum, f) => sum + (f.size || 0), 0);
      return {
        fileCount,
        totalSize,
        isDisplay: fileCount > 0,
      };
    },
  },
  watch: {
    runId: {
      immediate: true,
      handler() {
        this.tryLoadWorkspace();
      },
    },
    threadId: {
      immediate: true,
      handler() {
        this.tryLoadWorkspace();
      },
    },
    initialData: {
      immediate: true,
      handler(newVal) {
        if (newVal) {
          this.currentPath = '';
          this.workspaceInfo = {
            fileCount: newVal.fileCount || 0,
            totalSize: newVal.totalSize || 0,
            isDisplay: newVal.isDisplay || false,
          };
          this.rootFiles = newVal.files || [];
          this.files = this.processFiles(this.rootFiles);
        }
      },
    },
  },
  methods: {
    isDirectory(file) {
      return file.type === 'directory' || file.type === 'dir';
    },

    tryLoadWorkspace() {
      if (!this.threadId) {
        return;
      }
      const currentRequestId = ++this.loadRequestId;
      setTimeout(() => {
        if (currentRequestId === this.loadRequestId) {
          this.loadWorkspace();
        }
      }, 50);
    },

    refreshCurrent() {
      this.loadWorkspace();
    },

    navigateTo(path) {
      this.currentPath = path || '';
      this.files = this.getFilesAtPath(this.currentPath);
      this.files = this.processFiles(this.files);
    },

    getFilesAtPath(path) {
      if (!path) {
        return this.rootFiles || [];
      }
      const parts = path.split('/').filter(p => p);
      let current = this.rootFiles || [];
      for (const part of parts) {
        const dir = current.find(f => f.name === part && this.isDirectory(f));
        if (dir && dir.children) {
          current = dir.children;
        } else {
          return [];
        }
      }
      return current;
    },

    processFiles(files) {
      if (!files || !Array.isArray(files)) return [];
      const sorted = [...files].sort((a, b) => {
        const aIsDir = this.isDirectory(a);
        const bIsDir = this.isDirectory(b);
        if (aIsDir && !bIsDir) return -1;
        if (!aIsDir && bIsDir) return 1;
        return (a.name || '').localeCompare(b.name || '');
      });
      return sorted;
    },

    async loadWorkspace() {
      if (!this.threadId) {
        this.loading = false;
        return;
      }

      this.loading = true;
      this.currentPath = '';
      try {
        const params = {
          threadId: this.threadId,
          runId: this.runId,
        };
        const res = await getGeneralAgentWorkspace(params);
        if (res.code === 0 && res.data) {
          this.workspaceInfo = {
            fileCount: res.data.fileCount || 0,
            totalSize: res.data.totalSize || 0,
            isDisplay: res.data.isDisplay || false,
          };
          this.rootFiles = res.data.files || [];
          this.files = this.processFiles(this.rootFiles);
        } else if (res.code !== 0) {
          console.error('[WorkspacePanel] API error:', res.msg);
          this.$message.error(
            res.msg || this.$t('generalAgent.workspace.loadFailed'),
          );
          this.workspaceInfo = {
            fileCount: 0,
            totalSize: 0,
            isDisplay: false,
          };
          this.rootFiles = [];
          this.files = [];
        }
      } catch (error) {
        console.error('[WorkspacePanel] 加载工作空间失败:', error);
        this.$message.error(this.$t('generalAgent.workspace.loadFailed'));
        this.workspaceInfo = {
          fileCount: 0,
          totalSize: 0,
          isDisplay: false,
        };
        this.rootFiles = [];
        this.files = [];
      } finally {
        this.loading = false;
      }
    },

    async handleFileClick(file) {
      if (this.isDirectory(file)) {
        const newPath = this.currentPath
          ? `${this.currentPath}/${file.name}`
          : file.name;
        if (file.children && file.children.length > 0) {
          this.currentPath = newPath;
          this.files = this.processFiles(file.children);
        } else {
          this.currentPath = newPath;
          this.files = [];
        }
        return;
      }

      const filePath = this.currentPath
        ? `${this.currentPath}/${file.name}`
        : file.name;

      this.$emit('preview-file', {
        file,
        filePath,
        threadId: this.threadId,
        runId: this.runId,
      });
    },

    async downloadFile(file) {
      try {
        const filePath = this.currentPath
          ? `${this.currentPath}/${file.name}`
          : file.name;
        const blob = await downloadGeneralAgentWorkspace({
          threadId: this.threadId,
          runId: this.runId,
          path: filePath,
        });
        // 如果是文件夹，文件名添加.zip后缀
        const fileName = this.isDirectory(file)
          ? `${file.name}.zip`
          : file.name;
        resDownloadFile(blob, fileName);
        this.$message.success(
          this.$t('generalAgent.workspace.downloadSuccess'),
        );
      } catch (error) {
        console.error('下载文件失败:', error);
        this.$message.error(this.$t('generalAgent.workspace.downloadFailed'));
      }
    },
  },
};
</script>

<style scoped>
.workspace-panel {
  position: relative;
  display: flex;
  flex-direction: column;
  width: 400px;
  height: 100%;
  background: #fff;
  border-left: 1px solid #e4e7ed;
  box-shadow: -2px 0 8px rgba(0, 0, 0, 0.05);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #e4e7ed;
  background: #fafafa;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  color: #10a37f;
}

.header-title {
  font-size: 14px;
  font-weight: 500;
  color: #1a1a1a;
}

.header-actions {
  display: flex;
  gap: 4px;
}

.header-btn {
  padding: 6px;
  background: transparent;
  border: none;
  cursor: pointer;
  color: #606266;
  border-radius: 4px;
  transition: all 0.2s;
}

.header-btn:hover {
  background: #f0f2f5;
  color: #10a37f;
}

.header-btn:disabled {
  color: #c0c4cc;
  cursor: not-allowed;
}

.panel-body {
  flex: 1;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #909399;
  font-size: 14px;
}

.loading-state i,
.empty-state .empty-icon {
  font-size: 48px;
  margin-bottom: 12px;
  color: #c0c4cc;
}

.empty-state .empty-text {
  font-size: 14px;
  color: #606266;
  margin-bottom: 4px;
}

.empty-state .empty-hint {
  font-size: 12px;
  color: #909399;
}
</style>
