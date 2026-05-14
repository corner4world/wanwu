<template>
  <div class="api-key-table-wrapper">
    <div v-if="localList.length" class="api-key-card-list">
      <div
        v-for="(item, index) in localList"
        :key="item._localKey"
        class="api-key-card"
      >
        <template v-if="item.isEditing">
          <div class="api-key-card__header">
            <div class="api-key-card__title">
              {{
                item.isNew
                  ? $t('tempSquare.skills.apiKeyConfig.button.add')
                  : item.name || '-'
              }}
            </div>
            <div class="api-key-card__actions">
              <el-button
                type="text"
                size="small"
                :class="['save-btn', { 'is-disabled': isRowInvalid(item) }]"
                :disabled="isRowInvalid(item)"
                @click="handleSave(item)"
              >
                {{ $t('common.button.save') }}
              </el-button>
              <el-button
                type="text"
                size="small"
                class="delete-btn"
                @click="handleDelete(index)"
              >
                {{ $t('common.button.delete') }}
              </el-button>
            </div>
          </div>

          <div class="api-key-card__form">
            <div class="field-grid">
              <div class="field-item">
                <div class="field-label">
                  {{ $t('tempSquare.skills.apiKeyConfig.table.name') }}
                </div>
                <el-input
                  v-model="item.name"
                  size="small"
                  :placeholder="
                    $t('tempSquare.skills.apiKeyConfig.placeholder.name')
                  "
                ></el-input>
              </div>
              <div class="field-item">
                <div class="field-label">
                  {{ $t('tempSquare.skills.apiKeyConfig.table.variableKey') }}
                </div>
                <el-input
                  v-model="item.variableKey"
                  size="small"
                  :placeholder="
                    $t('tempSquare.skills.apiKeyConfig.placeholder.variableKey')
                  "
                ></el-input>
              </div>
            </div>

            <div class="field-item">
              <div class="field-label">
                {{ $t('tempSquare.skills.apiKeyConfig.table.desc') }}
              </div>
              <el-input
                v-model="item.desc"
                size="small"
                :placeholder="
                  $t('tempSquare.skills.apiKeyConfig.placeholder.desc')
                "
              ></el-input>
            </div>

            <div class="field-item">
              <div class="field-label">
                {{ $t('tempSquare.skills.apiKeyConfig.table.variableValue') }}
              </div>
              <el-input
                v-model="item.variableValue"
                size="small"
                show-password
                :placeholder="
                  $t('tempSquare.skills.apiKeyConfig.placeholder.variableValue')
                "
              ></el-input>
            </div>
          </div>
        </template>

        <template v-else>
          <div class="api-key-card__header">
            <div class="api-key-card__identity">
              <div class="api-key-card__title">
                {{ item.name || '-' }}
              </div>
              <div class="api-key-card__key">
                {{ item.variableKey || '-' }}
              </div>
            </div>
            <div class="api-key-card__actions">
              <el-button
                type="text"
                size="small"
                :disabled="hasEditingRow"
                @click="handleEdit(item)"
              >
                {{ $t('common.button.edit') }}
              </el-button>
              <el-button
                type="text"
                size="small"
                class="delete-btn"
                :disabled="hasEditingRow"
                @click="handleDelete(index)"
              >
                {{ $t('common.button.delete') }}
              </el-button>
            </div>
          </div>

          <div class="api-key-card__meta">
            <div class="meta-item">
              <span class="meta-label">
                {{ $t('tempSquare.skills.apiKeyConfig.table.desc') }}
              </span>
              <span class="meta-value meta-value--multiline">
                {{ item.desc || '-' }}
              </span>
            </div>
            <div class="meta-item">
              <span class="meta-label">
                {{ $t('tempSquare.skills.apiKeyConfig.table.variableValue') }}
              </span>
              <span class="meta-value">
                {{ item.variableValue ? '********' : '-' }}
              </span>
            </div>
          </div>
        </template>
      </div>
    </div>

    <div v-else class="api-key-empty">
      {{ $t('tempSquare.skills.apiKeyConfig.table.emptyText') }}
    </div>

    <div class="add-row-container">
      <el-button
        type="primary"
        plain
        icon="el-icon-plus"
        size="small"
        :disabled="hasEditingRow"
        @click="handleAdd"
      >
        {{ $t('tempSquare.skills.apiKeyConfig.button.add') }}
      </el-button>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ApiKeyTable',
  props: {
    dataList: {
      type: Array,
      default: () => [],
    },
  },
  data() {
    return {
      localList: [],
      pendingDataList: null,
    };
  },
  computed: {
    hasEditingRow() {
      return this.localList.some(item => item.isEditing);
    },
  },
  watch: {
    dataList: {
      handler(val) {
        if (this.hasEditingRow) {
          this.pendingDataList = JSON.parse(JSON.stringify(val));
          return;
        }
        this.initLocalList(val);
      },
      immediate: true,
      deep: true,
    },
    hasEditingRow(val) {
      if (!val && this.pendingDataList) {
        const nextDataList = this.pendingDataList;
        this.pendingDataList = null;
        this.initLocalList(nextDataList);
      }
    },
  },
  methods: {
    isRowInvalid(row) {
      if (!row) return true;

      return ['name', 'desc', 'variableKey', 'variableValue'].some(field => {
        const value = row[field];
        return typeof value !== 'string' || !value.trim();
      });
    },
    initLocalList(val) {
      this.localList = JSON.parse(JSON.stringify(val)).map(item => ({
        ...item,
        _localKey:
          item._localKey ||
          `existing-${item.name || ''}-${item.variableKey || ''}`,
        isEditing: item.isEditing || false,
        isNew: false,
      }));
    },
    handleAdd() {
      if (this.hasEditingRow) return;

      this.localList.push({
        name: '',
        desc: '',
        variableKey: '',
        variableValue: '',
        _localKey: `new-${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        isEditing: true,
        isNew: true,
      });
    },
    handleEdit(row) {
      if (this.hasEditingRow) return;
      row.isEditing = true;
    },
    handleSave(row) {
      if (this.isRowInvalid(row)) return;

      const variable = {
        name: row.name,
        desc: row.desc,
        variableKey: row.variableKey,
        variableValue: row.variableValue,
      };

      if (row.id) {
        variable.id = row.id;
      }

      if (row.isNew) {
        this.$emit('create-variable', variable);
        row.isNew = false;
      } else {
        this.$emit('update-variable', variable);
      }
      row.isEditing = false;
    },
    handleDelete(index) {
      const row = this.localList[index];
      if (!row || (this.hasEditingRow && !row.isEditing)) return;

      if (row.isNew) {
        this.localList.splice(index, 1);
        return;
      }

      this.$emit('delete-variable', {
        id: row.id,
        name: row.name,
      });
    },
  },
};
</script>

<style lang="scss" scoped>
.api-key-table-wrapper {
  margin-top: 8px;
  width: 100%;
}

.api-key-card-list {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.api-key-card {
  padding: 14px 16px;
  border: 1px solid #e5eaf3;
  border-radius: 8px;
  background: #fff;
}

.api-key-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
}

.api-key-card__identity {
  min-width: 0;
  flex: 1;
}

.api-key-card__title {
  color: #1f2a37;
  font-size: 14px;
  font-weight: 600;
  line-height: 20px;
}

.api-key-card__key {
  margin-top: 4px;
  color: #6b7280;
  font-size: 12px;
  line-height: 18px;
  word-break: break-all;
}

.api-key-card__actions {
  display: inline-flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.api-key-card__meta {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.meta-item {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  font-size: 12px;
  line-height: 18px;
}

.meta-label {
  color: #909399;
  flex-shrink: 0;
}

.meta-value {
  color: #303133;
  min-width: 0;
  word-break: break-word;
}

.meta-value--multiline {
  white-space: pre-wrap;
}

.api-key-card__form {
  margin-top: 12px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.field-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.field-item {
  min-width: 0;
}

.field-label {
  margin-bottom: 6px;
  color: #606266;
  font-size: 12px;
  line-height: 18px;
}

.api-key-empty {
  padding: 20px 16px;
  border: 1px dashed #d7deea;
  border-radius: 8px;
  background: #fafbfd;
  color: #909399;
  font-size: 13px;
  line-height: 20px;
  text-align: center;
}

.save-btn {
  color: #10a37f;

  &:hover {
    color: #0d8a6a;
  }

  &.is-disabled {
    color: #c0c4cc;
  }
}

.delete-btn {
  color: #f56c6c;

  &:hover {
    color: #f78989;
  }
}

.add-row-container {
  margin-top: 16px;
  display: flex;
  justify-content: flex-start;
}

@media (max-width: 768px) {
  .api-key-card {
    padding: 12px;
  }

  .api-key-card__header {
    flex-direction: column;
    gap: 10px;
  }

  .api-key-card__actions {
    gap: 16px;
  }

  .field-grid {
    grid-template-columns: 1fr;
  }

  .meta-item {
    flex-direction: column;
    gap: 2px;
  }
}
</style>
