<template>
  <div class="section page-wrapper" v-loading="loading.itemStatus">
    <div class="title">
      <i
        class="el-icon-arrow-left"
        @click="$router.go(-1)"
        style="margin-right: 20px; font-size: 20px; cursor: pointer"
      ></i>
      {{ obj.name }}
    </div>
    <div class="container">
      <el-descriptions
        class="margin-top"
        title=""
        :column="3"
        :size="''"
        border
      >
        <el-descriptions-item
          :label="$t('knowledgeManage.communityReport.name')"
        >
          {{ $t('knowledgeManage.communityReport.communityReport') }}
        </el-descriptions-item>
        <el-descriptions-item
          :label="$t('knowledgeManage.communityReport.segmentTotalNum')"
        >
          {{ res.total }}
        </el-descriptions-item>
        <el-descriptions-item
          :label="$t('knowledgeManage.communityReport.uploadTime')"
        >
          {{ formatDate(res.createdAt) }}
        </el-descriptions-item>
        <el-descriptions-item
          :label="$t('knowledgeManage.communityReport.segmentType')"
        >
          {{ communityReportStatus[res.status] }}
        </el-descriptions-item>
        <el-descriptions-item
          :label="$t('knowledgeManage.communityReport.lastImportStatus')"
          v-if="res.status === STATUS_FINISHED"
        >
          {{ communityImportStatus[res.lastImportStatus] }}
        </el-descriptions-item>
      </el-descriptions>

      <div class="btnRow">
        <el-button
          type="primary"
          icon="el-icon-refresh"
          @click="refreshData"
          size="mini"
          :loading="loading.itemStatus"
        >
          {{ $t('common.gpuDialog.reload') }}
        </el-button>
        <el-button
          type="primary"
          @click="generateReport"
          size="mini"
          :loading="loading.stop"
          :disabled="!res.canGenerate || permissionType === POWER_TYPE_READ"
        >
          {{
            res.generateLabel === ''
              ? $t('knowledgeManage.communityReport.generate')
              : res.generateLabel
          }}
        </el-button>
        <el-button
          type="primary"
          @click="createReport"
          size="mini"
          :loading="loading.stop"
          :disabled="!res.canAddReport || permissionType === POWER_TYPE_READ"
        >
          {{ $t('knowledgeManage.communityReport.addCommunityReport') }}
        </el-button>
      </div>

      <div class="card">
        <el-row :gutter="20" v-if="res && res.list && res.list.length > 0">
          <el-col
            :span="6"
            v-for="(item, index) in res.list"
            :key="index"
            class="card-box"
          >
            <el-card class="box-card">
              <div slot="header" class="clearfix">
                <el-tooltip
                  :content="item.title"
                  placement="top"
                  :disabled="item.title.length <= 10"
                >
                  <span>
                    {{
                      item.title.length > 10
                        ? item.title.substring(0, 10) + '...'
                        : item.title
                    }}
                  </span>
                </el-tooltip>
                <div>
                  <el-dropdown
                    @command="handleCommand"
                    placement="bottom"
                    v-if="permissionType !== POWER_TYPE_READ"
                  >
                    <span class="el-dropdown-link">
                      <i class="el-icon-more more"></i>
                    </span>
                    <el-dropdown-menu slot="dropdown">
                      <el-dropdown-item
                        class="card-delete"
                        :command="{ type: 'delete', item }"
                      >
                        <i class="el-icon-delete card-opera-icon" />
                        {{ $t('common.button.delete') }}
                      </el-dropdown-item>
                    </el-dropdown-menu>
                  </el-dropdown>
                </div>
              </div>
              <div class="text item" @click="handleClick(item, index)">
                {{ item.content }}
              </div>
            </el-card>
          </el-col>
        </el-row>
        <el-empty v-else :description="$t('knowledgeManage.noData')"></el-empty>
      </div>

      <div class="list-common" style="text-align: right">
        <el-pagination
          background
          @size-change="handleSizeChange"
          @current-change="handleCurrentChange"
          :current-page.sync="page.pageNo"
          :page-sizes="page.pageSizeList"
          :page-size="page.pageSize"
          layout="total, prev, pager, next, jumper"
          :total="page.total"
        ></el-pagination>
      </div>
    </div>
    <createReport ref="createReport" @refreshData="refreshData"></createReport>
  </div>
</template>
<script>
import {
  getCommunityReportList,
  delCommunityReport,
  generateCommunityReport,
} from '@/api/knowledge';
import {
  COMMUNITY_REPORT_STATUS,
  COMMUNITY_IMPORT_STATUS,
} from '@/views/knowledge/config';
import { mapGetters } from 'vuex';
import commonMixin from '@/mixins/common';
import createReport from './create.vue';
import {
  STATUS_FINISHED,
  INITIAL,
  POWER_TYPE_READ,
  POWER_TYPE_EDIT,
  POWER_TYPE_ADMIN,
  POWER_TYPE_SYSTEM_ADMIN,
} from '@/views/knowledge/constants';

export default {
  name: 'KnowledgeCommunityReport',
  components: { createReport },
  mixins: [commonMixin],
  data() {
    return {
      obj: {},
      page: {
        pageNo: 1,
        pageSize: 8,
        pageSizeList: [10, 15, 20, 50],
        total: 0,
      },
      loading: {
        stop: false,
        itemStatus: false,
      },
      res: {
        contentList: [],
      },
      communityReportStatus: COMMUNITY_REPORT_STATUS,
      communityImportStatus: COMMUNITY_IMPORT_STATUS,
      STATUS_FINISHED,
      POWER_TYPE_READ,
      POWER_TYPE_EDIT,
      POWER_TYPE_ADMIN,
      POWER_TYPE_SYSTEM_ADMIN,
    };
  },
  computed: {
    ...mapGetters('app', ['permissionType']),
  },
  created() {
    this.obj = this.$route.query;
    this.getList();
  },
  methods: {
    formatDate(value) {
      if (value === null || value === undefined || value === '') {
        return '-';
      }
      let dateValue = value;
      if (
        typeof value === 'number' ||
        (typeof value === 'string' && /^\d+$/.test(value))
      ) {
        const timestamp = typeof value === 'string' ? parseInt(value) : value;
        if (timestamp.toString().length === 10) {
          dateValue = timestamp * 1000;
        } else {
          dateValue = timestamp;
        }
      }
      const dateFormatFilter =
        (this.$options.filters && this.$options.filters.dateFormat) || null;
      return dateFormatFilter ? dateFormatFilter(dateValue) : dateValue;
    },
    refreshData() {
      setTimeout(() => {
        this.getList();
      }, 500);
    },
    createReport() {
      this.$refs.createReport.showDialog(this.obj.knowledgeId, 'add');
    },
    generateReport() {
      generateCommunityReport({ knowledgeId: this.obj.knowledgeId }).then(
        res => {
          if (res.code === 0) {
            this.$message.success(
              this.$t('knowledgeManage.communityReport.generateSuccess'),
            );
            this.getList();
          }
        },
      );
    },
    handleCommand(value) {
      const { type, item } = value || {};
      switch (type) {
        case 'delete':
          this.delReport(item);
          break;
      }
    },
    delReport(item) {
      delCommunityReport({
        contentId: item.contentId,
        knowledgeId: this.obj.knowledgeId,
      }).then(res => {
        if (res.code === 0) {
          this.$message.success(
            this.$t('knowledgeManage.communityReport.deleteSuccess'),
          );
          this.getList();
        }
      });
    },
    getList() {
      this.loading.itemStatus = true;
      getCommunityReportList({
        knowledgeId: this.obj.knowledgeId,
        pageNo: this.page.pageNo,
        pageSize: this.page.pageSize,
      })
        .then(res => {
          this.loading.itemStatus = false;
          this.res = res.data;
          this.page.total = this.res.total;
          if (
            (!this.res.list || this.res.list.length === 0) &&
            this.page.pageNo > 1
          ) {
            this.page.pageNo = 1;
            this.getList();
          }
        })
        .catch(() => {
          this.loading.itemStatus = false;
        });
    },
    handleClick(item, index) {
      if (this.permissionType === 0) return;
      // 点击卡片事件，可根据需求添加功能
      this.$refs.createReport.showDialog(this.obj.knowledgeId, 'edit', item);
    },
    handleCurrentChange(val) {
      this.page.pageNo = val;
      this.getList();
    },
    handleSizeChange(val) {
      this.page.pageSize = val;
      this.getList();
    },
  },
};
</script>
<style lang="scss" scoped>
.section {
  width: 100%;
  height: 100%;
  padding: 20px 20px 30px 20px;
  margin: auto;
  overflow: auto;

  .el-divider--horizontal {
    margin: 30px 0;
  }

  .title {
    font-size: 18px;
    font-weight: bold;
    color: #333;
    padding: 10px 0;
  }

  .container {
    display: block;
    min-width: 980px;
    padding: 15px;
    height: calc(100% - 45px);
    /*background: #fff;
    box-shadow: 0 1px 6px rgba(0, 0, 0, 0.3);*/
    border-radius: 5px;
    overflow: auto;

    ::v-deep .el-descriptions :not(.is-bordered) .el-descriptions-item__cell {
      &:nth-child(even) {
        width: 25%;
      }

      padding: 10px;
    }

    .btnRow {
      padding: 10px 0;
      text-align: right;
    }

    .card {
      flex-wrap: wrap;

      ::v-deep .el-row {
        margin: 0 !important;
      }

      .text {
        font-size: 14px;
      }

      .item {
        height: 120px;
        margin-bottom: 18px;
        display: -webkit-box;
        -webkit-line-clamp: 6;
        -webkit-box-orient: vertical;
        overflow: hidden;
        text-overflow: ellipsis;
      }

      .clearfix {
        display: flex;
        justify-content: space-between;
        align-items: center;
      }

      .card-box {
        margin-bottom: 10px;

        .box-card {
          &:hover {
            cursor: pointer;
            transform: scale(1.03);
          }

          .more {
            margin-left: 5px;
            cursor: pointer;
            transform: rotate(90deg);
            font-size: 16px;
            color: #8c8c8f;
          }
        }

        .segment-type {
          margin: 0 5px;
          color: #999;
          font-size: 12px;
        }

        .segment-length {
          color: #999;
          font-size: 12px;
        }

        .segment-child {
          color: #999;
          font-size: 12px;
          padding-left: 5px;
        }
      }

      ::v-deep .el-card__header {
        padding: 8px 20px;
      }
    }
  }
}
</style>
