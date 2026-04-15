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
              :placeholder="$t('common.hint.modelName')"
            ></el-input>
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.from')" prop="from">
            <el-input v-model="ruleForm.from"></el-input>
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.desc')" prop="desc">
            <el-input
              type="textarea"
              rows="5"
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
              placeholder="Streamable HTTP URL为空"
            ></el-input>
          </el-form-item>
          <el-form-item label=" " style="text-align: right">
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

export default {
  components: { uploadAvatar },
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
      },
      rules: {
        name: [
          {
            pattern: /^(?!_)[a-zA-Z0-9-_.\u4e00-\u9fa5]+$/,
            message: this.$t('common.hint.modelName'),
            trigger: 'blur',
          },
          {
            min: 2,
            max: 50,
            message: this.$t('common.hint.modelNameLimit'),
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
            message: 'Streamable HTTP URL为空',
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
      },
      toolsLoading: false,
      publishLoading: false,
    };
  },
  watch: {
    // 监听初始数据变化，更新本地副本
    initialData: {
      handler(newVal) {
        this.ruleForm = { ...newVal };
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
    handleTransportChange() {
      // 切换 transport 类型时清空工具列表
      this.mcpList = [];
    },
    handleCancel() {
      this.$emit('handleClose', false);
      this.$refs['ruleForm'].resetFields();
      this.mcpList = [];
    },
    submitForm() {
      this.$refs['ruleForm'].validate(valid => {
        if (valid) {
          this.publishLoading = true;
          if (this.initialData.mcpId) {
            setUpdate({
              ...this.ruleForm,
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
            setCreate(this.ruleForm)
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
      this.toolsLoading = true;
      // 根据 transport 类型选择 URL
      const serverUrl =
        this.ruleForm.transport === 'streamable'
          ? this.ruleForm.streamableUrl
          : this.ruleForm.sseUrl;
      getTools({
        serverUrl: serverUrl,
        transport: this.ruleForm.transport,
      })
        .then(res => {
          if (res.code === 0) this.mcpList = res.data.tools;
        })
        .finally(() => (this.toolsLoading = false));
    },
  },
  computed: {
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
}
</style>
