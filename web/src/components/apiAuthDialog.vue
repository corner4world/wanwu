<template>
  <el-dialog
    :title="$t('tool.custom.auth.title')"
    :visible.sync="dialogVisible"
    width="600px"
    append-to-body
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <div class="action-form">
      <el-form
        :rules="apiAuthRules"
        ref="apiAuthForm"
        :inline="false"
        :model="form"
      >
        <el-form-item :label="$t('tool.custom.auth.authType')">
          <el-select v-model="form.authType">
            <el-option label="None" value="none" />
            <el-option
              :label="$t('tool.custom.auth.headerType')"
              value="api_key_header"
            />
            <el-option
              :label="$t('tool.custom.auth.queryType')"
              value="api_key_query"
            />
          </el-select>
        </el-form-item>
        <!--请求头-->
        <div v-if="form.authType === 'api_key_header'">
          <el-form-item
            :label="$t('tool.custom.auth.prefix')"
            prop="apiKeyHeaderPrefix"
          >
            <el-select v-model="form.apiKeyHeaderPrefix">
              <el-option label="Basic" value="basic" />
              <el-option label="Bearer" value="bearer" />
              <el-option label="Custom" value="custom" />
            </el-select>
          </el-form-item>
          <el-form-item prop="apiKeyHeader">
            <template #label>
              {{ $t('tool.custom.auth.header') }}
              <el-tooltip
                effect="dark"
                :content="$t('tool.custom.auth.headerHint')"
                placement="top-start"
              >
                <span class="el-icon-question tips" />
              </el-tooltip>
            </template>
            <el-input
              class="desc-input"
              v-model="form.apiKeyHeader"
              placeholder="Authorization"
              clearable
            />
          </el-form-item>
          <el-form-item
            :label="$t('tool.custom.auth.value')"
            prop="apiKeyValue"
          >
            <el-input
              class="desc-input"
              v-model="form.apiKeyValue"
              placeholder="API key"
              clearable
            />
          </el-form-item>
        </div>
        <!--查询参数-->
        <div v-if="form.authType === 'api_key_query'">
          <el-form-item prop="apiKeyQueryParam">
            <template #label>
              {{ $t('tool.custom.auth.query') }}
              <el-tooltip
                effect="dark"
                :content="$t('tool.custom.auth.queryHint')"
                placement="top-start"
              >
                <span class="el-icon-question tips" />
              </el-tooltip>
            </template>
            <el-input
              class="desc-input"
              v-model="form.apiKeyQueryParam"
              clearable
            />
          </el-form-item>
          <el-form-item
            :label="$t('tool.custom.auth.value')"
            prop="apiKeyValue"
          >
            <el-input
              class="desc-input"
              v-model="form.apiKeyValue"
              placeholder="API key"
              clearable
            />
          </el-form-item>
        </div>
      </el-form>
    </div>
    <span slot="footer" class="dialog-footer">
      <el-button @click="handleClose">
        {{ $t('common.button.cancel') }}
      </el-button>
      <el-button type="primary" @click="handleConfirm">
        {{ $t('common.button.confirm') }}
      </el-button>
    </span>
  </el-dialog>
</template>

<script>
export default {
  name: 'ApiAuthDialog',
  props: {
    visible: {
      type: Boolean,
      default: false,
    },
    apiAuth: {
      type: Object,
      default: () => ({
        authType: 'none',
        apiKeyValue: '',
        apiKeyHeader: '',
        apiKeyHeaderPrefix: 'basic',
        apiKeyQueryParam: '',
      }),
    },
  },
  data() {
    return {
      dialogVisible: false,
      form: {
        authType: 'none',
        apiKeyValue: '',
        apiKeyHeader: '',
        apiKeyHeaderPrefix: 'basic',
        apiKeyQueryParam: '',
      },
    };
  },
  computed: {
    apiAuthRules() {
      const isQuery = this.form.authType === 'api_key_query';
      const isHeader = this.form.authType === 'api_key_header';
      return {
        ...(isQuery && {
          apiKeyQueryParam: [
            {
              required: true,
              message: this.$t('common.input.placeholder'),
              trigger: 'blur',
            },
          ],
        }),
        ...(isHeader && {
          apiKeyHeaderPrefix: [
            {
              required: true,
              message: this.$t('common.input.placeholder'),
              trigger: 'blur',
            },
          ],
          apiKeyHeader: [
            {
              required: true,
              message: this.$t('common.input.placeholder'),
              trigger: 'blur',
            },
          ],
        }),
        ...((isQuery || isHeader) && {
          apiKeyValue: [
            {
              required: true,
              message: this.$t('common.input.placeholder'),
              trigger: 'blur',
            },
          ],
        }),
      };
    },
  },
  watch: {
    visible(val) {
      this.dialogVisible = val;
      if (val) {
        this.form = {
          authType: 'none',
          apiKeyValue: '',
          apiKeyHeader: '',
          apiKeyHeaderPrefix: 'basic',
          apiKeyQueryParam: '',
          ...this.apiAuth,
        };
      }
    },
    'form.authType': {
      handler() {
        this.$nextTick(() => {
          if (this.$refs.apiAuthForm) {
            this.$refs.apiAuthForm.clearValidate();
          }
        });
      },
      immediate: false,
    },
  },
  methods: {
    handleClose() {
      if (this.$refs.apiAuthForm) {
        this.$refs.apiAuthForm.clearValidate();
      }
      this.$emit('close');
    },
    handleConfirm() {
      this.$refs.apiAuthForm.validate(valid => {
        if (!valid) return;
        // 根据类型清理无用字段
        const result = { ...this.form };
        switch (result.authType) {
          case 'none':
            result.apiKeyValue = '';
            result.apiKeyHeader = '';
            result.apiKeyHeaderPrefix = '';
            result.apiKeyQueryParam = '';
            break;
          case 'api_key_query':
            result.apiKeyHeader = '';
            result.apiKeyHeaderPrefix = '';
            break;
          case 'api_key_header':
            result.apiKeyQueryParam = '';
            break;
        }
        this.$emit('confirm', result);
        this.handleClose();
      });
    },
  },
};
</script>
