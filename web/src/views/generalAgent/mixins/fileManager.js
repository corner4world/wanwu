/**
 * 文件管理 Mixin - 管理上传文件的逻辑
 */

export default {
  data() {
    return {
      uploadedFiles: [],
    };
  },

  methods: {
    /**
     * 处理文件上传完成
     */
    handleSetFileId(fileInfo) {
      if (fileInfo && fileInfo.length > 0) {
        fileInfo.forEach(file => {
          this.uploadedFiles.push({
            name: file.fileName,
            fileName: file.oldFileName,
            url: file.fileUrl,
            displayUrl: file.imgUrl,
            type: this.getFileTypeFromName(file.fileName),
            size: file.fileSize || 0,
            uploading: false,
            uploadProgress: 100,
          });
        });
      }
    },

    /**
     * 根据文件名判断文件类型
     */
    getFileTypeFromName(fileName) {
      const ext = fileName.split('.').pop().toLowerCase();
      const imageExts = ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'];
      if (imageExts.includes(ext)) {
        return 'image/' + ext;
      }
      return 'application/octet-stream';
    },

    /**
     * 移除文件
     */
    removeFile(index) {
      this.uploadedFiles.splice(index, 1);
    },

    /**
     * 清空所有文件
     */
    clearFiles() {
      this.uploadedFiles = [];
    },

    /**
     * 构建用户消息内容（包含文件）
     */
    buildUserMessage(content) {
      const message = { id: this.generateId(), role: 'user' };

      // 如果没有文件，直接返回文本
      if (this.uploadedFiles.length === 0) {
        message.content = content;
        return message;
      }

      // 有文件时，构建多部分内容
      const contentArray = [];

      // 添加文本内容（如果有）
      if (content && content.trim()) {
        contentArray.push({ type: 'text', text: content.trim() });
      }

      // 添加文件内容 - 后端统一使用 type: 'binary'，根据 mimeType 判断具体类型
      this.uploadedFiles.forEach(file => {
        contentArray.push({
          type: 'binary',
          mimeType: file.type || 'application/octet-stream',
          url: file.url, // 使用服务器返回的 HTTP URL
          fileName: file.fileName, // 服务器返回的文件名
        });
      });

      message.content = contentArray;
      return message;
    },

    /**
     * 根据已存在的用户消息构建请求消息（用于重新生成）
     */
    buildRequestMessage(userMessage) {
      const message = { id: this.generateId(), role: 'user' };

      // 如果没有文件，直接返回文本
      if (!userMessage.files || userMessage.files.length === 0) {
        message.content = userMessage.content;
        return message;
      }

      // 有文件时，构建多部分内容
      const contentArray = [];

      // 添加文本内容（如果有）
      if (userMessage.content && userMessage.content.trim()) {
        contentArray.push({ type: 'text', text: userMessage.content.trim() });
      }

      // 添加文件内容
      userMessage.files.forEach(file => {
        contentArray.push({
          type: 'binary',
          mimeType: file.type || 'application/octet-stream',
          url: file.url,
          fileName: file.name,
        });
      });

      message.content = contentArray;
      return message;
    },
  },
};
