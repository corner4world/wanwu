<template>
  <div class="mcp-detail page-wrapper" id="timeScroll">
    <span class="back" @click="back">
      {{ $t('menu.back') + $t('menu.resource') }}
    </span>
    <div class="mcp-title">
      <img
        class="logo"
        v-if="detail.avatar && detail.avatar.path"
        :src="avatarSrc(detail.avatar.path)"
      />
      <div :class="['info', { fold: foldStatus }]">
        <p class="name">{{ detail.name }}</p>
        <p v-if="detail.desc && detail.desc.length > 260" class="desc">
          {{ foldStatus ? detail.desc : detail.desc.slice(0, 268) + '...' }}
          <span class="arrow" v-show="detail.desc.length > 260" @click="fold">
            {{
              foldStatus ? $t('common.button.fold') : $t('common.button.detail')
            }}
          </span>
        </p>
        <p v-else class="desc">{{ detail.desc }}</p>
      </div>
    </div>
    <div class="mcp-main">
      <div class="left-info">
        <!-- tabs -->
        <div class="tabs">
          <div
            :class="['tab', { active: tabActive === 0 }]"
            @click="tabClick(0)"
          >
            {{ $t('tool.builtIn.count', { count: detail.actionSum || 0 }) }}
          </div>
        </div>

        <div>
          <div class="tool" v-if="detail.needApiKeyInput">
            <div class="tool-item">
              <p class="title">{{ $t('tool.builtIn.api') }}</p>
              <div class="sse-url" style="display: flex">
                <el-input
                  v-model="apiKey"
                  style="margin-right: 20px"
                  showPassword
                />
                <el-button
                  style="width: 100px"
                  size="mini"
                  type="primary"
                  :disabled="detail.hasCustom"
                  @click="changeApiKey"
                >
                  {{
                    detail.apiKey
                      ? $t('tool.builtIn.update')
                      : $t('tool.builtIn.confirm')
                  }}
                </el-button>
              </div>
            </div>
          </div>
          <div class="overview" v-if="detail.detail">
            <div class="overview-item">
              <!--<div class="item-title">• &nbsp;详情</div>-->
              <div class="item-desc">
                <div
                  class="readme-content markdown-body mcp-markdown"
                  v-html="md.render(detail.detail || '')"
                ></div>
              </div>
            </div>
          </div>
        </div>
        <div class="tool" v-if="tools && tools.length">
          <div class="tool-item">
            <!--<p class="title">工具介绍:</p>-->
            <div class="tool-intro">
              <el-collapse class="mcp-el-collapse" v-model="activeNames">
                <el-collapse-item
                  v-for="(n, i) in tools"
                  :key="n.name + i"
                  :title="n.name"
                  :name="i"
                >
                  <div class="desc" v-if="n.description">
                    {{ $t('tool.builtIn.desc') }}
                    <span v-html="parseTxt(n.description)" />
                  </div>
                  <div class="params">
                    <p>{{ $t('tool.builtIn.params') }}</p>
                    <div
                      class="params-table"
                      v-for="(m, j) in n.params"
                      :key="m.name + j"
                    >
                      <div class="tr">
                        <div class="td">{{ m.name }}</div>
                        <div class="td color">{{ m.type }}</div>
                        <div class="td color">{{ m.requiredBadge }}</div>
                      </div>
                      <p
                        class="params-desc"
                        v-html="parseTxt(m.description)"
                      ></p>
                    </div>
                  </div>
                </el-collapse-item>
              </el-collapse>
            </div>
          </div>
        </div>
      </div>

      <div class="right-recommend">
        <p style="margin: 20px 0; color: #333">
          {{ $t('tool.builtIn.recommend') }}
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
            :src="avatarSrc(item.avatar.path)"
          />
          <p class="name">{{ item.name }}</p>
          <p class="intro">{{ item.desc }}</p>
        </div>
      </div>
    </div>
  </div>
</template>
<script>
import { md } from '@/mixins/markdown-it';
import { getRecommendsList, getToolDetail, changeApiKey } from '@/api/mcp';
import { avatarSrc, formatTools } from '@/utils/util';

export default {
  data() {
    return {
      md: md,
      toolSquareId: '',
      detail: {},
      tools: [],
      apiKey: '',
      foldStatus: false,
      tabActive: 0,
      recommendList: [],
      activeNames: [],
      dialogVisible: false,
    };
  },
  watch: {
    $route: {
      handler() {
        this.initData();
      },
      // 深度观察监听
      deep: true,
    },
  },
  mounted() {
    this.initData();
    this.getRecommendList();
  },
  methods: {
    avatarSrc,
    initData() {
      this.toolSquareId = this.$route.query.toolSquareId;
      this.tabActive = 0;
      this.getDetailData();

      //滚动到顶部
      const main = document.querySelector('.el-main > .page-container');
      if (main) main.scrollTop = 0;
    },
    getDetailData() {
      getToolDetail({ toolSquareId: this.toolSquareId }).then(res => {
        const data = res.data || {};
        this.detail = data;
        this.apiKey = data.apiKey || '';
        this.tools = formatTools(data.tools);
        this.activeNames = data.actionSum === 1 ? [0] : [];
      });
    },
    getRecommendList() {
      const params = {
        mcpSquareId: this.mcpSquareId,
      };
      getRecommendsList(params).then(res => {
        this.recommendList = res.data.list;
      });
    },
    handleClick(val) {
      this.$router.push(`/mcp/detail/square?mcpSquareId=${val.mcpSquareId}`);
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
    changeApiKey() {
      changeApiKey({
        apiKey: this.apiKey,
        toolSquareId: this.toolSquareId,
      }).then(res => {
        if (res.code === 0) {
          this.$message.success(this.$t('common.message.success'));
          this.getDetailData();
        }
      });
    },
    back() {
      this.$router.push({ path: '/tool?tool=builtIn' });
    },
  },
};
</script>
<style lang="scss" scoped>
@import '@/style/markdown.scss';
@import '@/style/tabs.scss';
@import '@/style/squareDetail.scss';
</style>
