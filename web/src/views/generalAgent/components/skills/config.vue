<template>
  <div class="skill-config">
    <div v-loading="loading" class="config-body">
      <ApiKeyTable
        :dataList="detailData.variables || []"
        @create-variable="handleCreateVariable"
        @update-variable="handleUpdateVariable"
        @delete-variable="handleDeleteVariable"
      />
    </div>
  </div>
</template>

<script>
import ApiKeyTable from '@/components/skills/ApiKeyTable.vue';
import {
  createCustomSkillConfig,
  deleteCustomSkillConfig,
  getCustomSkillInfo,
  updateCustomSkillConfig,
} from '@/api/templateSquare';

export default {
  name: 'SkillConfig',
  components: {
    ApiKeyTable,
  },
  props: {
    skillPreviewParams: {
      type: Object,
      default: () => ({}),
    },
  },
  data() {
    return {
      detailData: {},
      loading: false,
    };
  },
  watch: {
    'skillPreviewParams.customSkillId': {
      handler() {
        this.initDetailData();
      },
      immediate: true,
    },
  },
  methods: {
    async initDetailData() {
      const { customSkillId } = this.skillPreviewParams || {};

      if (!customSkillId) {
        this.detailData = {};
        return;
      }

      this.loading = true;
      try {
        const res = await getCustomSkillInfo({ skillId: customSkillId });
        this.detailData = res.data || {};
      } finally {
        this.loading = false;
      }
    },
    async handleCreateVariable(payload) {
      const { customSkillId } = this.skillPreviewParams || {};
      if (!customSkillId) return;

      await createCustomSkillConfig({
        skillId: customSkillId,
        variable: payload,
      });
      await this.initDetailData();
    },
    async handleUpdateVariable(payload) {
      if (!payload?.id) return;

      const { id, ...variable } = payload;
      await updateCustomSkillConfig({
        id,
        variable,
      });
      await this.initDetailData();
    },
    async handleDeleteVariable(payload) {
      if (!payload?.id) return;

      await deleteCustomSkillConfig({
        id: payload.id,
      });
      await this.initDetailData();
    },
  },
};
</script>

<style lang="scss" scoped>
.skill-config {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
  padding: 24px;
  overflow-y: auto;
}

.config-body {
  flex: 1;
  display: flex;
  justify-content: center;
}
</style>
