<template>
  <div class="workspace-activity" v-if="workspaceInfo">
    <div class="activity-header">
      <div class="activity-icon">
        <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
          <path
            d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"
          />
        </svg>
      </div>
      <div class="activity-title">
        <span class="title-text">工作空间已更新</span>
        <span class="activity-badge" v-if="fileCount > 0">
          {{ fileCount }} 个文件
        </span>
      </div>
    </div>

    <div class="activity-body">
      <div class="activity-stats">
        <div class="stat-item">
          <span class="stat-value">{{ fileCount }}</span>
          <span class="stat-label">文件</span>
        </div>
        <div class="stat-divider"></div>
        <div class="stat-item">
          <span class="stat-value">{{ formatSize(totalSize) }}</span>
          <span class="stat-label">大小</span>
        </div>
      </div>

      <div class="activity-actions">
        <el-button
          size="mini"
          type="primary"
          plain
          @click="handleViewWorkspace"
        >
          <i class="el-icon-folder-opened"></i>
          查看工作空间
        </el-button>
        <el-button size="mini" plain @click="handleDownloadAll">
          <i class="el-icon-download"></i>
          下载
        </el-button>
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'WorkspaceActivity',
  props: {
    // workspace 信息
    workspaceInfo: {
      type: Object,
      default: null,
    },
    // threadId
    threadId: {
      type: String,
      default: '',
    },
    // runId
    runId: {
      type: String,
      default: '',
    },
  },
  computed: {
    fileCount() {
      return this.workspaceInfo?.fileCount || 0;
    },
    totalSize() {
      return this.workspaceInfo?.totalSize || 0;
    },
  },
  methods: {
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

    handleViewWorkspace() {
      this.$emit('view-workspace', {
        threadId: this.threadId,
        runId: this.runId || this.workspaceInfo?.runId,
        fileCount: this.workspaceInfo?.fileCount || 0,
        totalSize: this.workspaceInfo?.totalSize || 0,
      });
    },

    handleDownloadAll() {
      this.$emit('download-all', {
        threadId: this.threadId,
        runId: this.runId || this.workspaceInfo?.runId,
      });
    },
  },
};
</script>

<style lang="scss" scoped>
$workspace-color: #3b82f6;
$workspace-light: #60a5fa;
$workspace-bg: rgba(59, 130, 246, 0.08);

.workspace-activity {
  margin: 16px 0;
  border-radius: 12px;
  background: linear-gradient(135deg, $workspace-bg 0%, #fafafa 100%);
  border: 1px solid rgba(59, 130, 246, 0.2);
  overflow: hidden;
  transition: all 0.3s ease;
}

.activity-header {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: linear-gradient(
    135deg,
    rgba(59, 130, 246, 0.1) 0%,
    rgba(59, 130, 246, 0.05) 100%
  );
  border-bottom: 1px solid rgba(59, 130, 246, 0.1);
}

.activity-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: linear-gradient(
    135deg,
    $workspace-color 0%,
    $workspace-light 100%
  );
  border-radius: 8px;
  color: #fff;
}

.activity-title {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 10px;
}

.title-text {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.activity-badge {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.15);
  border-radius: 10px;
  font-size: 12px;
  font-weight: 500;
  color: $workspace-color;
}

.activity-body {
  padding: 16px;
}

.activity-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 20px;
  margin-bottom: 16px;
}

.stat-item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.stat-value {
  font-size: 20px;
  font-weight: 600;
  color: $workspace-color;
  font-variant-numeric: tabular-nums;
}

.stat-label {
  font-size: 12px;
  color: #6b7280;
}

.stat-divider {
  width: 1px;
  height: 32px;
  background: #e5e7eb;
}

.activity-actions {
  display: flex;
  justify-content: center;
  gap: 12px;

  .el-button {
    display: flex;
    align-items: center;
    gap: 4px;

    i {
      font-size: 14px;
    }
  }
}
</style>
