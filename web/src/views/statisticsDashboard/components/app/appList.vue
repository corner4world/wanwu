<template>
  <div>
    <div class="table-wrap list-common wrap-fullheight">
      <div class="table-box">
        <el-button
          class="add-bt"
          size="mini"
          type="primary"
          @click="exportData"
        >
          <span>{{ $t('common.button.export') }}</span>
        </el-button>
        <el-table
          :data="tableData"
          :header-cell-style="{ background: '#F9F9F9', color: '#999999' }"
          v-loading="loading"
          style="width: 100%"
        >
          <el-table-column
            prop="appName"
            :label="$t('statisticsDashboard.appName')"
            align="left"
          >
            <template slot-scope="scope">
              {{ scope.row.appName || '--' }}
            </template>
          </el-table-column>
          <el-table-column
            prop="appType"
            :label="$t('statisticsDashboard.appType')"
            align="left"
          >
            <template slot-scope="scope">
              {{ appTypeObj[scope.row.appType] || '--' }}
            </template>
          </el-table-column>
          <el-table-column
            prop="orgName"
            :label="$t('statisticsDashboard.org')"
            align="left"
          >
            <template slot-scope="scope">
              {{ scope.row.orgName || '--' }}
            </template>
          </el-table-column>
          <el-table-column
            prop="userName"
            :label="$t('statisticsDashboard.userName')"
            align="left"
          >
            <template slot-scope="scope">
              {{ scope.row.userName || '--' }}
            </template>
          </el-table-column>
          <el-table-column
            prop="callCount"
            :label="
              $t('statisticsDashboard.appCallCount') +
              ` (${$t('statisticsDashboard.frequency')})`
            "
            align="left"
          >
            <template slot-scope="scope">
              {{ formatAmount(scope.row.callCount) }}
            </template>
          </el-table-column>
          <el-table-column
            prop="callFailure"
            :label="
              $t('statisticsDashboard.appCallFailure') +
              ` (${$t('statisticsDashboard.frequency')})`
            "
            align="left"
          >
            <template slot-scope="scope">
              {{ formatAmount(scope.row.callFailure) }}
            </template>
          </el-table-column>
          <el-table-column
            prop="failureRate"
            :label="$t('statisticsDashboard.failureRate')"
            align="left"
          />
          <el-table-column
            prop="avgStreamCosts"
            :label="$t('statisticsDashboard.avgFirstCosts') + ` (ms)`"
            align="left"
          >
            <template slot-scope="scope">
              {{ formatAmount(scope.row.avgStreamCosts) }}
            </template>
          </el-table-column>
          <el-table-column
            prop="avgNonStreamCosts"
            :label="$t('statisticsDashboard.avgCosts') + ` (ms)`"
            align="left"
          >
            <template slot-scope="scope">
              {{ formatAmount(scope.row.avgNonStreamCosts) }}
            </template>
          </el-table-column>
          <el-table-column
            prop="streamCount"
            :label="
              $t('statisticsDashboard.streamCount') +
              ` (${$t('statisticsDashboard.frequency')})`
            "
            align="left"
          >
            <template slot-scope="scope">
              {{ formatAmount(scope.row.streamCount) }}
            </template>
          </el-table-column>
          <el-table-column
            prop="nonStreamCount"
            :label="
              $t('statisticsDashboard.nonStreamCount') +
              ` (${$t('statisticsDashboard.frequency')})`
            "
            align="left"
          >
            <template slot-scope="scope">
              {{ formatAmount(scope.row.nonStreamCount) }}
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
  </div>
</template>

<script>
import Pagination from '@/components/pagination.vue';
import { formatAmount, resDownloadFile } from '@/utils/util.js';
import { fetchAppList, exportAppData } from '@/api/statisticsDashboard';
import { AppType } from '@/utils/commonSet';

export default {
  components: { Pagination },
  props: {
    params: {},
  },
  data() {
    return {
      listApi: fetchAppList,
      loading: false,
      tableData: [],
      appTypeObj: AppType,
    };
  },
  methods: {
    formatAmount,
    async getTableData(params) {
      if (this.$refs.pagination) {
        this.loading = true;
        try {
          this.tableData = await this.$refs.pagination.getTableData(params);
        } finally {
          this.loading = false;
        }
      }
    },
    refreshData(data) {
      this.tableData = data;
    },
    async exportData() {
      const response = await exportAppData(this.params);
      resDownloadFile(
        response,
        `${this.$t('statisticsDashboard.appData')}.xlsx`,
      );
    },
  },
};
</script>

<style lang="scss" scoped>
.table-wrap {
  padding: 0 12px;
  .add-bt {
    margin: 0 0 16px;
    float: right;
  }
}
</style>
