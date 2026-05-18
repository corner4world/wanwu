<template>
  <el-dialog
    :title="$t('tempSquare.skills.form.convertTitle')"
    :visible.sync="visible"
    width="500px"
    @close="handleClose"
  >
    <el-form ref="form" :model="form" :rules="rules" label-width="80px">
      <el-form-item :label="$t('tempSquare.skills.form.author')" prop="author">
        <el-input
          v-model="form.author"
          :placeholder="$t('tempSquare.skills.form.authorPlaceholder')"
        />
      </el-form-item>
      <el-form-item :label="$t('tempSquare.skills.form.model')" prop="modelId">
        <ModelSelect
          v-model="form.modelId"
          :options="modelList"
          :placeholder="$t('tempSquare.skills.form.modelPlaceholder')"
          :loading="modelLoading"
          :filterable="true"
        />
      </el-form-item>
    </el-form>
    <div slot="footer" class="dialog-footer">
      <el-button @click="visible = false">
        {{ $t('common.confirm.cancel') }}
      </el-button>
      <el-button type="primary" :loading="submitLoading" @click="handleSubmit">
        {{ $t('common.confirm.confirm') }}
      </el-button>
    </div>
  </el-dialog>
</template>

<script>
import ModelSelect from '@/components/modelSelect.vue';
import { selectModelList } from '@/api/modelAccess';
import { createConvertSkillConversation } from '@/api/generalAgent';

export default {
  name: 'ConvertSkillDialog',
  components: {
    ModelSelect,
  },

  data() {
    return {
      visible: false,
      submitLoading: false,
      modelLoading: false,
      modelList: [],
      form: {
        id: '',
        type: '',
        author: '',
        modelId: '',
      },
      rules: {
        author: [
          {
            required: true,
            message: this.$t('tempSquare.skills.form.authorRequired'),
            trigger: 'blur',
          },
        ],
        modelId: [
          {
            required: true,
            message: this.$t('tempSquare.skills.form.modelRequired'),
            trigger: 'change',
          },
        ],
      },
    };
  },
  methods: {
    async open({ id, type } = {}) {
      if (!id || !type) {
        this.$message.warning(this.$t('tempSquare.skills.form.missingTarget'));
        return;
      }
      // 校验允许的资源类型
      const allowedTypes = ['mcp', 'tool', 'agent', 'workflow', 'rag'];
      if (!allowedTypes.includes(type)) {
        this.$message.error(this.$t('generalAgent.skill.convertTypeError'));
        return;
      }
      this.form.id = id;
      this.form.type = type;
      this.visible = true;

      const userInfo = this.$store.getters['user/userInfo'];
      this.form.author = userInfo && userInfo.userName ? userInfo.userName : '';

      this.form.modelId = '';
      if (this.$refs.form) {
        this.$refs.form.clearValidate();
      }

      await this.fetchModelList();
    },
    handleClose() {
      this.visible = false;
      this.submitLoading = false;
      if (this.$refs.form) {
        this.$refs.form.resetFields();
      }
    },
    async fetchModelList() {
      this.modelLoading = true;
      try {
        const res = await selectModelList();
        if (res.code === 0 && res.data && res.data.list) {
          this.modelList = res.data.list.map(model => ({
            modelId: model.modelId,
            displayName: model.displayName,
            model: model.model,
            provider: model.provider,
            modelType: model.modelType,
            config: model.config,
            avatar: model.avatar,
            tags: model.tags || [],
          }));
          if (this.modelList.length > 0 && !this.form.modelId) {
            this.form.modelId = this.modelList[0].modelId;
          }
        }
      } catch (e) {
        console.error(e);
      } finally {
        this.modelLoading = false;
      }
    },
    async handleSubmit() {
      try {
        await this.$refs.form.validate();
      } catch (e) {
        return;
      }

      const selectedModelConfig = this.modelList.find(
        m => m.modelId === this.form.modelId,
      );
      if (!selectedModelConfig) {
        this.$message.warning(this.$t('tempSquare.skills.form.invalidModel'));
        return;
      }

      const modelConfig = {
        modelId: selectedModelConfig.modelId,
        model: selectedModelConfig.model,
        provider: selectedModelConfig.provider,
        displayName: selectedModelConfig.displayName,
        modelType: selectedModelConfig.modelType,
        config: selectedModelConfig.config,
      };

      this.submitLoading = true;
      try {
        const res = await createConvertSkillConversation({
          id: this.form.id,
          type: this.form.type,
          author: this.form.author,
          modelConfig,
        });

        if (res.code === 0) {
          const { customSkillId, threadId, previewId } = res.data || {};
          this.$message.success(this.$t('tempSquare.skills.form.convertStart'));
          this.visible = false;
          this.$router.push({
            path: '/skill/workshop',
            query: {
              chatType: 'skill',
              chatMode: 'convert',
              customSkillId,
            },
          });
        } else {
          this.$message.error(
            res.msg ||
              res.message ||
              this.$t('tempSquare.skills.form.convertFailed'),
          );
        }
      } catch (e) {
        console.error(e);
      } finally {
        this.submitLoading = false;
      }
    },
  },
};
</script>

<style scoped>
.dialog-footer {
  text-align: right;
  margin-top: 20px;
}
</style>
