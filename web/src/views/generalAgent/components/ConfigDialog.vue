<template>
  <div>
    <el-dialog
      :visible.sync="dialogVisible"
      width="50%"
      min-width="400px"
      custom-class="config-dialog"
      :close-on-click-modal="false"
      @close="handleClose"
    >
      <div slot="title" class="dialog-title">
        <h3>{{ $t('generalAgent.config.title') }}</h3>
      </div>

      <div class="dialog-body">
        <div class="drawer-section">
          <!-- 切换按钮组 -->
          <div class="tab-buttons">
            <div
              v-if="hasTools"
              :class="['tab-btn', { active: activeTab === 'tools' }]"
              @click="activeTab = 'tools'"
            >
              {{ $t('generalAgent.config.tools') }}
            </div>
            <!-- MCP 选择 -->
            <div
              v-if="hasMcp"
              :class="['tab-btn', { active: activeTab === 'mcp' }]"
              @click="activeTab = 'mcp'"
            >
              {{ $t('generalAgent.config.mcp') }}
            </div>
            <!-- 工作流选择 -->
            <div
              v-if="hasWorkflows"
              :class="['tab-btn', { active: activeTab === 'workflows' }]"
              @click="activeTab = 'workflows'"
            >
              {{ $t('generalAgent.config.workflows') }}
            </div>
            <!-- 智能体选择 -->
            <div
              v-if="hasAgents"
              :class="['tab-btn', { active: activeTab === 'assistants' }]"
              @click="activeTab = 'assistants'"
            >
              {{ $t('generalAgent.config.agents') }}
            </div>
          </div>

          <div class="config-content">
            <!-- 工具列表 - 按分类展示 -->
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
                  <el-tag
                    size="mini"
                    :type="getConditionType(category.condition)"
                  >
                    {{ getConditionLabel(category.condition) }}
                  </el-tag>
                  <span
                    v-if="validationErrors.has(categoryIndex)"
                    class="error-tip"
                  >
                    {{ $t('generalAgent.config.validationError') }}
                  </span>
                </div>
                <div class="tool-list">
                  <div
                    v-for="tool in category.toolList"
                    :key="tool.toolId"
                    :class="[
                      'tool-item',
                      {
                        selected: isItemSelected(tool.toolId),
                      },
                    ]"
                    @click="handleToggleItem(tool)"
                  >
                    <div class="tool-avatar">
                      <img
                        v-if="tool.avatar?.path"
                        :src="avatarSrc(tool.avatar.path)"
                      />
                      <i v-else class="el-icon-setting"></i>
                    </div>
                    <div class="tool-info">
                      <div class="tool-name">{{ tool.toolName }}</div>
                      <div class="tool-desc">{{ tool.desc }}</div>
                      <!-- API Key 提示 -->
                      <div
                        v-if="
                          tool.needApiKeyInput &&
                          (!tool.apiKey || tool.apiKey === '')
                        "
                        class="api-key-tip"
                      >
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

            <!-- MCP/工作流/智能体列表 - 统一渲染 -->
            <div
              v-else-if="activeTab !== 'tools'"
              :class="currentListConfig.className"
            >
              <div
                v-for="item in currentListConfig.list"
                :key="item[currentListConfig.idField]"
                :class="[
                  'tool-item',
                  {
                    selected: isItemSelected(item[currentListConfig.idField]),
                  },
                ]"
                @click="handleToggleItem(item)"
              >
                <div class="tool-avatar">
                  <img
                    v-if="item.avatar?.path"
                    :src="avatarSrc(item.avatar.path)"
                  />
                  <i v-else :class="currentListConfig.iconClass"></i>
                </div>
                <div class="tool-info">
                  <div class="tool-name">
                    {{ item[currentListConfig.nameField] }}
                  </div>
                  <div class="tool-desc">
                    {{ item[currentListConfig.descField] }}
                  </div>
                </div>
                <el-checkbox
                  :value="isItemSelected(item[currentListConfig.idField])"
                  @click.native.stop
                  @change="handleToggleItem(item)"
                />
              </div>
            </div>
          </div>
        </div>
      </div>

      <div slot="footer" class="dialog-footer">
        <el-button @click="handleCancel">
          {{ $t('generalAgent.config.cancel') }}
        </el-button>
        <el-button type="primary" @click="handleConfirm">
          {{ $t('generalAgent.config.confirm') }}
        </el-button>
      </div>
    </el-dialog>

    <!-- API Key 输入弹窗 -->
    <el-dialog
      :visible.sync="apiKeyModalVisible"
      width="500px"
      custom-class="api-key-dialog"
      :close-on-click-modal="false"
      title="请输入 API Key"
      @close="handleApiKeyModalClose"
    >
      <div class="api-key-input-container">
        <el-input
          v-model="apiKeyValue"
          placeholder="请输入 API Key"
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
  getGeneralAgentAssistantSelect,
  getGeneralAgentMcpSelect,
  getGeneralAgentWorkflowSelect,
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
  },
  data() {
    return {
      dialogVisible: this.visible,
      activeTab: 'tools', // 当前激活的tab: tools | mcp | workflows | assistants
      toolList: [],
      mcpList: [],
      workflowList: [],
      assistantList: [],
      selectedTools: [],
      selectedMcps: [],
      selectedWorkflows: [],
      selectedAssistants: [],
      validationErrors: new Set(),
      // API Key 弹窗相关状态
      apiKeyModalVisible: false,
      currentTool: null,
      apiKeyValue: '',
      submitting: false,
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
    // 判断是否有 MCP 数据
    hasMcp() {
      return this.mcpList.length > 0;
    },
    // 判断是否有工作流数据
    hasWorkflows() {
      return this.workflowList.length > 0;
    },
    // 判断是否有智能体数据
    hasAgents() {
      return this.assistantList.length > 0;
    },
    // 当前列表配置（根据 activeTab 动态返回）
    currentListConfig() {
      const configs = {
        mcp: {
          list: this.mcpList,
          idField: 'mcpId',
          nameField: 'name',
          descField: 'description',
          iconClass: 'el-icon-connection',
          className: 'mcp-list',
        },
        workflows: {
          list: this.workflowList,
          idField: 'appId',
          nameField: 'name',
          descField: 'desc',
          iconClass: 'el-icon-share',
          className: 'workflow-list',
        },
        assistants: {
          list: this.assistantList,
          idField: 'appId',
          nameField: 'name',
          descField: 'desc',
          iconClass: 'el-icon-user',
          className: 'assistant-list',
        },
      };
      return configs[this.activeTab] || {};
    },
  },
  methods: {
    avatarSrc,
    async fetchAllData() {
      await Promise.allSettled([
        this.fetchToolList(),
        this.fetchMcpList(),
        this.fetchWorkflowList(),
        this.fetchAssistantList(),
        this.fetchGlobalConfig(),
      ]);
      // 加载完成后,自动选中第一个有数据的tab
      this.selectFirstAvailableTab();
    },
    // 自动选择第一个有数据的tab
    selectFirstAvailableTab() {
      const tabs = ['tools', 'mcp', 'workflows', 'assistants'];
      for (const tab of tabs) {
        if (this[`has${tab.charAt(0).toUpperCase() + tab.slice(1)}`]) {
          this.activeTab = tab;
          return;
        }
      }
    },
    async fetchToolList() {
      const res = await getGeneralAgentToolSelect();
      this.toolList = res?.data?.list || [];
    },
    async fetchMcpList() {
      const res = await getGeneralAgentMcpSelect();
      this.mcpList = res?.data?.list || [];
    },
    async fetchWorkflowList() {
      const res = await getGeneralAgentWorkflowSelect();
      this.workflowList = res?.data?.list || [];
    },
    async fetchAssistantList() {
      const res = await getGeneralAgentAssistantSelect();
      this.assistantList = res?.data?.list || [];
    },
    async fetchGlobalConfig() {
      const res = await getGeneralAgentGlobalConfig();
      if (res.data) {
        // 初始化已选中的工具
        this.selectedTools = (res.data.toolList || []).map(tool => ({
          toolId: tool.toolId,
          toolType: tool.toolType,
        }));
        // 初始化已选中的 MCP
        this.selectedMcps = (res.data.mcpList || []).map(mcp => ({
          mcpId: mcp.mcpId,
          mcpType: mcp.toolType,
        }));
        // 初始化已选中的工作流
        this.selectedWorkflows = (res.data.workflowList || []).map(
          workflow => ({
            workflowId: workflow.workflowId,
          }),
        );
        // 初始化已选中的智能体
        this.selectedAssistants = (res.data.assistantList || []).map(
          assistant => ({
            assistantId: assistant.assistantId,
          }),
        );
      }
    },
    handleClose() {
      this.$emit('update:visible', false);
    },

    handleCancel() {
      this.validationErrors.clear();
      this.handleClose();
    },

    async handleConfirm() {
      if (this.hasTools) {
        const errors = new Set();

        // 验证每个分类的选择条件
        this.toolList.forEach((category, index) => {
          const selectedInCategory = category.toolList.filter(tool => {
            return this.isItemSelected(tool.toolId, 'tools');
          }).length;
          const totalInCategory = category.toolList.length;

          // 验证 condition
          if (
            category.condition === 'required' &&
            selectedInCategory !== totalInCategory
          ) {
            errors.add(index);
          } else if (
            category.condition === 'optional' &&
            selectedInCategory < 1
          ) {
            errors.add(index);
          }
          // none 类型不做限制
        });

        if (errors.size > 0) {
          this.activeTab = 'tools';
          this.validationErrors = errors;
          this.$message.warning(
            this.$t('generalAgent.config.validationWarning'),
          );
          return;
        }

        // 验证通过,清除错误状态
        this.validationErrors.clear();
      }

      // 收集所有选中的工具（遍历所有分类）
      const allSelectedTools = [];
      const toolsWithoutApiKey = []; // 记录没有 API Key 的工具

      this.toolList.forEach(category => {
        category.toolList.forEach(tool => {
          if (this.isItemSelected(tool.toolId, 'tools')) {
            // 检查需要 API Key 的工具是否已配置
            if (tool.needApiKeyInput && (!tool.apiKey || tool.apiKey === '')) {
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

      // 如果有工具缺少 API Key，提醒用户
      if (toolsWithoutApiKey.length > 0) {
        this.$message.warning(
          this.$t('generalAgent.config.missingApiKey', {
            names: toolsWithoutApiKey.join('、'),
          }),
        );
        return;
      }

      // 收集所有选中的 MCP
      const allSelectedMcps = [];
      this.mcpList.forEach(mcp => {
        if (this.isItemSelected(mcp.mcpId, 'mcp')) {
          allSelectedMcps.push({
            mcpId: mcp.mcpId,
            mcpType: mcp.toolType,
          });
        }
      });

      // 收集所有选中的工作流
      const allSelectedWorkflows = [];
      this.workflowList.forEach(workflow => {
        if (this.isItemSelected(workflow.appId, 'workflows')) {
          allSelectedWorkflows.push({
            workflowId: workflow.appId,
          });
        }
      });

      // 收集所有选中的智能体
      const allSelectedAssistants = [];
      this.assistantList.forEach(assistant => {
        const assistantId = assistant.appId;
        if (this.isItemSelected(assistantId, 'assistants')) {
          allSelectedAssistants.push({
            assistantId: assistantId,
          });
        }
      });

      const res = await updateGeneralAgentGlobalConfig({
        toolList: allSelectedTools,
        mcpList: allSelectedMcps,
        workflowList: allSelectedWorkflows,
        assistantList: allSelectedAssistants,
      });

      if (res.code === 0) {
        this.$message.success(this.$t('generalAgent.config.saveSuccess'));
        // 触发确认事件,传递选中的工具和智能体列表
        this.$emit('confirm', {
          tools: allSelectedTools,
          mcps: allSelectedMcps,
          workflows: allSelectedWorkflows,
          assistants: allSelectedAssistants,
        });
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
      if (itemType === 'mcp') {
        return this.selectedMcps.some(m => m.mcpId === itemId);
      }
      if (itemType === 'workflows') {
        return this.selectedWorkflows.some(w => w.workflowId === itemId);
      }
      // 智能体的选中状态判断
      return this.selectedAssistants.some(a => a.assistantId === itemId);
    },

    handleToggleItem(item) {
      if (this.activeTab === 'tools') {
        this.handleToggleTool(item);
      } else if (this.activeTab === 'mcp') {
        this.handleToggleMcp(item);
      } else if (this.activeTab === 'workflows') {
        this.handleToggleWorkflow(item);
      } else {
        this.handleToggleAssistant(item);
      }
    },

    handleToggleTool(tool) {
      // 如果需要 API Key（apiKey 为空），则弹出输入框
      if (tool.needApiKeyInput && (!tool.apiKey || tool.apiKey === '')) {
        this.currentTool = tool;
        this.apiKeyModalVisible = true;
        this.apiKeyValue = '';
        return;
      }

      // 在选中状态中切换
      const index = this.selectedTools.findIndex(t => t.toolId === tool.toolId);
      if (index > -1) {
        // 已选中，取消选中
        this.selectedTools.splice(index, 1);
      } else {
        // 未选中，添加选中
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
        // 调用更新 API Key 的接口
        await changeApiKey({
          apiKey: this.apiKeyValue,
          toolSquareId: toolId,
        });

        // 更新工具列表中的 apiKey
        this.updateToolApiKeyInList(toolId, this.apiKeyValue);

        this.$message.success(this.$t('generalAgent.config.apiKeySaveSuccess'));
        this.apiKeyModalVisible = false;
        this.currentTool = null;
        this.apiKeyValue = '';

        // API Key 设置成功后，自动选中该工具
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

    handleToggleAssistant(assistant) {
      // 智能体使用 appId 作为标识
      const assistantId = assistant.appId;
      // 在选中状态中切换
      const index = this.selectedAssistants.findIndex(
        a => a.assistantId === assistantId,
      );
      if (index > -1) {
        // 已选中，取消选中
        this.selectedAssistants.splice(index, 1);
      } else {
        // 未选中，添加选中
        this.selectedAssistants.push({
          assistantId: assistantId,
        });
      }
    },

    handleToggleMcp(mcp) {
      // 在选中状态中切换
      const index = this.selectedMcps.findIndex(m => m.mcpId === mcp.mcpId);
      if (index > -1) {
        // 已选中，取消选中
        this.selectedMcps.splice(index, 1);
      } else {
        // 未选中，添加选中
        this.selectedMcps.push({
          mcpId: mcp.mcpId,
          mcpType: mcp.toolType,
        });
      }
    },

    handleToggleWorkflow(workflow) {
      // 工作流使用 appId 作为标识
      const workflowId = workflow.appId;
      // 在选中状态中切换
      const index = this.selectedWorkflows.findIndex(
        w => w.workflowId === workflowId,
      );
      if (index > -1) {
        // 已选中，取消选中
        this.selectedWorkflows.splice(index, 1);
      } else {
        // 未选中，添加选中
        this.selectedWorkflows.push({
          workflowId: workflowId,
        });
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
  }

  .dialog-body {
    overflow-y: auto;
    padding: 16px 20px;
    max-height: calc(80vh - 60px);
  }

  .dialog-footer {
    text-align: right;
    padding: 16px 20px;
    border-top: 1px solid #e5e5e5;
  }

  .drawer-section {
    margin-bottom: 24px;

    .tab-buttons {
      display: flex;
      gap: 12px;
      margin-bottom: 16px;

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

    .section-header {
      display: flex;
      align-items: center;
      gap: 8px;
      margin-bottom: 16px;
      font-size: 14px;
      font-weight: 500;
      color: #1a1a1a;

      i {
        font-size: 16px;
        color: #10a37f;
      }
    }
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

  // 合并所有列表样式（工具、MCP、工作流、智能体）
  .tool-list,
  .mcp-list,
  .workflow-list,
  .assistant-list {
    display: flex;
    flex-direction: column;
    gap: 6px;
  }

  .tool-item {
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

    .tool-avatar {
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

    .tool-info {
      flex: 1;
      min-width: 0;

      .tool-name {
        font-size: 14px;
        font-weight: 500;
        color: #1a1a1a;
        margin-bottom: 2px;
      }

      .tool-desc {
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
  }
}

.tool-tooltip-popper {
  max-width: 360px !important;
  padding: 0 !important;
  border: 1px solid #e4e7ed !important;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.12) !important;
  border-radius: 8px !important;
}

.tool-detail-tooltip {
  padding: 12px 14px;

  .tooltip-title {
    font-size: 14px;
    font-weight: 600;
    color: #1a1a1a;
    margin-bottom: 8px;
    padding-bottom: 8px;
    border-bottom: 1px solid #f0f0f0;
  }

  .tooltip-desc {
    font-size: 13px;
    color: #666;
    line-height: 1.6;
    white-space: pre-wrap;
    max-height: 200px;
    overflow-y: auto;
  }
}

// API Key 弹窗样式
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
