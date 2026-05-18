<template>
  <div class="page-wrapper skill-management">
    <div class="common_bg">
      <!-- tabs -->
      <div class="tabs" style="margin: 0 20px">
        <div
          :class="['tab', { active: tabActive === SKILLBUILTIN }]"
          @click="tabClick(SKILLBUILTIN)"
        >
          {{ $t('tempSquare.skills.app.builtin') }}
        </div>
        <div
          :class="['tab', { active: tabActive === SKILLADDED }]"
          @click="tabClick(SKILLADDED)"
        >
          {{ $t('tempSquare.skills.app.myAdded') }}
        </div>
        <div
          :class="['tab', { active: tabActive === SKILLCUSTOM }]"
          @click="tabClick(SKILLCUSTOM)"
        >
          {{ $t('tempSquare.skills.app.myCreated') }}
        </div>
      </div>

      <Builtin ref="builtin" v-if="tabActive === SKILLBUILTIN" />
      <Acquired ref="acquired" v-if="tabActive === SKILLADDED" />
      <Custom ref="custom" v-if="tabActive === SKILLCUSTOM" />
    </div>
  </div>
</template>
<script>
import Builtin from './builtin/list';
import Acquired from './acquired.vue';
import Custom from './custom/list';
import { SKILLBUILTIN, SKILLADDED, SKILLCUSTOM } from '../constants';

export default {
  data() {
    return {
      SKILLBUILTIN,
      SKILLADDED,
      SKILLCUSTOM,
      tabActive: SKILLBUILTIN,
    };
  },
  watch: {
    $route: {
      handler() {
        this.setInitTab();
      },
      // 深度观察监听
      deep: true,
    },
  },
  mounted() {
    this.setInitTab();
  },
  methods: {
    setInitTab() {
      const { type } = this.$route.query || {};
      this.tabActive = type || SKILLBUILTIN;
    },
    tabClick(type) {
      this.tabActive = type;
      this.$router.replace({
        query: {
          ...this.$route.query,
          type,
        },
      });
    },
  },
  components: {
    Builtin,
    Acquired,
    Custom,
  },
};
</script>
<style lang="scss" scoped>
@import '@/style/tabs.scss';
::v-deep .scroll-card-container {
  max-height: calc(100vh - 165px);
  .scroll-card-pr {
    padding-bottom: 0;
  }
}

.skill-management {
  height: calc(100% - 50px);
  padding-top: 20px;

  .common_bg {
    height: 100%;
  }

  .title {
    font-size: 20px;
    margin: 0;
    padding: 0 0 20px 0;
    text-align: center;

    .svg-icon {
      width: 1.6em;
      height: 1.6em;
      color: $color;
      vertical-align: -0.25em;
    }
  }

  .mcp-content-box {
    height: calc(100% - 145px);
  }

  .mcp-content {
    padding: 0 20px;
    width: 100%;
    height: 100%;
  }

  .el-tabs__nav-wrap {
    text-align: center;
  }

  .el-tabs__nav-scroll {
    display: inline-block;
  }

  .el-tabs__nav-wrap::after {
    display: none;
  }

  .card-box {
    display: flex;
    flex-wrap: wrap;
    margin: 6px -10px 0;
    /*overflow: auto;*/
    .card {
      position: relative;
      padding: 20px 16px;
      border-radius: 12px;
      height: fit-content;
      background: #fff url('@/assets/imgs/card_bg.png');
      background-size: 100% 100%;
      display: flex;
      flex-direction: column;
      align-items: center;
      width: calc((100% / 4) - 20px);
      margin: 0 10px 20px;
      box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.15);
      border: 1px solid rgba(0, 0, 0, 0);

      &:hover {
        cursor: pointer;
        box-shadow:
          0 2px 8px #171a220d,
          0 4px 16px #0000000f;
        border: 1px solid $border_color;
      }

      .card-title {
        display: flex;
        width: 100%;
        height: 58px;
        padding-bottom: 7px;

        .svg-icon {
          width: 50px;
          height: 50px;
        }

        .mcp_detailBox {
          width: calc(100% - 70px);
          margin-left: 10px;
          display: flex;
          flex-direction: column;
          justify-content: space-between;
          padding: 0 0 3px 0;

          .mcp_name {
            min-height: 22px;
            display: block;
            font-size: 15px;
            font-weight: 700;
            overflow: hidden;
            white-space: nowrap;
            text-overflow: ellipsis;
            color: $create_card_text_color;
            line-height: 1;
          }

          .mcp_from {
            label {
              padding: 3px 7px;
              font-size: 12px;
              color: $tag_color;
              background: $tag_bg;
              border-radius: 3px;
              display: block;
              height: 22px;
              width: 100%;
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
            }
          }
        }

        margin-bottom: 13px;
      }

      .card-des {
        width: 100%;
        display: -webkit-box;
        text-overflow: ellipsis;
        color: #5d5d5d;
        font-weight: 400;
        overflow: hidden;
        -webkit-line-clamp: 3;
        line-clamp: 2;
        -webkit-box-orient: vertical;
        font-size: 13px;
        height: 55px;
        word-wrap: break-word;
      }
    }
  }

  .no-list {
    display: flex;
    justify-content: center;
    align-items: center;
    height: calc(100vh - 330px);
    min-height: 200px;
    font-size: 30px;
    // color: #ddd;
    text-align: center;

    i {
      font-size: 50px;
      color: $color;
      cursor: pointer;
    }

    span {
      padding-top: 20px;
      display: block;
    }
  }

  .card-search {
    text-align: right;
    padding: 10px 0;
  }

  .el-tabs__content {
    max-width: 1500px;
    margin: 0 auto;
  }

  .card-search-cust {
    display: flex;
    justify-content: space-between;
    align-items: center;

    .card-search-des {
      color: #585a73;
      font-size: 12px;

      .el-button {
        padding: 5px 12px;

        span {
          font-size: 12px;
        }
      }
    }

    .radio-box {
      margin: 10px 0;
      padding: 0;
    }
  }

  .el-radio__input.is-checked .el-radio__inner {
    border-color: $color;
    background: $color;
  }
}
</style>
