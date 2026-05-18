/**
 * 模式选择管理 Mixin - 管理深度研究、PPT等模式选择
 */

import { getGeneralAgentSubList } from '@/api/generalAgent';
import { avatarSrc } from '@/utils/util';

export default {
  data() {
    return {
      selectedMode: null,
      modeOptions: {}, // 从接口获取的可选模式列表
    };
  },

  created() {
    // 初始化空对象，后续通过接口填充
    this.modeOptions = {};
  },

  watch: {
    'selectedMode.value': {
      handler(newVal) {
        if (newVal === 'Skill Chat Agent') {
          this.chatType = 'skill';
        } else {
          this.chatType = '';
        }
      },
      immediate: true,
    },
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
      const mode = this.modeOptions[modeValue];
      if (mode) {
        this.selectedMode = { ...mode };
      }
    },

    /**
     * 移除模式
     */
    removeMode(modeValue) {
      if (this.selectedMode?.value === modeValue) {
        this.selectedMode = null;
      }
    },

    /**
     * 清空模式
     */
    clearModes() {
      this.selectedMode = null;
    },
  },
};
