<template>
  <div>
    <div>
      <el-button
        type="primary"
        icon="el-icon-plus"
        size="mini"
        @click="showDialog(null)"
      >
        {{ $t('common.button.add') }}
      </el-button>
      <el-table
        :data="tableData"
        style="width: 100%; margin-top: 15px"
        :header-cell-style="{ background: '#F9F9F9', color: '#999999' }"
      >
        <el-table-column
          prop="name"
          :label="$t('app.appName')"
        ></el-table-column>
        <el-table-column
          prop="description"
          :label="$t('oauth.desc')"
        ></el-table-column>
        <el-table-column prop="suffix" :label="$t('app.accessUrl')">
          <template slot-scope="scope">
            <span>{{ scope.row.suffix }}</span>
            <span
              class="el-icon-copy-document copy"
              @click="handleCopy(scope.row) && copycb()"
            ></span>
          </template>
        </el-table-column>
        <el-table-column
          prop="expiredAt"
          :label="$t('apiKeyManage.table.expiredAt')"
          width="180"
        ></el-table-column>
        <el-table-column prop="status" :label="$t('list.status')" width="180">
          <template slot-scope="scope">
            <el-switch
              v-model="scope.row.status"
              @change="statusChange($event, scope.row)"
              active-color="var(--color)"
            ></el-switch>
          </template>
        </el-table-column>
        <el-table-column :label="$t('knowledgeManage.operate')" width="260">
          <template slot-scope="scope">
            <el-button size="mini" round @click="showDialog(scope.row)">
              {{ $t('common.button.edit') }}
            </el-button>
            <el-button size="mini" round @click="handleDel(scope.row)">
              {{ $t('common.button.delete') }}
            </el-button>
          </template>
        </el-table-column>
      </el-table>
    </div>
    <el-dialog
      :title="title"
      :visible.sync="dialogVisible"
      width="30%"
      :before-close="handleClose"
    >
      <el-form ref="form" :model="form" class="formUrl" label-width="100px">
        <el-form-item
          :label="$t('app.appName')"
          prop="name"
          :rules="[
            { required: true, message: $t('app.appNameMsg'), trigger: 'blur' },
          ]"
        >
          <el-input
            v-model="form.name"
            :placeholder="$t('app.appNameMsg')"
          ></el-input>
        </el-form-item>
        <el-form-item :label="$t('app.appDesc')" prop="description">
          <el-input
            v-model="form.description"
            :placeholder="$t('app.appDescMsg')"
            type="textarea"
            :rows="2"
          ></el-input>
        </el-form-item>
        <el-form-item label="过期时间" prop="expiredAt" :rules="expiredAtRules">
          <el-date-picker
            v-model="form.expiredAt"
            type="datetime"
            value-format="yyyy-MM-dd HH:mm:ss"
            :placeholder="$t('app.urlEffectiveTime')"
            :picker-options="expiredAtPickerOptions"
            @change="handleExpiredAtChange"
          ></el-date-picker>
        </el-form-item>
        <!--<el-form-item label="知识库出处详情">
          <el-switch
            v-model="value"
            active-color="var(&#45;&#45;color)"
          >
          </el-switch>
        </el-form-item>
        <el-form-item label="工作流详情">
          <el-switch
            v-model="value"
            active-color="var(&#45;&#45;color)"
          >
          </el-switch>
        </el-form-item>-->
        <div class="online-item">
          <el-form-item prop="copyright">
            <template #label>
              <span>{{ $t('app.copyRight') }}</span>
              <el-tooltip
                class="item"
                effect="dark"
                :content="$t('app.copyrightEnable')"
                placement="top-start"
              >
                <span class="el-icon-question tips"></span>
              </el-tooltip>
            </template>
            <el-input
              v-model="form.copyright"
              :placeholder="$t('app.copyrightMsg')"
            ></el-input>
          </el-form-item>
          <el-form-item prop="copyrightEnable">
            <el-switch
              :disabled="!form.copyright"
              v-model="form.copyrightEnable"
              active-color="var(--color)"
            ></el-switch>
          </el-form-item>
        </div>
        <div class="online-item">
          <el-form-item prop="privacyPolicy">
            <template #label>
              <span>{{ $t('app.privacyPolicy') }}</span>
              <el-tooltip
                class="item"
                effect="dark"
                :content="$t('app.privacyPolicyEnable')"
                placement="top-start"
              >
                <span class="el-icon-question tips"></span>
              </el-tooltip>
            </template>
            <el-input
              v-model="form.privacyPolicy"
              :placeholder="$t('app.privacyPolicyMsg')"
              @blur="urlBlur"
            ></el-input>
          </el-form-item>
          <el-form-item prop="privacyPolicyEnable">
            <el-switch
              :disabled="!form.privacyPolicy"
              v-model="form.privacyPolicyEnable"
              active-color="var(--color)"
            ></el-switch>
          </el-form-item>
        </div>
        <div class="online-item">
          <el-form-item prop="disclaimer">
            <template #label>
              <span>{{ $t('app.disclaimer') }}</span>
              <el-tooltip
                class="item"
                effect="dark"
                :content="$t('app.disclaimerEnable')"
                placement="top-start"
              >
                <span class="el-icon-question tips"></span>
              </el-tooltip>
            </template>
            <el-input
              v-model="form.disclaimer"
              type="textarea"
              :rows="2"
              :placeholder="$t('app.disclaimerMsg')"
            ></el-input>
          </el-form-item>
          <el-form-item prop="disclaimerEnable">
            <el-switch
              :disabled="!form.disclaimer"
              v-model="form.disclaimerEnable"
              active-color="var(--color)"
            ></el-switch>
          </el-form-item>
        </div>
      </el-form>
      <span slot="footer" class="dialog-footer">
        <el-button @click="handleClose">
          {{ $t('common.button.cancel') }}
        </el-button>
        <el-button type="primary" @click="submit('form')">
          {{ $t('common.button.confirm') }}
        </el-button>
      </span>
    </el-dialog>
  </div>
</template>
<script>
import {
  getOpenurl,
  delOpenurl,
  editOpenurl,
  createOpenurl,
  switchOpenurl,
} from '@/api/agent';
export default {
  props: ['appId', 'appType'],
  data() {
    return {
      form: {
        appId: '',
        appType: '',
        copyright: '',
        copyrightEnable: false,
        disclaimer: '',
        disclaimerEnable: false,
        expiredAt: '',
        name: '',
        description: '',
        privacyPolicy: '',
        privacyPolicyEnable: false,
      },
      title: this.$t('app.createUrl'),
      dialogVisible: false,
      tableData: [],
      urlId: '',
      expiredAtPickerOptions: {
        disabledDate(time) {
          const todayStart = new Date();
          todayStart.setHours(0, 0, 0, 0);
          return time.getTime() < todayStart.getTime();
        },
      },
      expiredAtRules: [
        {
          validator: this.validateExpiredAt,
          trigger: 'change',
        },
      ],
    };
  },
  created() {
    this.form.appId = this.appId;
    this.form.appType = this.appType;
    this.getList();
  },
  methods: {
    urlBlur() {
      const text = this.form.privacyPolicy;
      if (!this.isValidUrl(text)) {
        this.$message.warning(this.$t('app.linkCheck'));
        this.form.privacyPolicy = '';
      }
    },
    isValidUrl(string) {
      const pattern =
        /^https?:\/\/(?:[-\w.])+(?:\:[0-9]+)?(?:\/(?:[\w/_.])*(?:\?(?:[\w&=%.])*)?(?:\#(?:[\w.])*)?)?$/;
      return pattern.test(string.trim());
    },
    getExpiredAtTime(value) {
      if (!value) return NaN;
      if (value instanceof Date) return value.getTime();
      return new Date(String(value).replace(/-/g, '/')).getTime();
    },
    isExpiredAtBeforeNow(value) {
      const expiredAtTime = this.getExpiredAtTime(value);
      return !Number.isNaN(expiredAtTime) && expiredAtTime < Date.now();
    },
    validateExpiredAt(rule, value, callback) {
      if (this.isExpiredAtBeforeNow(value)) {
        callback(new Error(this.$t('app.expiredAtBeforeNow')));
        return;
      }
      callback();
    },
    handleExpiredAtChange(value) {
      if (!value || !this.isExpiredAtBeforeNow(value)) return;

      this.form.expiredAt = '';
      this.$message.warning(this.$t('app.expiredAtBeforeNow'));
      this.$nextTick(() => {
        if (this.$refs.form) {
          this.$refs.form.clearValidate('expiredAt');
        }
      });
    },
    handleCopy(row) {
      let text = row.suffix;
      var textareaEl = document.createElement('textarea');
      textareaEl.setAttribute('readonly', 'readonly');
      textareaEl.value = text;
      document.body.appendChild(textareaEl);
      textareaEl.select();
      var res = document.execCommand('copy');
      document.body.removeChild(textareaEl);
      return res;
    },
    copycb() {
      this.$message.success(this.$t('agent.copyTips'));
    },
    getList() {
      getOpenurl({ appId: this.appId, appType: this.appType })
        .then(res => {
          if (res.code === 0) {
            this.tableData = res.data || [];
          }
        })
        .catch(() => {});
    },
    statusChange(status, row) {
      switchOpenurl({ status, urlId: row.urlId })
        .then(res => {
          if (res.code === 0) {
            this.$message.success(this.$t('common.message.success'));
            this.getList();
          }
        })
        .catch(() => {});
    },
    showDialog(row = null) {
      this.dialogVisible = true;
      if (row === null) {
        this.title = this.$t('app.createUrl');
        this.urlId = '';
        this.$nextTick(() => {
          if (this.$refs.form) {
            this.$refs.form.resetFields();
            this.$refs.form.clearValidate();
            this.clear();
          }
        });
      } else {
        this.title = this.$t('app.editUrl');
        this.urlId = row.urlId;
        Object.keys(row).forEach(key => {
          if (this.form.hasOwnProperty(key)) {
            this.form[key] = row[key];
          }
        });
      }
    },
    clear() {
      this.form.copyright = '';
      this.form.copyrightEnable = false;
      this.form.disclaimer = '';
      this.form.disclaimerEnable = false;
      this.form.expiredAt = '';
      this.form.name = '';
      this.form.privacyPolicy = '';
      this.form.privacyPolicyEnable = false;
    },
    submit(formName) {
      this.$refs[formName].validate(valid => {
        if (valid) {
          if (this.urlId === '') {
            this.createUrl();
          } else {
            this.eidtUrl();
          }
        } else {
          return false;
        }
      });
    },
    createUrl() {
      createOpenurl(this.form)
        .then(res => {
          if (res.code === 0) {
            this.$message.success(this.$t('common.message.success'));
            this.dialogVisible = false;
            this.getList();
          }
        })
        .catch(() => {});
    },
    eidtUrl() {
      const data = {
        ...this.form,
        urlId: this.urlId,
      };
      editOpenurl(data)
        .then(res => {
          if (res.code === 0) {
            this.$message.success(this.$t('common.message.success'));
            this.dialogVisible = false;
            this.getList();
          }
        })
        .catch(() => {});
    },
    handleDel(row) {
      this.$confirm(
        this.$t('app.delUrlConfirm'),
        this.$t('knowledgeManage.tip'),
        {
          confirmButtonText: this.$t('common.confirm.confirm'),
          cancelButtonText: this.$t('common.confirm.cancel'),
          type: 'warning',
        },
      )
        .then(() => {
          delOpenurl({ urlId: row.urlId }).then(res => {
            if (res.code === 0) {
              this.$message.success(this.$t('list.delSuccess'));
              this.getList();
            }
          });
        })
        .catch(error => {
          this.getList();
        });
    },
    handleClose() {
      this.dialogVisible = false;
    },
  },
};
</script>
<style lang="scss" scoped>
.copy {
  cursor: pointer;
  margin-left: 5px;
  color: $color;
}
.formUrl {
  .el-date-editor {
    width: 100%;
  }
  .online-item {
    display: flex;
    justify-content: space-between;
    .tips {
      margin-left: 2px;
      color: #aaadcc;
      cursor: pointer;
    }
  }
  .online-item > :nth-child(1) {
    width: 80%;
  }
  .online-item > :nth-child(2) {
    width: 20%;
    display: flex;
    justify-content: flex-end;
    ::v-deep.el-form-item__content {
      margin-left: 0 !important;
    }
  }
}
</style>
