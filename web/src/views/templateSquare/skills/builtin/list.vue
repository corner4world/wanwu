<template>
  <div class="tempSquare-management">
    <div class="tempSquare-content-box tempSquare-third">
      <div class="tempSquare-main">
        <div class="tempSquare-content">
          <div class="tempSquare-card-box">
            <div class="card-search card-search-cust">
              <SearchInput
                style="margin-right: 2px"
                :placeholder="$t('tempSquare.searchText')"
                ref="searchInput"
                @handleSearch="doGetSkillTempList"
              />
            </div>

            <div class="card-loading-box" v-if="list.length">
              <div class="card-box" v-loading="loading">
                <skill-card
                  v-for="(item, index) in list"
                  :key="index"
                  :info="item"
                  :type="1"
                  @download="handleDownload"
                >
                  <template v-slot:operations></template>
                </skill-card>
              </div>
            </div>
            <div v-else class="empty">
              <el-empty :description="$t('common.noData')"></el-empty>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>
<script>
import SkillCard from '../card.vue';
import SearchInput from '@/components/searchInput.vue';
import {
  getResourceBuiltinSkillList,
  downloadBuiltinSkill,
} from '@/api/templateSquare';
import { resDownloadFile } from '@/utils/util';

export default {
  components: { SearchInput, SkillCard },
  data() {
    return {
      list: [],
      loading: false,
    };
  },
  mounted() {
    this.doGetSkillTempList();
  },
  methods: {
    doGetSkillTempList() {
      const searchInput = this.$refs.searchInput;
      const params = {
        name: searchInput?.value,
      };

      this.loading = true;
      getResourceBuiltinSkillList(params)
        .then(res => {
          const { list } = res.data || {};
          this.list = list || [];
          this.loading = false;
        })
        .catch(() => (this.loading = false));
    },
    async handleDownload(info) {
      const { skillId, name } = info;
      if (!skillId) return;

      const res = await downloadBuiltinSkill({ skillId });
      resDownloadFile(res, `${name}.zip`);
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/tempSquare.scss';
.tempSquare-management {
  .card-search-cust {
    justify-content: flex-start;
    margin-top: 10px;
  }
}
</style>
