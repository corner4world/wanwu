/**
 * 消息管理 Mixin - 管理多会话的消息列表
 */

export default {
  data() {
    return {
      // 每个会话独立的消息列表 { threadId: messageList }
      messagesMap: {},
    };
  },

  computed: {
    // 当前会话的消息列表
    messageList: {
      get() {
        return this.messagesMap[this.currentThreadId] || [];
      },
      set(val) {
        this.$set(this.messagesMap, this.currentThreadId, val);
      },
    },
  },

  methods: {
    /**
     * 确保会话的消息列表存在
     */
    ensureMessageList(threadId) {
      if (!this.messagesMap[threadId]) {
        this.$set(this.messagesMap, threadId, []);
      }
      return this.messagesMap[threadId];
    },

    /**
     * 添加用户消息到指定会话
     */
    addUserMessage(threadId, content, files = []) {
      const messages = this.ensureMessageList(threadId);
      const userMessage = {
        id: this.generateId(),
        role: 'user',
        content: content,
        files: [...files],
      };
      messages.push(userMessage);
      return userMessage;
    },

    /**
     * 添加助手消息到指定会话
     */
    addAssistantMessage(threadId, assistantMessage) {
      const messages = this.ensureMessageList(threadId);
      messages.push(assistantMessage);
      return assistantMessage;
    },

    /**
     * 删除指定消息
     */
    removeMessage(threadId, messageId) {
      const messages = this.messagesMap[threadId];
      if (!messages) return;

      const index = messages.findIndex(m => m.id === messageId);
      if (index !== -1) {
        messages.splice(index, 1);
      }
    },

    /**
     * 清空指定会话的消息
     */
    clearMessages(threadId) {
      this.$set(this.messagesMap, threadId, []);
    },

    /**
     * 迁移消息（用于新建对话时从临时会话迁移到正式会话）
     */
    migrateMessages(fromThreadId, toThreadId) {
      const fromMessages = this.messagesMap[fromThreadId] || [];
      this.$set(this.messagesMap, toThreadId, fromMessages);
      this.$delete(this.messagesMap, fromThreadId);
    },

    /**
     * 获取工具结果
     */
    getToolResultsForMessage(message) {
      if (message.toolResults && message.toolResults.length > 0) {
        return message.toolResults;
      }
      return [];
    },
  },
};
