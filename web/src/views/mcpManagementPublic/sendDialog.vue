<template>
  <div class="add-dialog">
    <el-drawer
      :visible.sync="dialogVisible"
      size="45%"
      :before-close="handleCancel"
    >
      <div slot="title" class="send-title">
        <img src="@/assets/imgs/detail_send_title_icon.png" alt="" />
        <span>{{ $t('tool.square.send.title') }}</span>
      </div>
      <div class="send-content">
        <el-form
          :model="ruleForm"
          :rules="rules"
          ref="ruleForm"
          label-width="130px"
        >
          <el-form-item :label="$t('tool.integrate.name')">
            <div>{{ detail.name }}</div>
          </el-form-item>
          <el-form-item :label="$t('tool.integrate.from')">
            <div>{{ detail.from }}</div>
          </el-form-item>
          <el-form-item
            :label="$t('tool.integrate.desc')"
            class="description-text"
          >
            <div>{{ detail.desc }}</div>
          </el-form-item>
          <el-form-item label="MCP ServerURL" prop="serverUrl">
            <el-input v-model="ruleForm.serverUrl"></el-input>
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
        <ul class="mcpList" v-if="mcpList.length > 0">
          <li v-for="(item, index) in mcpList" :key="index">{{ item.name }}</li>
        </ul>
        <span class="send-footer">
          <el-button
            type="primary"
            size="mini"
            :disabled="mcpList.length === 0"
            @click="submitForm('ruleForm')"
            :loading="publishLoading"
          >
            {{ $t('tool.integrate.publish') }}
          </el-button>
          <el-button @click="handleCancel" size="mini">
            {{ $t('common.button.cancel') }}
          </el-button>
        </span>
      </div>
    </el-drawer>
  </div>
</template>
<script>
import { getTools, setCreate } from '@/api/mcp.js';
import { isValidURL } from '@/utils/util';

export default {
  props: ['dialogVisible', 'detail'],
  data() {
    const validateUrl = (rule, value, callback) => {
      if (!isValidURL(value)) {
        callback(new Error(this.$t('tool.integrate.serverUrlErr')));
      } else {
        callback();
      }
    };
    return {
      mcpList: [],
      ruleForm: {
        serverUrl: '',
      },
      rules: {
        serverUrl: [
          {
            required: true,
            message: this.$t('tool.integrate.serverUrlMsg'),
            trigger: 'blur',
          },
          { validator: validateUrl, trigger: 'blur' },
        ],
      },
      toolsLoading: false,
      publishLoading: false,
    };
  },
  methods: {
    handleCancel() {
      this.clearForm();
      this.$refs['ruleForm'].clearValidate();
      this.$emit('handleClose', false);
    },
    submitForm(formName) {
      this.$refs[formName].validate(valid => {
        if (valid) {
          const params = {
            name: this.detail.name,
            from: this.detail.from,
            sseUrl: this.ruleForm.serverUrl,
            desc: this.detail.desc,
            mcpSquareId: this.detail.mcpSquareId,
          };
          this.publishLoading = true;
          setCreate(params)
            .then(res => {
              if (res.code === 0) {
                this.$message.success(this.$t('common.info.publish'));
                this.publishLoading = false;
                this.handleCancel();
                // 更新发送按钮状态
                this.$emit('getIsCanSendStatus');
              }
            })
            .finally(() => (this.publishLoading = false));
        }
      });
    },
    clearForm() {
      this.ruleForm.serverUrl = '';
      this.mcpList = [];
    },
    handleTools() {
      this.toolsLoading = true;
      this.$refs['ruleForm'].validate(valid => {
        if (valid) {
          getTools({
            serverUrl: this.ruleForm.serverUrl,
          })
            .then(res => {
              this.mcpList = res.data.tools || [];
            })
            .finally(() => (this.toolsLoading = false));
        }
      });
    },
  },
  computed: {
    isGetMCP() {
      return !isValidURL(this.ruleForm.serverUrl);
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
    padding-bottom: 30px;
    li {
      padding: 5px 10px;
      border-radius: 5px;
      margin-bottom: 10px;
      background: $color_opacity;
      color: $color;
    }
  }
  .description-text .el-form-item__content {
    line-height: 24px !important;
    padding: 10px 0;
  }
  .send-title {
    display: flex;
    align-items: center;
    img {
      width: 25px;
      margin-right: 8px;
    }
    span {
      font-size: 16px;
      font-weight: bold;
      color: $color_title;
    }
  }
  ::v-deep .el-drawer__close-btn {
    margin-top: -14px;
  }
  .send-content {
    padding: 0 20px 50px;
    overflow-y: scroll;
  }
  .send-footer {
    padding: 20px;
    position: absolute;
    right: 0;
    bottom: 0;
    width: 100%;
    background-color: #fff;
    text-align: right;
  }
}
</style>
