<template>
  <el-dialog
    :title="$t('statisticsDashboard.detailTitle')"
    :visible.sync="dialogVisible"
    append-to-body
    :close-on-click-modal="false"
    width="800px"
  >
    <div v-if="type === 'list'" class="detail-infos">
      <p>
        <label>{{ $t('statisticsDashboard.name') }}:</label>
        <span>{{ row.name }}</span>
      </p>
      <p>
        <label>{{ 'API Key' }}:</label>
        <span>{{ row.apiKey ? row.apiKey.slice(0, 6) + '******' : '--' }}</span>
      </p>
      <!--<p>
        <label>{{ $t('statisticsDashboard.model') }}:</label>
        <span>{{ row.model || '--' }}</span>
      </p>-->
      <p>
        <label>{{ $t('statisticsDashboard.apiPath') }}:</label>
        <span>{{ row.methodPath || '--' }}</span>
      </p>
      <p>
        <label>
          {{
            $t('statisticsDashboard.appCallCount') +
            ` (${$t('statisticsDashboard.frequency')})`
          }}:
        </label>
        <span>
          {{ formatAmount(row.callCount) }}
        </span>
      </p>
      <p>
        <label>
          {{
            $t('statisticsDashboard.appCallFailure') +
            ` (${$t('statisticsDashboard.frequency')})`
          }}:
        </label>
        <span>{{ formatAmount(row.callFailure) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.avgStreamCosts') + ` (ms)` }}:</label>
        <span>{{ formatAmount(row.avgStreamCosts) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.avgCosts') + ` (ms)` }}:</label>
        <span>{{ formatAmount(row.avgNonStreamCosts) }}</span>
      </p>
      <p>
        <label>
          {{
            $t('statisticsDashboard.streamCount') +
            ` (${$t('statisticsDashboard.frequency')})`
          }}:
        </label>
        <span>{{ formatAmount(row.streamCount) }}</span>
      </p>
      <p>
        <label>
          {{
            $t('statisticsDashboard.nonStreamCount') +
            ` (${$t('statisticsDashboard.frequency')})`
          }}:
        </label>
        <span>{{ formatAmount(row.nonStreamCount) }}</span>
      </p>
      <!--<p>
        <label>{{ $t('statisticsDashboard.apiPromptTokens') }}:</label>
        <span>{{ formatAmount(row.promptTokens) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.apiCompletionTokens') }}:</label>
        <span>{{ formatAmount(row.completionTokens) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.apiTotalTokens') }}:</label>
        <span>{{ formatAmount(row.totalTokens) }}</span>
      </p>-->
    </div>
    <div v-else class="detail-infos">
      <p>
        <label>{{ $t('statisticsDashboard.name') }}:</label>
        <span>{{ row.name }}</span>
      </p>
      <p>
        <label>{{ 'API Key' }}:</label>
        <span>{{ row.apiKey ? row.apiKey.slice(0, 6) + '******' : '--' }}</span>
      </p>
      <!--<p>
        <label>{{ $t('statisticsDashboard.model') }}:</label>
        <span>{{ row.model || '&#45;&#45;' }}</span>
      </p>-->
      <p>
        <label>{{ $t('statisticsDashboard.apiPath') }}:</label>
        <span>{{ row.methodPath || '--' }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.responseStatus') }}:</label>
        <span>
          {{ row.responseStatus || '--' }}
        </span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.callTime') }}:</label>
        <span>{{ row.callTime || '--' }}</span>
      </p>
      <!--<p>
        <label>{{ $t('statisticsDashboard.reqStart') }}:</label>
        <span>{{ row.requestStartAt || '&#45;&#45;' }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.firstTokenCreated') }}:</label>
        <span>{{ row.firstTokenCreated || '&#45;&#45;' }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.lastTokenCreated') }}:</label>
        <span>{{ row.lastTokenCreated || '&#45;&#45;' }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.apiPromptTokens') }}:</label>
        <span>{{ formatAmount(row.promptTokens) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.apiCompletionTokens') }}:</label>
        <span>{{ formatAmount(row.completionTokens) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.apiTotalTokens') }}:</label>
        <span>{{ formatAmount(row.totalTokens) }}</span>
      </p>-->
      <p>
        <label>{{ $t('statisticsDashboard.streamCosts') + ` (ms)` }}:</label>
        <span>{{ formatAmount(row.streamCosts) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.nonStreamCosts') + ` (ms)` }}:</label>
        <span>{{ formatAmount(row.nonStreamCosts) }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.reqContent') }}:</label>
        <span>{{ row.requestBody || '--' }}</span>
      </p>
      <p>
        <label>{{ $t('statisticsDashboard.resContent') }}:</label>
        <span>{{ row.responseBody || '--' }}</span>
      </p>
      <!--<p>
        <label>{{ $t('statisticsDashboard.finishReason') }}:</label>
        <span>{{ row.finishReason || '&#45;&#45;' }}</span>
      </p>-->
    </div>
  </el-dialog>
</template>

<script>
import { formatAmount } from '@/utils/util.js';
export default {
  props: {
    type: '',
  },
  data() {
    return {
      dialogVisible: false,
      row: {},
    };
  },
  methods: {
    formatAmount,
    openDialog(row) {
      this.row = row;
      this.dialogVisible = true;
    },
  },
};
</script>

<style lang="scss" scoped>
.detail-infos {
  margin-top: -26px;
  p {
    padding: 10px 0;
    display: flex;
    label {
      display: inline-block;
      width: 150px;
    }
    span {
      display: inline-block;
      width: 0;
      flex: 1;
      position: relative;
      margin-left: 10px;
    }
  }
}
</style>
