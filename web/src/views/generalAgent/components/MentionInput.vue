<template>
  <div class="mention-input-wrapper" ref="inputWrapper">
    <el-popover
      ref="configPopover"
      placement="top-start"
      trigger="manual"
      :visible-arrow="false"
      popper-class="config-popover"
      v-model="showConfigPopover"
    >
      <div class="config-popover-content" @mousedown.prevent>
        <!-- Tab 切换 -->
        <div class="popover-tabs">
          <div
            v-for="tab in tabs"
            :key="tab.key"
            class="tab-item"
            :class="{ active: popoverTab === tab.key }"
            @click="popoverTab = tab.key"
          >
            {{ tab.label }}
          </div>
        </div>
        <div class="popover-list">
          <div
            v-for="(item, index) in currentFilteredList"
            :key="`${item.resourceType}_${item.id || item.name}_${index}`"
            class="popover-item"
            :class="{ selected: index === selectedIndex }"
            @click="selectConfigItem(item)"
          >
            <div class="item-avatar">
              <img
                v-if="item.avatar?.path"
                :src="avatarSrc(item.avatar.path)"
              />
            </div>
            <div class="item-info">
              <div class="item-name-wrapper">
                <span class="item-name">{{ item.name }}</span>
                <span
                  v-if="popoverTab === 'all' && item.resourceType"
                  class="tag"
                >
                  {{ $t(`generalAgent.config.${item.resourceType}`) }}
                </span>
              </div>
              <div v-if="item.author" class="item-desc">
                作者：{{ item.author }}
              </div>
              <div class="item-desc">{{ item.desc }}</div>
            </div>
          </div>
          <div v-if="currentFilteredList.length === 0" class="empty-tip">
            {{ $t('common.noData') }}
          </div>
        </div>
      </div>

      <div
        slot="reference"
        ref="senderRef"
        class="x-sender-container"
        :class="{ disabled: disabled }"
      ></div>
    </el-popover>
  </div>
</template>

<script>
import { getGeneralAgentResourceSelect } from '@/api/generalAgent';
import { avatarSrc } from '@/utils/util';
import XSender from 'x-sender';
import 'x-sender/style';
import PinyinMatch from 'pinyin-match';

export default {
  name: 'MentionInput',
  props: {
    value: {
      type: String,
      default: '',
    },
    placeholder: {
      type: String,
      default: '',
    },
    disabled: {
      type: Boolean,
      default: false,
    },
  },
  data() {
    return {
      inputValue: this.value,
      showConfigPopover: false,
      popoverTab: '',
      tabs: [],
      resourceList: {},
      mentionStartPos: -1,
      mentionSearchText: '',
      selectedIndex: 0,
      sender: null,
      ontologyId: null, // ontology单选
    };
  },
  computed: {
    availableResourceTypes() {
      return Object.keys(this.resourceList).filter(
        type => this.resourceList[type] && this.resourceList[type].length > 0,
      );
    },

    // 合并所有资源列表为"全部"
    allResourcesList() {
      const allList = [];
      this.availableResourceTypes.forEach(type => {
        const list = this.resourceList[type] || [];
        // 为每个item添加type标识
        const itemsWithType = list.map(item => ({
          ...item,
          resourceType: type,
        }));
        allList.push(...itemsWithType);
      });
      return allList;
    },

    currentConfig() {
      const type = this.popoverTab;
      if (!type) {
        return {};
      }

      // "all"类型返回所有资源的总和
      if (type === 'all') {
        return {
          list: this.allResourcesList,
        };
      }

      return {
        list: this.resourceList[type] || [],
      };
    },

    currentFilteredList() {
      const { list } = this.currentConfig;
      return this.filterList(list || []);
    },
  },
  watch: {
    value(newVal) {
      if (newVal === this.inputValue) {
        return;
      }
      this.clear();
      this.sender.setText(newVal);
    },
    inputValue(newVal) {
      this.$emit('input', newVal);
    },
    popoverTab() {
      this.selectedIndex = 0;
    },
    placeholder(newVal) {
      this.sender.updateConfig({
        placeholder: newVal,
      });
    },
    disabled(newVal) {
      newVal ? this.sender.disable() : this.sender.enable();
    },
    availableResourceTypes(newVal) {
      if (newVal.length > 0 && this.tabs.length === 0) {
        this.initTabs();
      }
    },
  },
  methods: {
    avatarSrc,

    initTabs() {
      // 第一列为"全部"选项
      this.tabs = [
        {
          key: 'all',
          label: this.$t('common.all'),
        },
        ...this.availableResourceTypes.map(type => ({
          key: type,
          label: this.$t(`generalAgent.config.${type}`),
        })),
      ];

      if (this.tabs.length > 0 && !this.popoverTab) {
        this.popoverTab = this.tabs[0].key;
      }
    },

    // 通用的列表过滤方法 - 支持中英文和拼音混合搜索
    filterList(list) {
      if (!this.mentionSearchText) {
        return list;
      }
      // 去除拼音中的撇号,使 ti'a 能匹配到 tia
      const searchText = this.mentionSearchText.trim().replaceAll("'", '');
      return list.filter(item => {
        // 检测是否为"中文+字母"的格式
        const mixedPattern = /^([\u4e00-\u9fa5]+)([a-zA-Z]+)$/;
        const match = searchText.match(mixedPattern);

        if (match) {
          // 中文+拼音模式:如 "地t" 匹配 "高德地图"
          const [, chinesePart, pinyinPart] = match;

          // 在目标名称中查找中文部分的位置
          const chineseIndex = item.name.indexOf(chinesePart);
          if (chineseIndex === -1) {
            return false;
          }

          // 获取中文部分之后的剩余文本,用 PinyinMatch 匹配拼音
          const remainingText = item.name.substring(
            chineseIndex + chinesePart.length,
          );
          return PinyinMatch.match(remainingText, pinyinPart);
        } else {
          return PinyinMatch.match(item.name, searchText);
        }
      });
    },

    initSender() {
      this.sender = new XSender(this.$refs.senderRef, {
        placeholder: this.placeholder,
        autoFocus: false,
      });

      const { EVENT_COMMON_CHANGE } = XSender.EventSet;
      this.sender.bus.on('XSender', EVENT_COMMON_CHANGE, () => {
        this.inputValue = this.sender.getText();
        if (this.showConfigPopover) {
          this.updateMentionPosition();
          this.$nextTick(() => {
            if (this.mentionStartPos === -1) return;
            const allList = this.filterList(this.allResourcesList);
            if (allList.length === 0) {
              this.showConfigPopover = false;
            }
          });
        }
      });

      this.sender.chatElement.richText.addEventListener(
        'keydown',
        e => {
          this.handleSenderKeydown(e);
        },
        true,
      );

      this.sender.chatElement.richText.addEventListener('blur', () => {
        this.handleSenderBlur();
      });

      this.sender.chatElement.richText.addEventListener('keyup', e => {
        if (e.key === '@' || +e.key === 2) {
          const { instance, offset } = this.sender.getCurrentNode();
          if (instance?.type !== 'Write') return;
          if (instance.text[offset - 1] !== '@') return;
          this.popoverTab = 'all';
          this.updateMentionPosition();
          this.showConfigPopover = true;
          this.selectedIndex = 0;
        }
      });
    },

    resetMentionState() {
      this.showConfigPopover = false;
      this.mentionStartPos = -1;
      this.mentionSearchText = '';
      this.selectedIndex = 0;
    },

    // 更新@提及的位置信息
    updateMentionPosition() {
      const { instance, offset } = this.sender.getCurrentNode();
      if (instance?.type !== 'Write') return;

      const currentText = instance.text;
      const lastAtIndex = currentText.substring(0, offset).lastIndexOf('@');

      this.mentionStartPos = lastAtIndex;
      if (this.mentionStartPos === -1) this.resetMentionState();
      else
        this.mentionSearchText = currentText.substring(lastAtIndex + 1, offset);
    },

    handleSenderKeydown(e) {
      if (this.showConfigPopover) {
        const keyHandlers = {
          Escape: () => {
            this.resetMentionState();
          },
          ArrowUp: () => this.handleKeyboardNavigation('ArrowUp'),
          ArrowDown: () => this.handleKeyboardNavigation('ArrowDown'),
          ArrowLeft: () => this.handleTabSwitch('ArrowLeft'),
          ArrowRight: () => this.handleTabSwitch('ArrowRight'),
          Enter: () =>
            this.selectConfigItem(this.currentFilteredList[this.selectedIndex]),
        };

        if (keyHandlers[e.key]) {
          e.preventDefault();
          e.stopPropagation();
          keyHandlers[e.key]();
        }
      } else if (e.key === 'Enter' && !e.shiftKey) {
        this.$emit('keydown-enter', e);
        this.clear();
      }
    },

    handleSenderBlur() {
      setTimeout(() => {
        const popover = this.$refs.configPopover?.$refs?.popper;
        if (popover?.contains(document.activeElement)) {
          return;
        }
        this.resetMentionState();
      }, 200);
    },

    async fetchConfigData() {
      const res = await getGeneralAgentResourceSelect();

      if (res?.data && Array.isArray(res.data)) {
        this.resourceList = {};

        res.data.forEach(item => {
          const { listType, list } = item;
          if (listType && Array.isArray(list)) {
            this.resourceList[listType] = list.map(resource => ({
              ...resource,
              resourceType: listType,
            }));
          }
        });
      }
    },

    handleKeyboardNavigation(key) {
      if (this.currentFilteredList.length === 0) return;

      const delta = key === 'ArrowUp' ? -1 : 1;
      this.selectedIndex =
        (this.selectedIndex + delta + this.currentFilteredList.length) %
        this.currentFilteredList.length;

      this.$nextTick(() => {
        this.scrollToSelected();
      });
    },

    handleTabSwitch(key) {
      if (this.tabs.length === 0) return;

      const currentIndex = this.tabs.findIndex(
        tab => tab.key === this.popoverTab,
      );
      const delta = key === 'ArrowLeft' ? -1 : 1;
      const newIndex =
        (currentIndex + delta + this.tabs.length) % this.tabs.length;
      this.popoverTab = this.tabs[newIndex].key;
    },

    scrollToSelected() {
      const selectedItem = document.querySelector('.popover-item.selected');
      selectedItem?.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
    },

    selectConfigItem(item) {
      this.sender.backspace(-(this.mentionSearchText.length + 1));

      this.sender.setMention({
        id: item.id,
        name: item.name + ' ', // @Amap-高德地图 帮我查询一下西安钟楼到大雁塔的骑行路线
      });

      this.resetMentionState();

      if (item.resourceType === 'ontology') {
        if (this.ontologyId) {
          // 如果已存在 ontology 提及，先删除它
          this.$message.warning(
            this.$t('generalAgent.config.ontologySingleWarning'),
          );
          this.sender.removeMention([this.ontologyId]);
        }
        this.ontologyId = item.id;
      }
    },

    clear() {
      this.sender.reset();
      this.ontologyId = null;
    },
  },
  mounted() {
    this.fetchConfigData();
    this.initSender();
  },
  beforeDestroy() {
    this.sender.destroy();
    this.sender = null;
  },
};
</script>

<style lang="scss">
@import '@/style/tag';
.x-sender-container {
  position: relative;
  * {
    font-size: 16px !important;
    font-style: normal;
  }

  *:focus,
  *:focus-visible,
  *:focus-within {
    outline: none !important;
    border: none !important;
    box-shadow: none !important;
  }
}

.config-popover {
  padding: 0 !important;
  width: 500px;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);

  .config-popover-content {
    height: 400px;
    display: flex;
    flex-direction: column;

    &::-webkit-scrollbar {
      width: 6px;
    }

    &::-webkit-scrollbar-track {
      background: transparent;
    }

    &::-webkit-scrollbar-thumb {
      background: #d1d5db;
      border-radius: 3px;

      &:hover {
        background: #9ca3af;
      }
    }

    .popover-tabs {
      display: flex;
      border-bottom: 1px solid #e8e8e8;
      padding: 0 8px;
      margin-bottom: 8px;
      background: #fff;
      flex-shrink: 0;

      .tab-item {
        padding: 12px 16px;
        font-size: 13px;
        color: #666;
        cursor: pointer;
        transition: all 0.2s;
        border-bottom: 2px solid transparent;
        white-space: nowrap;

        &:hover {
          color: #1890ff;
        }

        &.active {
          color: #1890ff;
          border-bottom-color: #1890ff;
          font-weight: 500;
        }
      }
    }

    .popover-list {
      flex: 1;
      overflow-y: auto;
      min-height: 0;
      padding: 0 8px 8px 8px;

      .popover-category {
        margin-bottom: 12px;

        &:last-child {
          margin-bottom: 0;
        }

        .popover-category-name {
          font-size: 12px;
          font-weight: 500;
          color: #666;
          padding: 8px 8px 4px;
          margin-bottom: 4px;
        }
      }

      .popover-item {
        display: flex;
        align-items: center;
        padding: 10px 12px;
        border-radius: 8px;
        cursor: pointer;
        transition: all 0.2s;
        margin-bottom: 4px;

        &:hover,
        &.selected {
          background: #f5f7fa;
        }

        &:last-child {
          margin-bottom: 0;
        }

        .item-avatar {
          width: 32px;
          height: 32px;
          border-radius: 6px;
          margin-right: 10px;
          display: flex;
          align-items: center;
          justify-content: center;
          background: #f0f0f0;
          overflow: hidden;
          flex-shrink: 0;

          img {
            width: 100%;
            height: 100%;
            object-fit: cover;
          }

          i {
            font-size: 16px;
            color: #999;
          }
        }

        .item-info {
          flex: 1;
          min-width: 0;

          .item-name-wrapper {
            display: flex;
            align-items: center;
            gap: 6px;
            margin-bottom: 2px;

            .item-name {
              font-size: 13px;
              font-weight: 500;
              color: #1a1a1a;
              overflow: hidden;
              text-overflow: ellipsis;
              white-space: nowrap;
            }
          }

          .item-desc {
            font-size: 11px;
            color: #999;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
          }
        }
      }

      .empty-tip {
        text-align: center;
        padding: 20px;
        color: #999;
        font-size: 13px;
      }
    }
  }
}
</style>
