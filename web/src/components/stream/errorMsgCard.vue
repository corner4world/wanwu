<template>
  <div class="session-error">
    <div class="session-error-icon">
      <el-tooltip effect="dark" placement="top">
        <template #content>
          <div style="white-space: pre-wrap; width: 20vw">{{ desc }}</div>
        </template>
        <i class="el-icon-warning-outline"></i>
      </el-tooltip>
    </div>
    <div class="session-error-body">
      <div class="session-error-title">
        {{ title || $t('rag.answerFailed') }}
      </div>
    </div>
  </div>
</template>

<script>
export default {
  name: 'ErrorMsgCard',
  props: {
    title: {
      type: String,
      default: '',
    },
    desc: {
      type: String,
      default: '',
    },
  },
};
</script>

<style lang="scss" scoped>
.session-error {
  display: inline-flex;
  align-items: flex-start;
  gap: 10px;
  max-width: 100%;
  padding: 10px 14px;
  border-radius: 10px;
  border: 1px solid rgba(245, 108, 108, 0.22);
  background: linear-gradient(
    180deg,
    rgba(245, 108, 108, 0.06) 0%,
    #fafafa 100%
  );
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);
  margin-top: 10px;

  .session-error-icon {
    flex-shrink: 0;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: rgba(245, 108, 108, 0.1);
    border: 1px solid rgba(245, 108, 108, 0.22);
    display: flex;
    align-items: center;
    justify-content: center;
    color: #d93025;
    margin-top: 1px;
    .el-icon-warning-outline {
      font-size: 13px;
      font-weight: bold;
    }
  }

  .session-error-body {
    flex: 1;
    min-width: 0;
  }

  .session-error-title {
    font-size: 13px;
    font-weight: 500;
    color: #1f2937;
    line-height: 20px;
  }
}
</style>
