<template>
  <div>
    <el-dialog
      :visible.sync="dialogVisible"
      width="50%"
      custom-class="config-dialog"
      :close-on-click-modal="false"
      @close="handleClose"
    >
      <div slot="title" class="dialog-title">
        <h3>{{ $t('generalAgent.config.title') }}</h3>
      </div>

      <div class="dialog-body">
        <div class="top-area">
          <div class="tab-buttons">
            <div
              v-if="hasTools"
              :class="['tab-btn', { active: activeTab === 'tools' }]"
              @click="activeTab = 'tools'"
            >
              {{ $t('generalAgent.config.tools') }}
            </div>
            <template v-for="type in availableResourceTypes">
              <div
                :key="type"
                v-if="shouldShowTab(type)"
                :class="['tab-btn', { active: activeTab === type }]"
                @click="activeTab = type"
              >
                {{ getTabLabel(type) }}
              </div>
            </template>
          </div>
          <div v-if="activeTab !== 'tools'" class="search-box">
            <el-input
              v-model="searchKeyword"
              :placeholder="$t('common.input.searchPlaceholder')"
              prefix-icon="el-icon-search"
              clearable
              size="small"
            />
          </div>
        </div>
        <div class="config-content">
          <div v-if="activeTab === 'tools'" class="tool-categories">
            <div
              v-for="(category, categoryIndex) in toolList"
              :key="category.category"
              class="tool-category"
              :class="{
                'validation-error': validationErrors.has(categoryIndex),
              }"
            >
              <div class="category-header">
                <span class="category-name">{{ category.category }}</span>
                <span
                  v-if="validationErrors.has(categoryIndex)"
                  class="error-tip"
                >
                  {{ $t('generalAgent.config.validationError') }}
                </span>
                <el-tag
                  size="mini"
                  :type="getConditionType(category.condition)"
                >
                  {{ getConditionLabel(category.condition) }}
                </el-tag>
              </div>
              <div class="resource-list">
                <div
                  v-for="tool in category.toolList"
                  :key="tool.toolId"
                  :class="[
                    'item-item',
                    {
                      selected: isItemSelected(tool.toolId),
                    },
                  ]"
                  @click="handleToggleItem(tool)"
                >
                  <div class="item-avatar">
                    <img
                      v-if="tool.avatar?.path"
                      :src="avatarSrc(tool.avatar.path)"
                    />
                  </div>
                  <div class="item-info">
                    <div class="item-name">{{ tool.toolName }}</div>
                    <div class="item-desc">{{ tool.desc }}</div>
                    <div v-if="needsApiKeyReminder(tool)" class="api-key-tip">
                      <i class="el-icon-warning"></i>
                      {{ $t('generalAgent.config.needApiKey') }}
                    </div>
                  </div>
                  <el-checkbox
                    :value="isItemSelected(tool.toolId)"
                    @click.native.stop
                    @change="handleToggleItem(tool)"
                  />
                </div>
              </div>
            </div>
          </div>

          <div v-else-if="activeTab !== 'tools'" class="resource-list">
            <div
              v-for="item in filteredList"
              :key="item.id"
              :class="[
                'item-item',
                {
                  selected: isItemSelected(item.id),
                },
              ]"
              @click="handleToggleItem(item)"
            >
              <div class="item-avatar">
                <img
                  v-if="item.avatar?.path"
                  :src="avatarSrc(item.avatar.path)"
                />
              </div>
              <div class="item-info">
                <div class="item-name">
                  {{ item.name }}
                </div>
                <div v-if="item.author" class="item-desc">
                  作者：{{ item.author }}
                </div>
                <div class="item-desc">
                  {{ item.desc }}
                </div>
              </div>
              <el-radio
                v-if="currentListConfig.type === 'ontology'"
                :label="item.id"
                :value="selectedResources[currentListConfig.type]?.[0]?.id"
                @change="handleToggleItem(item)"
                @click.native.stop
              >
                {{ '' }}
              </el-radio>
              <el-checkbox
                v-else
                :value="isItemSelected(item.id)"
                @click.native.stop
                @change="handleToggleItem(item)"
              />
            </div>
            <div v-if="filteredList.length === 0" class="empty-tip">
              {{ $t('common.noData') }}
            </div>
          </div>
        </div>
      </div>

      <div slot="footer" class="dialog-footer">
        <el-button @click="handleClose">
          {{ $t('generalAgent.config.cancel') }}
        </el-button>
        <el-button type="primary" @click="handleConfirm">
          {{ $t('generalAgent.config.confirm') }}
        </el-button>
      </div>
    </el-dialog>

    <el-dialog
      :visible.sync="apiKeyModalVisible"
      width="500px"
      custom-class="api-key-dialog"
      :close-on-click-modal="false"
      :title="$t('generalAgent.config.apiKeyTitle')"
      @close="handleApiKeyModalClose"
    >
      <div class="api-key-input-container">
        <el-input
          v-model="apiKeyValue"
          :placeholder="$t('generalAgent.config.apiKeyPlaceholder')"
          size="large"
          @keyup.enter.native="handleApiKeySubmit"
        />
      </div>
      <div slot="footer" class="dialog-footer">
        <el-button @click="handleApiKeyModalClose">
          {{ $t('generalAgent.config.cancel') }}
        </el-button>
        <el-button
          type="primary"
          :loading="submitting"
          @click="handleApiKeySubmit"
        >
          {{ $t('generalAgent.config.confirm') }}
        </el-button>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import { avatarSrc } from '@/utils/util';
import {
  getGeneralAgentToolSelect,
  getGeneralAgentResourceSelect,
  updateGeneralAgentGlobalConfig,
  getGeneralAgentGlobalConfig,
} from '@/api/generalAgent';
import { changeApiKey } from '@/api/mcp';

export default {
  name: 'ConfigDialog',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    agentId: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      dialogVisible: this.visible,
      activeTab: 'tools',
      toolList: [],
      resourceList: {},
      selectedTools: [],
      selectedResources: {},
      validationErrors: new Set(),
      apiKeyModalVisible: false,
      currentTool: null,
      apiKeyValue: '',
      submitting: false,
      searchKeyword: '',
    };
  },
  mounted() {
    this.fetchAllData();
  },
  watch: {
    visible(val) {
      this.dialogVisible = val;
      if (val) {
        this.fetchAllData();
      }
    },
  },
  computed: {
    // 判断是否有工具数据
    hasTools() {
      return this.toolList.length > 0;
    },
    // 动态获取所有可用的资源类型
    availableResourceTypes() {
      return Object.keys(this.resourceList).filter(
        type => this.resourceList[type] && this.resourceList[type].length > 0,
      );
    },
    // 当前列表配置（根据 activeTab 动态返回）
    currentListConfig() {
      if (this.activeTab === 'tools') {
        return {};
      }

      // 直接使用 activeTab 作为资源类型
      const resourceType = this.activeTab;
      if (!this.resourceList[resourceType]) {
        return {};
      }

      return {
        list: this.resourceList[resourceType],
        type: resourceType,
      };
    },
    // 过滤后的列表（支持搜索）
    filteredList() {
      if (!this.searchKeyword) {
        return this.currentListConfig.list || [];
      }
      const keyword = this.searchKeyword.toLowerCase();
      return (this.currentListConfig.list || []).filter(item => {
        const name = item.name?.toLowerCase() || '';
        return name.includes(keyword);
      });
    },
  },
  methods: {
    avatarSrc,
    async fetchAllData() {
      await Promise.allSettled([
        this.fetchToolList(),
        this.fetchResourceList(),
      ]);
      await this.fetchGlobalConfig();
      this.selectFirstAvailableTab();
    },
    // 自动选择第一个有数据的tab
    selectFirstAvailableTab() {
      // 如果有工具，优先选择 tools
      if (this.hasTools) {
        this.activeTab = 'tools';
        return;
      }

      // 否则选择第一个可用的资源类型
      if (this.availableResourceTypes.length > 0) {
        this.activeTab = this.availableResourceTypes[0];
      }
    },
    // 判断是否应该显示某个 tab
    shouldShowTab(resourceType) {
      return (
        this.resourceList[resourceType] &&
        this.resourceList[resourceType].length > 0
      );
    },
    // 获取 tab 标签文本
    getTabLabel(resourceType) {
      return this.$t(`generalAgent.config.${resourceType}`);
    },
    async fetchToolList() {
      const res = await getGeneralAgentToolSelect({ agentId: this.agentId });
      this.toolList = res?.data?.list || [];
    },
    async fetchResourceList() {
      const res = await getGeneralAgentResourceSelect();
      if (res?.data && Array.isArray(res.data)) {
        this.resourceList = {};
        const newSelectedResources = {};
        res.data.forEach(item => {
          const { listType, list } = item;
          if (listType && Array.isArray(list)) {
            this.resourceList[listType] = list.map(resource => ({
              ...resource,
              resourceType: listType,
            }));
            newSelectedResources[listType] = [];
          }
        });
        this.selectedResources = newSelectedResources;
      }
    },
    async fetchGlobalConfig() {
      const res = await getGeneralAgentGlobalConfig();
      if (res.data && Array.isArray(res.data)) {
        this.selectedTools = [];
        const newSelectedResources = {};
        Object.keys(this.resourceList).forEach(type => {
          newSelectedResources[type] = [];
        });

        res.data.forEach(item => {
          const { list, listType } = item;
          if (listType === 'tool') {
            this.selectedTools = (list || []).map(tool => ({
              toolId: tool.toolId,
              toolType: tool.toolType,
            }));
          } else if (listType && this.resourceList[listType]) {
            newSelectedResources[listType] = (list || []).map(resource => ({
              id: resource.id,
              type: resource.type,
            }));
          }
        });
        this.selectedResources = newSelectedResources;
      }
    },
    handleClose() {
      this.validationErrors.clear();
      this.$emit('update:visible', false);
    },

    validateTools() {
      if (!this.hasTools) {
        this.validationErrors.clear();
        return true;
      }

      const errors = new Set();

      this.toolList.forEach((category, index) => {
        const selectedInCategory = category.toolList.filter(tool =>
          this.isItemSelected(tool.toolId, 'tools'),
        ).length;
        const totalInCategory = category.toolList.length;

        if (
          (category.condition === 'required' &&
            selectedInCategory !== totalInCategory) ||
          (category.condition === 'optional' && selectedInCategory < 1)
        ) {
          errors.add(index);
        }
      });

      if (errors.size > 0) {
        this.activeTab = 'tools';
        this.validationErrors = errors;
        this.$message.warning(this.$t('generalAgent.config.validationWarning'));
        return false;
      }

      this.validationErrors.clear();
      return true;
    },

    async handleConfirm() {
      if (!this.validateTools()) {
        return;
      }

      const allSelectedTools = [];
      const toolsWithoutApiKey = [];

      this.toolList.forEach(category => {
        category.toolList.forEach(tool => {
          if (this.isItemSelected(tool.toolId, 'tools')) {
            if (this.needsApiKeyReminder(tool)) {
              toolsWithoutApiKey.push(tool.toolName);
            } else {
              allSelectedTools.push({
                toolId: tool.toolId,
                toolType: tool.toolType,
              });
            }
          }
        });
      });

      if (toolsWithoutApiKey.length > 0) {
        this.$message.warning(
          this.$t('generalAgent.config.missingApiKey', {
            names: toolsWithoutApiKey.join('、'),
          }),
        );
        return;
      }

      const selectedResourcesMap = {};
      Object.keys(this.resourceList).forEach(type => {
        const selectedList = [];
        this.resourceList[type].forEach(resource => {
          if (this.isItemSelected(resource.id, type)) {
            selectedList.push({
              id: resource.id,
              type: resource.type,
            });
          }
        });
        selectedResourcesMap[type] = selectedList;
      });

      const submitData = {};

      if (allSelectedTools.length > 0) {
        submitData.tool = allSelectedTools.map(tool => ({
          toolId: tool.toolId,
          toolType: tool.toolType,
        }));
      }

      Object.keys(selectedResourcesMap).forEach(type => {
        if (selectedResourcesMap[type].length > 0) {
          submitData[type] = selectedResourcesMap[type].map(resource => ({
            id: resource.id,
            type: resource.type,
          }));
        }
      });

      const res = await updateGeneralAgentGlobalConfig(submitData);

      if (res.code === 0) {
        this.$message.success(this.$t('generalAgent.config.saveSuccess'));
        this.handleClose();
      } else {
        this.$message.error(res.msg);
      }
    },

    isItemSelected(itemId, type) {
      const itemType = type || this.activeTab;
      if (itemType === 'tools') {
        return this.selectedTools.some(t => t.toolId === itemId);
      }
      if (this.selectedResources[itemType]) {
        return this.selectedResources[itemType].some(r => r.id === itemId);
      }
      return false;
    },

    handleToggleItem(item) {
      if (this.activeTab === 'tools') {
        this.handleToggleTool(item);
      } else {
        this.handleToggleResource(item, this.currentListConfig.type);
      }
    },

    handleToggleTool(tool) {
      if (this.needsApiKeyReminder(tool)) {
        this.currentTool = tool;
        this.apiKeyModalVisible = true;
        this.apiKeyValue = '';
        return;
      }

      const index = this.selectedTools.findIndex(t => t.toolId === tool.toolId);
      if (index > -1) {
        this.selectedTools.splice(index, 1);
      } else {
        this.selectedTools.push({
          toolId: tool.toolId,
          toolType: tool.toolType,
        });
      }
    },

    // 处理 API Key 提交
    async handleApiKeySubmit() {
      if (!this.currentTool) return;

      if (!this.apiKeyValue.trim()) {
        this.$message.warning(this.$t('generalAgent.config.apiKeyRequired'));
        return;
      }

      const toolId = this.currentTool.toolId;
      const toolType = this.currentTool.toolType;

      this.submitting = true;
      try {
        await changeApiKey({
          apiKey: this.apiKeyValue,
          toolSquareId: toolId,
        });

        this.updateToolApiKeyInList(toolId, this.apiKeyValue);

        this.$message.success(this.$t('generalAgent.config.apiKeySaveSuccess'));
        this.apiKeyModalVisible = false;
        this.currentTool = null;
        this.apiKeyValue = '';

        const index = this.selectedTools.findIndex(t => t.toolId === toolId);
        if (index === -1) {
          this.selectedTools.push({
            toolId: toolId,
            toolType: toolType,
          });
        }
      } catch (error) {
        console.error('保存 API Key 失败:', error);
        this.$message.error(
          error.msg || this.$t('generalAgent.config.apiKeySaveFailed'),
        );
      } finally {
        this.submitting = false;
      }
    },

    handleToggleResource(item, resourceType) {
      const itemId = item.id;

      // ontology 类型使用单选逻辑
      if (resourceType === 'ontology') {
        this.$set(this.selectedResources, resourceType, [
          {
            id: itemId,
            type: item.type,
          },
        ]);
        return;
      }

      // 其他类型使用多选逻辑
      const selectedList = this.selectedResources[resourceType];
      const index = selectedList.findIndex(r => r.id === itemId);

      if (index > -1) {
        const newList = [...selectedList];
        newList.splice(index, 1);
        this.$set(this.selectedResources, resourceType, newList);
      } else {
        const newList = [
          ...selectedList,
          {
            id: itemId,
            type: item.type,
          },
        ];
        this.$set(this.selectedResources, resourceType, newList);
      }
    },

    getConditionLabel(condition) {
      const labels = {
        none: this.$t('generalAgent.config.conditionLabels.none'),
        optional: this.$t('generalAgent.config.conditionLabels.optional'),
        required: this.$t('generalAgent.config.conditionLabels.required'),
      };
      return labels[condition] || condition;
    },

    getConditionType(condition) {
      const types = {
        none: 'info',
        optional: 'warning',
        required: 'danger',
      };
      return types[condition] || 'info';
    },

    // 更新工具列表中的 API Key
    updateToolApiKeyInList(toolId, apiKey) {
      this.toolList.forEach(category => {
        category.toolList.forEach(tool => {
          if (tool.toolId === toolId) {
            tool.apiKey = apiKey;
          }
        });
      });
    },

    // 处理 API Key 弹窗关闭
    handleApiKeyModalClose() {
      this.apiKeyModalVisible = false;
      this.currentTool = null;
      this.apiKeyValue = '';
    },

    // 检查工具是否需要 API Key 提醒
    needsApiKeyReminder(tool) {
      return tool.needApiKeyInput && (!tool.apiKey || tool.apiKey === '');
    },
  },
};
</script>

<style lang="scss">
.config-dialog {
  min-width: 1000px;
  .el-dialog__header {
    padding: 16px 20px;
    border-bottom: 1px solid #e5e5e5;
    margin: 0;

    .dialog-title {
      h3 {
        margin: 0;
        font-size: 16px;
        font-weight: 500;
        color: #1a1a1a;
      }
    }
  }

  .el-dialog__body {
    padding: 0;
    height: 50vh;
    display: flex;
    flex-direction: column;
  }

  .dialog-body {
    padding: 16px 20px 0 20px;
    height: 100%;
    display: flex;
    flex-direction: column;

    .top-area {
      background: #fff;
      margin-bottom: 16px;
      flex-shrink: 0;
      display: flex;
      align-items: center;
      justify-content: space-between;

      .tab-buttons {
        display: flex;
        gap: 12px;

        .tab-btn {
          padding: 8px 20px;
          border-radius: 8px;
          font-size: 14px;
          color: #666;
          background: #fff;
          border: 1px solid #e4e7ed;
          cursor: pointer;
          transition: all 0.2s;
          user-select: none;

          &:hover {
            border-color: #409eff;
            color: #409eff;
          }

          &.active {
            background: #409eff;
            border-color: #409eff;
            color: #fff;
            font-weight: 500;
          }
        }
      }

      .search-box {
        margin-left: auto;

        .el-input {
          max-width: 300px;
        }
      }
    }
    .config-content {
      flex: 1;
      overflow-y: auto;
      min-height: 0;
    }
  }

  .dialog-footer {
    padding-top: 16px;
    text-align: right;
    border-top: 1px solid #e5e5e5;
  }

  .tool-categories {
    .tool-category {
      margin-bottom: 16px;

      &.validation-error {
        border: 1px solid #f56c6c;
        background-color: #fef0f0;
        border-radius: 4px;
        padding: 8px;
      }

      .category-header {
        display: flex;
        align-items: center;
        justify-content: space-between;
        margin-bottom: 10px;
        padding-bottom: 8px;
        border-bottom: 1px solid #f0f0f0;

        .category-name {
          font-size: 13px;
          font-weight: 500;
          color: #1a1a1a;
        }

        .error-tip {
          font-size: 12px;
          color: #f56c6c;
          font-weight: 500;
          margin-left: 8px;
        }
      }
    }
  }

  .resource-list {
    display: flex;
    flex-direction: column;
    gap: 6px;

    .empty-tip {
      text-align: center;
      padding: 20px;
      color: #999;
      font-size: 13px;
    }
  }

  .item-item {
    display: flex;
    align-items: center;
    padding: 10px 12px;
    border-radius: 10px;
    cursor: pointer;
    transition: all 0.2s;
    border: 1px solid transparent;

    &:hover {
      background: #f5f7fa;
      border-color: #e4e7ed;
    }

    &.selected {
      background: rgba(16, 163, 127, 0.08);
      border-color: rgba(16, 163, 127, 0.2);
    }

    .item-avatar {
      width: 36px;
      height: 36px;
      border-radius: 8px;
      margin-right: 12px;
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
        font-size: 18px;
        color: #999;
      }
    }

    .item-info {
      flex: 1;
      min-width: 0;

      .item-name {
        font-size: 14px;
        font-weight: 500;
        color: #1a1a1a;
        margin-bottom: 2px;
      }

      .item-desc {
        font-size: 12px;
        color: #666;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }

      .api-key-tip {
        display: flex;
        align-items: center;
        gap: 4px;
        margin-top: 6px;
        padding: 4px 8px;
        font-size: 12px;
        color: #e6a23c;
        background-color: #fdf6ec;
        border-radius: 4px;

        i {
          font-size: 14px;
        }
      }
    }

    .el-checkbox {
      margin-left: 8px;
    }

    .el-radio {
      margin-left: 8px;
    }
  }
}

.api-key-dialog {
  .el-dialog__body {
    padding: 20px;
  }

  .api-key-input-container {
    padding: 10px 0;
  }

  .dialog-footer {
    text-align: right;
    padding: 16px 20px;
    border-top: 1px solid #e5e5e5;
  }
}
</style>
