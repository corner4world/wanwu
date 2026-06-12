<template>
  <div class="page-wrapper">
    <div class="page-title">
      <i class="el-icon-arrow-left" @click="$router.go(-1)" />
      <img class="page-title-img" src="@/assets/imgs/org.png" alt="" />
      <span class="page-title-name">{{ $t('menu.setting') }}</span>
    </div>
    <!-- tabs: UI 改版统计分析、OAuth 提出到菜单，无需切换 tab -->
    <div class="tabs tabs-spacing" v-if="checkPerm(settingPerm)">
      <div :class="['tab', { active: tabActive === 0 }]" @click="tabClick(0)">
        {{ $t('org.title') }}
      </div>
      <div :class="['tab', { active: tabActive === 1 }]" @click="tabClick(1)">
        {{ $t('infoSetting.title') }}
      </div>
    </div>

    <div v-if="tabActive === 0" style="margin: 0 20px">
      <div style="margin-bottom: -30px">
        <span
          v-for="item in list"
          :key="item.key"
          :class="['tab-span', { 'is-active': radio === item.key }]"
          v-if="checkPerm(item.perm)"
          @click="changeTab(item.key)"
        >
          {{ item.name }}
        </span>
      </div>
      <User v-if="radio === 'user'" />
      <Role v-if="radio === 'role'" />
      <Org v-if="radio === 'org'" />
    </div>
    <div v-if="tabActive === 1" style="margin: 16px 20px 0 20px">
      <InfoSetting />
    </div>
  </div>
</template>

<script>
import User from './user/index.vue';
import Role from './role/index.vue';
import Org from './org/index.vue';
import InfoSetting from '@/views/infoSetting/index.vue';
import { checkPerm, PERMS } from '@/router/permission';

export default {
  name: 'Permission',
  components: { User, Role, Org, InfoSetting },
  data() {
    return {
      radio: '',
      tabActive: 0,
      settingPerm: PERMS.SETTING,
      statisticsPerm: PERMS.STATISTIC,
      oauthPerm: PERMS.OAUTH,
      list: [
        {
          name: this.$t('user.name'),
          key: 'user',
          perm: PERMS.PERMISSION_USER,
        },
        {
          name: this.$t('role.name'),
          key: 'role',
          perm: PERMS.PERMISSION_ROLE,
        },
        { name: this.$t('org.name'), key: 'org', perm: PERMS.PERMISSION_ORG },
      ],
    };
  },
  created() {
    for (let item of this.list) {
      if (checkPerm(item.perm)) {
        this.radio = item.key;
        break;
      }
    }
  },
  methods: {
    checkPerm,
    changeTab(key) {
      this.radio = key;
    },
    tabClick(status) {
      this.tabActive = status;
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/tabs.scss';
.tab-span {
  display: inline-block;
  vertical-align: middle;
  padding: 6px 12px;
  border-radius: 6px;
  color: $color_title;
  cursor: pointer;
  margin-top: 10px;
}
.tab-span.is-active {
  color: $color;
  background: $color_opacity;
  font-weight: bold;
  border-radius: 16px;
}
.page-title {
  .el-icon-arrow-left {
    margin-right: 10px;
    font-size: 15px;
    cursor: pointer;
    color: $color_title;
  }
}
</style>
