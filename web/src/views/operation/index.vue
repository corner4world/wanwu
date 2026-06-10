<template>
  <div class="page-wrapper">
    <div class="page-title">
      <i class="el-icon-arrow-left" @click="$router.go(-1)" />
      <img
        class="page-title-img"
        src="@/assets/imgs/operationManage.svg"
        alt=""
      />
      <span class="page-title-name">{{ $t('menu.operationManage') }}</span>
    </div>
    <div
      class="tabs tabs-x-top"
      style="margin-bottom: -20px"
      v-if="checkPerm(operationPerm)"
    >
      <div
        :class="['tab', { active: tabActive === 0 }]"
        @click="tabClick(0)"
        v-if="checkPerm(oauthPerm)"
      >
        {{ $t('oauth.title') }}
      </div>
      <div
        :class="['tab', { active: tabActive === 1 }]"
        @click="tabClick(1)"
        v-if="checkPerm(statisticsPerm)"
      >
        {{ $t('statistics.title') }}
      </div>
    </div>

    <div v-if="tabActive === 0" style="margin: 0 20px 0 20px">
      <Oauth />
    </div>
    <div v-if="tabActive === 1" style="margin: 30px 20px 0 20px">
      <Statistics />
    </div>
  </div>
</template>

<script>
import Statistics from '@/views/permission/statistics';
import Oauth from '@/views/permission/oauth';
import { checkPerm, PERMS } from '@/router/permission';

export default {
  name: 'Operation',
  components: { Statistics, Oauth },
  data() {
    return {
      radio: '',
      tabActive: 0,
      operationPerm: PERMS.OPERATION,
      oauthPerm: PERMS.OAUTH,
      statisticsPerm: PERMS.STATISTIC,
    };
  },
  methods: {
    checkPerm,
    tabClick(status) {
      this.tabActive = status;
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/tabs.scss';
.page-title {
  .el-icon-arrow-left {
    margin-right: 10px;
    font-size: 15px;
    cursor: pointer;
    color: $color_title;
  }
}
</style>
