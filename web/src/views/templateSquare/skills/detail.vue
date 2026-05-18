<template>
  <SkillDetail
    :detail="detail"
    :recommendList="recommendList"
    :isPublic="isPublic"
    :bgColor="bgColor"
    :backText="backText"
    :visibleVariableConfig="true"
    @init="initData"
    @back="handleBack"
    @download="handleDownload"
    @click-recommend="handleClickRecommend"
    @create-variable="handleCreateVariable"
    @update-variable="handleUpdateVariable"
    @delete-variable="handleDeleteVariable"
  />
</template>

<script>
import SkillDetail from '@/components/skills/skillDetail.vue';
import {
  createAcquiredSkillConfig,
  createCustomSkillConfig,
  createResourceBuiltinSkillConfig,
  deleteAcquiredSkillConfig,
  deleteCustomSkillConfig,
  downloadBuiltinSkill,
  deleteResourceBuiltinSkillConfig,
  getCustomSkillInfo,
  getCustomSkillList,
  getResourceBuiltinSkillDetail,
  getResourceBuiltinSkillList,
  updateAcquiredSkillConfig,
  updateCustomSkillConfig,
  updateResourceBuiltinSkillConfig,
} from '@/api/templateSquare';
import {
  getAcquiredSkillList,
  getAcquiredSkillDetail,
} from '@/api/skillResource/added';
import { SKILL, SKILLCUSTOM, SKILLADDED, SKILLBUILTIN } from '../constants';
import { directDownload, resDownloadFile } from '@/utils/util';

export default {
  components: {
    SkillDetail,
  },
  data() {
    return {
      templateSquareId: '',
      type: SKILL,
      detail: {},
      recommendList: [],
      isPublic: false,
      bgColor:
        'linear-gradient(1deg, rgb(247, 252, 255) 50%, rgb(233, 246, 254) 98%)',
      backText: '',
    };
  },
  created() {
    this.isPublic = this.$route.path.includes('/public/');
    this.backText = this.$t('menu.back') + this.$t('menu.resource');
  },
  watch: {
    $route: {
      handler() {
        this.initData();
      },
      deep: true,
    },
  },
  methods: {
    initData() {
      const { templateSquareId, type } = this.$route.query || {};
      this.templateSquareId = templateSquareId;
      this.type = type || SKILL;

      this.getDetailData();
      this.getRecommendList();

      // 滚动到顶部
      const main = document.querySelector('.el-main > .page-container');
      if (main) main.scrollTop = 0;
    },
    async getDetailData() {
      let res;
      if (this.type === SKILLCUSTOM) {
        res = await getCustomSkillInfo({ skillId: this.templateSquareId });
      } else if (this.type === SKILLBUILTIN) {
        res = await getResourceBuiltinSkillDetail({
          skillId: this.templateSquareId,
        });
      } else {
        res = await getAcquiredSkillDetail({ skillId: this.templateSquareId });
      }
      this.detail = res.data || {};
    },
    async getRecommendList() {
      let res;
      if (this.type === SKILLCUSTOM) {
        res = await getCustomSkillList();
      } else if (this.type === SKILLBUILTIN) {
        res = await getResourceBuiltinSkillList();
      } else {
        res = await getAcquiredSkillList();
      }

      const list = res.data?.list || [];
      this.recommendList = list.filter(
        item => item.skillId !== this.templateSquareId,
      );
    },
    async handleDownload(item) {
      if (this.type === SKILLCUSTOM) {
        if (item.zipUrl) {
          directDownload(item.zipUrl);
        }
      } else if (this.type === SKILLBUILTIN) {
        const res = await downloadBuiltinSkill({
          skillId: item.skillId,
        });
        resDownloadFile(res, `${item.name}.zip`);
      } else if (this.type === SKILLADDED || this.type === SKILL) {
        if (item.downloadUrl) {
          directDownload(item.downloadUrl);
        }
      }
    },
    getVariableConfigApi(action) {
      const apiMap = {
        [SKILLCUSTOM]: {
          create: createCustomSkillConfig,
          update: updateCustomSkillConfig,
          delete: deleteCustomSkillConfig,
        },
        [SKILLBUILTIN]: {
          create: createResourceBuiltinSkillConfig,
          update: updateResourceBuiltinSkillConfig,
          delete: deleteResourceBuiltinSkillConfig,
        },
        [SKILLADDED]: {
          create: createAcquiredSkillConfig,
          update: updateAcquiredSkillConfig,
          delete: deleteAcquiredSkillConfig,
        },
        [SKILL]: {
          create: createAcquiredSkillConfig,
          update: updateAcquiredSkillConfig,
          delete: deleteAcquiredSkillConfig,
        },
      };

      return apiMap[this.type]?.[action];
    },
    async handleCreateVariable(payload) {
      const api = this.getVariableConfigApi('create');
      if (!api) return;

      await api(payload);
      await this.getDetailData();
    },
    async handleUpdateVariable(payload) {
      const api = this.getVariableConfigApi('update');
      if (!api) return;

      await api(payload);
      await this.getDetailData();
    },
    async handleDeleteVariable(payload) {
      const api = this.getVariableConfigApi('delete');
      if (!api) return;

      await api(payload);
      await this.getDetailData();
    },
    handleBack() {
      this.$router.push({
        path: '/skill',
        query: { type: this.type },
      });
    },
    handleClickRecommend(val) {
      const skillId = val.skillId;
      this.$router.push({
        path: this.$route.path,
        query: {
          templateSquareId: skillId,
          type: this.type,
        },
      });
    },
  },
};
</script>

<style lang="scss" scoped></style>
