/**
 * 模式选择管理 Mixin - 管理深度研究、PPT等模式选择
 */

export default {
  data() {
    return {
      selectedModes: [],
      modeOptions: null, // 在 created 中初始化以支持 i18n
    };
  },

  created() {
    // 在 created 钩子中初始化 modeOptions，确保 $t 可用
    this.modeOptions = {
      research: {
        label: this.$t('generalAgent.modeManager.research'),
        icon: 'el-icon-aim',
        value: 'research',
        placeholder: this.$t('generalAgent.modeManager.researchPlaceholder'),
      },
      analysis: {
        label: this.$t('generalAgent.modeManager.analysis'),
        icon: 'el-icon-data-analysis',
        value: 'analysis',
        placeholder: this.$t('generalAgent.modeManager.analysisPlaceholder'),
      },
      ppt: {
        label: this.$t('generalAgent.modeManager.ppt'),
        icon: 'el-icon-document',
        value: 'ppt',
        placeholder: this.$t('generalAgent.modeManager.pptPlaceholder'),
      },
      excel: {
        label: this.$t('generalAgent.modeManager.excel'),
        icon: 'el-icon-s-grid',
        value: 'excel',
        placeholder: this.$t('generalAgent.modeManager.excelPlaceholder'),
      },
      web: {
        label: this.$t('generalAgent.modeManager.web'),
        icon: 'el-icon-monitor',
        value: 'web',
        placeholder: this.$t('generalAgent.modeManager.webPlaceholder'),
      },
      // video: {
      //   label: '创建视频',
      //   icon: 'el-icon-video-camera',
      //   value: 'video',
      // },
      // skill: {
      //   label: '创建skill',
      //   icon: 'el-icon-cpu',
      //   value: 'skill',
      // },
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
