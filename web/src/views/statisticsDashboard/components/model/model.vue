<template>
  <div class="statistics_common list-common statistics_client_wrapper">
    <div>
      <div style="padding: 5px 24px">
        <label>{{ $t('statisticsDashboard.modelSelect') }}:</label>
        <el-select
          v-model="modelParams.modelType"
          :placeholder="$t('modelAccess.table.modelType')"
          class="no-border-select"
          style="margin-left: 15px"
          clearable
          @change="changeModelType()"
        >
          <el-option
            v-for="item in modelTypeList"
            :key="item.key"
            :label="item.name"
            :value="item.key"
          />
        </el-select>
        <el-select
          v-model="modelParams.models"
          :placeholder="$t('statisticsDashboard.model')"
          class="no-border-select scroll-select"
          style="margin-left: 15px; width: 600px"
          clearable
          multiple
          filterable
        >
          <el-option
            v-for="item in modelList"
            v-if="item.displayName"
            :key="item.modelId"
            :label="item.displayName"
            :value="item.modelId"
          >
            <div class="model-option-content">
              <div class="model-option-content-left">
                <img
                  class="model-img"
                  :src="convertModelIcon(item?.avatar.path)"
                />
                <span class="model-name">
                  {{ item.displayName }}
                </span>
              </div>

              <div
                class="model-select-tags"
                v-if="item.tags && item.tags.length > 0"
              >
                <span
                  v-for="(tag, tagIdx) in item.tags"
                  :key="tagIdx"
                  class="model-select-tag"
                >
                  {{ tag.text }}
                </span>
              </div>
            </div>
          </el-option>
        </el-select>
      </div>
      <div>
        <Search
          v-show="searchShow"
          ref="search"
          @handleSetTime="handleSetTime"
        ></Search>
      </div>
    </div>
    <div class="statistics_content_box scroll-card-container">
      <div class="item_box">
        <div class="dataOverview">
          <span class="title">
            {{ $t('statistics.overview') }}
          </span>
          <div class="client_dataOverview_content" v-loading="loading">
            <div v-for="(item, index) in count" :key="index" class="card">
              <span>
                {{ item.name }}
                <i
                  :style="{
                    background: item.des_value < 0 ? '#1afa29' : '#d81e06',
                  }"
                  :class="{
                    defaultBg: item.des_value === 0 || item.des_value === -9999,
                  }"
                ></i>
              </span>
              <strong>{{ formatAmount(item.value) }}{{ item.unit }}</strong>
              <span>
                {{ item.des }}
                <label
                  :style="{
                    color: item.des_value < 0 ? '#1afa29' : '#d81e06',
                  }"
                  :class="{
                    defaultColor:
                      item.des_value === 0 || item.des_value === -9999,
                  }"
                >
                  {{ item.des_value === -9999 ? '-' : item.des_value + '%' }}
                </label>
                <img
                  v-if="item.des_value < 0 && item.des_value !== -9999"
                  src="@/assets/imgs/descend.png"
                  alt=""
                />
                <img
                  v-if="item.des_value > 0 && item.des_value !== -9999"
                  src="@/assets/imgs/rise.png"
                  alt=""
                />
              </span>
            </div>
          </div>
        </div>
        <div class="data_echart_box">
          <div class="data_echart">
            <UserEchart
              :content="
                echartContent.modelCalls ? echartContent.modelCalls.lines : []
              "
              :name="
                echartContent.modelCalls
                  ? echartContent.modelCalls.tableName
                  : ''
              "
              v-loading="loading"
            ></UserEchart>
          </div>
          <div class="data_echart">
            <UserEchart
              :content="
                echartContent.tokensUsage ? echartContent.tokensUsage.lines : []
              "
              :name="
                echartContent.tokensUsage
                  ? echartContent.tokensUsage.tableName
                  : ''
              "
              v-loading="loading"
            ></UserEchart>
          </div>
        </div>
        <div class="dataOverview">
          <span class="title">
            {{ $t('statisticsDashboard.modelList') }}
          </span>
          <div style="margin-top: -20px">
            <ModelList
              :params="formatParams({ ...params, ...modelParams })"
              ref="modelList"
            />
          </div>
        </div>
      </div>
    </div>
    <el-backtop target=".statistics_content_box"></el-backtop>
  </div>
</template>
<script>
import Search from '@/components/searchDate.vue';
import UserEchart from '@/components/echart/userEchart.vue';
import ModelList from './modelList.vue';
import { avatarSrc, formatAmount, getModelDefaultIcon } from '@/utils/util.js';
import { getModelData } from '@/api/statisticsDashboard';
import { MODEL_TYPE, LLM } from '@/views/modelAccess/constants';
import { fetchModelList } from '@/api/modelAccess';

export default {
  components: {
    UserEchart,
    Search,
    ModelList,
  },
  data() {
    return {
      modelTypeList: MODEL_TYPE,
      modelList: [],
      loading: false,
      content: {}, // 存储返回的总揽数据
      echartContent: {}, // 存储返回的echart数据
      count: [
        {
          name: this.$t('statisticsDashboard.tokenTotals'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'totalTokensTotal',
          des_value: -9999,
          unit: this.$t('statisticsDashboard.quantity'),
        },
        {
          name: this.$t('statisticsDashboard.promptTokensTotals'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'promptTokensTotal',
          des_value: -9999,
          unit: this.$t('statisticsDashboard.quantity'),
        },
        {
          name: this.$t('statisticsDashboard.completionTokensTotals'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'completionTokensTotal',
          des_value: -9999,
          unit: this.$t('statisticsDashboard.quantity'),
        },
        {
          name: this.$t('statisticsDashboard.avgCosts'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'avgCosts',
          des_value: -9999,
          unit: 'ms',
        },
        {
          name: this.$t('statisticsDashboard.callCount'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'callCountTotal',
          des_value: -9999,
          unit: this.$t('statisticsDashboard.frequency'),
        },
        {
          name: this.$t('statisticsDashboard.callFailure'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'callFailureTotal',
          des_value: -9999,
          unit: this.$t('statisticsDashboard.frequency'),
        },
        {
          name: this.$t('statisticsDashboard.avgFirstTokenLatency'),
          value: 0,
          des: this.$t('statistics.percentage'),
          key: 'avgFirstTokenLatency',
          des_value: -9999,
          unit: 'ms',
        },
      ],
      searchShow: true,
      searchTime: {
        time: [],
      },
      modelParams: {
        modelType: LLM,
        models: [],
      },
    };
  },
  computed: {
    params() {
      return {
        endDate: this.searchTime.time[1],
        startDate: this.searchTime.time[0],
      };
    },
  },
  mounted() {
    this.fetchModels();
  },
  methods: {
    formatAmount,
    formatParams(params) {
      return {
        ...params,
        models: params.models ? params.models.toString() : '',
      };
    },
    changeModelType() {
      this.fetchModels();
      this.fetchData(
        this.formatParams({ ...this.params, ...this.modelParams }),
      );
    },
    async fetchModels() {
      this.modelList = [];
      this.modelParams.models = [];

      const res = await fetchModelList({
        filterScope: '',
        modelType: this.modelParams.modelType,
      });
      const modelList = res.data ? res.data.list || [] : [];
      this.modelList = modelList.filter(item => item.allowEdit);
    },
    fetchData(params) {
      getModelData(params)
        .then(res => {
          const { overview, trend } = res.data || {};
          this.content = overview || {};
          this.echartContent = trend || {};
          // 解构后台返回的数据，暂存和 count 数组中key对应的数据
          this.count.map(item => {
            item.value = overview[item.key] ? overview[item.key].value : 0;
            item.des_value = overview[item.key]
              ? overview[item.key].periodOverPeriod
              : -9999;
          });
        })
        .finally(() => {
          this.loading = false;
        });
      this.$refs.modelList.getTableData(params);
    },
    handleSetTime(val) {
      this.loading = true;
      this.searchTime = val;

      const params = this.formatParams({
        startDate: val.time[0],
        endDate: val.time[1],
        ...this.modelParams,
      });
      this.fetchData(params);
    },
    convertModelIcon(iconPath) {
      return iconPath ? avatarSrc(iconPath) : getModelDefaultIcon();
    },
  },
};
</script>
<style lang="scss" scoped>
@import '@/style/modelSelect.scss';
@import '@/style/statisticsDashboard.scss';
</style>
