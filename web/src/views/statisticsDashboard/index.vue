<template>
  <div class="page-wrapper">
    <div class="tabs tabs-spacing">
      <div
        v-for="item in tabList"
        :key="item.type"
        :class="['tab', { active: tabActive === item.type }]"
        @click="tabClick(item.type)"
      >
        {{ item.name }}
      </div>
    </div>

    <div v-if="tabActive === STATISTIC.APP">
      <App />
    </div>

    <div v-if="tabActive === STATISTIC.MODEL">
      <Model />
    </div>

    <div v-if="tabActive === STATISTIC.API">
      <API />
    </div>
  </div>
</template>

<script>
import Model from './components/model/model.vue';
import App from './components/app/app.vue';
import API from './components/api/api.vue';
import { STATISTIC } from './constants';

export default {
  components: { Model, App, API },
  data() {
    return {
      STATISTIC,
      radio: '',
      tabActive: STATISTIC.APP,
      tabList: [
        {
          name: this.$t('statisticsDashboard.app'),
          type: STATISTIC.APP,
        },
        {
          name: this.$t('statisticsDashboard.model'),
          type: STATISTIC.MODEL,
        },
        {
          name: 'API',
          type: STATISTIC.API,
        },
      ],
    };
  },
  methods: {
    tabClick(status) {
      this.tabActive = status;
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/tabs.scss';
::v-deep .scroll-card-container {
  max-height: calc(100vh - 200px);
}
</style>
