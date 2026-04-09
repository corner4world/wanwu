<template>
  <div class="file-tree">
    <!-- 面包屑导航 -->
    <div class="breadcrumb">
      <span
        :class="['breadcrumb-item', { active: currentPath === '' }]"
        @click="navigateTo('')"
      >
        <i class="el-icon-folder-opened"></i>
        {{ $t('generalAgent.workspace.rootDir') }}
      </span>
      <template v-for="(part, index) in pathParts">
        <span :key="'sep-' + index" class="breadcrumb-sep">/</span>
        <span
          :key="'part-' + index"
          :class="[
            'breadcrumb-item',
            { active: index === pathParts.length - 1 },
          ]"
          @click="navigateTo(getPathByIndex(index))"
        >
          {{ part }}
        </span>
      </template>
    </div>

    <div class="tree-info">
      <span class="info-item">
        <i class="el-icon-document"></i>
        {{
          $t('generalAgent.workspace.fileCount', {
            count: workspaceInfo.fileCount,
          })
        }}
      </span>
      <span class="info-divider">|</span>
      <span class="info-item">
        <i class="el-icon-coin"></i>
        {{ formatSize(workspaceInfo.totalSize) }}
      </span>
    </div>

    <!-- 文件列表 -->
    <div class="file-list">
      <!-- 返回上一级 -->
      <div
        v-if="currentPath !== ''"
        class="file-item back-item"
        @click="navigateToParent"
      >
        <i class="el-icon-back"></i>
        <span class="file-name">..</span>
      </div>
      <div
        v-for="(file, index) in files"
        :key="index"
        :class="['file-item', { 'is-directory': isDirectory(file) }]"
      >
        <div class="file-item-main" @click="handleFileClick(file)">
          <i :class="getFileIconClass(file)"></i>
          <span class="file-name">{{ file.name }}</span>
          <span v-if="!isDirectory(file)" class="file-size">
            {{ formatSize(file.size) }}
          </span>
        </div>
        <button class="file-download-btn" @click.stop="$emit('download', file)">
          <i class="el-icon-download"></i>
        </button>
      </div>
    </div>
  </div>
</template>

<script>
import { formatFileSize, getFileIconClass } from '@/utils/util';

export default {
  name: 'FileTree',
  props: {
    files: {
      type: Array,
      default: () => [],
    },
    currentPath: {
      type: String,
      default: '',
    },
    workspaceInfo: {
      type: Object,
      default: () => ({}),
    },
  },
  computed: {
    pathParts() {
      return this.currentPath
        ? this.currentPath.split('/').filter(Boolean)
        : [];
    },
  },
  methods: {
    isDirectory(file) {
      return file.type === 'directory' || file.type === 'dir' || file.isDir;
    },

    getFileIconClass,

    formatSize: formatFileSize,

    navigateTo(path) {
      this.$emit('navigate', path);
    },

    navigateToParent() {
      const parts = this.currentPath.split('/').filter(Boolean);
      parts.pop();
      this.navigateTo(parts.join('/'));
    },

    getPathByIndex(index) {
      return this.pathParts.slice(0, index + 1).join('/');
    },

    handleFileClick(file) {
      if (this.isDirectory(file)) {
        const newPath = this.currentPath
          ? `${this.currentPath}/${file.name}`
          : file.name;
        this.navigateTo(newPath);
      } else {
        this.$emit('preview', file);
      }
    },
  },
};
</script>

<style scoped>
.file-tree {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  padding: 12px 16px;
  background: #f5f7fa;
  border-bottom: 1px solid #e4e7ed;
  font-size: 13px;
}

.breadcrumb-item {
  cursor: pointer;
  color: #606266;
  transition: color 0.2s;
  display: flex;
  align-items: center;
  gap: 4px;
}

.breadcrumb-item:hover {
  color: #10a37f;
}

.breadcrumb-item.active {
  color: #10a37f;
  font-weight: 500;
}

.breadcrumb-sep {
  margin: 0 8px;
  color: #c0c4cc;
}

.tree-info {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 16px;
  background: #fafafa;
  border-bottom: 1px solid #e4e7ed;
  font-size: 12px;
  color: #909399;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.info-divider {
  color: #dcdfe6;
}

.file-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.file-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 16px;
  cursor: pointer;
  transition: background-color 0.2s;
}

.file-item:hover {
  background-color: #f5f7fa;
}

.file-item.is-directory {
  color: #10a37f;
}

.file-item.back-item {
  color: #909399;
}

.file-item-main {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.file-name {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 14px;
}

.file-size {
  color: #909399;
  font-size: 12px;
  margin-left: auto;
  padding-left: 16px;
}

.file-download-btn {
  margin-left: 4px;
  padding: 4px 8px;
  background: transparent;
  border: 1px solid #dcdfe6;
  border-radius: 4px;
  cursor: pointer;
  opacity: 0;
  transition: all 0.2s;
  color: #606266;
}

.file-item:hover .file-download-btn {
  opacity: 1;
}

.file-download-btn:hover {
  color: #10a37f;
  border-color: #10a37f;
}
</style>
