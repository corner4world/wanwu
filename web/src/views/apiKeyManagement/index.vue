<template>
  <div class="page-wrapper api-key-management">
    <!--<div class="page-title">
      <img
        class="page-title-img"
        :src="require('@/assets/imgs/api_key_management.svg')"
        alt=""
      />
      <span class="page-title-name">{{ $t('menu.apiKey') }}</span>
    </div>-->
    <div class="table-wrap list-common wrap-fullheight">
      <div class="table-box">
        <el-button
          class="add-bt"
          size="mini"
          type="primary"
          @click="preUpdate()"
        >
          <span>{{ $t('common.button.create') }}</span>
        </el-button>
        <el-table
          :data="tableData"
          :header-cell-style="{ background: '#F9F9F9', color: '#999999' }"
          v-loading="loading"
          style="width: 100%"
        >
          <el-table-column
            prop="name"
            :label="$t('apiKeyManage.table.name')"
            align="left"
          />
          <el-table-column
            prop="desc"
            :label="$t('apiKeyManage.table.desc')"
            align="left"
          >
            <template slot-scope="scope">
              <span>{{ scope.row.desc || '--' }}</span>
            </template>
          </el-table-column>
          <el-table-column
            prop="key"
            :label="$t('apiKeyManage.table.apiKey')"
            align="left"
          >
            <template slot-scope="scope">
              <span>{{ scope.row.key.slice(0, 6) + '******' }}</span>
              <i
                class="el-icon-copy-document copy-icon"
                :title="$t('common.button.copy')"
                @click="
                  () => {
                    copy(scope.row.key) && copyCb();
                  }
                "
              ></i>
            </template>
          </el-table-column>
          <el-table-column
            prop="expiredAt"
            :label="$t('apiKeyManage.table.expiredAt')"
            align="left"
          >
            <template slot-scope="scope">
              <span>
                {{ scope.row.expiredAt || $t('apiKeyManage.table.permanent') }}
              </span>
            </template>
          </el-table-column>
          >
          <el-table-column
            prop="createdAt"
            :label="$t('apiKeyManage.table.createdAt')"
            align="left"
          />
          <el-table-column
            prop="creator"
            :label="$t('apiKeyManage.table.creator')"
            align="left"
          />
          <el-table-column
            align="left"
            :label="$t('apiKeyManage.table.status')"
          >
            <template slot-scope="scope">
              <div style="height: 26px">
                <el-switch
                  @change="
                    val => {
                      changeStatus(scope.row, val);
                    }
                  "
                  style="display: block; height: 22px"
                  v-model="scope.row.status"
                  :active-text="$t('common.switch.start')"
                  :inactive-text="$t('common.switch.stop')"
                />
              </div>
            </template>
          </el-table-column>
          <el-table-column
            align="center"
            :label="$t('common.table.operation')"
            width="100"
          >
            <template slot-scope="scope">
              <el-button
                class="operation"
                type="text"
                @click="preUpdate(scope.row)"
              >
                {{ $t('common.button.edit') }}
              </el-button>
              <el-button type="text" @click="preDel(scope.row)">
                {{ $t('common.button.delete') }}
              </el-button>
            </template>
          </el-table-column>
        </el-table>
      </div>
      <Pagination
        class="pagination"
        ref="pagination"
        :listApi="listApi"
        @refreshData="refreshData"
      />
    </div>
    <CreateApiKeyDialog ref="createApiKeyDialog" @reloadData="reloadData" />
  </div>
</template>

<script>
import Pagination from '@/components/pagination.vue';
import CreateApiKeyDialog from './components/createApiKeyDialog.vue';
import {
  fetchApiKeyList,
  changeApiKeyStatus,
  deleteApiKey,
} from '@/api/apiKeyManagement';
import { copy, copyCb } from '@/utils/util';
export default {
  name: 'ApiKeyManagement',
  components: { Pagination, CreateApiKeyDialog },
  data() {
    return {
      listApi: fetchApiKeyList,
      loading: false,
      tableData: [],
    };
  },
  mounted() {
    this.getTableData();
  },
  methods: {
    copy,
    copyCb,
    reloadData(params) {
      this.getTableData(params);
    },
    async getTableData(params) {
      this.loading = true;
      try {
        this.tableData = await this.$refs.pagination.getTableData(params);
      } finally {
        this.loading = false;
      }
    },
    refreshData(data) {
      this.tableData = data;
    },
    preUpdate(row) {
      this.$refs.createApiKeyDialog.openDialog(row);
    },
    preDel(row) {
      this.$confirm(
        this.$t('apiKeyManage.confirm.delete'),
        this.$t('common.confirm.title'),
        {
          confirmButtonText: this.$t('common.confirm.confirm'),
          cancelButtonText: this.$t('common.confirm.cancel'),
          type: 'warning',
        },
      ).then(async () => {
        let res = await deleteApiKey({ keyId: row.keyId });
        if (res.code === 0) {
          this.$message.success(this.$t('common.message.success'));
          await this.getTableData();
        }
      });
    },
    changeStatus(row, val) {
      this.$confirm(
        val
          ? this.$t('apiKeyManage.switch.startHint')
          : this.$t('apiKeyManage.switch.stopHint'),
        this.$t('common.confirm.title'),
        {
          confirmButtonText: this.$t('common.confirm.confirm'),
          cancelButtonText: this.$t('common.confirm.cancel'),
          type: 'warning',
        },
      )
        .then(async () => {
          const res = await changeApiKeyStatus({
            keyId: row.keyId,
            status: val,
          });
          if (res.code === 0) {
            this.$message.success(this.$t('common.message.success'));
            await this.getTableData();
          }
        })
        .catch(() => {
          this.getTableData();
        });
    },
  },
};
</script>

<style lang="scss" scoped>
.table-wrap {
  padding: 20px 18px;
}
.table-box {
  .table-header {
    font-size: 16px;
    font-weight: bold;
    color: #555;
  }
  .add-bt {
    margin: 0 0 20px;
    img {
      width: 16px;
      margin-right: 5px;
      display: inline-block;
      vertical-align: middle;
    }
    span {
      display: inline-block;
      vertical-align: middle;
    }
  }
  ::v-deep .el-switch__label * {
    font-size: 13px;
  }
}
::v-deep .operation.el-button--text.el-button {
  padding: 3px 10px 3px 0;
  border-right: 1px solid #eaeaea !important;
}
</style>
