<template>
  <div class="graph-page page-wrapper">
    <graphMap
      ref="graphMap"
      :data="graphData"
      :knowledge-id="knowledgeId"
      @goBack="goBack(`/knowledge/doclist/${knowledgeId}`)"
      @refresh="handleRefresh"
    />
  </div>
</template>

<script>
import { getGraphDetail } from '@/api/knowledge';
import { goBack } from '@/utils/util';
import graphMap from '@/components/graphMap.vue';

export default {
  name: 'KnowledgeGraph',
  components: {
    graphMap,
  },
  data() {
    return {
      knowledgeId: null,
      graphData: {
        nodes: [],
        edges: [],
        processingCount: 0,
        successCount: 0,
        failCount: 0,
      },
      loading: false,
    };
  },
  created() {
    this.knowledgeId = this.$route.params.id;
    this.getGraphData();
  },
  methods: {
    goBack,
    getGraphData() {
      if (!this.knowledgeId) return;

      this.loading = true;
      getGraphDetail({ knowledgeId: this.knowledgeId })
        .then(res => {
          this.loading = false;
          if (res.code === 0 && res.data) {
            this.graphData = res.data;
          }
        })
        .catch(() => {
          this.loading = false;
        });
    },
    handleRefresh(callback) {
      this.getGraphData();
      if (typeof callback === 'function') {
        setTimeout(() => {
          callback();
        }, 500);
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.graph-page {
  width: 100%;
  height: 100%;
  overflow: hidden;
}
</style>
