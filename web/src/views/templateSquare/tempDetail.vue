<template>
  <div
    class="tempSquare-detail page-wrapper"
    :style="isPublic ? `background: ${bgColor}; min-height: 100%` : ''"
  >
    <span class="back" @click="back">
      {{
        $t('menu.back') +
        (type === workflow ? $t('menu.templateSquare') : $t('menu.resource'))
      }}
    </span>
    <div class="tempSquare-title">
      <div class="tempSquare-title-left">
        <img
          class="logo"
          v-if="detail.avatar && detail.avatar.path"
          :src="
            type === workflow
              ? detail.avatar.path
              : avatarSrc(detail.avatar.path)
          "
        />
        <div :class="['info', { fold: foldStatus }]">
          <p class="name">{{ detail.name }}</p>
          <p v-if="detail.desc && detail.desc.length > 260" class="desc">
            {{ foldStatus ? detail.desc : detail.desc.slice(0, 268) + '...' }}
            <span class="arrow" v-show="detail.desc.length > 260" @click="fold">
              {{
                foldStatus
                  ? $t('common.button.fold')
                  : $t('common.button.detail')
              }}
            </span>
          </p>
          <p v-else class="desc">{{ detail.desc }}</p>
        </div>
      </div>
      <div style="margin-left: 10px">
        <el-button
          v-if="type === workflow"
          type="primary"
          size="mini"
          @click="copyTemplate(detail)"
        >
          {{ $t('tempSquare.copy') }}
        </el-button>
        <el-button type="primary" size="mini" @click="downloadTemplate(detail)">
          {{ $t('tempSquare.download') }}
        </el-button>
      </div>
    </div>
    <div class="tempSquare-main">
      <div class="left-info">
        <div class="tabs">
          <div
            :class="['tab', { active: tabActive === 0 }]"
            @click="tabClick(0)"
          >
            {{ $t('square.info') }}
          </div>
        </div>

        <div>
          <div
            class="overview bg-border"
            v-if="detail.summary || detail.feature || detail.scenario"
          >
            <div class="overview-item" v-if="detail.summary">
              <div class="item-title">
                <img src="@/assets/imgs/detail_title_icon.png" alt="" />
                <span>{{ $t('square.summary') }}</span>
              </div>
              <div class="item-desc" v-html="parseTxt(detail.summary)"></div>
            </div>
            <div class="overview-item" v-if="detail.feature">
              <div class="item-title">
                <img src="@/assets/imgs/detail_title_icon.png" alt="" />
                <span>{{ $t('square.feature') }}</span>
              </div>
              <div class="item-desc" v-html="parseTxt(detail.feature)"></div>
            </div>
            <div class="overview-item" v-if="detail.scenario">
              <div class="item-title">
                <img src="@/assets/imgs/detail_title_icon.png" alt="" />
                <span>{{ $t('square.scenario') }}</span>
              </div>
              <div class="item-desc">
                <div v-html="parseTxt(detail.scenario)"></div>
              </div>
            </div>
          </div>
          <div class="overview bg-border" v-if="detail.note">
            <div class="overview-item">
              <div class="item-title">
                <img src="@/assets/imgs/detail_title_icon.png" alt="" />
                <span>{{ $t('square.note') }}</span>
              </div>
              <div class="item-desc" v-html="parseTxt(detail.note)"></div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="recommendList.length" class="right-recommend">
        <p style="margin: 20px 0; color: #333">
          {{ $t('tempSquare.otherTemp') }}
        </p>
        <div
          class="recommend-item"
          v-for="(item, i) in recommendList"
          :key="`${i}rc`"
          @click="handleClick(item)"
        >
          <img
            class="logo"
            v-if="item.avatar && item.avatar.path"
            :src="
              type === workflow ? item.avatar.path : avatarSrc(item.avatar.path)
            "
          />
          <p class="name">{{ item.name }}</p>
          <p class="intro">{{ item.desc }}</p>
        </div>
      </div>
    </div>
    <CreateWorkflow type="clone" ref="cloneWorkflowDialog" />
  </div>
</template>
<script>
import {
  downloadWorkflow,
  getWorkflowRecommendsList,
  getWorkflowTempInfo,
} from '@/api/templateSquare';
import { WORKFLOW } from './constants';
import { avatarSrc, directDownload, resDownloadFile } from '@/utils/util';
import CreateWorkflow from '@/components/createApp/createWorkflow.vue';
import MdRender from '@/components/mdRender.vue';

export default {
  components: { CreateWorkflow, MdRender },
  data() {
    return {
      basePath: this.$basePath,
      isPublic: true,
      bgColor:
        'linear-gradient(1deg, rgb(247, 252, 255) 50%, rgb(233, 246, 254) 98%)',
      type: '',
      workflow: WORKFLOW,
      isFromSquare: true,
      templateSquareId: '',
      detail: {},
      foldStatus: false,
      tabActive: 0,
      recommendList: [],
      dialogVisible: false,
    };
  },
  watch: {
    $route: {
      handler() {
        this.initData();
        this.getRecommendList();
      },
      // 深度观察监听
      deep: true,
    },
  },
  created() {
    this.isPublic = this.$route.path.includes('/public/');
  },
  mounted() {
    this.initData();
    this.getRecommendList();
  },
  methods: {
    avatarSrc,
    initData() {
      const { type, templateSquareId } = this.$route.query || {};
      this.templateSquareId = templateSquareId;
      this.type = type || WORKFLOW;
      this.getDetailData();

      // 滚动到顶部
      const main = document.querySelector('.el-main > .page-container');
      if (main) main.scrollTop = 0;
    },
    async getDetailData() {
      const res = await getWorkflowTempInfo({
        templateId: this.templateSquareId,
      });
      this.detail = res.data || {};
    },
    async getRecommendList() {
      const res = await getWorkflowRecommendsList({
        templateId: this.templateSquareId,
      });
      this.recommendList = res.data.list || [];
    },
    copyTemplate(item) {
      this.$refs.cloneWorkflowDialog.openDialog(item);
    },
    async downloadTemplate(item) {
      const res = await downloadWorkflow({ templateId: item.templateId });
      resDownloadFile(res, `${item.name}.json`);
    },
    getPath() {
      return this.isPublic ? '/public/templateSquare' : '/templateSquare';
    },
    handleClick(val) {
      this.$router.push(
        `${this.getPath()}/detail?templateSquareId=${val.templateId}`,
      );
    },
    // 解析文本，遇到.换行等
    parseTxt(txt) {
      if (!txt) return '';
      const text = txt
        .replaceAll('\n\t', '<br/>&nbsp;')
        .replaceAll('\n', '<br/>')
        .replaceAll('\t', '   &nbsp;');
      return text;
    },
    tabClick(status) {
      this.tabActive = status;
    },
    fold() {
      this.foldStatus = !this.foldStatus;
    },
    back() {
      this.$router.push({ path: this.getPath() });
    },
  },
};
</script>
<style lang="scss" scoped>
@import '@/style/tabs.scss';
@import '@/style/squareDetail.scss';
</style>
