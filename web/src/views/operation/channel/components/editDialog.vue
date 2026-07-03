<template>
  <el-dialog
    :visible.sync="dialogVisible"
    :title="$t('channel.editDialog.title')"
    width="640px"
    append-to-body
    :close-on-click-modal="false"
    :before-close="handleClose"
    custom-class="channel-edit-dialog"
  >
    <el-form
      ref="formRef"
      :model="form"
      :rules="rules"
      label-position="top"
      size="small"
      style="margin-top: -20px"
    >
      <el-form-item :label="$t('channel.channelName')" prop="name">
        <el-input v-model="form.name" maxlength="50" />
      </el-form-item>

      <el-form-item :label="$t('channel.appType')" prop="appType">
        <el-select v-model="form.appType" style="width: 100%">
          <el-option
            v-for="item in appTypeOptions"
            :key="item.value"
            :label="item.label"
            :value="item.value"
          />
        </el-select>
      </el-form-item>

      <el-form-item :label="$t('channel.bindApp')" prop="appId">
        <el-select
          v-model="form.appId"
          :placeholder="$t('channel.bindAppPlaceholder')"
          filterable
          style="width: 100%"
        >
          <el-option
            v-for="item in appList"
            :key="item.appId"
            :label="item.name"
            :value="item.appId"
          />
        </el-select>
      </el-form-item>

      <el-form-item :label="$t('channel.bindApiKey')" prop="apiKeyId">
        <el-select
          v-model="form.apiKeyId"
          :placeholder="$t('common.select.placeholder')"
          filterable
          style="width: 100%"
        >
          <el-option
            v-for="item in apiKeyList"
            :key="item.keyId"
            :label="item.name"
            :value="item.keyId"
          />
        </el-select>
      </el-form-item>
      <!-- 渠道配置展示 -->
      <div v-if="justifyConfig()">
        <el-form-item
          v-for="item in Object.keys(form.config)"
          :label="item"
          :key="item"
          :prop="item"
        >
          <el-input v-model="form.config[item]" placeholder="" disabled />
        </el-form-item>
      </div>
    </el-form>

    <template #footer>
      <el-button type="primary" @click="handleSave" :loading="saveLoading">
        {{ $t('channel.saveConfig') }}
      </el-button>
      <el-button @click="handleClose">
        {{ $t('common.button.cancel') }}
      </el-button>
    </template>
  </el-dialog>
</template>

<script>
import { AGENT } from '@/utils/commonSet';
import { editChannel, getApiSelect, getAppSelect } from '@/api/channel';

export default {
  name: 'ChannelEditDialog',
  data() {
    return {
      dialogVisible: false,
      saveLoading: false,
      form: {
        name: '',
        appType: AGENT,
        appId: '',
        apiKeyId: '',
        config: {},
      },
      rules: {
        name: [
          {
            pattern: this.$config.commonTextReg,
            message: this.$t('common.hint.text'),
            trigger: 'blur',
          },
          {
            required: true,
            message: this.$t('common.input.placeholder'),
            trigger: 'blur',
          },
          {
            min: 2,
            max: 50,
            message: this.$t('common.hint.textLimit'),
            trigger: 'blur',
          },
        ],
        appId: [
          {
            required: true,
            message: this.$t('common.select.placeholder'),
            trigger: 'change',
          },
        ],
        appType: [
          {
            required: true,
            message: this.$t('common.select.placeholder'),
            trigger: 'change',
          },
        ],
        apiKeyId: [
          {
            required: true,
            message: this.$t('common.select.placeholder'),
            trigger: 'change',
          },
        ],
      },
      appTypeOptions: [{ value: AGENT, label: this.$t('channel.agent') }],
      appList: [],
      apiKeyList: [],
      row: {},
    };
  },
  watch: {
    'form.appType': {
      handler() {
        this.fetchAppList();
      },
      immediate: false,
    },
  },
  methods: {
    open(row) {
      this.row = row;
      this.form = {
        name: row.name || '',
        appType: row.appType || AGENT,
        appId: row.appId || '',
        apiKeyId: row.apiKeyId || '',
        config: row.config || {},
      };
      this.fetchAppList();
      this.fetchApiKeyList();
      this.dialogVisible = true;
      this.$nextTick(() => {
        this.$refs.formRef && this.$refs.formRef.clearValidate();
      });
    },
    handleClose() {
      this.dialogVisible = false;
      this.$refs.formRef && this.$refs.formRef.resetFields();
    },
    justifyConfig() {
      return Object.keys(this.form.config || {}).length > 0;
    },
    handleSave() {
      this.$refs.formRef.validate(valid => {
        if (!valid) return;
        this.saveLoading = true;
        editChannel(this.row.id, this.form)
          .then(() => {
            this.saveLoading = false;
            this.$message.success(this.$t('common.message.success'));
            this.handleClose();
            this.$emit('success');
          })
          .catch(() => {
            this.saveLoading = false;
          });
      });
    },
    async fetchAppList() {
      const res = await getAppSelect(this.form.appType);
      this.appList = res.data?.list || [];
    },
    async fetchApiKeyList() {
      const res = await getApiSelect();
      this.apiKeyList = res.data?.list || [];
    },
  },
};
</script>

<style lang="scss" scoped>
.channel-edit-dialog {
  border-radius: 8px;
  .el-dialog__body {
    padding: 16px 24px 0;
  }
  .el-dialog__footer {
    padding: 12px 24px 20px;
    text-align: right;
  }
}
</style>
