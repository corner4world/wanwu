/**
 * 模式选择管理 Mixin - 管理深度研究、PPT等模式选择
 */

import { getGeneralAgentSubList } from '@/api/generalAgent';
import { avatarSrc } from '@/utils/util';

export default {
  data() {
    return {
      selectedModes: [],
      modeOptions: {}, // 从接口获取的可选模式列表
    };
  },

  created() {
    // 初始化空对象，后续通过接口填充
    this.modeOptions = {};
  },

  methods: {
    /**
     * 获取可选模式列表
     */
    async fetchModeOptions() {
      const res = await getGeneralAgentSubList();
      if (res.code === 0 && res.data?.wgaAgentList) {
        // 将接口返回的数据转换为 modeOptions 格式
        this.modeOptions = res.data.wgaAgentList.reduce((acc, agent) => {
          acc[agent.agentId] = {
            label: agent.agentName,
            value: agent.agentId,
            placeholder: agent.placeholder,
            avatar: avatarSrc(agent.avatar?.path),
          };
          return acc;
        }, {});
      }
    },

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
