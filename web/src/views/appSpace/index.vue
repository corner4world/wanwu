<template>
  <div class="page-wrapper page-wrapper-pr-none">
    <!--<div class="page-title">
      <img
        class="page-title-img"
        :src="
          typeObj[type] ? typeObj[type].img : require('@/assets/imgs/task.png')
        "
        alt=""
      />
      <span class="page-title-name">
        {{ typeObj[type] ? typeObj[type].title : $t('appSpace.title') }}
      </span>
    </div>-->
    <div class="hide-loading-bg" style="padding: 20px" v-loading="loading">
      <div class="header-form-pr" style="padding-bottom: 20px">
        <div class="header-left">
          <search-input
            :placeholder="$t('appSpace.search')"
            ref="searchInput"
            @handleSearch="handleSearch"
          />
          <el-select
            v-if="type === workflow"
            v-model="subTypeFilter"
            size="small"
            clearable
            :placeholder="$t('appSpace.selectWorkflowType')"
            @change="handleSearch"
            class="subtype-filter no-border-input"
          >
            <el-option
              v-for="item in workflowTypeList"
              :key="item.value"
              :label="item.name"
              :value="item.value"
            />
          </el-select>
        </div>
        <div class="header-right">
          <el-button
            size="mini"
            type="primary"
            @click="showImport"
            v-if="type === workflow"
          >
            {{ $t('common.button.import') }}
          </el-button>
          <el-button
            size="mini"
            type="primary"
            @click="showCreate"
            icon="el-icon-plus"
          >
            {{ $t('common.button.create') }}
          </el-button>
        </div>
      </div>
      <AppList
        :type="type"
        :showCreate="showCreate"
        :appData="listData"
        :isShowPublished="true"
        :isShowTool="true"
        :isDev="true"
        @reloadData="getTableData"
        @convertToSkill="handleConvertToSkill"
      />
      <CreateTotalDialog ref="createTotalDialog" />
      <UploadFileDialog
        @reloadData="reloadData"
        :appType="type"
        :title="$t('appSpace.workflowExport')"
        ref="uploadFileDialog"
      />
      <ConvertSkillDialog ref="convertSkillDialog" />
    </div>
  </div>
</template>

<script>
import SearchInput from '@/components/searchInput.vue';
import AppList from '@/components/appList.vue';
import CreateTotalDialog from '@/components/createTotalDialog.vue';
import UploadFileDialog from '@/components/uploadFileDialog.vue';
import ConvertSkillDialog from '@/components/skills/convertSkillDialog.vue';
import { getAppSpaceList } from '@/api/appspace';
import { WORKFLOW, RAG, AGENT, WorkflowTypeList } from '@/utils/commonSet';
import { mapGetters } from 'vuex';
import { fetchPermFirPath } from '@/utils/util';

export default {
  components: {
    SearchInput,
    CreateTotalDialog,
    UploadFileDialog,
    AppList,
    ConvertSkillDialog,
  },
  data() {
    return {
      type: '',
      workflow: WORKFLOW,
      workflowTypeList: WorkflowTypeList,
      subTypeFilter: '',
      loading: false,
      listData: [],
      typeObj: {
        [WORKFLOW]: {
          title: this.$t('appSpace.workflow'),
          img: require('@/assets/imgs/workflow_icon.svg'),
        },
        [RAG]: {
          title: this.$t('appSpace.rag'),
          img: require('@/assets/imgs/rag.svg'),
        },
        [AGENT]: {
          title: this.$t('appSpace.agent'),
          img: require('@/assets/imgs/agent.svg'),
        },
      },
      currentTypeObj: {},
    };
  },
  watch: {
    $route: {
      handler(val) {
        this.listData = [];
        this.$refs.searchInput.value = '';
        this.initialPage(val);
      },
      // 深度观察监听
      deep: true,
    },
    fromList: {
      handler(val) {
        if (val !== '') {
          this.type = val;
          this.getTableData();
        }
      },
    },
  },
  computed: {
    ...mapGetters('app', ['fromList']),
  },
  mounted() {
    this.initialPage(this.$route);
  },
  methods: {
    initialPage(val) {
      const route = val || this.$route || {};
      const { type } = route.params || {};

      this.type = type;
      this.subTypeFilter = '';

      this.justifyRenderPage(type);
      this.getTableData();
    },
    justifyRenderPage(type) {
      if (![WORKFLOW, AGENT, RAG].includes(type)) {
        const { path } = fetchPermFirPath();
        this.$router.push({ path });
      }
    },
    reloadData() {
      this.$refs.searchInput.value = '';
      this.subTypeFilter = '';
      this.getTableData();
    },
    handleSearch() {
      this.getTableData();
    },
    getTableData() {
      this.loading = true;
      const searchInput = this.$refs.searchInput;
      const searchInfo = {
        appType: this.type === 'all' ? '' : this.type,
        ...(searchInput.value && { name: searchInput.value }),
      };
      let reqAppType = '';
      if (this.type === 'all') {
        reqAppType = 'app';
      } else if (this.type === AGENT) {
        reqAppType = 'assistant';
        delete searchInfo.appType;
      } else if (this.type === WORKFLOW) {
        reqAppType = WORKFLOW;
        delete searchInfo.appType;
        if (this.subTypeFilter) searchInfo.appType = this.subTypeFilter;
      } else {
        reqAppType = this.type;
        delete searchInfo.appType;
      }
      getAppSpaceList(reqAppType, searchInfo)
        .then(res => {
          this.loading = false;
          this.listData = res.data ? res.data.list || [] : [];
        })
        .catch(() => {
          this.loading = false;
          this.listData = [];
        });
    },
    showImport() {
      this.$refs.uploadFileDialog.openDialog();
    },
    showCreate() {
      switch (this.type) {
        case AGENT:
          this.$refs.createTotalDialog.showCreateIntelligent();
          break;
        case RAG:
          this.$refs.createTotalDialog.showCreateTxtQues();
          break;
        case WORKFLOW:
          this.$refs.createTotalDialog.showCreateWorkflow();
          break;
        default:
          this.$refs.createTotalDialog.openDialog();
          break;
      }
    },
    // 转化为 Skill
    handleConvertToSkill(row) {
      this.$refs.convertSkillDialog.open({
        id: row.appId,
        type: row.appType,
      });
    },
  },
};
</script>
<style lang="scss" scoped>
.scroll-card-container {
  max-height: calc(100vh - 125px);
}
.header-right {
  display: inline-block;
  float: right;
}
.header-left {
  display: inline-flex;
  align-items: center;
}
.subtype-filter {
  margin-left: 12px;
}
</style>
