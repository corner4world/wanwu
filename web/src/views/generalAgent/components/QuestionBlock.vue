<template>
  <div :class="['question-block', statusClass]">
    <!-- 折叠头部 -->
    <div class="question-header" @click="toggleExpand">
      <div class="header-left">
        <i :class="isExpanded ? 'el-icon-arrow-up' : 'el-icon-arrow-down'"></i>
        <div class="question-icon-wrapper">
          <i class="el-icon-question question-icon"></i>
        </div>
        <span class="question-title">
          {{ $t('generalAgent.question.pleaseSelect') }}
        </span>
        <span v-if="questionCount > 1" class="question-count">
          {{ questionCount }} {{ $t('generalAgent.question.questions') }}
        </span>
      </div>
      <div class="header-right">
        <div :class="['status-badge', status]">
          <span class="status-dot"></span>
          <span class="status-text">{{ statusText }}</span>
        </div>
      </div>
    </div>

    <!-- 折叠内容 -->
    <el-collapse-transition>
      <div v-show="isExpanded" class="question-body">
        <div v-for="(q, index) in questions" :key="index" class="question-item">
          <div class="question-meta">
            <h4 v-if="q.header" class="question-header-label">
              {{ q.header }}
            </h4>
            <span class="question-type-badge">
              <template v-if="q.multiple">
                {{ $t('generalAgent.question.multiSelect') }}
              </template>
              <template v-else>
                {{ $t('generalAgent.question.singleSelect') }}
              </template>
            </span>
          </div>
          <p class="question-text">{{ q.question }}</p>

          <div v-if="q.options && q.options.length" class="options">
            <div
              v-for="opt in q.options"
              :key="opt.label"
              class="option-item"
              :class="{
                selected: isSelected(q, opt.label),
                disabled: status !== 'pending',
              }"
              @click="status === 'pending' && selectOption(q, opt.label)"
            >
              <span class="option-indicator">
                <span
                  v-if="q.multiple"
                  :class="{ checked: isSelected(q, opt.label) }"
                  class="checkbox"
                >
                  <i v-if="isSelected(q, opt.label)" class="el-icon-check"></i>
                </span>
                <span
                  v-else
                  :class="{ checked: isSelected(q, opt.label) }"
                  class="radio"
                ></span>
              </span>
              <div class="option-content">
                <span class="option-label">{{ opt.label }}</span>
                <span v-if="opt.description" class="option-desc">
                  {{ opt.description }}
                </span>
              </div>
            </div>
            <!-- 自定义输入选项 -->
            <div
              v-if="q.custom"
              :class="{
                selected: isSelected(q, '__CUSTOM__'),
                disabled: status !== 'pending',
              }"
              class="option-item"
              @click="status === 'pending' && selectOption(q, '__CUSTOM__')"
            >
              <span class="option-indicator">
                <span
                  v-if="q.multiple"
                  :class="{ checked: isSelected(q, '__CUSTOM__') }"
                  class="checkbox"
                >
                  <i
                    v-if="isSelected(q, '__CUSTOM__')"
                    class="el-icon-check"
                  ></i>
                </span>
                <span
                  v-else
                  :class="{ checked: isSelected(q, '__CUSTOM__') }"
                  class="radio"
                ></span>
              </span>
              <div class="option-content">
                <span class="option-label">
                  {{ $t('generalAgent.question.other') }}
                </span>
                <el-input
                  ref="customInput"
                  v-model="customAnswers[index]"
                  :placeholder="$t('generalAgent.question.customPlaceholder')"
                  class="custom-input-inline"
                />
              </div>
            </div>
          </div>
        </div>

        <!-- 按钮区域（仅 pending 时显示） -->
        <div v-if="status === 'pending'" class="actions">
          <button
            :disabled="!canSubmit"
            class="btn-primary"
            @click="submitReply"
          >
            {{ $t('generalAgent.question.submit') }}
          </button>
          <button class="btn-secondary" @click="reject">
            {{ $t('generalAgent.question.cancel') }}
          </button>
        </div>
      </div>
    </el-collapse-transition>
  </div>
</template>

<script>
import { replyQuestion, rejectQuestion } from '@/api/generalAgent';

export default {
  name: 'QuestionBlock',
  props: {
    questionId: { type: String, required: true },
    runId: { type: String, required: true },
    status: { type: String, default: 'pending' },
    questions: { type: Array, default: () => [] },
    answers: { type: Array, default: null },
  },
  data() {
    return {
      isExpanded: this.status === 'pending',
      selectedOptions: {},
      customAnswers: {},
    };
  },
  computed: {
    statusClass() {
      return `status-${this.status}`;
    },
    statusText() {
      const texts = {
        pending: this.$t('generalAgent.question.pending'),
        answered: this.$t('generalAgent.question.answered'),
        rejected: this.$t('generalAgent.question.rejected'),
      };
      return texts[this.status] || this.status;
    },
    questionCount() {
      return this.questions ? this.questions.length : 0;
    },
    canSubmit() {
      if (this.questions.length === 0) return false;
      return this.questions.every((q, index) => {
        const selected = this.selectedOptions[index] || [];
        const custom = this.customAnswers[index];
        const hasCustomInput = custom && custom.trim().length > 0;

        // 过滤掉"__CUSTOM__"标记
        const filteredSelected = selected.filter(s => s !== '__CUSTOM__');
        const hasSelection = filteredSelected.length > 0;
        const hasCustomSelected = selected.includes('__CUSTOM__');

        // 选了普通选项就可以提交
        if (hasSelection) return true;
        // 如果选择了且有自定义输入，也可以提交
        if (hasCustomSelected && hasCustomInput) return true;
        return false;
      });
    },
  },
  watch: {
    status: {
      immediate: true,
      handler(val, oldVal) {
        // 只有 pending 状态自动展开，其他状态收起
        if (val === 'pending') {
          this.isExpanded = true;
        } else if (!oldVal) {
          // 初始加载时，非 pending 状态收起
          this.isExpanded = false;
        }
        // 从 pending 变为其他状态时自动收起
        if (
          oldVal === 'pending' &&
          (val === 'answered' || val === 'rejected')
        ) {
          this.isExpanded = false;
        }
      },
    },
    answers: {
      immediate: true,
      handler(val) {
        if (val && Array.isArray(val)) {
          this.restoreAnswers(val);
        }
      },
    },
  },
  methods: {
    toggleExpand() {
      this.isExpanded = !this.isExpanded;
    },
    restoreAnswers(answers) {
      answers.forEach((answer, index) => {
        if (this.questions[index]) {
          const q = this.questions[index];

          // 检查答案中是否包含自定义输入
          const optionLabels = q.options.map(opt => opt.label);
          const customAnswer = answer.find(a => !optionLabels.includes(a));
          const optionAnswers = answer.filter(a => optionLabels.includes(a));

          // 设置普通选项
          if (optionAnswers.length > 0) {
            this.$set(this.selectedOptions, index, optionAnswers);
          }

          // 如果有自定义输入，设置选项和自定义内容
          if (customAnswer) {
            const currentSelected = this.selectedOptions[index] || [];
            this.$set(this.selectedOptions, index, [
              ...currentSelected,
              '__CUSTOM__',
            ]);
            this.$set(this.customAnswers, index, customAnswer);
          }
        }
      });
    },
    isSelected(question, label) {
      const index = this.questions.indexOf(question);
      return (this.selectedOptions[index] || []).includes(label);
    },
    selectOption(question, label) {
      const index = this.questions.indexOf(question);
      if (!this.selectedOptions[index]) {
        this.$set(this.selectedOptions, index, []);
      }
      const selected = this.selectedOptions[index];
      if (question.multiple) {
        const idx = selected.indexOf(label);
        if (idx >= 0) {
          selected.splice(idx, 1);
        } else {
          selected.push(label);
        }
      } else {
        this.$set(this.selectedOptions, index, [label]);
      }

      // 如果选择的是自定义选项，自动聚焦到输入框
      if (label === '__CUSTOM__') {
        this.$nextTick(() => {
          const customInputs = this.$refs.customInput;

          if (Array.isArray(customInputs)) {
            // v-for 中的 ref 会返回数组
            const targetInput = customInputs[index];

            if (targetInput && targetInput.$el) {
              const inputElement = targetInput.$el.querySelector('input');
              if (inputElement) {
                inputElement.focus();
              }
            }
          } else if (customInputs && customInputs.$el) {
            // 单个 ref 的情况
            const inputElement = customInputs.$el.querySelector('input');
            if (inputElement) {
              inputElement.focus();
            }
          }
        });
      }
    },
    async submitReply() {
      const answers = this.questions.map((q, index) => {
        const selected = this.selectedOptions[index] || [];
        const custom = this.customAnswers[index];

        // 过滤掉"__CUSTOM__"标记
        const filteredSelected = selected.filter(s => s !== '__CUSTOM__');

        // 如果选择了且有自定义输入，将自定义内容加入答案
        if (
          selected.includes('__CUSTOM__') &&
          custom &&
          custom.trim().length > 0
        ) {
          return [...filteredSelected, custom.trim()];
        }

        // 否则返回普通选项
        if (filteredSelected.length > 0) {
          return filteredSelected;
        }

        return [];
      });

      try {
        await replyQuestion({
          runId: this.runId,
          questionId: this.questionId,
          answers,
        });
        this.$message.success(this.$t('generalAgent.question.replySuccess'));
        this.$emit('reply', { questionId: this.questionId, answers });
      } catch (error) {
        console.error('Failed to reply question:', error);
        this.$message.error(this.$t('generalAgent.question.replyFailed'));
      }
    },
    async reject() {
      try {
        await rejectQuestion({
          runId: this.runId,
          questionId: this.questionId,
        });
        this.$message.success(this.$t('generalAgent.question.rejectSuccess'));
        this.$emit('reject', { questionId: this.questionId });
      } catch (error) {
        console.error('Failed to reject question:', error);
        this.$message.error(this.$t('generalAgent.question.rejectFailed'));
      }
    },
  },
};
</script>

<style lang="scss" scoped>
@import '../styles/_variables.scss';

$question-color: #10a37f;
$question-dark: #0d8a6a;
$question-pending: #3b82f6;
$question-pending-dark: #2563eb;
$question-rejected: #ef4444;
$question-rejected-dark: #dc2626;

.question-block {
  margin-bottom: 14px;
  border-radius: 14px;
  border: 1px solid rgba(16, 163, 127, 0.2);
  overflow: hidden;
  transition: all 0.3s ease;
  background: #fff;
  font-family: $font-sans;
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.04);

  // pending 状态
  &.status-pending {
    border-color: rgba(59, 130, 246, 0.4);
    box-shadow: 0 4px 16px rgba(59, 130, 246, 0.15);

    .question-header {
      background: linear-gradient(
        135deg,
        rgba(59, 130, 246, 0.06) 0%,
        #fafafa 100%
      );

      .question-title {
        color: $question-pending-dark;
        font-weight: 600;
      }
    }

    .question-icon {
      animation: spin 1s linear infinite;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(59, 130, 246, 0.12) 0%,
        rgba(59, 130, 246, 0.08) 100%
      );
      border: 1px solid rgba(59, 130, 246, 0.2);

      .status-dot {
        background: $question-pending;
        animation: pulse 1.5s infinite;
        box-shadow: 0 0 6px rgba(59, 130, 246, 0.5);
      }

      .status-text {
        color: $question-pending-dark;
      }
    }
  }

  // answered 状态
  &.status-answered {
    border-color: rgba(16, 163, 127, 0.3);
    box-shadow: 0 2px 8px rgba(16, 163, 127, 0.08);

    .question-header {
      background: linear-gradient(
        135deg,
        rgba(16, 163, 127, 0.06) 0%,
        #fafafa 100%
      );
    }

    .question-icon {
      color: $question-color;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(16, 163, 127, 0.12) 0%,
        rgba(16, 163, 127, 0.08) 100%
      );
      border: 1px solid rgba(16, 163, 127, 0.2);

      .status-dot {
        background: $question-color;
        box-shadow: 0 0 4px rgba(16, 163, 127, 0.4);
      }

      .status-text {
        color: $question-dark;
      }
    }
  }

  // rejected 状态
  &.status-rejected {
    border-color: rgba(239, 68, 68, 0.3);
    box-shadow: 0 2px 8px rgba(239, 68, 68, 0.08);

    .question-header {
      background: linear-gradient(
        135deg,
        rgba(239, 68, 68, 0.06) 0%,
        #fafafa 100%
      );
    }

    .question-icon {
      color: $question-rejected;
    }

    .status-badge {
      background: linear-gradient(
        135deg,
        rgba(239, 68, 68, 0.12) 0%,
        rgba(239, 68, 68, 0.08) 100%
      );
      border: 1px solid rgba(239, 68, 68, 0.2);

      .status-dot {
        background: $question-rejected;
      }

      .status-text {
        color: $question-rejected-dark;
      }
    }
  }
}

.question-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  background: linear-gradient(180deg, #fafbfc 0%, #f8f9fa 100%);
  cursor: pointer;
  user-select: none;
  transition: background 0.2s ease;

  &:hover {
    background: linear-gradient(180deg, #f5f7f9 0%, #f3f4f6 100%);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: 12px;
    flex: 1;
    min-width: 0;

    i.el-icon-arrow-down,
    i.el-icon-arrow-up {
      color: $text-muted;
      font-size: 12px;
      transition: transform 0.2s ease;
    }

    .question-icon-wrapper {
      display: flex;
      align-items: center;
      justify-content: center;
      width: 24px;
      height: 24px;

      .question-icon {
        font-size: 18px;
        color: $text-muted;
      }
    }

    .question-title {
      font-size: 14px;
      font-weight: 600;
      color: $text-primary;
      letter-spacing: 0.01em;
    }

    .question-count {
      font-size: 12px;
      color: $text-muted;
      background: rgba(0, 0, 0, 0.04);
      padding: 3px 8px;
      border-radius: 8px;
    }
  }

  .header-right {
    display: flex;
    align-items: center;
    gap: 12px;
    flex-shrink: 0;

    .status-badge {
      display: flex;
      align-items: center;
      gap: 6px;
      padding: 4px 12px;
      border-radius: 14px;
      font-size: 12px;
      font-weight: 500;
      letter-spacing: 0.02em;

      .status-dot {
        width: 6px;
        height: 6px;
        border-radius: 50%;
      }

      .status-text {
        line-height: 1;
      }
    }
  }
}

.question-body {
  padding: 16px 18px;
  background: #fff;
  border-top: 1px solid #e8ecf0;
}

.question-item {
  &:not(:last-child) {
    border-bottom: 1px solid #e8ecf0;
    padding-bottom: 16px;
    margin-bottom: 16px;
  }
}

.question-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 8px;
}

.question-header-label {
  font-size: 11px;
  font-weight: 600;
  color: $text-secondary;
  text-transform: uppercase;
  letter-spacing: 0.08em;
  margin: 0;
}

.question-type-badge {
  font-size: 11px;
  color: $accent-color;
  background: rgba(16, 163, 127, 0.1);
  padding: 2px 8px;
  border-radius: 4px;
  font-weight: 500;
}

.question-text {
  font-size: 15px;
  color: $text-primary;
  margin: 0 0 16px 0;
  line-height: 1.5;
}

.options {
  display: flex;
  flex-direction: column;
  gap: 10px;
  margin-bottom: 12px;
}

.option-item {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 12px 14px;
  background: #fafafa;
  border: 1px solid #e5e7eb;
  border-radius: 10px;
  cursor: pointer;
  transition: all 0.2s ease;

  &:hover:not(.disabled) {
    border-color: $accent-color;
    background: rgba(16, 163, 127, 0.05);
  }

  &.selected {
    border-color: $accent-color;
    background: rgba(16, 163, 127, 0.1);

    .option-label {
      color: $accent-dark;
      font-weight: 500;
    }
  }

  &.disabled {
    cursor: not-allowed;
    opacity: 0.7;
  }
}

.option-indicator {
  flex-shrink: 0;
  margin-top: 2px;
}

.radio {
  display: block;
  width: 16px;
  height: 16px;
  border: 2px solid #d1d5db;
  border-radius: 50%;
  transition: all 0.2s ease;
  position: relative;

  &.checked {
    border-color: $accent-color;

    &::after {
      content: '';
      position: absolute;
      top: 50%;
      left: 50%;
      transform: translate(-50%, -50%);
      width: 8px;
      height: 8px;
      background: $accent-color;
      border-radius: 50%;
    }
  }
}

.checkbox {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border: 2px solid #d1d5db;
  border-radius: 4px;
  transition: all 0.2s ease;

  &.checked {
    border-color: $accent-color;
    background: $accent-color;

    i {
      color: white;
      font-size: 10px;
      font-weight: bold;
    }
  }
}

.option-content {
  flex: 1;
  min-width: 0;
}

.option-label {
  font-size: 14px;
  color: $text-primary;
  display: block;
}

.option-desc {
  font-size: 12px;
  color: $text-muted;
  display: block;
  margin-top: 4px;
  line-height: 1.4;
}

.custom-input-inline {
  flex: 1;
  min-width: 0;
  margin-top: 4px;

  ::v-deep .el-input__inner {
    padding: 4px 8px;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    font-size: 13px;
    transition: all 0.2s ease;
    background: transparent;

    &:focus {
      border-color: $accent-color !important;
      box-shadow: 0 0 0 2px rgba(16, 163, 127, 0.1);
    }
  }
}

.actions {
  display: flex;
  gap: 12px;
  margin-top: 16px;
  padding-top: 16px;
  border-top: 1px solid #e8ecf0;

  button {
    flex: 1;
    padding: 10px 20px;
    border-radius: 10px;
    font-size: 14px;
    font-weight: 500;
    cursor: pointer;
    transition: all 0.2s ease;
  }

  .btn-primary {
    background: $accent-color;
    color: white;
    border: none;

    &:hover:not(:disabled) {
      background: $accent-dark;
    }

    &:disabled {
      background: #9ca3af;
      cursor: not-allowed;
    }
  }

  .btn-secondary {
    background: white;
    color: $text-primary;
    border: 1px solid #d1d5db;

    &:hover {
      background: #f3f4f6;
    }
  }
}

@keyframes spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@keyframes pulse {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.5;
  }
}
</style>
