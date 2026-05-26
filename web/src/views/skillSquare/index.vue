<template>
  <div class="page-wrapper">
    <div class="app-header">
      <div class="header-top">
        <div class="taglist_warp">
          <div
            v-for="item in tagList"
            :key="item.value"
            class="tagList"
            @click="handleTagClick(item)"
            :class="{ white: item.value === active }"
          >
            <img
              :src="item.value === active ? item.activeImg : item.unactiveImg"
              class="h-icon"
            />
            <span>{{ item.name }}</span>
          </div>
        </div>
        <SearchInput
          :placeholder="placeholder"
          style="width: 200px"
          @handleSearch="handleSearch"
        />
      </div>
      <div class="explore-tab-pane">
        <SkillList
          :list="listData"
          :type="active"
          :loading="loading"
          :showShared="active !== 'builtin'"
          @download="handleDownload"
          @link-more="handleLinkMore"
          @card-click="handleCardClick"
          @send-to-resource="handleSendToResource"
        />
      </div>
    </div>
  </div>
</template>

<script>
import SearchInput from '@/components/searchInput.vue';
import SkillList from './components/list.vue';
import CreateTotalDialog from '@/components/createTotalDialog.vue';
import {
  getBuiltinSquareSkillList,
  downloadSquareSkill,
  sendSquareSkillToResource,
  getSharedSquareSkillList,
  addSharedSkillToResource,
  downloadSharedSquareSkill,
  getCreatedSquareSkillList,
} from '@/api/skillSquare';
import { downloadBuiltinSkill } from '@/api/templateSquare';
import { resDownloadFile } from '@/utils/util';

export default {
  components: { SearchInput, CreateTotalDialog, SkillList },
  data() {
    return {
      placeholder: this.$t('skillSpace.search'),
      searchValue: '',
      loading: false,
      active: 'builtin',
      tagList: [
        // {
        //   name: this.$t('explore.tag.all'),
        //   value: 'all',
        //   activeImg: require('@/assets/imgs/all_active.svg'),
        //   unactiveImg: require('@/assets/imgs/all_unactive.svg'),
        // },
        {
          name: this.$t('skillSpace.builtin'),
          value: 'builtin',
          activeImg: require('@/assets/imgs/mine_active.svg'),
          unactiveImg: require('@/assets/imgs/mine_unactive.svg'),
        },
        {
          name: this.$t('skillSpace.shared'),
          value: 'shared',
          activeImg: require('@/assets/imgs/all_active.svg'),
          unactiveImg: require('@/assets/imgs/all_unactive.svg'),
        },
        // {
        //   name: this.$t('explore.tag.favorite'),
        //   value: 'favorite',
        //   activeImg: require('@/assets/imgs/mine_active.svg'),
        //   unactiveImg: require('@/assets/imgs/mine_unactive.svg'),
        // },
        {
          name: this.$t('skillSpace.mine'),
          value: 'mine',
          activeImg: require('@/assets/imgs/start_active.svg'),
          unactiveImg: require('@/assets/imgs/start_unactive.svg'),
        },
      ],
      listData: [],
    };
  },
  created() {
    this.initActiveByRouteType();
    this.getExplorationList();
  },
  mounted() {},
  methods: {
    initActiveByRouteType() {
      const { type } = this.$route.query;
      const targetTag = this.tagList.find(item => item.value === type);
      if (targetTag) {
        this.active = targetTag.value;
      }
    },
    handleSearch(value) {
      this.searchValue = value;
      this.getExplorationList();
    },
    handleTagClick(item) {
      this.active = item.value;
      this.getExplorationList();
      this.updateRouteType(item.value);
    },
    updateRouteType(type) {
      if (this.$route.query.type === type) {
        return;
      }

      const route = {
        query: {
          ...this.$route.query,
          type,
        },
      };

      if (this.$route.name) {
        route.name = this.$route.name;
        route.params = this.$route.params;
      } else {
        route.path = this.$route.path;
      }

      this.$router.replace(route).catch(err => {
        if (err && err.name !== 'NavigationDuplicated') {
          throw err;
        }
      });
    },
    getExplorationList() {
      const params = {
        name: this.searchValue,
      };
      let requestApi = getBuiltinSquareSkillList;
      if (this.active === 'builtin') {
        requestApi = getBuiltinSquareSkillList;
      } else if (this.active === 'shared') {
        requestApi = getSharedSquareSkillList;
      } else if (this.active === 'mine') {
        requestApi = getCreatedSquareSkillList;
      }

      this.loading = true;
      requestApi(params)
        .then(res => {
          const { list } = res.data || {};
          this.listData = list || [];
          this.loading = false;
        })
        .catch(() => {
          this.loading = false;
        });
    },
    handleDownload(info) {
      let downloadApi = downloadSquareSkill;
      if (this.active === 'builtin') {
        downloadApi = downloadBuiltinSkill;
      } else if (this.active === 'shared') {
        downloadApi = downloadSharedSquareSkill;
      }

      downloadApi({ skillId: info.skillId }).then(response => {
        resDownloadFile(response, `${info.name}.zip`);
      });
    },
    handleSendToResource(info) {
      const requestApi =
        this.active === 'shared'
          ? addSharedSkillToResource
          : sendSquareSkillToResource;

      requestApi({ skillId: info.skillId }).then(res => {
        if (res.code === 0) {
          this.$message.success(this.$t('common.info.send'));
          info.isShared = true;
        }
      });
    },
    handleLinkMore() {
      window.open('https://clawhub.ai/skills?sort=downloads', '_blank');
    },
    handleCardClick(info) {
      const path = '/skillSquare/detail';
      const query = { skillId: info.skillId };
      if (this.active === 'builtin') {
        query.skillType = 'builtin';
      } else if (this.active === 'shared') {
        query.skillType = 'shared';
      } else if (this.active === 'mine') {
        query.skillType = 'mine';
      }
      this.$router.push({
        path,
        query: query,
      });
    },
  },
};
</script>
<style lang="scss" scoped>
@import '@/style/tabs.scss';
::v-deep {
  .el-tabs__content {
    overflow: unset;
  }

  .table-search-input {
    height: 30px;
  }
}

.white {
  font-weight: bold;
  color: $color;
  border-bottom: 2.5px solid $color !important;
}

.page-wrapper {
  padding: 10px 30px 20px;
  box-sizing: border-box;

  .header-top {
    display: flex;
    justify-content: space-between;
    padding: 15px 0 6px 0;
    box-sizing: border-box;

    .tagList:nth-child(1) {
      margin-left: 0 !important;
    }

    .taglist_warp {
      display: flex;
      margin-top: -20px;

      .tagList {
        margin: 10px;
        padding: 0 3px;
        height: 36px;
        line-height: 36px;
        cursor: pointer;
        display: flex;
        align-items: center;
        border-bottom: 2.5px solid rgba(255, 255, 255, 0);

        .h-icon {
          margin-right: 5px;
          width: 14px;
        }
      }
    }
  }
}
.explore-tab-pane ::v-deep {
  .el-tabs__nav-wrap::after,
  .el-tabs__active-bar {
    background-color: rgba(255, 255, 255, 0) !important;
  }
  .el-tabs__item {
    font-size: 13px;
    height: 32px;
    line-height: 32px;
    padding: 0 10px !important;
    margin-right: 6px;
    &.is-active {
      background-color: $color-opacity !important;
      border-radius: 16px;
      font-weight: bold;
    }
  }
  .el-tabs__header {
    margin: 0 !important;
  }
}
</style>
