<template>
  <div class="config-panel-overlay" @click.self="$emit('close')">
    <div class="config-panel">
      <div class="panel-header">
        <h3>对话配置</h3>
        <i class="el-icon-close" @click="$emit('close')"></i>
      </div>
      <div class="panel-body">
        <!-- 模型配置 -->
        <div class="config-section">
          <div class="section-title">
            <i class="el-icon-cpu"></i>
            模型配置
          </div>
          <div class="section-content">
            <el-form label-width="80px" size="small">
              <el-form-item label="选择模型">
                <el-select
                  v-model="config.modelConfig.modelId"
                  placeholder="请选择模型"
                  @change="handleModelChange"
                  style="width: 100%"
                >
                  <el-option
                    v-for="model in modelList"
                    :key="model.modelId"
                    :label="model.modelName"
                    :value="model.modelId"
                  />
                </el-select>
              </el-form-item>
            </el-form>
          </div>
        </div>

        <!-- 智能体选择 -->
        <div class="config-section">
          <div class="section-title">
            <i class="el-icon-user"></i>
            智能体选择
          </div>
          <div class="section-content">
            <div v-if="loadingAssistants" class="loading-text">
              <i class="el-icon-loading"></i>
              加载中...
            </div>
            <div v-else class="assistant-list">
              <div
                v-for="item in assistantList"
                :key="item.appId"
                :class="[
                  'assistant-item',
                  { selected: isAssistantSelected(item.appId) },
                ]"
                @click="toggleAssistant(item)"
              >
                <div class="assistant-avatar">
                  <img v-if="item.avatar?.path" :src="item.avatar.path" />
                  <i v-else class="el-icon-user"></i>
                </div>
                <div class="assistant-info">
                  <div class="assistant-name">{{ item.name }}</div>
                  <div class="assistant-desc">{{ item.description }}</div>
                </div>
                <el-checkbox :value="isAssistantSelected(item.appId)" />
              </div>
            </div>
          </div>
        </div>

        <!-- 工具选择 -->
        <div class="config-section">
          <div class="section-title">
            <i class="el-icon-setting"></i>
            工具选择
            <el-tag size="mini" type="info" style="margin-left: 8px">
              {{ toolConditionText }}
            </el-tag>
          </div>
          <div class="section-content">
            <div v-if="loadingTools" class="loading-text">
              <i class="el-icon-loading"></i>
              加载中...
            </div>
            <div v-else class="tool-categories">
              <div
                v-for="category in toolCategories"
                :key="category.category"
                class="tool-category"
              >
                <div class="category-header">
                  <span class="category-name">{{ category.category }}</span>
                  <el-tag
                    size="mini"
                    :type="getConditionType(category.condition)"
                  >
                    {{ getConditionLabel(category.condition) }}
                  </el-tag>
                </div>
                <div class="tool-list">
                  <div
                    v-for="tool in category.toolList"
                    :key="tool.toolId"
                    :class="[
                      'tool-item',
                      { selected: isToolSelected(tool.toolId) },
                    ]"
                    @click="toggleTool(tool)"
                  >
                    <div class="tool-avatar">
                      <img v-if="tool.avatar?.path" :src="tool.avatar.path" />
                      <i v-else class="el-icon-setting"></i>
                    </div>
                    <div class="tool-info">
                      <div class="tool-name">{{ tool.toolName }}</div>
                      <div class="tool-desc">{{ tool.desc }}</div>
                    </div>
                    <el-checkbox :value="isToolSelected(tool.toolId)" />
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- 配置检查 -->
        <div class="config-section" v-if="configCheck">
          <div class="section-title">
            <i class="el-icon-warning-outline"></i>
            配置检查
          </div>
          <div class="section-content">
            <div class="check-result">
              <div
                :class="['check-item', configCheck.modelMeet ? 'pass' : 'fail']"
              >
                <i
                  :class="
                    configCheck.modelMeet ? 'el-icon-success' : 'el-icon-error'
                  "
                ></i>
                模型配置
              </div>
              <div
                v-for="cat in configCheck.toolsMeet"
                :key="cat.category"
                :class="['check-item', cat.meet ? 'pass' : 'fail']"
              >
                <i :class="cat.meet ? 'el-icon-success' : 'el-icon-error'"></i>
                {{ cat.category }}
                <span v-if="!cat.meet" class="check-detail">
                  (需要 {{ cat.condition }})
                </span>
              </div>
            </div>
          </div>
        </div>
      </div>
      <div class="panel-footer">
        <el-button size="small" @click="$emit('close')">取消</el-button>
        <el-button
          type="primary"
          size="small"
          :loading="saving"
          @click="saveConfig"
        >
          保存配置
        </el-button>
      </div>
    </div>
  </div>
</template>

<script>
import {
  getGeneralAgentAssistantSelect,
  getGeneralAgentToolSelect,
  getGeneralAgentConfig,
  updateGeneralAgentConfig,
  checkGeneralAgentConfig,
  getLlmModelSelect,
} from '@/api/generalAgent';

export default {
  name: 'ConfigPanel',
  props: {
    threadId: {
      type: String,
      required: true,
    },
  },
  data() {
    return {
      config: {
        modelConfig: {
          modelId: '',
          model: '',
          provider: '',
          modelType: '',
          config: '',
        },
        toolList: [],
        assistantList: [],
      },
      modelList: [],
      assistantList: [],
      toolCategories: [],
      configCheck: null,
      loadingAssistants: false,
      loadingTools: false,
      saving: false,
    };
  },
  computed: {
    toolConditionText() {
      const conditions = this.toolCategories.map(c => c.condition);
      if (conditions.includes('required')) {
        return '必须选择部分工具';
      }
      return '可选工具';
    },
  },
  mounted() {
    this.loadConfig();
    this.loadAssistants();
    this.loadTools();
    this.loadModelList();
  },
  methods: {
    async loadModelList() {
      try {
        const res = await getLlmModelSelect();
        if (res.code === 0 && res.data?.list) {
          this.modelList = res.data.list.map(model => ({
            modelId: model.modelId || model.model,
            modelName: model.displayName || model.model,
            model: model.model,
            provider: model.provider,
            modelType: model.modelType,
            config: model.config,
          }));
        }
      } catch (error) {
        console.error('获取模型列表失败:', error);
      }
    },

    async loadConfig() {
      try {
        const res = await getGeneralAgentConfig({ threadId: this.threadId });
        if (res.code === 0 && res.data) {
          this.config = {
            modelConfig: res.data.modelConfig || {},
            toolList: res.data.toolList || [],
            assistantList: res.data.assistantList || [],
          };
        }
      } catch (error) {
        console.error('加载配置失败:', error);
      }
    },

    async loadAssistants() {
      this.loadingAssistants = true;
      try {
        const res = await getGeneralAgentAssistantSelect({});
        if (res.code === 0) {
          this.assistantList = res.data?.list || [];
        }
      } catch (error) {
        console.error('加载智能体列表失败:', error);
      } finally {
        this.loadingAssistants = false;
      }
    },

    async loadTools() {
      this.loadingTools = true;
      try {
        const res = await getGeneralAgentToolSelect();
        if (res.code === 0) {
          // 后端返回的是数组，不是 {list: []} 格式
          this.toolCategories = res.data || [];
        }
      } catch (error) {
        console.error('加载工具列表失败:', error);
      } finally {
        this.loadingTools = false;
      }
    },

    isToolSelected(toolId) {
      return this.config.toolList.some(t => t.toolId === toolId);
    },

    toggleTool(tool) {
      const index = this.config.toolList.findIndex(
        t => t.toolId === tool.toolId,
      );
      if (index > -1) {
        this.config.toolList.splice(index, 1);
      } else {
        this.config.toolList.push({
          toolId: tool.toolId,
          toolType: tool.toolType,
        });
      }
      this.checkConfig();
    },

    isAssistantSelected(assistantId) {
      return this.config.assistantList.some(a => a.assistantId === assistantId);
    },

    toggleAssistant(item) {
      const index = this.config.assistantList.findIndex(
        a => a.assistantId === item.appId,
      );
      if (index > -1) {
        this.config.assistantList.splice(index, 1);
      } else {
        this.config.assistantList.push({
          assistantId: item.appId,
          assistantType: item.type || 'default',
        });
      }
    },

    handleModelChange() {
      const model = this.modelList.find(
        m => m.modelId === this.config.modelConfig.modelId,
      );
      if (model) {
        this.config.modelConfig.model = model.model;
        this.config.modelConfig.provider = model.provider;
        this.config.modelConfig.modelType = model.modelType;
      }
      this.checkConfig();
    },

    async checkConfig() {
      try {
        const res = await checkGeneralAgentConfig({
          threadId: this.threadId,
          modelConfig: this.config.modelConfig,
          toolList: this.config.toolList,
          assistantList: this.config.assistantList,
        });
        if (res.code === 0) {
          this.configCheck = res.data;
        }
      } catch (error) {
        console.error('检查配置失败:', error);
      }
    },

    async saveConfig() {
      this.saving = true;
      try {
        const res = await updateGeneralAgentConfig({
          threadId: this.threadId,
          modelConfig: this.config.modelConfig,
          toolList: this.config.toolList,
          assistantList: this.config.assistantList,
        });
        if (res.code === 0) {
          this.$message.success('保存成功');
          this.$emit('close');
          this.$emit('config-changed');
        }
      } catch (error) {
        console.error('保存配置失败:', error);
        this.$message.error('保存失败');
      } finally {
        this.saving = false;
      }
    },

    getConditionLabel(condition) {
      const labels = {
        none: '可选',
        optional: '推荐',
        required: '必选',
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
  },
};
</script>

<style lang="scss" scoped>
.config-panel-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  justify-content: flex-end;
  z-index: 1000;
}

.config-panel {
  width: 400px;
  background: #fff;
  display: flex;
  flex-direction: column;
  box-shadow: -2px 0 8px rgba(0, 0, 0, 0.1);
}

.panel-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #e4e7ed;

  h3 {
    margin: 0;
    font-size: 16px;
    font-weight: 500;
  }

  .el-icon-close {
    cursor: pointer;
    font-size: 18px;
    color: #909399;

    &:hover {
      color: #409eff;
    }
  }
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px 20px;
}

.config-section {
  margin-bottom: 20px;

  .section-title {
    display: flex;
    align-items: center;
    font-size: 14px;
    font-weight: 500;
    color: #303133;
    margin-bottom: 12px;

    i {
      margin-right: 6px;
    }
  }

  .section-content {
    .loading-text {
      text-align: center;
      color: #909399;
      padding: 16px;
    }
  }
}

.assistant-list,
.tool-list {
  .assistant-item,
  .tool-item {
    display: flex;
    align-items: center;
    padding: 10px 12px;
    border: 1px solid #e4e7ed;
    border-radius: 6px;
    margin-bottom: 8px;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      border-color: #409eff;
    }

    &.selected {
      border-color: #409eff;
      background: #ecf5ff;
    }

    .assistant-avatar,
    .tool-avatar {
      width: 36px;
      height: 36px;
      border-radius: 6px;
      margin-right: 10px;
      display: flex;
      align-items: center;
      justify-content: center;
      background: #f5f7fa;
      overflow: hidden;

      img {
        width: 100%;
        height: 100%;
        object-fit: cover;
      }

      i {
        font-size: 18px;
        color: #909399;
      }
    }

    .assistant-info,
    .tool-info {
      flex: 1;
      min-width: 0;

      .assistant-name,
      .tool-name {
        font-size: 14px;
        color: #303133;
        margin-bottom: 2px;
      }

      .assistant-desc,
      .tool-desc {
        font-size: 12px;
        color: #909399;
        overflow: hidden;
        text-overflow: ellipsis;
        white-space: nowrap;
      }
    }
  }
}

.tool-categories {
  .tool-category {
    margin-bottom: 16px;

    .category-header {
      display: flex;
      align-items: center;
      margin-bottom: 8px;

      .category-name {
        font-size: 13px;
        color: #606266;
        margin-right: 8px;
      }
    }
  }
}

.check-result {
  .check-item {
    display: flex;
    align-items: center;
    padding: 8px 12px;
    border-radius: 4px;
    margin-bottom: 4px;
    font-size: 13px;

    i {
      margin-right: 8px;
    }

    &.pass {
      background: #f0f9eb;
      color: #67c23a;
    }

    &.fail {
      background: #fef0f0;
      color: #f56c6c;
    }

    .check-detail {
      margin-left: 8px;
      font-size: 12px;
      opacity: 0.8;
    }
  }
}

.panel-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  padding: 16px 20px;
  border-top: 1px solid #e4e7ed;
}
</style>
