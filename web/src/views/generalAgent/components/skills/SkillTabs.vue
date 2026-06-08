<template>
  <aside class="skill-workspace-container">
    <div class="workspace-header">
      <div class="workspace-title">
        <i class="el-icon-folder-opened"></i>
        <span>
          {{ $t('generalAgent.skill.panel.skillWorkspace') }}
        </span>
      </div>
      <div class="header-actions">
        <AppPublishActions
          :appId="skillPreviewParams.customSkillId"
          :appType="SKILL"
          :appName="assistantInfo.name"
          :publishType="publishType"
          @reload-data="reloadData"
          @preview-version="previewVersion"
        />
      </div>
    </div>

    <div class="workspace-body" :class="{ 'disable-clicks': disableClick }">
      <SkillWorkspaceExplorer
        ref="explorer"
        :customSkillId="skillPreviewParams.customSkillId"
        :activeGitDiffId="activeGitDiffId"
        @open-file="openFile"
        @open-search-result="openSearchResult"
        @open-git-diff="openGitDiff"
        @close-tabs-by-path="closeTabsByPath"
        @discard-file="handleDiscardFile"
      />
      <SkillWorkbench
        ref="workbench"
        :skillPreviewParams="skillPreviewParams"
        @active-git-diff-change="activeGitDiffId = $event"
        @file-saved="handleFileSaved"
        @view-workspace="$emit('view-workspace', $event)"
        @download-all="$emit('download-all')"
      />
    </div>
  </aside>
</template>

<script>
import SkillWorkspaceExplorer from './SkillWorkspaceExplorer.vue';
import SkillWorkbench from './SkillWorkbench.vue';
import AppPublishActions from '@/components/appPublishActions.vue';
import { getCustomSkillInfo } from '@/api/templateSquare';
import { SKILL } from '@/utils/commonSet';

export default {
  name: 'SkillTabs',
  components: {
    SkillWorkspaceExplorer,
    SkillWorkbench,
    AppPublishActions,
  },
  props: {
    skillPreviewParams: {
      type: Object,
      default: () => ({}),
    },
  },
  data() {
    return {
      SKILL,
      publishType: '',
      disableClick: false,
      version: '',
      assistantInfo: {},
      activeGitDiffId: '',
    };
  },
  watch: {
    'skillPreviewParams.customSkillId': {
      handler(val) {
        if (val) {
          this.getAppDetail();
        }
      },
      immediate: true,
    },
  },
  methods: {
    openFile(file) {
      if (this.$refs.workbench) {
        this.$refs.workbench.openFile(file);
      }
    },
    openSearchResult(payload) {
      if (this.$refs.workbench) {
        this.$refs.workbench.openSearchResult(payload);
      }
    },
    openGitDiff(payload) {
      if (this.$refs.workbench) {
        this.$refs.workbench.openGitDiff(payload);
      }
    },
    refreshFiles() {
      if (this.$refs.explorer) {
        this.$refs.explorer.refreshFiles();
      }
    },
    closeTabsByPath(path) {
      if (this.$refs.workbench) {
        this.$refs.workbench.closeTabsByPath(path);
      }
    },
    async handleDiscardFile(payload) {
      if (this.$refs.workbench) {
        await this.$refs.workbench.refreshOpenedFileByPath(payload);
      }
    },
    async refreshWorkspace() {
      this.refreshFiles();
      if (this.$refs.workbench) {
        await this.$refs.workbench.refreshOpenedFiles({ force: true });
      }
      if (this.$refs.explorer) {
        this.refreshGit();
      }
    },
    refreshGit() {
      if (this.$refs.explorer) {
        this.$refs.explorer.refreshGit();
      }
    },
    hasUnsavedFiles() {
      return this.$refs.workbench
        ? this.$refs.workbench.hasUnsavedFiles()
        : false;
    },
    async discardUnsavedFiles() {
      if (!this.$refs.workbench) return [];
      return this.$refs.workbench.discardUnsavedFiles();
    },
    reloadData() {
      this.disableClick = false;
      this.getAppDetail();
      this.$emit('refresh-workspace');
      this.refreshWorkspace();
    },
    previewVersion(item) {
      this.disableClick = !item.isCurrent;
      this.version = item.version || '';
      this.getAppDetail();
    },
    handleFileSaved() {
      this.refreshGit();
    },
    async getAppDetail() {
      const params = {
        skillId: this.skillPreviewParams.customSkillId,
      };
      const res = await getCustomSkillInfo(params);

      if (res.code === 0 && res.data) {
        this.assistantInfo = res.data;
        this.publishType = res.data.publishType;
      }
    },
  },
};
</script>

<style lang="scss" scoped>
@import '../../styles/variables';

.skill-workspace-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  min-width: 0;
  background: #fff;
  border-left: 1px solid #f0f0f0;
  position: relative;
  z-index: 10;
}

.workspace-header {
  height: $header-height;
  position: relative;
  padding: 0 16px;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  flex-shrink: 0;

  .workspace-title {
    display: flex;
    align-items: center;
    gap: 8px;
    min-width: 0;
    color: #333;
    font-size: 14px;
    font-weight: 600;

    i {
      color: #10a37f;
      font-size: 16px;
      flex-shrink: 0;
    }
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;
    flex-shrink: 0;
  }
}

.workspace-body {
  flex: 1;
  min-height: 0;
  min-width: 0;
  overflow: hidden;
  position: relative;
  display: flex;
  background: #fff;

  &.disable-clicks {
    pointer-events: none;
    opacity: 0.7;
    filter: grayscale(0.5);
  }
}
</style>
