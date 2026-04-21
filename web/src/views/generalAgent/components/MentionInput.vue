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

        <!-- 列表内容 - 使用动态渲染 -->
        <div class="popover-list">
          <div
            v-for="(item, index) in currentFilteredList"
            :key="getItemKey(item)"
            class="popover-item"
            :class="{ selected: index === selectedIndex }"
            @click="selectConfigItem(item, currentType)"
          >
            <div class="item-avatar">
              <img v-if="item.avatar?.path" :src="avatarSrc(item.avatar.path)" />
              <i v-else :class="currentIcon"></i>
            </div>
            <div class="item-info">
              <div class="item-name">{{ item.name }}</div>
              <div class="item-desc">{{ getItemDesc(item) }}</div>
            </div>
          </div>
          <div v-if="currentFilteredList.length === 0" class="empty-tip">
            {{ $t('common.noData') }}
          </div>
        </div>
      </div>

      <div slot="reference" ref="senderRef" class="x-sender-container"></div>
    </el-popover>
  </div>
</template>

<script>
import {
  getGeneralAgentAssistantSelect,
  getGeneralAgentMcpSelect,
  getGeneralAgentSkillSelect,
  getGeneralAgentWorkflowSelect,
} from '@/api/generalAgent';
import { avatarSrc } from '@/utils/util';
import XSender from 'x-sender';
import 'x-sender/style';

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
      popoverTab: 'mcp', // 当前 popover 显示的 tab
      tabs: [
        { key: 'mcp', label: this.$t('generalAgent.config.mcp') },
        { key: 'workflows', label: this.$t('generalAgent.config.workflows') },
        { key: 'skills', label: this.$t('generalAgent.config.skills') },
        { key: 'assistants', label: this.$t('generalAgent.config.agents') },
      ],
      mcpList: [],
      workflowList: [],
      skillList: [],
      assistantList: [],
      mentionStartPos: -1, // @ 符号的位置
      mentionSearchText: '', // @ 后的搜索文本
      selectedIndex: 0, // 当前选中的索引
      sender: null, // XSender 实例
    };
  },
  computed: {
    // 统一的配置项映射
    configTypeMap() {
      return {
        mcp: { list: this.mcpList, keyField: 'mcpId', descField: 'description', icon: 'el-icon-connection' },
        workflows: { list: this.workflowList, keyField: 'appId', descField: 'desc', icon: 'el-icon-share' },
        skills: { list: this.skillList, keyField: 'skillId', descField: 'desc', icon: 'el-icon-document' },
        assistants: { list: this.assistantList, keyField: 'appId', descField: 'desc', icon: 'el-icon-user' },
      };
    },

    // 当前类型配置
    currentConfig() {
      return this.configTypeMap[this.popoverTab] || {};
    },

    // 当前过滤后的列表
    currentFilteredList() {
      const { list, descField } = this.currentConfig;
      return this.filterList(list || [], descField);
    },

    // 当前类型
    currentType() {
      const typeMap = {
        mcp: 'mcp',
        workflows: 'workflow',
        skills: 'skill',
        assistants: 'assistant',
      };
      return typeMap[this.popoverTab];
    },

    // 当前图标
    currentIcon() {
      return this.currentConfig.icon || 'el-icon-document';
    },

    filteredMcpList() {
      return this.filterList(this.mcpList, 'description');
    },

    filteredWorkflowList() {
      return this.filterList(this.workflowList, 'desc');
    },

    filteredSkillList() {
      return this.filterList(this.skillList, 'desc');
    },

    filteredAssistantList() {
      return this.filterList(this.assistantList, 'desc');
    },
  },
  watch: {
    value(newVal) {
      this.inputValue = newVal;
    },
    inputValue(newVal) {
      this.$emit('input', newVal);
    },
    popoverTab() {
      this.selectedIndex = 0;
    },
    placeholder(newVal) {
      if (this.sender) {
        this.sender.updateConfig({
          placeholder: newVal,
        });
      }
    },
  },
  methods: {
    avatarSrc,

    // 通用的列表过滤方法
    filterList(list, descField) {
      if (!this.mentionSearchText) {
        return list;
      }
      const searchText = this.mentionSearchText.toLowerCase();
      return list.filter(
        item =>
          item.name?.toLowerCase().includes(searchText) ||
          item[descField]?.toLowerCase().includes(searchText),
      );
    },

    // 获取列表项的 key
    getItemKey(item) {
      const { keyField } = this.currentConfig;
      return item[keyField] || item.name;
    },

    // 获取列表项的描述
    getItemDesc(item) {
      const { descField } = this.currentConfig;
      return item[descField];
    },

    initSender() {
      if (this.$refs.senderRef) {
        this.sender = new XSender(this.$refs.senderRef, {
          placeholder: this.placeholder,
          autoFocus: false,
          disabled: this.disabled,
        });

        // 使用 XSender 官方事件总线监听内容变化（包括输入和删除）
        const { EVENT_COMMON_CHANGE } = XSender.EventSet;
        const busKey = 'XSender';

        this.sender.bus.on(busKey, EVENT_COMMON_CHANGE, () => {
          this.inputValue = this.sender.getText();
          if (this.showConfigPopover) {
            this.updateMentionSearch();
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

            this.triggerMentionPopover();
          }
        });
      }
    },

    // 重置提及搜索状态
    resetMentionState() {
      this.showConfigPopover = false;
      this.mentionStartPos = -1;
      this.mentionSearchText = '';
      this.selectedIndex = 0;
    },

    triggerMentionPopover() {
      this.showConfigPopover = true;
      this.selectedIndex = 0;
      this.updateMentionSearch();
    },

    getCursorPosition() {
      try {
        const selection = window.getSelection();
        if (!selection || selection.rangeCount === 0) return 0;

        const range = selection.getRangeAt(0);
        const preCaretRange = range.cloneRange();
        preCaretRange.selectNodeContents(this.sender.chatElement.richText);
        preCaretRange.setEnd(range.endContainer, range.endOffset);
        return preCaretRange.toString().length;
      } catch (e) {
        console.error('获取光标位置失败:', e);
        return 0;
      }
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
          Enter: () => this.selectCurrentItem(),
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

    updateMentionSearch() {
      if (!this.inputValue || this.inputValue.length === 0) {
        this.resetMentionState();
        return;
      }

      const cursorPos = this.getCursorPosition();
      const beforeCursor = this.inputValue.substring(0, cursorPos);
      const lastAtIndex = beforeCursor.lastIndexOf('@');

      if (lastAtIndex !== -1) {
        this.mentionStartPos = lastAtIndex;
        this.mentionSearchText = this.inputValue.substring(lastAtIndex + 1, cursorPos);

        this.$nextTick(() => {
          this.$refs.configPopover?.updatePopper();
        });
      } else {
        this.resetMentionState();
      }
    },

    handleSenderBlur() {
      setTimeout(() => {
        const popover = this.$refs.configPopover?.$refs?.popper;
        if (popover && popover.contains(document.activeElement)) {
          return;
        }
        this.resetMentionState();
      }, 200);
    },

    async fetchConfigData() {
      try {
        const [mcpRes, workflowRes, skillRes, assistantRes] =
          await Promise.allSettled([
            getGeneralAgentMcpSelect(),
            getGeneralAgentWorkflowSelect(),
            getGeneralAgentSkillSelect(),
            getGeneralAgentAssistantSelect(),
          ]);

        // 统一处理响应数据
        const handleResponse = (res, targetProp) => {
          if (res.status === 'fulfilled' && res.value?.data?.list) {
            this[targetProp] = res.value.data.list;
          }
        };

        handleResponse(mcpRes, 'mcpList');
        handleResponse(workflowRes, 'workflowList');
        handleResponse(skillRes, 'skillList');
        handleResponse(assistantRes, 'assistantList');
      } catch (error) {
        console.error('获取配置数据失败:', error);
      }
    },

    getCurrentList() {
      const typeMap = {
        mcp: this.filteredMcpList,
        workflows: this.filteredWorkflowList,
        skills: this.filteredSkillList,
        assistants: this.filteredAssistantList,
      };
      return typeMap[this.popoverTab] || [];
    },

    handleTabSwitch(key) {
      const tabs = ['mcp', 'workflows', 'skills', 'assistants'];
      const currentIndex = tabs.indexOf(this.popoverTab);
      const delta = key === 'ArrowLeft' ? -1 : 1;
      const newIndex = (currentIndex + delta + tabs.length) % tabs.length;
      this.popoverTab = tabs[newIndex];
    },

    handleKeyboardNavigation(key) {
      const currentList = this.getCurrentList();
      if (currentList.length === 0) return;

      const delta = key === 'ArrowUp' ? -1 : 1;
      this.selectedIndex = (this.selectedIndex + delta + currentList.length) % currentList.length;

      this.$nextTick(() => {
        this.scrollToSelected();
      });
    },

    scrollToSelected() {
      const selectedItem = document.querySelector('.popover-item.selected');
      if (selectedItem) {
        selectedItem.scrollIntoView({ block: 'nearest', behavior: 'smooth' });
      }
    },

    selectCurrentItem() {
      const currentList = this.getCurrentList();
      if (currentList.length === 0 || this.selectedIndex < 0) return;

      const typeMap = {
        mcp: 'mcp',
        workflows: 'workflow',
        skills: 'skill',
        assistants: 'assistant',
      };

      this.selectConfigItem(currentList[this.selectedIndex], typeMap[this.popoverTab]);
    },

    selectConfigItem(item, type) {
      if (!this.inputValue || this.mentionStartPos === -1) return;

      // 删除 @ 和搜索文本
      this.sender.backspace(-(this.mentionSearchText.length + 1));

      // 插入提及项
      this.sender.setMention({
        id: item.mcpId || item.appId || item.skillId,
        name: item.name,
        type,
      });

      // 重置状态并聚焦
      this.resetMentionState();
      this.$nextTick(() => {
        this.sender?.focus();
      });

      // 更新输入值
      this.inputValue = this.sender.getText();
      this.$emit('input', this.inputValue);
    },

    async loadData() {
      await this.fetchConfigData();
    },

    focus() {
      if (this.sender) {
        this.sender.focus();
      }
    },

    blur() {
      if (this.sender) {
        this.sender.blur();
      }
    },

    clear() {
      if (this.sender) {
        this.sender.chatElement.richText.innerHTML = '';
        this.inputValue = '';
        this.$emit('input', '');
      }
    },
  },
  mounted() {
    this.loadData();
    this.initSender();
  },
  beforeDestroy() {
    if (this.sender) {
      this.sender.destroy();
      this.sender = null;
    }
  },
};
</script>

<style lang="scss">
.config-popover {
  padding: 0 !important;
  width: 400px;
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

          .item-name {
            font-size: 13px;
            font-weight: 500;
            color: #1a1a1a;
            margin-bottom: 2px;
            overflow: hidden;
            text-overflow: ellipsis;
            white-space: nowrap;
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
