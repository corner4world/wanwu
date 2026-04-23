<template>
  <div
    class="tempSquare-detail page-wrapper"
    :style="isPublic ? `background: ${bgColor}; min-height: 100%` : ''"
  >
    <!-- 返回按钮，点击抛出 back 事件 -->
    <span class="back" @click="$emit('back')">
      <!-- 可通过 prop 灵活定义返回按钮文字 -->
      <slot name="back-text">
        {{ backText || $t('menu.back') }}
      </slot>
    </span>

    <div class="tempSquare-title">
      <div class="tempSquare-title-left">
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
                foldStatus
                  ? $t('common.button.fold')
                  : $t('common.button.detail')
              }}
            </span>
          </p>
          <p v-else class="desc">{{ detail.desc }}</p>
        </div>
      </div>
      <div style="margin-left: 10px; flex-shrink: 0">
        <!-- 默认的下载按钮，点击时将当前 detail 抛出，由外部调用具体 API 或者下载逻辑 -->
        <el-button
          type="primary"
          size="mini"
          @click="$emit('download', detail)"
        >
          {{ $t('tempSquare.download') }}
        </el-button>
        <!-- 预留插槽，如果是工作流页面可能需要增加一些像“克隆”一样的按钮 -->
        <slot name="extra-buttons"></slot>
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
              <div class="item-title">• &nbsp;{{ $t('square.summary') }}</div>
              <div class="item-desc" v-html="parseTxt(detail.summary)"></div>
            </div>
            <div class="overview-item" v-if="detail.feature">
              <div class="item-title">• &nbsp;{{ $t('square.feature') }}</div>
              <div class="item-desc" v-html="parseTxt(detail.feature)"></div>
            </div>
            <div class="overview-item" v-if="detail.scenario">
              <div class="item-title">• &nbsp;{{ $t('square.scenario') }}</div>
              <div class="item-desc">
                <div v-html="parseTxt(detail.scenario)"></div>
              </div>
            </div>
          </div>
          <div class="overview bg-border" v-if="detail.note">
            <div class="overview-item">
              <div class="item-title">• &nbsp;{{ $t('square.note') }}</div>
              <div class="item-desc" v-html="parseTxt(detail.note)"></div>
            </div>
          </div>
          <div class="overview bg-border" v-if="detail.skillMarkdown">
            <div class="overview-item">
              <div class="item-desc">
                <div class="tempSquare-markdown">
                  <MdRender :content="detail.skillMarkdown" />
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>

      <div v-if="recommendList.length" class="right-recommend">
        <!-- 右侧标题也可以扩展成插槽，以支持更多定制化 -->
        <slot name="recommend-title">
          <p style="margin: 20px 0; color: #333">
            {{ $t('skillSpace.detail.otherSkill') }}
          </p>
        </slot>
        <div
          class="recommend-item"
          v-for="(item, i) in recommendList"
          :key="`${i}rc`"
          @click="$emit('click-recommend', item)"
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
// 这里不导入任何 API 文件，仅仅作为一个单纯的展示组件 (Dump Component)
import { avatarSrc } from '@/utils/util';
import MdRender from '@/components/mdRender.vue';

export default {
  name: 'SkillDetail',
  components: { MdRender },
  props: {
    // 详情主数据
    detail: {
      type: Object,
      default: () => ({}),
    },
    // 右侧推荐列表数据
    recommendList: {
      type: Array,
      default: () => [],
    },
    // 是否为公开分享页面（控制一些背景等视觉样式）
    isPublic: {
      type: Boolean,
      default: false,
    },
    // 公开页面的背景由于有时需要自定义，所以提升为 prop
    bgColor: {
      type: String,
      default:
        'linear-gradient(1deg, rgb(247, 252, 255) 50%, rgb(233, 246, 254) 98%)',
    },
    // 返回按钮文本
    backText: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      foldStatus: false,
      tabActive: 0,
    };
  },
  // 可以在挂载时发起事件，通知父组件初始化数据
  mounted() {
    this.$emit('init');
  },
  methods: {
    avatarSrc,
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
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/tabs.scss';
.tempSquare-detail {
  padding: 20px;
  overflow: auto;
  .back {
    color: $color;
    cursor: pointer;
  }
  .tempSquare-title {
    padding: 20px 0;
    display: flex;
    border-bottom: 1px solid #bfbfbf;
    justify-content: space-between;
    align-items: center;
    .tempSquare-title-left {
      display: flex;
      align-items: center;
    }
    .logo {
      width: 54px;
      height: 54px;
      object-fit: cover;
    }
    .info {
      position: relative;
      margin-left: 15px;
      .name {
        font-size: 16px;
        color: #5d5d5d;
        font-weight: bold;
      }
      .desc {
        margin-top: 10px;
        line-height: 22px;
        color: #9f9f9f;
        word-break: break-all;
      }
      .arrow {
        position: absolute;
        display: block;
        right: 0;
        bottom: -5px;
        cursor: pointer;
        color: $color;
        margin-left: 10px;
        font-size: 13px;
      }
    }
    .fold {
      height: auto;
    }
  }
  .tempSquare-main {
    display: flex;
    margin: 10px 0 0 0;
    .left-info {
      width: calc(100% - 420px);
      margin-right: 20px;
      .overview {
        .overview-item {
          display: flex;
          padding: 15px 0;
          border-bottom: 1px solid #eee;
          line-height: 24px;
          .item-title {
            width: 80px;
            color: $color;
            font-weight: bold;
          }
          .item-desc {
            width: calc(100% - 100px);
            margin-left: 10px;
            flex: 1;
            color: #333;
          }
        }
        .overview-item:last-child {
          border-bottom: none;
        }
      }
    }
    .right-recommend {
      width: 400px;
      overflow-y: auto;
      border-left: 1px solid #eee;
      padding: 20px;
      max-height: 900px;
      .recommend-item {
        position: relative;
        border: 1px solid $border_color;
        background: $color_opacity;
        margin-bottom: 15px;
        border-radius: 10px;
        padding: 20px 20px 20px 80px;
        text-align: left;
        cursor: pointer;
        min-height: 100px;
        .logo {
          width: 46px;
          height: 46px;
          object-fit: cover;
          position: absolute;
          left: 20px;
          border: 1px solid #fff;
          border-radius: 4px;
        }
        .name {
          color: #5d5d5d;
          font-weight: bold;
        }
        .intro {
          max-height: 36px;
          color: #5d5d5d;
          margin-top: 8px;
          font-size: 13px;
          overflow: hidden;
          display: -webkit-box;
          -webkit-box-orient: vertical;
          text-overflow: ellipsis;
          -webkit-line-clamp: 2;
          line-clamp: 2;
        }
      }
    }
  }
  .bg-border {
    margin-top: 20px;
    background-color: rgba(255, 255, 255, 1);
    box-sizing: border-box;
    border-radius: 10px;
    padding: 10px 20px;
    box-shadow: 2px 2px 15px $color_opacity;
  }
  .overview-item .item-desc {
    line-height: 28px;
  }
}
.tempSquare-markdown {
  ::v-deep .code-header {
    padding: 0 0 5px 0;
  }
}
</style>
