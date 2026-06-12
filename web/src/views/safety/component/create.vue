<template>
  <div>
    <el-dialog
      top="10vh"
      :title="title"
      :close-on-click-modal="false"
      :visible.sync="dialogVisible"
      width="40%"
      :before-close="handleClose"
    >
      <el-form
        :model="ruleForm"
        ref="ruleForm"
        label-width="140px"
        class="demo-ruleForm"
        :rules="rules"
        @submit.native.prevent
      >
        <el-form-item :label="$t('safety.create.tableName')" prop="tableName">
          <el-input
            v-model="ruleForm.tableName"
            :placeholder="$t('common.hint.text')"
            maxlength="50"
            show-word-limit
          />
        </el-form-item>
        <el-form-item :label="$t('safety.create.remark')" prop="remark">
          <el-input
            v-model="ruleForm.remark"
            type="textarea"
            :rows="4"
            maxlength="200"
            show-word-limit
            :placeholder="$t('safety.create.remarkPlaceholder')"
          />
        </el-form-item>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="handleClose()">
          {{ $t('common.confirm.cancel') }}
        </el-button>
        <el-button type="primary" @click="submitForm('ruleForm')">
          {{ $t('common.confirm.confirm') }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>
<script>
import { createSensitive, editSensitive } from '@/api/safety';

export default {
  props: {
    type: '',
  },
  data() {
    return {
      title: this.$t('safety.create.createTitle'),
      dialogVisible: false,
      ruleForm: {
        tableName: '',
        remark: '',
      },
      rules: {
        tableName: [
          {
            required: true,
            message: this.$t('safety.create.tableNameMsg'),
            trigger: 'blur',
          },
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
        ],
        remark: [
          {
            required: true,
            message: this.$t('safety.create.remarkMsg'),
            trigger: 'blur',
          },
        ],
      },
      tableId: '',
    };
  },
  methods: {
    handleClose() {
      this.dialogVisible = false;
      this.clearform();
    },
    clearform() {
      this.tableId = '';
      this.$refs.ruleForm.resetFields();
      this.$refs.ruleForm.clearValidate();
    },
    submitForm(formName) {
      this.$refs[formName].validate(valid => {
        if (valid) {
          if (this.tableId !== '') {
            this.editSensitive();
          } else {
            this.createSensitive();
          }
        } else {
          return false;
        }
      });
    },
    createSensitive() {
      createSensitive({ ...this.ruleForm, type: this.type }).then(res => {
        if (res.code === 0) {
          this.$message.success(this.$t('common.info.create'));
          this.$emit('reloadData');
          this.dialogVisible = false;
          this.$router.push({ path: `/safety/wordList/${res.data.tableId}` });
        }
      });
    },
    editSensitive() {
      const data = {
        ...this.ruleForm,
        tableId: this.tableId,
      };
      editSensitive(data)
        .then(res => {
          if (res.code === 0) {
            this.$message.success(this.$t('common.info.edit'));
            this.$emit('reloadData');
            this.clearform();
            this.dialogVisible = false;
          }
        })
        .catch(error => {
          this.$message.error(error);
        });
    },
    showDialog(row = null) {
      this.dialogVisible = true;
      if (row) {
        this.title = this.$t('safety.create.editTitle');
        this.tableId = row.tableId;
        this.ruleForm = {
          tableName: row.tableName,
          remark: row.remark,
        };
      } else {
        this.title = this.$t('safety.create.createTitle');
        this.ruleForm = {
          tableName: '',
          remark: '',
        };
      }
    },
  },
};
</script>
