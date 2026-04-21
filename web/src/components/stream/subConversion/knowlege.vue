<!-- 子会话-知识库 -->
<template>
  <div class="sub-conversion-knowledge">
    <div
      v-for="(knowledgeItem, knowledgeIndex) in searchList"
      :key="knowledgeItem.id"
      class="knowledge-item"
      :data-index="knowledgeIndex"
    >
      <div class="knowledge-header">
        <span
          class="index-badge"
          data-citation-type="sub"
          :data-sub-id="conversion.id"
          :data-parents-index="parentsIndex"
        >
          {{ knowledgeIndex + 1 }}
        </span>
        <p
          class="doc-title"
          :title="knowledgeItem.title"
          @click="$emit('handleDocLink', knowledgeItem)"
        >
          {{ knowledgeItem.title }}
        </p>
        <svg-icon
          v-if="
            knowledgeItem.meta_data && knowledgeItem.meta_data.download_link
          "
          icon-class="download"
          class="download-icon"
          @click="handleDownloadknowledgeFile(knowledgeItem)"
        />
      </div>
      <div class="knowledge-meta">
        <span class="pill-tag kb-name" v-if="knowledgeItem.user_kb_name">
          {{ knowledgeItem.user_kb_name }}
        </span>
        <span class="pill-tag score">
          Score: {{ formatScore(knowledgeItem.score) }}
        </span>
      </div>
      <div class="knowledge-content">
        <div
          :ref="'snippet-' + knowledgeIndex"
          class="snippet"
          :class="{ 'is-collapsed': !expandedMap[knowledgeIndex] }"
          v-html="md.render(knowledgeItem.snippet || '')"
        ></div>
        <div
          v-if="isOverflowMap[knowledgeIndex]"
          class="expand-btn"
          @click="toggleExpand(knowledgeIndex)"
        >
          {{
            expandedMap[knowledgeIndex]
              ? $t('common.button.fold')
              : $t('common.button.viewAll')
          }}
        </div>
      </div>
    </div>
  </div>
</template>

<script>
import { formatScore, fetchDownload } from '@/utils/util';
import { md } from '@/mixins/markdown-it';

export default {
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
    // 父级在 history 中的索引
    parentsIndex: {
      type: Number,
      default: 0,
    },
  },
  computed: {
    searchList() {
      return this.conversion.searchList;
    },
  },
  data() {
    return {
      md: md,
      formatScore,
      expandedMap: {}, // 用于记录每个条目的展开状态
      isOverflowMap: {}, // 用于记录每个条目是否溢出
    };
  },
  watch: {
    searchList: {
      handler() {
        this.$nextTick(() => {
          this.checkOverflow();
        });
      },
      immediate: true,
      deep: true,
    },
    conversion: {
      handler(val) {
        if (val.status === 3) {
          this.$nextTick(() => {
            this.checkOverflow();
          });
        }
      },
      deep: true,
      immediate: true,
    },
  },
  methods: {
    checkOverflow() {
      if (!this.searchList) return;
      this.searchList.forEach((item, index) => {
        const el =
          this.$refs[`snippet-${index}`] && this.$refs[`snippet-${index}`][0];
        if (el) {
          const isOverflow = el.scrollHeight > el.clientHeight;
          this.$set(this.isOverflowMap, index, isOverflow);
        }
      });
    },
    toggleExpand(index) {
      this.$set(this.expandedMap, index, !this.expandedMap[index]);
    },
    // 下载知识库文件
    handleDownloadknowledgeFile(knowledgeItem) {
      const { download_link, file_name } = knowledgeItem.meta_data;
      fetchDownload(download_link, file_name);
    },
  },
};
</script>

<style lang="scss" scoped>
.sub-conversion-knowledge {
  display: flex;
  flex-direction: column;
  gap: 12px;

  .knowledge-item {
    background: transparent;
    display: flex;
    flex-direction: column;
    gap: 8px;
    font-size: 14px;
    color: #666;
    line-height: 1.5;
    background: #fff;
    padding: 10px;
    border-radius: 6px;

    img,
    ::v-deep img {
      align-self: center;
      width: 100% !important;
      max-height: 50vh;
      min-height: 50px;
      background: #ccc;
      object-fit: contain;
    }

    .knowledge-header {
      display: flex;
      align-items: center;
      gap: 8px;

      .index-badge {
        display: flex;
        align-items: center;
        justify-content: center;
        width: 16px;
        height: 16px;
        border: 1px solid $color;
        border-radius: 50%;
        font-size: 12px;
        color: $color;
        flex-shrink: 0;
      }

      .doc-title {
        margin: 0;
        font-size: 14px;
        font-weight: 500;
        color: #303133;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;
        flex: 0 1 auto;
        cursor: pointer;
        &:hover {
          color: $color;
        }
      }

      .download-icon {
        flex-shrink: 0;
        cursor: pointer;
        color: rgb(102, 102, 102);
      }
    }

    .knowledge-meta {
      display: flex;
      align-items: center;
      gap: 8px;

      .pill-tag {
        display: inline-block;
        padding: 2px 10px;
        background-color: #f2f3f5;
        border-radius: 20px;
        font-size: 12px;
        color: #606266;
        white-space: nowrap;
      }

      .kb-name {
        max-width: calc(100% * 2 / 3);
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .score {
        flex-shrink: 0;
      }
    }

    .knowledge-content {
      .snippet {
        font-size: 13px;
        line-height: 1.6;
        color: #606266;
        word-break: break-all;

        &.is-collapsed {
          display: -webkit-box;
          -webkit-box-orient: vertical;
          -webkit-line-clamp: 8;
          line-clamp: 8;
          overflow: hidden;
          text-overflow: ellipsis;
        }
      }

      .expand-btn {
        margin-top: 4px;
        font-size: 12px;
        color: #409eff;
        cursor: pointer;
        display: inline-block;
        user-select: none;

        &:hover {
          opacity: 0.8;
        }
      }
    }
  }
}
</style>
