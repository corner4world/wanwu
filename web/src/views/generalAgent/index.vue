<template>
  <div class="wp">
    <iframe id="inlineFrameExample" :src="wgaUrl" allow="clipboard-read; clipboard-write"></iframe>
  </div>
</template>

<script>
export default {
  data() {
    return {
      wgaUrl:
        process.env.NODE_ENV === 'development'
          ? `http://localhost:3000/wga`
          : window.location.origin + this.$basePath + `/wga`,
    };
  },
};
</script>

<style lang="scss" scoped>
.wp {
  width: 100%;
  height: calc(100vh - 40px);
  iframe {
    border-radius: 10px;
    width: 100%;
    height: 100%;
    border: none;
  }
}
</style>
