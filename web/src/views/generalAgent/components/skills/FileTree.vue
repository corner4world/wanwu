<template>
  <div class="file-tree-wrapper">
    <div class="file-tree-header">
      <div class="header-icons">
        <i
          :class="['header-icon', { active: activeView === 'files' }]"
          class="el-icon-folder"
          :title="$t('generalAgent.skill.skillWorkBench.common.files')"
          @click="$emit('switch-view', 'files')"
        ></i>
        <i
          :class="['header-icon', { active: activeView === 'search' }]"
          class="el-icon-search"
          :title="$t('generalAgent.skill.skillWorkBench.common.search')"
          @click="$emit('switch-view', 'search')"
        ></i>
        <svg-icon
          :class="[
            'header-icon svg-icon-btn',
            { active: activeView === 'git' },
          ]"
          icon-class="gitBranch"
          :title="$t('generalAgent.skill.skillWorkBench.common.git')"
          @click.native="$emit('switch-view', 'git')"
        />
      </div>
      <i
        class="el-icon-refresh header-icon"
        :title="$t('generalAgent.skill.skillWorkBench.common.refresh')"
        :class="{ spinning: manualLoading }"
        @click="refreshFiles"
      ></i>
    </div>
    <div class="file-tree-content" @scroll="hideContextMenu">
      <el-tree
        v-if="treeData.length > 0"
        :data="treeData"
        :props="treeProps"
        node-key="path"
        :expand-on-click-node="false"
        :default-expanded-keys="defaultExpandedKeys"
        @node-click="handleNodeClick"
        @node-contextmenu="handleNodeContextMenu"
        ref="fileTree"
      >
        <span
          class="custom-tree-node"
          slot-scope="{ node, data }"
          :class="{ 'is-placeholder': data.isEmptyPlaceholder }"
        >
          <i
            :class="getFileIcon(data).icon"
            class="file-icon"
            :style="{
              color: data.isEmptyPlaceholder ? '#999' : getFileIcon(data).color,
            }"
          ></i>
          <el-tooltip :content="node.label" placement="top" :open-delay="300">
            <span class="file-name">{{ node.label }}</span>
          </el-tooltip>
        </span>
      </el-tree>
      <div v-else class="empty-state">
        <i class="el-icon-folder-opened"></i>
        <p>
          {{
            manualLoading
              ? $t('generalAgent.skill.skillWorkBench.fileTree.loading')
              : $t('generalAgent.skill.skillWorkBench.fileTree.empty')
          }}
        </p>
      </div>
    </div>

    <!-- 右键菜单-->
    <el-dropdown
      ref="contextDropdown"
      trigger="click"
      size="mini"
      class="context-menu-dropdown"
      placement="bottom-start"
      @command="handleContextMenuCommand"
      :style="{ left: contextMenuX + 'px', top: contextMenuY + 'px' }"
    >
      <span ref="contextTrigger" class="dropdown-trigger-node"></span>
      <el-dropdown-menu slot="dropdown" class="file-tree-dropdown-menu">
        <el-dropdown-item command="download">
          {{ $t('common.button.download') }}
        </el-dropdown-item>
        <el-dropdown-item command="delete" class="text-danger">
          {{ $t('common.button.delete') }}
        </el-dropdown-item>
      </el-dropdown-menu>
    </el-dropdown>
  </div>
</template>

<script>
import { getSkillWorkspaceFiles } from '@/api/skillResource/skillWorkSpace';
import { getFileIcon } from '@/utils/fileIcons';

export default {
  name: 'FileTree',
  props: {
    customSkillId: {
      type: String,
      required: true,
    },
    activeView: {
      type: String,
      default: 'files',
    },
    polling: {
      type: Boolean,
      default: false,
    },
    refreshInterval: {
      type: Number,
      default: 2000,
    },
  },
  data() {
    return {
      treeData: [],
      manualLoading: false,
      pollingTimer: null,
      contextMenuX: 0,
      contextMenuY: 0,
      contextMenuTarget: null,
      defaultExpandedKeys: [], // 保存展开的节点路径
      treeProps: {
        children: 'children',
        label: 'name',
      },
    };
  },
  mounted() {
    this.fetchFiles(true);
    window.addEventListener('resize', this.hideContextMenu);
  },
  beforeDestroy() {
    this.stopPolling();
    window.removeEventListener('resize', this.hideContextMenu);
  },
  methods: {
    async fetchFiles(showLoading = false) {
      if (!this.customSkillId) return;
      if (showLoading) this.manualLoading = true;
      try {
        const res = await getSkillWorkspaceFiles(this.customSkillId);
        if (res.code === 0 && res.data) {
          // 预处理：为空目录添加占位子节点，以便显示展开箭头
          const newTreeData = this.processTreeData(res.data.files || []);

          // 保存当前展开状态
          this.saveExpandedState();

          // 更新数据
          this.treeData = newTreeData;

          // 如果是首次加载且没有保存的展开状态，默认展开所有目录
          if (showLoading && this.defaultExpandedKeys.length === 0) {
            this.$nextTick(() => {
              this.expandAllDirectories();
            });
          }
        }
      } catch (error) {
        console.error('Failed to fetch files:', error);
      } finally {
        if (showLoading) this.manualLoading = false;
      }
    },
    // 保存当前展开状态
    saveExpandedState() {
      if (this.$refs.fileTree) {
        const nodesMap = this.$refs.fileTree.store.nodesMap;
        this.defaultExpandedKeys = [];
        for (const path in nodesMap) {
          if (nodesMap[path].expanded && !nodesMap[path].isLeaf) {
            this.defaultExpandedKeys.push(path);
          }
        }
      }
    },
    // 展开所有目录
    expandAllDirectories() {
      const expandedPaths = [];
      const collectPaths = nodes => {
        if (!Array.isArray(nodes)) return;
        nodes.forEach(node => {
          if (node.isDir) {
            expandedPaths.push(node.path);
            if (node.children) {
              collectPaths(node.children);
            }
          }
        });
      };
      collectPaths(this.treeData);
      this.defaultExpandedKeys = expandedPaths;
    },
    // 递归处理树数据，为空目录添加占位节点
    processTreeData(nodes) {
      if (!Array.isArray(nodes)) return nodes;

      return nodes.map(node => {
        if (node.isDir) {
          const children = node.children || [];
          if (children.length === 0) {
            // 空目录：添加一个隐藏的占位节点
            return {
              ...node,
              children: [
                {
                  path: `${node.path}/.empty`,
                  name: this.$t(
                    'generalAgent.skill.skillWorkBench.fileTree.emptyDir',
                  ),
                  isDir: false,
                  isEmptyPlaceholder: true,
                },
              ],
            };
          } else {
            return {
              ...node,
              children: this.processTreeData(children),
            };
          }
        }
        return node;
      });
    },
    refreshFiles() {
      this.fetchFiles(true);
    },
    startPolling() {
      if (!this.customSkillId) {
        return;
      }
      this.stopPolling();
      this.pollingTimer = setInterval(() => {
        this.fetchFiles(false); // 轮询不显示 loading
      }, this.refreshInterval);
    },
    stopPolling() {
      if (this.pollingTimer) {
        clearInterval(this.pollingTimer);
        this.pollingTimer = null;
      }
    },
    handleNodeContextMenu(event, data) {
      event.preventDefault();
      event.stopPropagation();
      if (!data || data.isEmptyPlaceholder) {
        this.hideContextMenu();
        return;
      }

      this.contextMenuTarget = data;
      // 稍微偏移，确保菜单在鼠标右下方，且不被光标完全遮挡
      this.contextMenuX = event.clientX + 2;
      this.contextMenuY = event.clientY + 2;

      this.$nextTick(() => {
        if (this.$refs.contextTrigger) {
          this.$refs.contextTrigger.click();
        }
      });
    },
    handleContextMenuCommand(command) {
      if (command === 'download') {
        if (!this.contextMenuTarget) return;
        this.$emit('download-file', this.contextMenuTarget);
      } else if (command === 'delete') {
        if (!this.contextMenuTarget) return;
        this.$emit('delete-file', this.contextMenuTarget);
      }
      this.hideContextMenu();
    },
    hideContextMenu() {
      const dropdown = this.$refs.contextDropdown;
      if (dropdown && dropdown.visible) {
        dropdown.hide();
      }
      this.contextMenuTarget = null;
    },
    handleNodeClick(data, node) {
      this.hideContextMenu();
      // 忽略空目录占位节点点击
      if (data.isEmptyPlaceholder) {
        return;
      }
      if (data.isDir) {
        // 点击目录时切换展开/折叠状态
        if (node.expanded) {
          node.collapse();
        } else {
          node.expand();
        }
      } else {
        this.$emit('file-click', data);
      }
    },
    getFileIcon(data) {
      if (data.isEmptyPlaceholder) {
        return { icon: 'el-icon-document', color: '#999' };
      }
      if (data.isDir) return { icon: 'el-icon-folder', color: '#dcb67a' };
      return getFileIcon(data.name);
    },
  },
  watch: {
    customSkillId(newVal, oldVal) {
      if (newVal !== oldVal) {
        this.treeData = [];
        this.defaultExpandedKeys = []; // 重置展开状态
        if (newVal) {
          this.fetchFiles(true);
          // 如果 polling 已经是 true，启动轮询
          if (this.polling) {
            this.startPolling();
          }
        }
      }
    },
    polling: {
      handler(newVal) {
        if (newVal) {
          this.startPolling();
        } else {
          this.stopPolling();
        }
      },
      immediate: true,
    },
  },
};
</script>

<style lang="scss" scoped>
.file-tree-wrapper {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #f3f3f3;
  color: #333;

  .file-tree-header {
    padding: 6px 8px;
    display: flex;
    justify-content: space-between;
    align-items: center;
    border-bottom: 1px solid #e0e0e0;
    background: #f8f8f8;

    .header-icons {
      display: flex;
      gap: 4px;
    }

    .header-icon {
      width: 24px;
      height: 24px;
      display: flex;
      align-items: center;
      justify-content: center;
      font-size: 15px;
      color: #666;
      cursor: pointer;
      border-radius: 4px;

      &:hover {
        color: #444;
        background: rgba(0, 0, 0, 0.05);
      }
      &.active {
        color: #5983ff;
        background: rgba(89, 131, 255, 0.1);
      }
      &.spinning {
        animation: spin 0.6s linear;
      }

      &.svg-icon-btn {
        font-size: 15px;
        ::v-deep svg {
          width: 15px;
          height: 15px;
        }
      }
    }
  }

  .file-tree-content {
    flex: 1;
    overflow-y: auto;
    padding: 4px 0;

    ::v-deep .el-tree {
      background: transparent;
      color: #333;

      .el-tree-node__content {
        height: 22px;
        line-height: 22px;

        &:hover {
          background-color: #e8e8e8;
        }
      }

      .el-tree-node.is-current > .el-tree-node__content {
        background-color: rgba(89, 131, 255, 0.12);
      }

      // 文件节点（非目录）隐藏展开箭头
      .el-tree-node.is-leaf .el-tree-node__expand-icon {
        display: none;
      }

      .el-tree-node__expand-icon {
        color: #666;
        font-size: 12px;
        &.is-leaf {
          display: none;
        }
      }

      .custom-tree-node {
        display: flex;
        align-items: center;
        font-size: 13px;
        min-width: 0;
        width: 100%;

        .file-icon {
          margin-right: 4px;
          font-size: 14px;
          flex-shrink: 0;
        }

        .file-name {
          display: block;
          min-width: 0;
          flex: 1;
          overflow: hidden;
          text-overflow: ellipsis;
          white-space: nowrap;
        }

        // 空目录占位节点样式
        &.is-placeholder {
          opacity: 0.6;
          font-style: italic;
          color: #999;
          cursor: default;

          .file-icon {
            display: none;
          }
        }
      }
    }

    .empty-state {
      display: flex;
      flex-direction: column;
      align-items: center;
      justify-content: center;
      height: 200px;
      color: #999;

      i {
        font-size: 48px;
        margin-bottom: 12px;
      }
      p {
        font-size: 13px;
        margin: 0;
      }
    }
  }

  .context-menu-dropdown {
    position: fixed;
    visibility: hidden;
    pointer-events: none;

    .dropdown-trigger-node {
      display: block;
      width: 1px;
      height: 1px;
    }
  }
}

::v-deep .file-tree-dropdown-menu.el-dropdown-menu {
  padding: 4px 0;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);

  .el-dropdown-menu__item {
    font-size: 13px;
    line-height: 32px;
    padding: 0 16px;
    display: flex;
    align-items: center;
    gap: 8px;

    i {
      margin: 0;
      font-size: 14px;
    }
    &:hover {
      background-color: #f5f7fa;
      color: #5983ff;
    }
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

.text-danger {
  color: #f56c6c !important;
}
</style>
