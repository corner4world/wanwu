<template>
  <SkillDetail
    :detail="detail"
    :recommendList="recommendList"
    :isPublic="isPublic"
    :bgColor="bgColor"
    :backText="backText"
    :visibleVariableConfig="false"
    @init="initData"
    @back="handleBack"
    @download="handleDownload"
    @click-recommend="handleClickRecommend"
  >
    <template #info-header>
      <div class="apiKeyConfig-tips">
        <i class="el-icon-info"></i>
        <span>
          {{
            skillType === 'builtin'
              ? $t('skillSpace.detail.apiKeyEmptyTips_builtin')
              : $t('skillSpace.detail.apiKeyEmptyTips')
          }}
        </span>
      </div>
    </template>
    <template #extra-buttons>
      <el-button
        v-if="skillType !== 'builtin'"
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
import SkillDetail from '@/components/skills/skillDetail.vue';
import {
  getSquareSkillList,
  getBuiltinSquareSkillList,
  sendSquareSkillToResource,
  getSquareSkillDetail,
  downloadSquareSkill,
} from '@/api/skillSquare';
import { downloadBuiltinSkill } from '@/api/templateSquare';
import { resDownloadFile } from '@/utils/util';

export default {
  components: {
    SkillDetail,
  },
  data() {
    return {
      skillId: '',
      skillType: '',
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
      const { skillId, skillType } = this.$route.query || {};
      this.skillId = skillId;
      this.skillType = skillType;

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
      const requestApi =
        this.skillType === 'builtin'
          ? getBuiltinSquareSkillList
          : getSquareSkillList;
      const res = await requestApi();
      this.recommendList =
        res.data.list.filter(item => item.skillId !== this.skillId) || [];
    },
    async handleDownload(item) {
      const downloadApi =
        this.skillType === 'builtin'
          ? downloadBuiltinSkill
          : downloadSquareSkill;
      const res = await downloadApi({ skillId: item.skillId });
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
      const query = { skillId: skillId };
      if (this.skillType === 'builtin') {
        query.skillType = 'builtin';
      }
      this.$router.push({
        path: this.$route.path,
        query,
      });
    },
  },
};
</script>

<style lang="scss" scoped>
.apiKeyConfig-tips {
  display: flex;
  align-items: flex-start;
  background: #f0f7ff;
  border: 1px solid #ddecff;
  border-left: 4px solid #409eff;
  color: #5e6d82;
  font-size: 13px;
  padding: 12px 16px;
  border-radius: 4px;
  margin: 15px 0;
  line-height: 1.6;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.02);

  i {
    font-size: 16px;
    color: #409eff;
    margin-right: 10px;
    margin-top: 2px;
    flex-shrink: 0;
  }

  span {
    flex: 1;
  }
}
</style>
