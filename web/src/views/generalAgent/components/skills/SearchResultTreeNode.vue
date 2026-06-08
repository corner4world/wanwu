<template>
  <div class="tree-node">
    <div class="tree-node-header" :style="headerStyle" @click="handleNodeClick">
      <i
        :class="getNodeIconClass(node)"
        :style="getNodeIconStyle(node)"
        class="node-icon"
      ></i>
      <span class="node-name">{{ node.name }}</span>
      <span v-if="!node.isDir && node.matches" class="match-count">
        ({{ node.matches.length }})
      </span>
      <button
        v-if="node.isDir"
        type="button"
        class="expand-button"
        @click.stop="$emit('toggle-node', node)"
      >
        <i
          :class="[
            'expand-icon',
            node.expanded ? 'el-icon-arrow-down' : 'el-icon-arrow-right',
          ]"
        ></i>
      </button>
    </div>

    <div
      v-if="node.isDir && node.expanded && node.children"
      class="tree-children"
    >
      <SearchResultTreeNode
        v-for="child in node.children"
        :key="child.path"
        :node="child"
        :level="level + 1"
        @toggle-node="$emit('toggle-node', $event)"
        @result-click="$emit('result-click', $event)"
        @match-click="$emit('match-click', $event)"
      />
    </div>

    <div v-if="!node.isDir && node.expanded && node.matches" class="match-list">
      <div
        v-for="(match, idx) in node.matches"
        :key="idx"
        class="match-item"
        :style="matchItemStyle"
        @click="$emit('match-click', match)"
      >
        <span class="match-line">:{{ match.line }}</span>
        <span class="match-content">{{ match.content.trim() }}</span>
      </div>
    </div>
  </div>
</template>

<script>
import { getFileIcon } from '@/utils/fileIcons';

export default {
  name: 'SearchResultTreeNode',
  props: {
    node: {
      type: Object,
      required: true,
    },
    level: {
      type: Number,
      default: 0,
    },
  },
  computed: {
    headerStyle() {
      return {
        paddingLeft: `${8 + this.level * 12}px`,
      };
    },
    matchItemStyle() {
      return {
        paddingLeft: `${28 + this.level * 12}px`,
      };
    },
  },
  methods: {
    handleNodeClick() {
      if (this.node.isDir) {
        this.$emit('toggle-node', this.node);
      } else {
        this.$emit('result-click', this.node);
      }
    },
    getNodeIconClass(node) {
      if (node.isDir) {
        return node.expanded ? 'el-icon-folder-opened' : 'el-icon-folder';
      }
      return getFileIcon(node.name).icon;
    },
    getNodeIconStyle(node) {
      return {
        color: node.isDir ? '#dcb67a' : getFileIcon(node.name).color,
      };
    },
  },
};
</script>

<style lang="scss" scoped>
.tree-node-header {
  display: flex;
  align-items: center;
  padding: 4px 8px;
  cursor: pointer;
  font-size: 12px;
  color: #333;
  min-width: 0;

  &:hover {
    background: #e8e8e8;
  }

  .node-icon {
    margin-right: 4px;
    font-size: 14px;
    flex-shrink: 0;
  }

  .node-name {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .match-count {
    color: #888;
    font-size: 11px;
    margin-right: 4px;
  }

  .expand-button {
    width: 22px;
    height: 22px;
    padding: 0;
    border: 0;
    border-radius: 3px;
    background: transparent;
    color: #666;
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;

    &:hover {
      background: #dedede;
      color: #333;
    }
  }

  .expand-icon {
    font-size: 12px;
    line-height: 1;
  }
}

.match-list {
  background: #fafafa;

  .match-item {
    display: flex;
    align-items: flex-start;
    padding: 3px 8px 3px 28px;
    cursor: pointer;
    font-size: 11px;

    &:hover {
      background: #e0e0e0;
    }

    .match-line {
      color: #5983ff;
      flex-shrink: 0;
      margin-right: 4px;
    }

    .match-content {
      color: #666;
      overflow: hidden;
      text-overflow: ellipsis;
      white-space: nowrap;
      flex: 1;
    }
  }
}
</style>
