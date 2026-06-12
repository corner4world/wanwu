<template>
  <div class="mark__render">
    <div class="markdown-body" v-html="md.render(content || '')"></div>
  </div>
</template>
<script>
import { md } from '@/mixins/markdown-it';

export default {
  props: {
    content: '',
  },
  data() {
    return {
      md: md,
    };
  },
  watch: {
    content: {
      handler(val) {
        if (val) this.addCopy();
      },
    },
    deep: true,
    immediate: true,
  },
  mounted() {
    this.addCopy();
  },
  beforeDestroy() {
    this.clearTimer();
  },
  methods: {
    addCopy() {
      this.timer = setTimeout(() => {
        this.addCopyClick();
        this.clearTimer();
      }, 1000);
    },
    clearTimer() {
      if (this.timer) clearTimeout(this.timer);
    },
    addCopyClick() {
      let copyList = document.getElementsByClassName('mk-copy-btn') || [];
      for (let i = 0; i < copyList.length; i++) {
        copyList[i].addEventListener('click', e => {
          let innerText = e.target.parentNode.nextElementSibling.innerText;
          this.$copy(innerText);
          e.target.innerText = this.$t('common.copy.copySuccess');
          this.timer = setTimeout(() => {
            e.target.innerText = this.$t('common.button.copy');
            this.clearTimer();
          }, 1500);
        });
      }
    },
  },
};
</script>

<style lang="scss">
@import '@/style/markdown.scss';
.mark__render .markdown-body {
  background: rgba(255, 255, 255, 0);
  color: #333;
}
</style>
