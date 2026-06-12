<template>
  <div class="mcp-management">
    <div class="common_bg">
      <!-- tabs -->
      <div class="tabs tabs-x-top">
        <div :class="['tab', { active: tabActive === 0 }]" @click="tabClick(0)">
          {{ $t('common.button.import') }}MCP
        </div>
        <div :class="['tab', { active: tabActive === 1 }]" @click="tabClick(1)">
          {{ $t('common.button.add') }}MCP
        </div>
      </div>

      <customize ref="customize" v-if="tabActive === 0" />
      <server ref="server" v-if="tabActive === 1" />
    </div>
  </div>
</template>
<script>
import customize from './integrate';
import server from './server';
export default {
  name: 'McpTabs',
  data() {
    return {
      tabActive: 0,
      mcpTabObj: {
        integrate: 0,
        server: 1,
      },
    };
  },
  watch: {
    $route: {
      handler(val) {
        // keep-alive 下组件不会销毁，离开当前页面 watcher 仍然触发触发，需忽略非当前页面的路由
        if (val.path === '/mcpService') this.setInitTab();
      },
      deep: true,
    },
  },
  mounted() {
    this.setInitTab();
  },
  methods: {
    setInitTab() {
      const { mcp } = this.$route.query || {};
      this.tabActive = this.mcpTabObj[mcp] || 0;
    },
    tabClick(status) {
      this.tabActive = status;
    },
  },
  components: {
    customize,
    server,
  },
};
</script>
<style lang="scss" scoped>
::v-deep .scroll-card-container {
  max-height: calc(100vh - 160px);
}
</style>
