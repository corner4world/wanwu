/**
 * 模式选择管理 Mixin - 管理深度研究、PPT等模式选择
 */

export default {
  data() {
    return {
      selectedModes: [],
      modeOptions: {
        research: {
          label: '深度研究',
          icon: 'el-icon-aim',
          value: 'research',
        },
        ppt: {
          label: '创建ppt',
          icon: 'el-icon-document',
          value: 'ppt',
        },
        excel: {
          label: '创建excel',
          icon: 'el-icon-s-grid',
          value: 'excel',
        },
        web: {
          label: '创建网页',
          icon: 'el-icon-monitor',
          value: 'web',
        },
      },
    };
  },

  methods: {
    /**
     * 添加模式
     */
    addMode(modeValue) {
      // 避免重复添加
      if (this.selectedModes.find(m => m.value === modeValue)) {
        return;
      }
      const mode = this.modeOptions[modeValue];
      if (mode) {
        this.selectedModes.push({ ...mode });
      }
    },

    /**
     * 移除模式
     */
    removeMode(modeValue) {
      this.selectedModes = this.selectedModes.filter(
        m => m.value !== modeValue,
      );
    },

    /**
     * 清空所有模式
     */
    clearModes() {
      this.selectedModes = [];
    },
  },
};
