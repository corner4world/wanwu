<template>
  <div>
    <el-dialog
      :title="titleMap[type]"
      :visible.sync="dialogVisible"
      width="750"
      append-to-body
      :close-on-click-modal="false"
    >
      <!-- 类型选择：仅创建模式显示 -->
      <div class="workflow-type-list" v-if="type === 'create'">
        <div
          v-for="typeItem in workflowTypeList"
          :key="typeItem.key"
          :class="[
            'workflow-type-item',
            form.appType === typeItem.key ? 'active' : '',
          ]"
          @click="changeAppType(typeItem.key)"
        >
          <div class="item-img">
            <svg-icon class="item-icon" :icon-class="typeItem.icon" />
          </div>
          <div class="item-content">
            <p class="item-text">{{ typeItem.text }}</p>
            <h3 class="item-desc">{{ typeItem.desc }}</h3>
          </div>
        </div>
      </div>

      <el-form ref="form" :model="form" label-width="120px" :rules="rules">
        <el-form-item :label="$t('list.pluginPic') + ':'" prop="avatar">
          <el-upload
            class="avatar-uploader"
            action=""
            name="files"
            :show-file-list="false"
            :http-request="handleUploadImage"
            accept=".png,.jpg,.jpeg"
          >
            <!--:on-error="handleUploadError"-->
            <img
              class="upload-img"
              :src="
                form.avatar && form.avatar.path
                  ? form.avatar.path
                  : defaultIcon || defaultLogo
              "
            />
            <p class="upload-hint">
              {{ $t('common.fileUpload.clickUploadImg') }}
            </p>
          </el-upload>
        </el-form-item>
        <el-form-item :label="$t('list.pluginName') + ':'" prop="name">
          <el-input
            :placeholder="$t('common.hint.text')"
            v-model="form.name"
            maxlength="50"
            show-word-limit
          ></el-input>
        </el-form-item>
        <el-form-item :label="$t('list.pluginDesc') + ':'" prop="desc">
          <el-input
            type="textarea"
            :placeholder="$t('list.descplaceholder')"
            v-model="form.desc"
            show-word-limit
            maxlength="200"
          ></el-input>
        </el-form-item>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="dialogVisible = false">
          {{ $t('list.cancel') }}
        </el-button>
        <el-button type="primary" @click="doPublish">
          {{ $t('list.confirm') }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>

<script>
import { createWorkFlow, uploadFile } from '@/api/workflow';
import { copyWorkflowTemplate } from '@/api/templateSquare';
import { avatarSrc } from '@/utils/util';
import { WORKFLOW, CHAT } from '@/utils/commonSet';

export default {
  props: {
    type: {
      type: String,
      default: 'create',
    },
  },
  data() {
    return {
      dialogVisible: false,
      defaultLogo: require('@/assets/imgs/bg-logo.png'),
      defaultIcon: '',
      form: {
        name: '',
        desc: '',
        avatar: {
          key: '',
          path: '',
        },
        // 创建时选择的类型：workflow 或 chatflow
        appType: WORKFLOW,
      },
      titleMap: {
        edit: this.$t('list.editplugin'),
        create: this.$t('list.createplugin'),
        clone: this.$t('list.copy_Demo'),
      },
      workflowID: '',
      templateId: '',
      // 工作流/对话流选项列表
      workflowTypeList: [
        {
          key: WORKFLOW,
          icon: 'workflow_icon',
          text: this.$t('uploadDialog.workflow'),
          desc: this.$t('uploadDialog.workflowDesc'),
        },
        {
          key: CHAT,
          icon: 'chatflow_icon',
          text: this.$t('uploadDialog.chat'),
          desc: this.$t('uploadDialog.chatDesc'),
        },
      ],
      rules: {
        name: [
          {
            required: true,
            message: this.$t('list.nameRules'),
            trigger: 'change',
          },
          {
            pattern: this.$config.commonTextReg,
            message: this.$t('common.hint.text'),
            trigger: 'change',
          },
          {
            min: 2,
            max: 50,
            message: this.$t('common.hint.textLimit'),
            trigger: 'blur',
          },
        ],
        desc: [
          {
            required: true,
            message: this.$t('list.pluginDescRules'),
            trigger: 'blur',
          },
          {
            max: 200,
            message: this.$t('list.pluginLimitRules'),
            trigger: 'blur',
          },
        ],
      },
    };
  },
  created() {
    const { defaultIcon = {} } = this.$store.state.user.commonInfo.data || {};
    this.defaultIcon = avatarSrc(defaultIcon.workflowIcon);
  },
  methods: {
    changeAppType(appType) {
      this.form.appType = appType;
      const { defaultIcon = {} } = this.$store.state.user.commonInfo.data || {};
      this.defaultIcon = avatarSrc(
        appType === WORKFLOW
          ? defaultIcon.workflowIcon
          : defaultIcon.chatflowIcon,
      );
    },
    getBase64(file) {
      return new Promise((resolve, reject) => {
        const fileReader = new FileReader();
        fileReader.onload = event => {
          const result = event.target ? event.target.result : '';
          if (!result || typeof result !== 'string') {
            reject('file read fail');
            return;
          }
          resolve(result.replace(/^.*?,/, ''));
        };
        fileReader.readAsDataURL(file);
      });
    },
    getFileExtension(name) {
      const index = name.lastIndexOf('.');
      return name.slice(index + 1).toLowerCase();
    },
    async handleUploadImage(data) {
      if (data.file) {
        const base64 = await this.getBase64(data.file).catch(() => '');

        if (!base64) {
          this.handleUploadError();
          return;
        }
        const res = await uploadFile({
          file_head: {
            file_type: this.getFileExtension(data.file.name),
            biz_type: 6,
          },
          data: base64,
        });
        const { upload_uri, upload_url } = res.data || {};
        this.form.avatar = { key: upload_uri || '', path: upload_url || '' };
      }
    },
    handleUploadError() {
      this.$message.error(this.$t('common.message.uploadError'));
    },
    openDialog(row) {
      this.clearForm();
      if (row) {
        const { templateId, desc, avatar } = row;
        this.templateId = templateId;
        this.form = { name: templateId, desc, avatar };
      }
      this.dialogVisible = true;
      this.$nextTick(() => {
        this.$refs['form'].clearValidate();
      });
    },
    clearForm() {
      this.form = {
        name: '',
        desc: '',
        avatar: {
          key: '',
          path: '',
        },
        appType: WORKFLOW,
      };
    },
    async doPublish() {
      let valid = false;
      await this.$refs.form.validate(vv => {
        if (vv) {
          valid = true;
        }
      });
      if (!valid) return;
      if (this.type === 'clone') {
        let res = await copyWorkflowTemplate({
          ...this.form,
          templateId: this.templateId,
        });
        if (res.code === 0) {
          this.$message.success(this.$t('list.copySuccess'));
          this.dialogVisible = false;
          this.$router.push({ path: '/appSpace/workflow' });
        }
        return;
      }
      const res = await createWorkFlow(this.form);
      if (res.code === 0) {
        this.$message.success(this.$t('list.createSuccess'));
        this.dialogVisible = false;
        const { workflow_id } = res.data || {};
        const querys = { id: workflow_id };
        this.$router.push({ path: '/workflow', query: querys });
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.avatar-uploader {
  position: relative;
  width: 98px;
  .upload-img {
    object-fit: cover;
    width: 100%;
    height: 98px;
    background: #eee;
    border-radius: 8px;
    border: 1px solid #dcdfe6;
    display: inline-block;
    vertical-align: middle;
  }
  .upload-hint {
    position: absolute;
    width: 100%;
    bottom: 0;
    background: $color_opacity;
    color: $color;
    font-size: 12px;
    line-height: 26px;
    z-index: 10;
    border-radius: 0 0 8px 8px;
  }
}

.workflow-type-list {
  display: flex;
  margin-bottom: 20px;
  gap: 15px;

  .workflow-type-item {
    display: flex;
    align-items: center;
    cursor: pointer;
    border: 1px solid #ddd;
    padding: 10px;
    border-radius: 6px;
    gap: 15px;
    width: 50%;

    &.active {
      border-color: $color;
    }

    .item-img {
      width: 45px;
      height: 45px;
      border: 1px solid #eeeded;
      border-radius: 8px;
      display: flex;
      justify-content: center;
      align-items: center;
      box-shadow: 0px 2px 4px -2px rgba(16, 24, 40, 0.06);

      .item-icon {
        color: $color;
        font-size: 22px;
      }
    }

    .item-content {
      width: calc(100% - 45px);
    }

    .item-text {
      font-size: 14px;
      font-weight: 600;
      line-height: 1.8;
    }

    .item-desc {
      line-height: 1.2;
      color: #b4b3b3;
      font-weight: unset;
    }
  }
}
</style>
