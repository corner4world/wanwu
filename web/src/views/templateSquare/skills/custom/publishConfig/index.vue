<template>
  <div class="publish-config-container page-wrapper right-page-content-body">
    <div class="publish-config-title">
      <span class="el-icon-arrow-left goback" @click="goback"></span>
      <span class="publish-config-title-text">
        {{ name }} - {{ $t('agent.form.publishConfig') }}
      </span>
    </div>
    <CommonLayout
      :showAside="true"
      :showTitle="false"
      asideWidth="260px"
      class="publish-config-content"
    >
      <template #aside>
        <div class="tab-item active-tab">
          <h3>{{ $t('app.publishSetting') }}</h3>
          <p>{{ $t('tempSquare.skills.publishTypeDesc') }}</p>
        </div>
      </template>
      <template #main-content>
        <CreateScope ref="CreateScope" :appId="appId" :appType="appType" />
      </template>
    </CommonLayout>
  </div>
</template>

<script>
import CommonLayout from '@/components/exploreContainer.vue';
import CreateScope from './createScope.vue';
import { SKILL } from '@/views/templateSquare/constants';

export default {
  name: 'SkillPublishConfig',
  components: { CommonLayout, CreateScope },
  data() {
    return {
      name: '',
      appId: '',
      appType: SKILL,
    };
  },
  created() {
    const { appId, appType, name } = this.$route.query;
    this.appId = appId;
    this.appType = appType || SKILL;
    this.name = name;
  },
  methods: {
    goback() {
      this.$router.back();
    },
  },
};
</script>

<style lang="scss" scoped>
.publish-config-container {
  width: 100%;
  height: 100%;
  padding: 0 10px;
  .publish-config-title {
    width: 100%;
    height: 60px;
    padding: 0 20px;
    border-bottom: 1px solid #dbdbdb;
    display: flex;
    align-items: center;
    .goback {
      font-size: 20px;
      margin-right: 10px;
      cursor: pointer;
    }
    .publish-config-title-text {
      font-size: 18px;
      font-weight: bold;
    }
  }
  .publish-config-content {
    width: 100%;
    padding: 10px;
    height: calc(100% - 60px);
    .tab-item {
      cursor: default;
      border: 1px solid #dbdbdb;
      text-align: center;
      width: 90%;
      height: 80px;
      border-radius: 6px;
      padding: 10px;
      margin: 20px auto;
      h3 {
        font-size: 16px;
      }
      p {
        color: #666;
        padding-top: 10px;
      }
      &.active-tab {
        border: 1px solid $color !important;
      }
    }
  }
  ::v-deep .explore-container {
    .page-wrapper {
      min-height: 0 !important;
      padding-left: 0;
    }
  }
}
</style>
