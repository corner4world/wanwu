<template>
  <!-- 子会话渲染列表 -->
  <div
    class="sub-conversion-item"
    :data-sub-id="conversion.id"
    :data-conversationType="conversion.conversationType"
  >
    <div class="sub-conversion-header">
      <div class="left-info">
        <img
          :class="[
            'logo',
            conversion.conversationType ===
              AGENT_MESSAGE_CONFIG.SUB_AGENT.CONVERSATION_TYPE && 'logo-large',
          ]"
          :src="avatarSrc(conversion.profile)"
        />
        <el-tooltip effect="dark" :content="conversion.name" placement="top">
          <span class="conversion-name">{{ conversion.name }}</span>
        </el-tooltip>
        <span class="conversion-status">
          <i
            v-if="conversion.status === 1 || conversion.status === 2"
            class="el-icon-loading"
          ></i>
          <i
            v-if="conversion.status === 3"
            class="el-icon-circle-check"
            style="color: #67c23a"
          ></i>
          <i
            v-if="conversion.status === 4"
            class="el-icon-circle-close"
            style="color: #f56c6c"
          ></i>
        </span>
      </div>
      <div class="right-info">
        <div v-show="conversion.timeCost && !conversion.isOpen">
          <span class="conversion-time">
            {{ `${$t('agent.runCompleted')}： ${conversion.timeCost}` }}
          </span>
        </div>
        <div class="right-action" @click="toggleSubConversion(conversion)">
          <i
            class="el-icon-arrow-left expand-btn"
            :class="[conversion.isOpen && 'expand-btn_active']"
          ></i>
        </div>
      </div>
    </div>
    <div v-show="conversion.isOpen" class="sub-conversion-content-wrapper">
      <Knowlege
        v-if="
          conversion.conversationType ===
          AGENT_MESSAGE_CONFIG.AGENT_KNOWLEDGE.CONVERSATION_TYPE
        "
        :conversion="conversion"
        :parents-index="parentsIndex"
      />
      <div
        v-if="conversion.response"
        class="sub-conversion-content"
        :class="{
          'is-think':
            conversion.conversationType ===
            AGENT_MESSAGE_CONFIG.AGENT_THINK.CONVERSATION_TYPE,
        }"
      >
        <template>
          <template
            v-if="
              (conversion.stableChunks && conversion.stableChunks.length) ||
              conversion.activeResponse
            "
          >
            <div
              v-for="(chunk, idx) in conversion.stableChunks"
              :key="idx"
              class="chunk_stable"
              v-html="chunk"
            ></div>
            <div
              v-if="conversion.activeResponse"
              class="chunk_active"
              v-html="conversion.activeResponse"
            ></div>
          </template>
          <div
            v-else
            class="markdown-body"
            v-html="md.render(conversion.response)"
          ></div>
        </template>
      </div>
      <!-- 子会话出处 -->
      <div
        v-if="conversion.searchList && conversion.searchList.length"
        class="search-list subConversionSearchList"
        style="padding: 10px 0 0 0"
      >
        <template v-for="(searchItem, searchIndex) in conversion.searchList">
          <div
            :key="`${searchIndex}subsl`"
            :data-citation-index="searchIndex + 1"
            v-if="(conversion.citationsTagList || []).includes(searchIndex + 1)"
            class="search-list-item"
          >
            <div class="serach-list-item">
              <span @click="collapseClick(searchItem, searchIndex)">
                <i
                  :class="[
                    searchItem.collapse
                      ? 'el-icon-caret-bottom'
                      : 'el-icon-caret-right',
                  ]"
                ></i>
                {{ $t('agent.source') }}：
              </span>

              <a
                v-if="searchItem.link"
                :href="searchItem.link"
                target="_blank"
                rel="noopener noreferrer"
                class="link"
              >
                {{ searchItem.link }}
              </a>

              <span v-if="searchItem.title">
                <i
                  class="subTag"
                  data-citation-type="sub"
                  :data-pid="conversion.id"
                  :data-parents-index="parentsIndex"
                  :data-collapse="searchItem.collapse ? 'true' : 'false'"
                >
                  {{ searchIndex + 1 }}
                </i>
                {{ searchItem.title }}
              </span>
            </div>

            <el-collapse-transition>
              <div v-show="searchItem.collapse" class="snippet">
                <p v-html="md.render(searchItem.snippet || '')"></p>
              </div>
            </el-collapse-transition>
          </div>
        </template>
      </div>
    </div>
    <div
      class="sub-conversion-footer"
      v-if="conversion.timeCost && conversion.isOpen"
    >
      <span class="conversion-time">
        {{ `${$t('agent.runCompleted')}： ${conversion.timeCost}` }}
      </span>
    </div>
  </div>
</template>

<script>
import { md } from '@/mixins/markdown-it';
import { avatarSrc } from '@/utils/util';
import Knowlege from './knowlege.vue';
import { AGENT_MESSAGE_CONFIG } from '@/components/stream/constants';

export default {
  name: 'SubConversion',
  components: {
    Knowlege,
  },
  props: {
    /**
     * 子会话数据
     * @property {string} response - 渲染后的HTML回复内容
     * @property {Array} searchList - 引用结果列表
     * @property {string} parentId - 父会话ID(仅作为未来区分工具的上级是mainAgent还是subAgent用)
     * @property {string} id - 子会话唯一ID
     * @property {string} name - 子会话名称
     * @property {string} profile - 子会话头像路径
     * @property {string} timeCost - 消耗时长
     * @property {number} status - 状态 (1:进行中, 2:输出中, 3:已完成, 4:失败)
     * @property {string} conversationType - 会话类型 ('subAgent'子智能体|'agentTool'主智能体工具|'subAgentTool'子智能体工具)
     * @property {Array<number>} citationsTagList - 提取的引用tag列表(引用下标需-1计算)
     */
    conversion: {
      type: Object,
      required: true,
    },
    // 父会话索引
    parentsIndex: {
      type: Number,
      required: true,
    },
  },
  emits: ['toggle-conversion', 'collapse-click'],
  data() {
    return {
      AGENT_MESSAGE_CONFIG,
      md: md,
    };
  },
  methods: {
    avatarSrc,
    toggleSubConversion(conversion) {
      this.$emit('toggle-conversion', conversion);
    },
    collapseClick(searchItem, index) {
      this.$emit('collapse-click', this.conversion, searchItem, index);
    },
  },
};
</script>

<style scoped lang="scss">
.sub-conversion-item {
  background: #f2f3f8;
  border-radius: 8px;
  border: 1px solid #eef0f5;

  .logo {
    width: 18px;
    height: 18px;
    border-radius: 6px;
    object-fit: cover;
    flex-shrink: 0; /* 防止头像被压缩 */
    background: none; /* 头像无背景色 */

    &.logo-large {
      width: 30px;
      height: 30px;
    }
  }

  ::v-deep li {
    list-style-position: inside !important;
  }

  .sub-conversion-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 8px 12px;
    background: #f2f3f8;
    border-bottom: 1px solid #eef0f5;
    font-size: 13px;
    gap: 4px;

    .left-info {
      display: flex;
      align-items: center;
      gap: 8px;
      font-weight: 500;
      color: #333;
      flex: 1;
      min-width: 0;
      min-height: 30px;

      .conversion-name {
        min-width: 0;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
      }

      .conversion-status {
        flex-shrink: 0;
      }
    }

    .right-info {
      display: flex;
      align-items: center;
      gap: 8px;
    }

    .right-action {
      cursor: pointer;
      color: #999;
      padding: 4px;
      &:hover {
        color: #666;
      }
      .expand-btn {
        transition: all 0.3s;
        &_active {
          transform: rotate(-90deg);
        }
      }
    }
  }

  ::v-deep .sub-conversion-content-wrapper {
    padding: 10px 12px;
    background: #edeef5;
    p:has(img) {
      display: flex;
      flex-direction: column;
      align-items: flex-start;
    }
    img {
      align-self: center;
      max-width: 100%;
      max-height: 50vh;
      min-height: 50px;
      background: #ccc;
      object-fit: contain;
    }
  }

  .sub-conversion-content {
    font-size: 14px;
    color: #666;
    line-height: 1.5;
    background: #fff;
    padding: 10px;
    border-radius: 6px;

    &.is-think {
      color: #999;
    }

    p:has(img) {
      display: flex;
      flex-direction: column;
      align-items: flex-start;
    }

    img {
      align-self: center;
      width: 100% !important;
      max-height: 50vh;
      min-height: 50px;
      background: #ccc;
      object-fit: contain;
    }

    ::v-deep p {
      margin: 0;
    }

    ::v-deep .citation {
      display: inline-flex;
      color: $color;
      border-radius: 50%;
      width: 18px;
      height: 18px;
      border: 1px solid $color;
      cursor: pointer;
      line-height: 18px;
      vertical-align: middle;
      margin-left: 5px;
      justify-content: center;
      align-items: center;
      font-size: 14px;
      overflow: hidden;
      white-space: nowrap;
      margin-bottom: 2px;
      transform: scale(0.8);
      top: 0;
    }
  }

  .sub-conversion-footer {
    background: #f2f3f8;
    text-align: left;
    padding: 10px 14px;
  }

  .conversion-time {
    border-radius: 8px;
    padding: 4px 12px;
    color: #4b7902;
    font-size: 12px;
    background: #c9e9d7;
  }
}

.search-list {
  padding: 10px 20px 3px 54px;
  .qa_content {
    display: flex;
    gap: 10px;
    margin-top: 5px;
  }
  .recommended-question-title {
    border-bottom: 1px solid #e5e5e5;
    padding: 5px 0;
  }
  .search-list-item {
    margin-bottom: 5px;
    line-height: 22px;
    p:nth-child(1) {
      white-space: normal;
    }
    a,
    span {
      color: #666;
      cursor: pointer;
      white-space: normal;
      overflow-wrap: break-word;
    }
    a {
      text-decoration: underline;
    }
    a:hover {
      color: deepskyblue;
    }
    .snippet {
      padding: 5px 14px;
    }
  }
}
.serach-list-item {
  .link:hover {
    color: $color !important;
  }
  .search-doc {
    margin-left: 10px;
    cursor: pointer;
    color: $color !important;
  }
  .subTag {
    display: inline-flex;
    color: $color;
    border-radius: 50%;
    width: 18px;
    height: 18px;
    border: 1px solid $color;
    line-height: 18px;
    vertical-align: middle;
    margin-left: 2px;
    justify-content: center;
    align-items: center;
    font-size: 14px;
    overflow: hidden;
    white-space: nowrap;
    margin-bottom: 2px;
    transform: scale(0.8);
    font-style: normal;
  }
}
</style>
