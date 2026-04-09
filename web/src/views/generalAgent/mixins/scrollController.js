/**
 * 滚动控制 Mixin - 管理消息区域的滚动行为
 */

export default {
  data() {
    return {
      // 滚动控制
      userHasScrolled: false,
      showScrollToBottom: false,
      isAutoScrolling: false,
    };
  },

  methods: {
    /**
     * 滚动到底部
     */
    scrollToBottom(force = false) {
      if (!force && this.userHasScrolled) {
        this.showScrollToBottom = true;
        return;
      }
      this.isAutoScrolling = true;
      const container = this.$refs.messageArea;
      if (container) {
        container.scrollTop = container.scrollHeight;
      }
      setTimeout(() => {
        this.isAutoScrolling = false;
      }, 100);
    },

    /**
     * 处理消息区域滚动
     */
    handleMessageAreaScroll() {
      if (this.isAutoScrolling) return;

      const container = this.$refs.messageArea;
      if (!container) return;

      const { scrollTop, scrollHeight, clientHeight } = container;
      const distanceFromBottom = scrollHeight - scrollTop - clientHeight;
      const threshold = 150;

      const isNearBottom = distanceFromBottom < threshold;

      if (isNearBottom) {
        this.userHasScrolled = false;
        this.showScrollToBottom = false;
      } else {
        this.userHasScrolled = true;
        this.showScrollToBottom = true;
      }
    },

    /**
     * 点击滚动到底部按钮
     */
    handleScrollToBottomClick() {
      this.userHasScrolled = false;
      this.showScrollToBottom = false;
      this.scrollToBottom(true);
    },

    /**
     * 重置滚动状态
     */
    resetScrollState() {
      this.userHasScrolled = false;
      this.showScrollToBottom = false;
    },
  },
};
