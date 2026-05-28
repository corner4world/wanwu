<template>
  <div class="add-dialog">
    <el-dialog
      :title="title"
      :visible.sync="dialogVisible"
      width="50%"
      :show-close="false"
      :close-on-click-modal="false"
    >
      <div>
        <el-form
          :model="ruleForm"
          :rules="rules"
          ref="ruleForm"
          label-width="130px"
          class="demo-ruleForm"
        >
          <el-form-item :label="$t('tool.integrate.avatar')" prop="avatar">
            <upload-avatar
              v-model="ruleForm.avatar"
              :default-avatar="defaultAvatar"
            />
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.name')" prop="name">
            <el-input
              v-model="ruleForm.name"
              :placeholder="$t('common.hint.text')"
              show-word-limit
              maxlength="50"
            ></el-input>
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.from')" prop="from">
            <el-input v-model="ruleForm.from"></el-input>
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.desc')" prop="desc">
            <el-input
              type="textarea"
              rows="5"
              show-word-limit
              maxlength="200"
              v-model="ruleForm.desc"
            ></el-input>
          </el-form-item>
          <el-form-item label="MCP Url" prop="transport">
            <el-radio-group
              v-model="ruleForm.transport"
              @change="handleTransportChange"
            >
              <el-radio label="sse">SSE</el-radio>
              <el-radio label="streamable">Streamable HTTP</el-radio>
            </el-radio-group>
          </el-form-item>
          <el-form-item
            v-if="ruleForm.transport === 'sse'"
            label="sse Url"
            prop="sseUrl"
          >
            <el-input
              v-model="ruleForm.sseUrl"
              :placeholder="$t('tool.integrate.sseUrlMsg')"
            ></el-input>
          </el-form-item>
          <el-form-item
            v-if="ruleForm.transport === 'streamable'"
            label="Streamable URL"
            prop="streamableUrl"
          >
            <el-input
              v-model="ruleForm.streamableUrl"
              :placeholder="$t('tool.integrate.streamableUrlMsg')"
            ></el-input>
          </el-form-item>
          <el-form-item
            :label="$t('tool.custom.apiAuth')"
            prop="apiAuth"
            required
          >
            <div class="api-auth-trigger" @click="preAuthorize">
              <div class="api-auth-display">
                {{ authTypeMap[ruleForm.apiAuth.authType] }}
              </div>
              <img
                class="auth-icon"
                :src="require('@/assets/imgs/auth.png')"
                alt=""
              />
            </div>
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.customParams')">
            <div class="custom-params">
              <div class="custom-params-header">
                <span>{{ $t('tool.integrate.paramName') }}</span>
                <span>{{ $t('tool.integrate.paramValue') }}</span>
                <span style="width: 30px; flex: none"></span>
              </div>
              <div
                v-for="(item, index) in ruleForm.customParams"
                :key="index"
                class="custom-param-row"
              >
                <el-input
                  v-model="item.name"
                  :placeholder="$t('tool.integrate.example') + 'Authorization'"
                  size="mini"
                />
                <el-input
                  v-model="item.value"
                  :placeholder="$t('tool.integrate.example') + 'Bearer token'"
                  size="mini"
                />
                <i class="el-icon-delete" @click="removeCustomParam(index)"></i>
              </div>
              <el-button
                v-if="ruleForm.customParams.length < 20"
                type="text"
                size="mini"
                class="add-param-btn"
                @click="addCustomParam"
              >
                + {{ $t('tool.integrate.addParams') }}
              </el-button>
            </div>
          </el-form-item>
          <el-form-item label="" style="text-align: right; margin-top: -10px">
            <el-button
              type="primary"
              size="mini"
              @click="handleTools"
              :disabled="isGetMCP"
              :loading="toolsLoading"
            >
              {{ $t('tool.integrate.action') }}
            </el-button>
          </el-form-item>
        </el-form>
        <ApiAuthDialog
          :visible.sync="dialogAuthVisible"
          :api-auth="ruleForm.apiAuth"
          @close="beforeApiAuthClose"
          @confirm="handleApiAuthConfirm"
        />
        <el-divider v-if="mcpList.length > 0"></el-divider>
        <ul class="mcpList" v-if="mcpList.length > 0">
          <li v-for="(item, index) in mcpList" :key="index">{{ item.name }}</li>
        </ul>
      </div>
      <span slot="footer" class="dialog-footer">
        <el-button @click="handleCancel" size="mini">
          {{ $t('common.button.cancel') }}
        </el-button>
        <el-button
          type="primary"
          size="mini"
          :disabled="mcpList.length === 0"
          @click="submitForm"
          :loading="publishLoading"
        >
          {{ $t('tool.integrate.publish') }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { getTools, setCreate, setUpdate } from '@/api/mcp.js';
import { isValidURL } from '@/utils/util';
import uploadAvatar from '@/components/uploadAvatar.vue';
import ApiAuthDialog from '@/components/apiAuthDialog.vue';

export default {
  components: { uploadAvatar, ApiAuthDialog },
  props: {
    title: {
      type: String,
      required: true,
    },
    dialogVisible: {
      type: Boolean,
      required: true,
    },
    initialData: {
      type: Object,
      default: () => ({
        name: '',
        from: '',
        sseUrl: '',
        streamableUrl: '',
        transport: 'sse',
        desc: '',
        mcpId: '',
        avatar: {
          key: '',
          path: '',
        },
        apiAuth: {
          authType: 'none',
          apiKeyValue: '',
          apiKeyHeader: '',
          apiKeyHeaderPrefix: 'basic',
          apiKeyQueryParam: '',
        },
        customParams: [],
      }),
    },
  },
  data() {
    const validateUrl = (rule, value, callback) => {
      if (!value) {
        callback(new Error(this.$t('tool.integrate.sseUrlMsg')));
      } else if (!isValidURL(value)) {
        callback(new Error(this.$t('tool.integrate.sseUrlErr')));
      } else {
        callback();
      }
    };
    return {
      mcpList: [],
      defaultAvatar: require('@/assets/imgs/mcp_active.svg'),
      ruleForm: {
        name: '',
        from: '',
        sseUrl: '',
        streamableUrl: '',
        transport: 'sse',
        desc: '',
        avatar: {
          key: '',
          path: '',
        },
        apiAuth: {
          authType: 'none',
          apiKeyValue: '',
          apiKeyHeader: '',
          apiKeyHeaderPrefix: 'basic',
          apiKeyQueryParam: '',
        },
        customParams: [],
      },
      rules: {
        name: [
          {
            pattern: this.$config.commonTextReg,
            message: this.$t('common.hint.text'),
            trigger: 'blur',
          },
          {
            min: 2,
            max: 50,
            message: this.$t('common.hint.textLimit'),
            trigger: 'blur',
          },
          {
            required: true,
            message: this.$t('common.input.placeholder'),
            trigger: 'blur',
          },
        ],
        from: [
          {
            required: true,
            message:
              this.$t('common.input.placeholder') +
              this.$t('tool.integrate.from'),
            trigger: 'blur',
          },
        ],
        sseUrl: [
          {
            required: true,
            message: this.$t('tool.integrate.sseUrlMsg'),
            trigger: 'blur',
          },
          { validator: validateUrl, trigger: 'blur' },
        ],
        streamableUrl: [
          {
            required: true,
            message: this.$t('tool.integrate.streamableUrlMsg'),
            trigger: 'blur',
          },
          { validator: validateUrl, trigger: 'blur' },
        ],
        desc: [
          {
            required: true,
            message:
              this.$t('common.input.placeholder') +
              this.$t('tool.integrate.desc'),
            trigger: 'blur',
          },
        ],
        apiAuth: [
          {
            validator: (rule, value, callback) => {
              if (
                this.ruleForm.apiAuth.authType === 'api_key_header' &&
                (!this.ruleForm.apiAuth.apiKeyValue ||
                  !this.ruleForm.apiAuth.apiKeyHeader)
              ) {
                callback(new Error(this.$t('tool.custom.apiAuthPlaceholder')));
              } else if (
                this.ruleForm.apiAuth.authType === 'api_key_query' &&
                (!this.ruleForm.apiAuth.apiKeyValue ||
                  !this.ruleForm.apiAuth.apiKeyQueryParam)
              ) {
                callback(new Error(this.$t('tool.custom.apiAuthPlaceholder')));
              } else {
                callback();
              }
            },
            trigger: 'blur',
          },
        ],
      },
      dialogAuthVisible: false,
      toolsLoading: false,
      publishLoading: false,
    };
  },
  watch: {
    // 监听初始数据变化，更新本地副本
    initialData: {
      handler(newVal) {
        const initialValue = {
          name: '',
          from: '',
          sseUrl: '',
          streamableUrl: '',
          transport: 'sse',
          desc: '',
          avatar: {
            key: '',
            path: '',
          },
          apiAuth: {
            authType: 'none',
            apiKeyValue: '',
            apiKeyHeader: '',
            apiKeyHeaderPrefix: 'basic',
            apiKeyQueryParam: '',
          },
          customParams: [],
          ...newVal,
        };
        if (!initialValue.apiAuth.authType) {
          initialValue.apiAuth.authType = 'none';
        }
        this.ruleForm = initialValue;
        // 如果没有 transport 字段，默认为 sse
        if (!this.ruleForm.transport) {
          this.ruleForm.transport = 'sse';
        }
      },
      immediate: true,
    },
    // 监听 sseUrl 变化
    'ruleForm.sseUrl': {
      handler(newVal, oldVal) {
        if (oldVal && newVal !== oldVal) {
          this.mcpList = [];
        }
      },
    },
    // 监听 streamableUrl 变化
    'ruleForm.streamableUrl': {
      handler(newVal, oldVal) {
        if (oldVal && newVal !== oldVal) {
          this.mcpList = [];
        }
      },
    },
  },
  methods: {
    preAuthorize() {
      this.ruleForm.apiAuth.apiKeyHeaderPrefix =
        this.ruleForm.apiAuth.apiKeyHeaderPrefix || 'basic';
      this.dialogAuthVisible = true;
    },
    beforeApiAuthClose() {
      this.dialogAuthVisible = false;
    },
    handleApiAuthConfirm(data) {
      this.ruleForm.apiAuth = data;
    },
    addCustomParam() {
      if (this.ruleForm.customParams.length >= 20) {
        this.$message.warning(this.$t('tool.integrate.addLimitTips'));
        return;
      }
      const names = this.ruleForm.customParams.map(p => p.name.trim());
      if (names.includes('')) {
        this.$message.warning(this.$t('tool.integrate.addEmptyTips'));
        return;
      }
      const uniqueNames = new Set(names);
      if (names.length !== uniqueNames.size) {
        this.$message.warning(this.$t('tool.integrate.addDuplicateTips'));
        return;
      }
      this.ruleForm.customParams.push({ name: '', value: '' });
    },
    removeCustomParam(index) {
      this.ruleForm.customParams.splice(index, 1);
    },
    handleTransportChange() {
      // 切换 transport 类型时清空工具列表
      this.mcpList = [];
    },
    handleCancel() {
      this.$emit('handleClose', false);
      this.$refs['ruleForm'].resetFields();
      this.ruleForm.customParams = [];
      this.mcpList = [];
    },
    checkCustomParams() {
      const customParams = this.ruleForm.customParams || [];
      const names = customParams.map(p => p.name.trim()).filter(n => n);
      const uniqueNames = new Set(names);
      if (names.length !== uniqueNames.size) {
        this.$message.warning(this.$t('tool.integrate.addDuplicateTips'));
        return false;
      }
      return true;
    },
    getHeaders() {
      const customParams = this.ruleForm.customParams || [];
      return customParams.reduce((acc, { name, value }) => {
        acc[name] = value;
        return acc;
      }, {});
    },
    submitForm() {
      this.$refs['ruleForm'].validate(valid => {
        if (valid) {
          if (!this.checkCustomParams()) {
            return;
          }
          this.publishLoading = true;
          const params = {
            ...this.ruleForm,
            headers: this.getHeaders(),
          };
          delete params.customParams;
          if (this.initialData.mcpId) {
            setUpdate({
              ...params,
              mcpId: this.initialData.mcpId,
            })
              .then(res => {
                if (res.code === 0) {
                  this.$message.success(this.$t('common.info.edit'));
                  this.$emit('handleFetch', false);
                  this.handleCancel();
                }
              })
              .finally(() => (this.publishLoading = false));
          } else
            setCreate(params)
              .then(res => {
                if (res.code === 0) {
                  this.$message.success(this.$t('common.info.publish'));
                  this.$emit('handleFetch', false);
                  this.handleCancel();
                }
              })
              .finally(() => (this.publishLoading = false));
        }
      });
    },
    handleTools() {
      if (!this.checkCustomParams()) {
        return;
      }
      this.$refs['ruleForm'].validate(valid => {
        if (valid) {
          this.toolsLoading = true;
          // 根据 transport 类型选择 URL
          const serverUrl =
            this.ruleForm.transport === 'streamable'
              ? this.ruleForm.streamableUrl
              : this.ruleForm.sseUrl;
          getTools({
            serverUrl: serverUrl,
            transport: this.ruleForm.transport,
            apiAuth: this.ruleForm.apiAuth,
            headers: this.getHeaders(),
          })
            .then(res => {
              if (res.code === 0) this.mcpList = res.data.tools;
            })
            .finally(() => (this.toolsLoading = false));
        }
      });
    },
  },
  computed: {
    authTypeMap() {
      return {
        none: 'None',
        api_key_header: this.$t('tool.custom.auth.headerType'),
        api_key_query: this.$t('tool.custom.auth.queryType'),
      };
    },
    isGetMCP() {
      const url =
        this.ruleForm.transport === 'streamable'
          ? this.ruleForm.streamableUrl
          : this.ruleForm.sseUrl;
      return !isValidURL(url);
    },
  },
};
</script>
<style lang="scss" scoped>
.add-dialog {
  .el-button.is-disabled {
    &:active {
      background: transparent !important;
      border-color: #ebeef5 !important;
    }
  }
  .mcpList {
    list-style: none;
    li {
      padding: 10px;
      border: 1px solid #ddd;
      border-radius: 5px;
      margin-bottom: 10px;
      background: #fff;
    }
  }
  .api-auth-trigger {
    display: flex;
    align-items: center;
    cursor: pointer;
    border: 1px solid #dcdfe6;
    border-radius: 4px;
    margin-top: 5px;
    .api-auth-display {
      flex: 1;
      height: 30px;
      line-height: 30px;
      padding: 0 15px;
      color: #606266;
    }
    .auth-icon {
      width: 30px;
      height: 30px;
      padding: 4px;
      border-left: 1px solid #dcdfe6;
      margin-left: -1px;
    }
  }
  .custom-params {
    .custom-params-header {
      display: flex;
      margin-bottom: 8px;
      span {
        flex: 1;
        font-size: 12px;
        color: #909399;
        &:last-child {
          width: 30px;
          flex: none;
        }
      }
    }
    .custom-param-row {
      display: flex;
      align-items: center;
      margin-bottom: 8px;
      .el-input {
        flex: 1;
        margin-right: 8px;
      }
      .el-icon-delete {
        color: #f56c6c;
        cursor: pointer;
        font-size: 16px;
      }
    }
    .add-param-btn {
      margin-top: 4px;
      width: 100%;
      color: $color !important;
      border: 1px solid $color !important;
    }
  }
}
</style>
