<template>
  <aside class="skill-tabs-container">
    <div class="tabs-header-wrapper">
      <el-tabs v-model="activeTab" class="skill-custom-tabs">
        <el-tab-pane
          :label="$t('generalAgent.skill.panel.preview')"
          name="preview"
        ></el-tab-pane>
        <el-tab-pane
          :label="$t('generalAgent.skill.panel.variableConfig')"
          name="config"
        ></el-tab-pane>
      </el-tabs>
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

    <div
      class="tabs-content-wrapper"
      :class="{ 'disable-clicks': disableClick }"
    >
      <keep-alive>
        <component
          :is="activeTabComponent"
          :skillPreviewParams="skillPreviewParams"
          v-bind="$attrs"
          v-on="$listeners"
        />
      </keep-alive>
    </div>
  </aside>
</template>

<script>
import PreviewChat from './preview.vue';
import SkillConfig from './config.vue';
import AppPublishActions from '@/components/appPublishActions.vue';
import { getCustomSkillInfo } from '@/api/templateSquare';
import { AGENT, SKILL } from '@/utils/commonSet';

export default {
  name: 'SkillTabs',
  components: {
    PreviewChat,
    SkillConfig,
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
      AGENT,
      SKILL,
      activeTab: 'preview',
      publishType: '',
      disableClick: false,
      version: '',
      assistantInfo: {},
    };
  },
  computed: {
    activeTabComponent() {
      return this.activeTab === 'preview' ? 'PreviewChat' : 'SkillConfig';
    },
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
    reloadData() {
      this.disableClick = false;
      this.getAppDetail();
      this.$emit('refresh-workspace');
    },
    previewVersion(item) {
      this.disableClick = !item.isCurrent;
      this.version = item.version || '';
      this.getAppDetail();
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
.skill-tabs-container {
  width: 650px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  min-height: 0;
  background: #fff;
  border-left: 1px solid #f0f0f0;
  position: relative;
  z-index: 10;
}

.tabs-header-wrapper {
  position: relative;
  padding: 0 16px;
  border-bottom: 1px solid #f0f0f0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;

  .skill-custom-tabs {
    flex: 1;
    ::v-deep .el-tabs__header {
      margin: 0;
    }
    ::v-deep .el-tabs__nav-wrap::after {
      display: none;
    }
    ::v-deep .el-tabs__item {
      height: 52px;
      line-height: 52px;
      font-size: 14px;
      font-weight: 500;
      color: #666;

      &.is-active {
        color: #10a37f;
      }
    }
    ::v-deep .el-tabs__active-bar {
      background-color: #10a37f;
      height: 2px;
    }
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 8px;

    .close-btn {
      font-size: 18px;
      color: #999;
      padding: 8px;
      &:hover {
        color: #666;
        background: #f5f5f5;
        border-radius: 4px;
      }
    }
  }
}

.tabs-content-wrapper {
  flex: 1;
  min-height: 0;
  overflow: hidden;
  position: relative;
  &.disable-clicks {
    pointer-events: none;
    opacity: 0.7;
    filter: grayscale(0.5);
  }
}
</style>
