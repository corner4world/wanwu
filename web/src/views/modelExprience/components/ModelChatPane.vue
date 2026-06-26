<template>
  <div class="model-wrapper rl">
    <div class="model-setting">
      <i class="el-icon-loading" v-if="pending"></i>
      <img v-else class="avatar" :src="getModelAvatar(detail.avatar)" />
      <div class="model-info">
        <div class="model-name">{{ detail.displayName || '-' }}</div>
        <div class="model-desc">
          <span v-for="(tag, index) in detail.tags" :key="index">
            {{ tag }}
          </span>
        </div>
      </div>
      <el-tooltip
        v-if="supportDelete"
        effect="light"
        :content="$t('modelExprience.tip.deleteModel')"
        placement="top"
      >
        <img
          @click="handleModelDelete"
          class="icon delete-btn"
          :src="require('@/assets/imgs/model-delete.svg')"
        />
      </el-tooltip>
      <!--不支持更换模型-->
      <!--<el-tooltip
        effect="light"
        :content="$t('modelExprience.tip.replaceModel')"
        placement="top"
      >
        <img
          @click="handleModelReplace"
          class="icon delete-btn"
          :src="require('@/assets/imgs/config-replace.svg')"
        />
      </el-tooltip>-->
      <el-tooltip
        effect="light"
        :content="$t('modelExprience.tip.modelConfig')"
        placement="top"
      >
        <img
          @click="handleModelSet"
          class="icon model-cfg"
          :src="require('@/assets/imgs/model-config.svg')"
        />
      </el-tooltip>
    </div>
    <StreamMessageField
      ref="session-com"
      :modelSessionStatus="modelSessionStatus"
      :modelIconUrl="modelIconUrl"
      :supportStop="supportStop"
      :supportClear="false"
      @queryCopy="handleSetQuery"
      @refresh="refresh"
      @preStop="preStop"
    />
  </div>
</template>
<script>
import StreamMessageField from '@/components/stream/streamMessageField.vue';
import sseMethod from '@/mixins/sseMethod.js';
import { avatarSrc, getModelDefaultIcon } from '@/utils/util';

export default {
  name: 'ModelChatPane',
  mixins: [sseMethod],
  components: {
    StreamMessageField,
  },
  props: {
    // 获取模型信息的接口是否在加载中
    pending: {
      type: Boolean,
      default: false,
    },
    modelExperienceId: {
      type: [String, Number],
      default: '',
    },
    sessionId: {
      type: String,
      default: '',
    },
    modelId: {
      type: String,
      default: '',
    },
    modelDetail: {
      // 模型详情
      type: Object,
      default: () => {},
    },
    modelSetting: {
      // 模型参数配置
      type: Object,
      default: () => {},
    },
    modelSessionStatus: {
      // 模型体验会话状态
      type: Number,
      default: -1, // 0:会话中，-1:非会话
    },
    supportDelete: {
      // 是否支持删除（模型对比模式下支持删除）
      type: Boolean,
      default: false,
    },
    supportStop: {
      // 是否支持单个停止体验
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      // 组件数据
    };
  },
  computed: {
    detail() {
      if (!this.modelDetail) {
        return {
          tags: [],
        };
      }
      const tagArr = [];
      const { modelType, tags = [] } = this.modelDetail;
      modelType && tagArr.push(modelType);
      tags.forEach(item => {
        tagArr.push(item.text);
      });
      return {
        ...this.modelDetail,
        tags: tagArr,
      };
    },
    modelIconUrl() {
      if (
        this.modelDetail &&
        this.modelDetail.avatar &&
        this.modelDetail.avatar.path
      ) {
        return avatarSrc(this.modelDetail.avatar.path);
      }
      return getModelDefaultIcon();
    },
    apiParams() {
      return {
        ...this.formatChatModelSetting(this.modelSetting),
        modelId: this.modelId,
        sessionId: this.sessionId,
        modelExperienceId: this.modelExperienceId || '0',
        role: 'user',
      };
    },
  },
  beforeDestroy() {
    this.stopEventSource();
  },
  methods: {
    formatChatModelSetting(modelSetting) {
      const newModelSetting = JSON.parse(JSON.stringify(modelSetting));
      if (newModelSetting.thinkingSupport !== undefined)
        delete newModelSetting.thinkingSupport;
      return newModelSetting;
    },
    // 该函数会覆盖sseMethod中的setStoreSessionStatus方法，以适配当前组件需求!!!!!!!!
    setStoreSessionStatus(val) {
      this.$emit('update:modelSessionStatus', val);
    },
    getModelAvatar(avatar) {
      if (!avatar || !avatar.path) {
        return getModelDefaultIcon();
      }
      return avatarSrc(avatar.path);
    },
    handleModelSet() {
      this.$emit('modelSet');
    },
    handleModelReplace() {
      this.$emit('modelReplace');
    },
    handleModelDelete() {
      this.$confirm(
        this.$t('modelExprience.warning.deleteModel'),
        this.$t('common.confirm.title'),
        {
          confirmButtonText: this.$t('common.confirm.confirm'),
          cancelButtonText: this.$t('common.confirm.cancel'),
          type: 'warning',
        },
      ).then(() => {
        this.$emit('modelDelete');
      });
    },
    initHistoryList(list) {
      this.$refs['session-com'].initHistoryList(list);
    },
    reconnectModelExperienceStream() {
      this.connectModelExperienceEventSource();
    },
    preSend(inputVal, fileList, fileInfo) {
      this.inputVal = inputVal;
      this.fileList = fileList;
      this.doExprienceSend({ inputVal, fileList, fileInfo });
    },
    // 创建会话前
    beforeCreateChat(params) {
      const { inputVal } = params;
      this.$refs['session-com'].replaceHistory([
        {
          query: inputVal,
          pending: true,
          responseLoading: true,
          requestFileUrls: [],
          fileList: [],
          pendingResponse: '',
        },
      ]);
    },
    // 创建会话后
    afterCreateChat() {
      this.$refs['session-com'].removeLastHistory();
    },
    handleSetQuery(query) {
      this.$emit('queryCopy', query);
    },
  },
};
</script>
<style scoped lang="scss">
.model-wrapper {
  word-break: break-all;
  height: 100%;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  .model-setting {
    background-color: #fff;
    position: relative;
    flex-shrink: 0;
    display: flex;
    align-items: center;
    border: 1px solid #d3d7dd;
    border-radius: 8px;
    padding: 8px 12px;
    .avatar {
      flex-shrink: 0;
      width: 35px;
      height: 35px;
      margin-right: 8px;
      border-radius: 4px;
    }
    .model-info {
      flex: 1;
      .model-name {
        font-size: 16px;
        font-weight: bold;
        margin-bottom: 6px;
      }
      .model-desc {
        display: flex;
        flex-wrap: wrap;
        gap: 8px;
        span {
          display: inline-block;
          padding: 2px 12px;
          border-radius: 2px;
          color: $color;
          background: $color_opacity;
        }
      }
    }
    .icon {
      width: 18px;
      cursor: pointer;
      margin-left: 12px;
      flex-shrink: 0;
    }
    .el-icon-loading {
      font-size: 18px;
      font-weight: 600;
      color: $color;
      margin-right: 8px;
    }
  }
}
</style>
