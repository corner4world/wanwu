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
  >
    <template #extra-buttons>
      <el-button
        type="primary"
        size="mini"
        plain
        :disabled="detail.isShared"
        @click="handleSendToResource(detail)"
      >
        {{
          detail.isShared
            ? $t('skillSpace.isShared')
            : $t('skillSpace.toResource')
        }}
      </el-button>
    </template>
  </SkillDetail>
</template>

<script>
import SkillDetail from '@/views/templateSquare/skillDetail.vue';
import {
  getSquareSkillList,
  sendSquareSkillToResource,
  getSquareSkillDetail,
  downloadSquareSkill,
} from '@/api/skillSquare';
import { resDownloadFile } from '@/utils/util';

export default {
  components: {
    SkillDetail,
  },
  data() {
    return {
      skillId: '',
      detail: {},
      recommendList: [],
      isPublic: false,
      bgColor:
        'linear-gradient(1deg, rgb(247, 252, 255) 50%, rgb(233, 246, 254) 98%)',
      backText: this.$t('skillSpace.detail.backText'),
    };
  },
  created() {
    this.isPublic = this.$route.path.includes('/public/');
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
      const { skillId } = this.$route.query || {};
      this.skillId = skillId;

      this.getDetailData();
      this.getRecommendList();

      // 滚动到顶部
      const main = document.querySelector('.el-main > .page-container');
      if (main) main.scrollTop = 0;
    },
    async getDetailData() {
      const res = await getSquareSkillDetail({
        skillId: this.skillId,
      });
      this.detail = res.data || {};
    },
    async getRecommendList() {
      const res = await getSquareSkillList();
      this.recommendList = res.data?.list || [];
    },
    async handleDownload(item) {
      const res = await downloadSquareSkill({ skillId: item.skillId });
      resDownloadFile(res, `${item.name}.zip`);
    },
    handleSendToResource(info) {
      sendSquareSkillToResource({ skillId: info.skillId }).then(res => {
        if (res.code === 0) {
          this.$message.success(this.$t('common.info.send'));
          this.$set(this.detail, 'isShared', true);
        }
      });
    },
    handleBack() {
      const path = '/skillSquare';
      this.$router.push({ path });
    },
    handleClickRecommend(val) {
      const skillId = val.skillId;
      this.$router.push({
        path: this.$route.path,
        query: { skillId: skillId },
      });
    },
  },
};
</script>

<style lang="scss" scoped></style>
