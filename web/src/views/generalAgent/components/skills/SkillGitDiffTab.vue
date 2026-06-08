<template>
  <div class="skill-git-diff-tab">
    <div v-if="loading" class="diff-loading">
      <i class="el-icon-loading"></i>
      <span>{{ $t('generalAgent.skill.skillWorkBench.diff.loading') }}</span>
    </div>

    <template v-else-if="diff.type === 'commit'">
      <div class="diff-file-list-header">
        <div class="diff-file-list-info">
          <span class="diff-commit-msg">{{ diff.commit.message }}</span>
          <span class="diff-commit-hash">
            {{ diff.commit.hash.substring(0, 7) }}
          </span>
        </div>
        <div class="diff-toolbar">
          <el-tooltip :content="diffModeTooltip" placement="bottom">
            <i
              :class="
                diffMode === 'side-by-side'
                  ? 'el-icon-notebook-2'
                  : 'el-icon-s-operation'
              "
              class="diff-mode-icon"
              @click="toggleDiffMode"
            ></i>
          </el-tooltip>
        </div>
      </div>
      <div class="diff-file-list">
        <div
          v-for="file in diff.changedFiles"
          :key="file.path"
          class="diff-file-group"
        >
          <div class="diff-file-header" @click="toggleDiffFile(file.path)">
            <i
              :class="
                expandedFiles[file.path]
                  ? 'el-icon-arrow-down'
                  : 'el-icon-arrow-right'
              "
              class="expand-icon"
            ></i>
            <span :class="['change-type-badge', file.changeType]">
              {{ file.changeType[0].toUpperCase() }}
            </span>
            <span class="diff-file-name">{{ file.path }}</span>
          </div>
          <div v-if="expandedFiles[file.path]" class="diff-file-content">
            <MonacoDiffEditor
              :original="fileDiffs[file.path]?.original || ''"
              :modified="fileDiffs[file.path]?.modified || ''"
              :language="getDiffLanguage(file.path)"
              :diffMode="diffMode"
            />
          </div>
        </div>
        <div v-if="diff.changedFiles.length === 0" class="diff-empty">
          {{ $t('generalAgent.skill.skillWorkBench.diff.noChangedFiles') }}
        </div>
      </div>
    </template>

    <template v-else>
      <div class="diff-main-header">
        <div class="diff-file-info">
          <span
            v-if="diff.file"
            :class="['change-type-badge', diff.file.changeType]"
          >
            {{ diff.file.changeType[0].toUpperCase() }}
          </span>
          <span v-if="diff.file" class="diff-file-name">
            {{ diff.file.path }}
          </span>
          <span v-if="diff.file && diff.file.staged" class="diff-label staged">
            {{ $t('generalAgent.skill.skillWorkBench.diff.staged') }}
          </span>
          <span v-else-if="diff.file" class="diff-label unstaged">
            {{ $t('generalAgent.skill.skillWorkBench.diff.unstaged') }}
          </span>
        </div>
        <div class="diff-toolbar">
          <el-tooltip :content="diffModeTooltip" placement="bottom">
            <i
              :class="
                diffMode === 'side-by-side'
                  ? 'el-icon-notebook-2'
                  : 'el-icon-s-operation'
              "
              class="diff-mode-icon"
              @click="toggleDiffMode"
            ></i>
          </el-tooltip>
        </div>
      </div>
      <div class="diff-editor-container">
        <MonacoDiffEditor
          :original="diff.original || ''"
          :modified="diff.modified || ''"
          :language="getDiffLanguage(diff.file ? diff.file.path : '')"
          :diffMode="diffMode"
        />
      </div>
    </template>
  </div>
</template>

<script>
import MonacoDiffEditor from '@/components/MonacoDiffEditor/index.vue';
import { getSkillWorkspaceGitFile } from '@/api/skillResource/skillWorkSpace';
import { getLanguageByPath } from './workspaceConstants';

export default {
  name: 'SkillGitDiffTab',
  components: {
    MonacoDiffEditor,
  },
  props: {
    customSkillId: {
      type: String,
      required: true,
    },
    diff: {
      type: Object,
      required: true,
    },
  },
  data() {
    return {
      diffMode: 'side-by-side',
      expandedFiles: {},
      fileDiffs: {},
      loadingFiles: {},
    };
  },
  computed: {
    loading() {
      return !!this.diff.loading;
    },
    diffModeTooltip() {
      return this.diffMode === 'side-by-side'
        ? this.$t('generalAgent.skill.skillWorkBench.diff.switchInline')
        : this.$t('generalAgent.skill.skillWorkBench.diff.switchSideBySide');
    },
  },
  watch: {
    'diff.id'() {
      this.expandedFiles = {};
      this.fileDiffs = {};
    },
  },
  methods: {
    toggleDiffMode() {
      this.diffMode =
        this.diffMode === 'side-by-side' ? 'inline' : 'side-by-side';
    },
    toggleDiffFile(path) {
      const expanded = !this.expandedFiles[path];
      this.$set(this.expandedFiles, path, expanded);
      if (expanded && !this.fileDiffs[path]) {
        this.loadCommitFileDiff(path);
      }
    },
    async loadCommitFileDiff(path) {
      if (!this.diff.commit || !this.customSkillId) return;
      const file = this.diff.changedFiles.find(item => item.path === path);
      if (!file) return;

      const isRenamed = file.changeType === 'renamed';
      const oldFilePath = isRenamed && file.oldPath ? file.oldPath : path;
      const isNew = file.changeType === 'added';
      const isDeleted = file.changeType === 'deleted';

      try {
        this.$set(this.loadingFiles, path, true);
        const [oldRes, newRes] = await Promise.all([
          isNew
            ? Promise.resolve({ code: 0, data: { content: '' } })
            : getSkillWorkspaceGitFile(this.customSkillId, {
                commitHash: this.diff.commit.hash + '~1',
                filePath: oldFilePath,
              }),
          isDeleted
            ? Promise.resolve({ code: 0, data: { content: '' } })
            : getSkillWorkspaceGitFile(this.customSkillId, {
                commitHash: this.diff.commit.hash,
                filePath: path,
              }),
        ]);
        const original = oldRes && oldRes.data ? oldRes.data.content || '' : '';
        const modified = newRes && newRes.data ? newRes.data.content || '' : '';
        this.$set(this.fileDiffs, path, { original, modified });
      } catch (e) {
        console.error('loadCommitFileDiff error', e);
        this.$set(this.fileDiffs, path, { original: '', modified: '' });
      } finally {
        this.$delete(this.loadingFiles, path);
      }
    },
    getDiffLanguage(path) {
      return getLanguageByPath(path);
    },
  },
};
</script>

<style lang="scss" scoped>
.skill-git-diff-tab {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;

  .diff-loading,
  .diff-empty {
    flex: 1;
    display: flex;
    align-items: center;
    justify-content: center;
    gap: 8px;
    color: #888;
    font-size: 13px;
  }

  .diff-main-header,
  .diff-file-list-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 6px 16px;
    background: #f8f8f8;
    border-bottom: 1px solid #e0e0e0;
    flex-shrink: 0;
  }

  .diff-file-info,
  .diff-file-list-info {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    overflow: hidden;
  }

  .diff-file-name,
  .diff-commit-msg {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: #333;
    font-size: 13px;
  }

  .diff-file-name {
    font-family: 'Menlo', 'Monaco', 'Courier New', monospace;
  }

  .diff-commit-hash {
    font-size: 12px;
    font-family: monospace;
    color: #888;
    flex-shrink: 0;
  }

  .diff-toolbar {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;

    .diff-mode-icon {
      font-size: 14px;
      color: #666;
      cursor: pointer;
      padding: 4px;
      border-radius: 4px;

      &:hover {
        color: #333;
        background: rgba(0, 0, 0, 0.06);
      }
    }
  }

  .diff-editor-container {
    flex: 1;
    overflow: hidden;
  }

  .diff-file-list {
    flex: 1;
    overflow-y: auto;

    .diff-file-group {
      border-bottom: 1px solid #f0f0f0;
    }

    .diff-file-header {
      display: flex;
      align-items: center;
      padding: 6px 12px;
      cursor: pointer;
      background: #fafafa;

      &:hover {
        background: #f0f0f0;
      }

      .expand-icon {
        font-size: 12px;
        color: #666;
        margin-right: 6px;
        flex-shrink: 0;
      }

      .diff-file-name {
        font-size: 12px;
        color: #333;
      }
    }

    .diff-file-content {
      height: 300px;
      overflow: hidden;
    }
  }

  .change-type-badge {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 16px;
    height: 16px;
    font-size: 10px;
    font-weight: bold;
    border-radius: 3px;
    margin-right: 8px;
    padding: 0 2px;
    flex-shrink: 0;

    &.added {
      background: #e6ffed;
      color: #22863a;
    }
    &.modified {
      background: #fff8c5;
      color: #b08800;
    }
    &.deleted {
      background: #ffebe9;
      color: #cb2431;
    }
    &.renamed {
      background: #ddf4ff;
      color: #0969da;
    }
    &.untracked {
      background: #f0f0f0;
      color: #666;
    }
  }

  .diff-label {
    font-size: 11px;
    padding: 1px 6px;
    border-radius: 3px;
    flex-shrink: 0;

    &.staged {
      background: #e6ffed;
      color: #22863a;
    }
    &.unstaged {
      background: #fff8c5;
      color: #b08800;
    }
  }
}
</style>
