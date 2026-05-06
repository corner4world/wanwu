<template>
  <div class="mcp-management">
    <div class="common_bg">
      <!-- tabs -->
      <div class="tabs tabs-x-top">
        <div :class="['tab', { active: tabActive === 0 }]" @click="tabClick(0)">
          {{ $t('menu.app.builtIn') }}
        </div>
        <div :class="['tab', { active: tabActive === 1 }]" @click="tabClick(1)">
          {{ $t('menu.app.custom') }}
        </div>
      </div>

      <builtIn ref="builtIn" v-if="tabActive === 0" />
      <custom ref="custom" v-if="tabActive === 1" />
    </div>
  </div>
</template>
<script>
import builtIn from './builtIn';
import custom from './custom';
export default {
  data() {
    return {
      tabActive: 0,
      toolTabObj: {
        builtIn: 0,
        custom: 1,
      },
    };
  },
  watch: {
    $route: {
      handler() {
        this.setInitTab();
      },
      // 深度观察监听
      deep: true,
    },
  },
  mounted() {
    this.setInitTab();
  },
  methods: {
    setInitTab() {
      const { tool } = this.$route.query || {};
      this.tabActive = this.toolTabObj[tool] || 0;
    },
    tabClick(status) {
      this.tabActive = status;
    },
  },
  components: {
    builtIn,
    custom,
  },
};
</script>
<style lang="scss" scoped>
::v-deep .scroll-card-container {
  max-height: calc(100vh - 160px);
}
</style>
