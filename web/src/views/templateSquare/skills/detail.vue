<template>
  <SkillDetail
    :detail="detail"
    :recommendList="recommendList"
    :isPublic="isPublic"
    :bgColor="bgColor"
    :backText="backText"
    @init="initData"
    @back="handleBack"
    @download="handleDownload"
    @click-recommend="handleClickRecommend"
  />
</template>

<script>
import SkillDetail from '@/components/skills/skillDetail.vue';
import { getCustomSkillInfo, getCustomSkillList } from '@/api/templateSquare';
import {
  getAcquiredSkillList,
  getAcquiredSkillDetail,
} from '@/api/skillResource/added';
import { SKILL, SKILLCUSTOM } from '../constants';
import { directDownload } from '@/utils/util';

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
      } else {
        res = await getAcquiredSkillDetail({ skillId: this.templateSquareId });
      }
      this.detail = res.data || {};
    },
    async getRecommendList() {
      let res;
      if (this.type === SKILLCUSTOM) {
        res = await getCustomSkillList();
      } else {
        res = await getAcquiredSkillList();
      }

      const list = res.data?.list || [];
      this.recommendList = list.filter(
        item => item.skillId !== this.templateSquareId,
      );
    },
    handleDownload(item) {
      if (this.type === SKILLCUSTOM) {
        if (item.zipUrl) {
          directDownload(item.zipUrl);
        }
      } else {
        if (item.downloadUrl) {
          directDownload(item.downloadUrl);
        }
      }
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
