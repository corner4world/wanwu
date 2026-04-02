<template>
  <span class="typing-cursor"></span>
</template>

<script>
export default {
  name: 'TypingCursor',
};
</script>

<style lang="scss" scoped>
.typing-cursor {
  display: inline-block;
  width: 2px;
  height: 1.2em;
  background: #10a37f;
  margin-left: 2px;
  vertical-align: text-bottom;
  animation: blink 1s infinite;
}

@keyframes blink {
  0%,
  45% {
    opacity: 1;
  }
  50%,
  95% {
    opacity: 0;
  }
  100% {
    opacity: 1;
  }
}
</style>
