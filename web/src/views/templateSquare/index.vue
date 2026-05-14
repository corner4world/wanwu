<template>
  <div class="templateSquare">
    <div
      class="page-wrapper page-wrapper-pr-none"
      :style="isPublic ? `background: ${bgColor}; height: 100%` : ''"
    >
      <!--<div class="page-title">
        <img
          class="page-title-img"
          :src="require('@/assets/imgs/template_square.svg')"
          alt=""
        />
        <span class="page-title-name">{{ $t('menu.templateSquare') }}</span>
      </div>-->
      <!-- tabs -->
      <div class="tabs tabs-spacing">
        <div
          v-for="item in tabList"
          :key="item.type"
          :class="['tab', { active: type === item.type }]"
          @click="tabClick(item.type)"
        >
          {{ item.name }}
        </div>
      </div>

      <TempSquare
        :isPublic="isPublic"
        :type="workflow"
        v-if="type === workflow"
      />
      <PromptTempSquare
        :isPublic="isPublic"
        :type="prompt"
        v-if="type === prompt"
      />
    </div>
  </div>
</template>
<script>
import TempSquare from './tempSquare.vue';
import PromptTempSquare from './prompt/promptTempSquare.vue';
import { WORKFLOW, PROMPT, SKILL } from './constants';

export default {
  components: { TempSquare, PromptTempSquare },
  data() {
    return {
      isPublic: true,
      bgColor:
        'linear-gradient(1deg, rgb(247, 252, 255) 50%, rgb(233, 246, 254) 98%)',
      workflow: WORKFLOW,
      prompt: PROMPT,
      type: '',
      tabList: [
        // { name: this.$t('tempSquare.workflow'), type: WORKFLOW },
        { name: this.$t('tempSquare.prompt'), type: PROMPT },
        // { name: 'Skills', type: SKILL },
      ],
    };
  },
  created() {
    this.isPublic = this.$route.path.includes('/public/');
    this.type = this.$route.query.type || PROMPT; // WORKFLOW
  },
  methods: {
    tabClick(type) {
      this.type = type;
      if (type === PROMPT) {
        this.$router.replace({ query: {} });
      } else {
        this.$router.replace({ query: { type } });
      }
    },
  },
};
</script>
<style lang="scss" scoped>
@import '@/style/tabs.scss';
.templateSquare {
  width: 100%;
  height: 100%;
}
</style>
