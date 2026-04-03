<template>
  <div class="workspace-panel">
    <div class="panel-header">
      <div class="header-left">
        <svg
          viewBox="0 0 24 24"
          width="18"
          height="18"
          fill="currentColor"
          class="header-icon"
        >
          <path
            d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"
          />
        </svg>
        <span class="header-title">工作空间</span>
      </div>
      <div class="header-actions">
        <el-tooltip content="刷新" placement="bottom">
          <button
            class="header-btn"
            @click="refreshCurrent"
            :disabled="loading"
          >
            <i :class="loading ? 'el-icon-loading' : 'el-icon-refresh'"></i>
          </button>
        </el-tooltip>
        <el-tooltip content="关闭" placement="bottom">
          <button class="header-btn" @click="$emit('close')">
            <i class="el-icon-close"></i>
          </button>
        </el-tooltip>
      </div>
    </div>

    <div class="panel-body">
      <!-- 加载状态 -->
      <div v-if="loading" class="loading-state">
        <i class="el-icon-loading"></i>
        <span>加载中...</span>
      </div>

      <!-- 空状态 -->
      <div v-else-if="!workspaceInfo.fileCount" class="empty-state">
        <svg
          viewBox="0 0 24 24"
          width="48"
          height="48"
          fill="currentColor"
          class="empty-icon"
        >
          <path
            d="M20 6h-8l-2-2H4c-1.1 0-1.99.9-1.99 2L2 18c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V8c0-1.1-.9-2-2-2zm0 12H4V8h16v10z"
            opacity="0.3"
          />
        </svg>
        <span class="empty-text">暂无文件</span>
        <span class="empty-hint">AI 生成的文件将显示在这里</span>
      </div>

      <!-- 文件树 -->
      <div v-else class="file-tree">
        <!-- 面包屑导航 -->
        <div class="breadcrumb">
          <span
            :class="['breadcrumb-item', { active: currentPath === '' }]"
            @click="navigateTo('')"
          >
            <i class="el-icon-folder-opened"></i>
            根目录
          </span>
          <template v-for="(part, index) in pathParts">
            <span :key="'sep-' + index" class="breadcrumb-sep">/</span>
            <span
              :key="'part-' + index"
              :class="[
                'breadcrumb-item',
                { active: index === pathParts.length - 1 },
              ]"
              @click="navigateTo(getPathByIndex(index))"
            >
              {{ part }}
            </span>
          </template>
        </div>

        <div class="tree-info">
          <span class="info-item">
            <i class="el-icon-document"></i>
            {{ workspaceInfo.fileCount }} 个文件
          </span>
          <span class="info-divider">|</span>
          <span class="info-item">
            <i class="el-icon-coin"></i>
            {{ formatSize(workspaceInfo.totalSize) }}
          </span>
        </div>

        <!-- 文件列表 -->
        <div class="file-list">
          <!-- 返回上一级 -->
          <div
            v-if="currentPath !== ''"
            class="file-item back-item"
            @click="navigateToParent"
          >
            <i class="el-icon-back"></i>
            <span class="file-name">..</span>
          </div>
          <div
            v-for="(file, index) in files"
            :key="index"
            :class="['file-item', { 'is-directory': isDirectory(file) }]"
          >
            <div class="file-item-main" @click="handleFileClick(file)">
              <i :class="getFileIcon(file)"></i>
              <span class="file-name">{{ file.name }}</span>
              <span v-if="!isDirectory(file)" class="file-size">
                {{ formatSize(file.size) }}
              </span>
            </div>
            <!-- 文件下载按钮 -->
            <button
              v-if="!isDirectory(file)"
              class="file-download-btn"
              @click.stop="downloadFile(file)"
              title="下载"
            >
              <i class="el-icon-download"></i>
            </button>
          </div>
        </div>

        <!-- 批量操作 -->
        <div class="tree-actions">
          <el-button size="small" plain @click="downloadAll">
            <i class="el-icon-download"></i>
            下载全部
          </el-button>
        </div>
      </div>
    </div>

    <!-- 文件预览抽屉 -->
    <el-drawer
      :visible.sync="previewVisible"
      :title="previewFile.name"
      direction="rtl"
      size="60%"
      :with-header="true"
      custom-class="preview-drawer"
      @close="closePreview"
    >
      <div class="preview-container">
        <!-- 加载中 -->
        <div v-if="previewLoading" class="preview-loading">
          <i class="el-icon-loading"></i>
          <span>加载中...</span>
        </div>

        <!-- 预览内容 -->
        <template v-else>
          <!-- 图片预览 -->
          <div v-if="previewType === 'image'" class="preview-image-wrapper">
            <img
              :src="previewUrl"
              class="preview-image"
              @error="handlePreviewError"
            />
          </div>

          <!-- 视频预览 -->
          <div
            v-else-if="previewType === 'video'"
            class="preview-video-wrapper"
          >
            <video :src="previewUrl" controls class="preview-video">
              您的浏览器不支持视频播放
            </video>
          </div>

          <!-- 音频预览 -->
          <div
            v-else-if="previewType === 'audio'"
            class="preview-audio-wrapper"
          >
            <div class="audio-cover">
              <i class="el-icon-headset"></i>
            </div>
            <audio :src="previewUrl" controls class="preview-audio">
              您的浏览器不支持音频播放
            </audio>
          </div>

          <!-- PDF 预览 -->
          <div v-else-if="previewType === 'pdf'" class="preview-pdf-wrapper">
            <iframe :src="previewUrl" class="preview-pdf"></iframe>
          </div>

          <!-- PPT 预览 -->
          <div v-else-if="previewType === 'ppt'" class="preview-ppt-wrapper">
            <ppt-preview
              :src="previewUrl"
              :file-name="previewFile.name"
              @download="downloadFile(previewFile)"
              @close="closePreview"
            />
          </div>

          <!-- HTML 预览 -->
          <div v-else-if="previewType === 'html'" class="preview-html-wrapper">
            <iframe
              :src="previewUrl"
              class="preview-html-frame"
              sandbox="allow-scripts allow-same-origin"
            ></iframe>
          </div>

          <!-- Markdown 预览 -->
          <div
            v-else-if="previewType === 'markdown'"
            class="preview-markdown-wrapper"
          >
            <markdown-renderer :content="previewContent" />
          </div>

          <!-- Office 文件预览 (Word/Excel 等) -->
          <div
            v-else-if="previewType === 'office'"
            class="preview-office-wrapper"
          >
            <div class="office-notice">
              <i class="el-icon-document"></i>
              <p>{{ previewFile.name }}</p>
              <p class="notice-text">此文件类型暂不支持在线预览</p>
              <el-button type="primary" @click="downloadFile(previewFile)">
                <i class="el-icon-download"></i>
                下载查看
              </el-button>
            </div>
          </div>

          <!-- 文本/代码预览 -->
          <div v-else-if="previewType === 'text'" class="preview-text-wrapper">
            <pre class="preview-text"><code>{{ previewContent }}</code></pre>
          </div>

          <!-- 不支持的格式 -->
          <div v-else class="preview-unsupported">
            <i class="el-icon-document"></i>
            <p class="file-name">{{ previewFile.name }}</p>
            <p class="notice-text">此文件类型暂不支持预览</p>
            <el-button type="primary" @click="downloadFile(previewFile)">
              <i class="el-icon-download"></i>
              下载文件
            </el-button>
          </div>
        </template>

        <!-- 预览工具栏 -->
        <div
          v-if="!previewLoading && previewType && previewType !== 'ppt'"
          class="preview-toolbar"
        >
          <el-button size="small" @click="downloadFile(previewFile)">
            <i class="el-icon-download"></i>
            下载
          </el-button>
          <el-button size="small" v-if="previewUrl" @click="openInNewTab">
            <i class="el-icon-link"></i>
            新窗口打开
          </el-button>
        </div>
      </div>
    </el-drawer>
  </div>
</template>

<script>
import {
  getGeneralAgentWorkspace,
  previewGeneralAgentWorkspace,
  downloadGeneralAgentWorkspace,
} from '@/api/generalAgent';
import PptPreview from './PptPreview.vue';
import MarkdownRenderer from './MarkdownRenderer.vue';

export default {
  name: 'WorkspacePanel',
  components: {
    PptPreview,
    MarkdownRenderer,
  },
  props: {
    threadId: {
      type: String,
      required: true,
    },
    runId: {
      type: String,
      default: '',
    },
    initialData: {
      type: Object,
      default: null,
    },
  },
  data() {
    return {
      loading: false,
      loadRequestId: 0, // 用于防止重复加载
      currentPath: '', // 当前目录路径
      rootFiles: [], // 根目录文件树（完整数据）
      files: [], // 当前显示的文件列表
      workspaceInfo: {
        fileCount: 0,
        totalSize: 0,
        isDisplay: false,
      },
      previewVisible: false,
      previewLoading: false,
      previewFile: {},
      previewUrl: '',
      previewContent: '',
      previewType: '', // image, video, audio, pdf, office, text, unsupported
      previewBlobUrl: '', // 用于清理
    };
  },
  computed: {
    // 面包屑路径部分
    pathParts() {
      if (!this.currentPath) return [];
      return this.currentPath.split('/').filter(p => p);
    },
  },
  watch: {
    runId: {
      immediate: true,
      handler(newVal, oldVal) {
        console.log('[WorkspacePanel] runId changed:', {
          new: newVal,
          old: oldVal,
          threadId: this.threadId,
        });
        this.tryLoadWorkspace();
      },
    },
    threadId: {
      immediate: true,
      handler(newVal, oldVal) {
        console.log('[WorkspacePanel] threadId changed:', {
          new: newVal,
          old: oldVal,
          runId: this.runId,
        });
        this.tryLoadWorkspace();
      },
    },
    initialData: {
      immediate: true,
      handler(newVal) {
        console.log('[WorkspacePanel] initialData changed:', newVal);
        if (newVal) {
          this.workspaceInfo = {
            fileCount: newVal.fileCount || 0,
            totalSize: newVal.totalSize || 0,
            isDisplay: newVal.isDisplay || false,
          };
        }
      },
    },
  },
  methods: {
    // 判断是否为目录
    isDirectory(file) {
      return file.type === 'directory' || file.type === 'dir';
    },

    // 统一的加载入口，防止重复加载
    tryLoadWorkspace() {
      if (!this.runId || !this.threadId) {
        console.log('[WorkspacePanel] Missing runId or threadId, skip loading');
        return;
      }
      // 使用请求ID防止重复加载
      const currentRequestId = ++this.loadRequestId;
      console.log(
        '[WorkspacePanel] tryLoadWorkspace requestId:',
        currentRequestId,
      );

      // 延迟执行，让两个 watch 合并为一次调用
      setTimeout(() => {
        if (currentRequestId === this.loadRequestId) {
          this.loadWorkspace();
        } else {
          console.log(
            '[WorkspacePanel] skipped duplicate load, current:',
            currentRequestId,
            'latest:',
            this.loadRequestId,
          );
        }
      }, 50);
    },

    // 刷新当前目录
    refreshCurrent() {
      this.loadWorkspace();
    },

    // 导航到指定路径（本地展开，不调用 API）
    navigateTo(path) {
      this.currentPath = path || '';
      // 从根文件树中获取指定路径的文件
      this.files = this.getFilesAtPath(this.currentPath);
      // 处理并排序
      this.files = this.processFiles(this.files);
      console.log(
        '[WorkspacePanel] navigateTo:',
        this.currentPath,
        'files:',
        this.files.length,
      );
    },

    // 返回上一级
    navigateToParent() {
      const parts = this.currentPath.split('/').filter(p => p);
      parts.pop();
      this.navigateTo(parts.join('/'));
    },

    // 根据索引获取路径
    getPathByIndex(index) {
      return this.pathParts.slice(0, index + 1).join('/');
    },

    // 根据路径获取文件列表（从缓存的文件树）
    getFilesAtPath(path) {
      if (!path) {
        // 根目录
        return this.rootFiles || [];
      }
      const parts = path.split('/').filter(p => p);
      let current = this.rootFiles || [];
      for (const part of parts) {
        const dir = current.find(f => f.name === part && this.isDirectory(f));
        if (dir && dir.children) {
          current = dir.children;
        } else {
          return [];
        }
      }
      return current;
    },

    async loadWorkspace() {
      console.log('[WorkspacePanel] loadWorkspace called', {
        runId: this.runId,
        threadId: this.threadId,
      });
      if (!this.runId || !this.threadId) {
        console.log('[WorkspacePanel] Missing runId or threadId');
        this.loading = false;
        return;
      }

      this.loading = true;
      this.currentPath = ''; // 重置到根目录
      console.log('[WorkspacePanel] loading set to true');
      try {
        const params = {
          threadId: this.threadId,
          runId: this.runId,
        };
        console.log('[WorkspacePanel] calling API with params:', params);
        const res = await getGeneralAgentWorkspace(params);
        console.log('[WorkspacePanel] API response:', res);
        if (res.code === 0 && res.data) {
          this.workspaceInfo = {
            fileCount: res.data.fileCount || 0,
            totalSize: res.data.totalSize || 0,
            isDisplay: res.data.isDisplay || false,
          };
          // 保存完整的文件树
          this.rootFiles = res.data.files || [];
          // 显示根目录文件
          this.files = this.processFiles(this.rootFiles);
          console.log(
            '[WorkspacePanel] loaded workspace:',
            this.workspaceInfo,
            'rootFiles:',
            this.rootFiles.length,
            'displayFiles:',
            this.files.length,
          );
        } else if (res.code !== 0) {
          console.error('[WorkspacePanel] API error:', res.msg);
          this.$message.error(res.msg || '加载工作空间失败');
          // 显示空状态
          this.workspaceInfo = {
            fileCount: 0,
            totalSize: 0,
            isDisplay: false,
          };
          this.rootFiles = [];
          this.files = [];
        }
      } catch (error) {
        console.error('[WorkspacePanel] 加载工作空间失败:', error);
        this.$message.error('加载工作空间失败，请稍后重试');
        // 显示空状态
        this.workspaceInfo = {
          fileCount: 0,
          totalSize: 0,
          isDisplay: false,
        };
        this.rootFiles = [];
        this.files = [];
      } finally {
        this.loading = false;
        console.log('[WorkspacePanel] loading set to false');
      }
    },

    // 处理文件列表，添加排序
    processFiles(files) {
      if (!files || !Array.isArray(files)) return [];
      // 排序：文件夹在前，然后按名称排序
      const sorted = [...files].sort((a, b) => {
        const aIsDir = this.isDirectory(a);
        const bIsDir = this.isDirectory(b);
        if (aIsDir && !bIsDir) return -1;
        if (!aIsDir && bIsDir) return 1;
        return (a.name || '').localeCompare(b.name || '');
      });
      return sorted;
    },

    async handleFileClick(file) {
      console.log(
        '[WorkspacePanel] handleFileClick:',
        file,
        'isDirectory:',
        this.isDirectory(file),
      );
      // 如果是文件夹，导航进入（本地展开，不调用 API）
      if (this.isDirectory(file)) {
        const newPath = this.currentPath
          ? `${this.currentPath}/${file.name}`
          : file.name;
        console.log('[WorkspacePanel] navigating to:', newPath);
        // 从 children 获取子文件
        if (file.children && file.children.length > 0) {
          this.currentPath = newPath;
          this.files = this.processFiles(file.children);
          console.log('[WorkspacePanel] loaded children:', this.files.length);
        } else {
          // 没有 children，显示空
          this.currentPath = newPath;
          this.files = [];
        }
        return;
      }

      // 文件预览
      this.previewFile = file;
      this.previewLoading = true;
      this.previewVisible = true;
      this.previewUrl = '';
      this.previewContent = '';
      this.previewType = '';
      this.previewBlobUrl = '';

      try {
        // 构建完整文件路径
        const filePath = this.currentPath
          ? `${this.currentPath}/${file.name}`
          : file.name;
        const blob = await previewGeneralAgentWorkspace({
          threadId: this.threadId,
          runId: this.runId,
          path: filePath,
        });

        // 根据文件类型设置预览方式
        this.previewType = this.getPreviewType(file);

        if (this.previewType === 'image') {
          this.previewBlobUrl = URL.createObjectURL(blob);
          this.previewUrl = this.previewBlobUrl;
        } else if (
          this.previewType === 'video' ||
          this.previewType === 'audio'
        ) {
          this.previewBlobUrl = URL.createObjectURL(blob);
          this.previewUrl = this.previewBlobUrl;
        } else if (this.previewType === 'pdf') {
          this.previewBlobUrl = URL.createObjectURL(blob);
          this.previewUrl = this.previewBlobUrl;
        } else if (this.previewType === 'ppt') {
          // PPT 预览 - 保存 blob 对象
          this.previewUrl = blob;
          this.previewBlobUrl = blob; // 保存 blob 引用，用于清理
        } else if (this.previewType === 'html') {
          // HTML 预览 - 创建 blob URL
          this.previewBlobUrl = URL.createObjectURL(blob);
          this.previewUrl = this.previewBlobUrl;
        } else if (this.previewType === 'markdown') {
          // Markdown 预览 - 读取文本内容
          this.previewContent = await blob.text();
        } else if (this.previewType === 'text') {
          this.previewContent = await blob.text();
        }
        // office 和 unsupported 不需要加载内容
      } catch (error) {
        console.error('预览文件失败:', error);
        this.$message.error('预览文件失败');
        this.previewType = 'unsupported';
      } finally {
        this.previewLoading = false;
      }
    },

    closePreview() {
      // 清理 blob URL
      if (this.previewBlobUrl) {
        URL.revokeObjectURL(this.previewBlobUrl);
        this.previewBlobUrl = '';
      }
    },

    getPreviewType(file) {
      if (!file || !file.name) return 'unsupported';
      const ext = file.name.split('.').pop().toLowerCase();

      // 图片
      if (
        ['png', 'jpg', 'jpeg', 'gif', 'svg', 'webp', 'bmp', 'ico'].includes(ext)
      ) {
        return 'image';
      }
      // 视频
      if (['mp4', 'webm', 'ogg', 'mov', 'm4v', 'avi', 'mkv'].includes(ext)) {
        return 'video';
      }
      // 音频
      if (['mp3', 'wav', 'ogg', 'm4a', 'flac', 'aac', 'wma'].includes(ext)) {
        return 'audio';
      }
      // PDF
      if (ext === 'pdf') {
        return 'pdf';
      }
      // PPT 文件（使用 vue-office 预览）
      if (['ppt', 'pptx'].includes(ext)) {
        return 'ppt';
      }
      // Office 文件 (Word/Excel 等)
      if (['doc', 'docx', 'xls', 'xlsx'].includes(ext)) {
        return 'office';
      }
      // HTML 文件
      if (['html', 'htm'].includes(ext)) {
        return 'html';
      }
      // Markdown 文件
      if (ext === 'md') {
        return 'markdown';
      }
      // 文本/代码
      if (
        [
          'txt',
          'json',
          'js',
          'ts',
          'jsx',
          'tsx',
          'vue',
          'py',
          'java',
          'go',
          'rs',
          'c',
          'cpp',
          'h',
          'hpp',
          'cs',
          'rb',
          'php',
          'swift',
          'kt',
          'scala',
          'css',
          'scss',
          'sass',
          'less',
          'xml',
          'yaml',
          'yml',
          'toml',
          'ini',
          'conf',
          'cfg',
          'sh',
          'bash',
          'zsh',
          'bat',
          'sql',
          'dockerfile',
          'makefile',
          'r',
          'm',
          'lua',
          'pl',
          'pm',
        ].includes(ext)
      ) {
        return 'text';
      }

      return 'unsupported';
    },

    async downloadFile(file) {
      try {
        // 构建完整文件路径
        const filePath = this.currentPath
          ? `${this.currentPath}/${file.name}`
          : file.name;
        const blob = await downloadGeneralAgentWorkspace({
          threadId: this.threadId,
          runId: this.runId,
          path: filePath,
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        a.download = file.name;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        this.$message.success('下载成功');
      } catch (error) {
        console.error('下载文件失败:', error);
        this.$message.error('下载文件失败');
      }
    },

    async downloadAll() {
      this.$message.info('正在打包下载...');
      try {
        const blob = await downloadGeneralAgentWorkspace({
          threadId: this.threadId,
          runId: this.runId,
          path: this.currentPath || '',
        });
        const url = URL.createObjectURL(blob);
        const a = document.createElement('a');
        a.href = url;
        const filename = this.currentPath
          ? `${this.currentPath.replace(/\//g, '-')}.zip`
          : `workspace-${this.runId}.zip`;
        a.download = filename;
        document.body.appendChild(a);
        a.click();
        document.body.removeChild(a);
        URL.revokeObjectURL(url);
        this.$message.success('下载成功');
      } catch (error) {
        console.error('下载失败:', error);
        this.$message.error('下载失败');
      }
    },

    openInNewTab() {
      if (this.previewUrl) {
        window.open(this.previewUrl, '_blank');
      }
    },

    handlePreviewError() {
      this.$message.error('文件加载失败');
    },

    getFileIcon(file) {
      if (!file || !file.name) return 'el-icon-document';
      // 文件夹图标
      if (file.type === 'directory' || file.isDir) {
        return 'el-icon-folder';
      }
      const ext = file.name.split('.').pop().toLowerCase();
      const iconMap = {
        // 文档
        pdf: 'el-icon-document',
        doc: 'el-icon-document',
        docx: 'el-icon-document',
        txt: 'el-icon-document',
        md: 'el-icon-document',
        // 表格
        xls: 'el-icon-s-grid',
        xlsx: 'el-icon-s-grid',
        csv: 'el-icon-s-grid',
        // 演示文稿
        ppt: 'el-icon-data-board',
        pptx: 'el-icon-data-board',
        // 代码
        json: 'el-icon-document',
        js: 'el-icon-document',
        ts: 'el-icon-document',
        vue: 'el-icon-document',
        py: 'el-icon-document',
        java: 'el-icon-document',
        go: 'el-icon-document',
        html: 'el-icon-document',
        css: 'el-icon-document',
        xml: 'el-icon-document',
        yaml: 'el-icon-document',
        yml: 'el-icon-document',
        sql: 'el-icon-document',
        sh: 'el-icon-document',
        // 图片
        png: 'el-icon-picture',
        jpg: 'el-icon-picture',
        jpeg: 'el-icon-picture',
        gif: 'el-icon-picture',
        svg: 'el-icon-picture',
        webp: 'el-icon-picture',
        bmp: 'el-icon-picture',
        ico: 'el-icon-picture',
        // 视频
        mp4: 'el-icon-video-camera',
        webm: 'el-icon-video-camera',
        mov: 'el-icon-video-camera',
        avi: 'el-icon-video-camera',
        mkv: 'el-icon-video-camera',
        // 音频
        mp3: 'el-icon-headset',
        wav: 'el-icon-headset',
        flac: 'el-icon-headset',
        aac: 'el-icon-headset',
        // 压缩包
        zip: 'el-icon-files',
        rar: 'el-icon-files',
        tar: 'el-icon-files',
        gz: 'el-icon-files',
        '7z': 'el-icon-files',
      };
      return iconMap[ext] || 'el-icon-document';
    },

    formatSize(bytes) {
      if (!bytes) return '0 B';
      const units = ['B', 'KB', 'MB', 'GB'];
      let size = bytes;
      let unitIndex = 0;
      while (size >= 1024 && unitIndex < units.length - 1) {
        size /= 1024;
        unitIndex++;
      }
      return `${size.toFixed(1)} ${units[unitIndex]}`;
    },
  },
};
</script>

<style lang="scss" scoped>
$workspace-color: #3b82f6;
$workspace-light: #60a5fa;
$workspace-bg: #f8fafc;
$border-color: #e2e8f0;

.workspace-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  background: #fff;
  border-left: 1px solid $border-color;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid $border-color;
  background: $workspace-bg;
}

.header-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.header-icon {
  color: $workspace-color;
}

.header-title {
  font-size: 14px;
  font-weight: 600;
  color: #1f2937;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.header-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  background: transparent;
  border-radius: 6px;
  cursor: pointer;
  color: #6b7280;
  transition: all 0.2s;

  &:hover:not(:disabled) {
    background: rgba($workspace-color, 0.1);
    color: $workspace-color;
  }

  &:disabled {
    cursor: not-allowed;
    opacity: 0.5;
  }

  i {
    font-size: 14px;
  }
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #9ca3af;

  i {
    font-size: 32px;
    margin-bottom: 8px;
  }

  .empty-icon {
    color: #d1d5db;
    margin-bottom: 12px;
  }

  .empty-text {
    font-size: 14px;
    color: #6b7280;
    margin-bottom: 4px;
  }

  .empty-hint {
    font-size: 12px;
    color: #9ca3af;
  }
}

.file-tree {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.breadcrumb {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 4px;
  padding: 8px 12px;
  background: $workspace-bg;
  border-radius: 8px;
  font-size: 13px;
}

.breadcrumb-item {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #4b5563;
  cursor: pointer;
  padding: 2px 6px;
  border-radius: 4px;
  transition: all 0.2s;

  i {
    font-size: 14px;
    color: $workspace-color;
  }

  &:hover:not(.active) {
    background: rgba($workspace-color, 0.1);
    color: $workspace-color;
  }

  &.active {
    color: $workspace-color;
    font-weight: 500;
    cursor: default;
  }
}

.breadcrumb-sep {
  color: #9ca3af;
  user-select: none;
}

.tree-info {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
  background: $workspace-bg;
  border-radius: 8px;
  font-size: 13px;
}

.info-item {
  display: flex;
  align-items: center;
  gap: 4px;
  color: #4b5563;

  i {
    font-size: 14px;
    color: $workspace-color;
  }
}

.info-divider {
  color: #e5e7eb;
}

.file-list {
  display: flex;
  flex-direction: column;
  gap: 4px;
  max-height: calc(100vh - 300px);
  overflow-y: auto;
}

.file-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-radius: 6px;
  cursor: pointer;
  transition: background 0.2s;

  &:hover {
    background: $workspace-bg;

    .file-download-btn {
      opacity: 1;
    }
  }

  .file-item-main {
    display: flex;
    align-items: center;
    gap: 8px;
    flex: 1;
    min-width: 0;
  }

  .file-name {
    flex: 1;
    font-size: 13px;
    color: #374151;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .file-size {
    font-size: 11px;
    color: #9ca3af;
    font-variant-numeric: tabular-nums;
    margin-left: auto;
  }

  .file-download-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border: none;
    background: rgba($workspace-color, 0.1);
    border-radius: 6px;
    cursor: pointer;
    color: $workspace-color;
    opacity: 0;
    transition: all 0.2s;
    flex-shrink: 0;

    i {
      font-size: 14px;
    }

    &:hover {
      background: rgba($workspace-color, 0.2);
    }
  }

  // 文件夹样式
  &.is-directory {
    i:first-child {
      color: #f59e0b;
    }

    .file-name {
      font-weight: 500;
    }
  }

  // 返回按钮样式
  &.back-item {
    i:first-child {
      color: #6b7280;
    }

    .file-name {
      color: #6b7280;
    }
  }
}

.tree-actions {
  display: flex;
  justify-content: center;
  padding-top: 12px;
  border-top: 1px solid $border-color;
  margin-top: 8px;

  .el-button {
    display: flex;
    align-items: center;
    gap: 4px;

    i {
      font-size: 14px;
    }
  }
}
</style>

<style lang="scss">
/* 预览抽屉样式 - 非 scoped */
.preview-drawer {
  .el-drawer__header {
    margin-bottom: 0;
    padding: 16px 20px;
    border-bottom: 1px solid #e5e7eb;
    font-size: 16px;
    font-weight: 600;
    color: #1f2937;

    > :first-child {
      outline: none;
    }
  }

  .el-drawer__body {
    padding: 0;
    display: flex;
    flex-direction: column;
    height: calc(100% - 53px);
  }
}

.preview-container {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  height: 100%;
}

.preview-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 12px;
  color: #6b7280;

  i {
    font-size: 40px;
  }
}

.preview-image-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: #f3f4f6;
  overflow: auto;
}

.preview-image {
  max-width: 100%;
  max-height: 100%;
  object-fit: contain;
  border-radius: 8px;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.1);
}

.preview-video-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: #000;
}

.preview-video {
  max-width: 100%;
  max-height: 100%;
  border-radius: 8px;
}

.preview-audio-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 24px;
  padding: 40px;
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);

  .audio-cover {
    width: 120px;
    height: 120px;
    background: rgba(255, 255, 255, 0.2);
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;

    i {
      font-size: 48px;
      color: #fff;
    }
  }
}

.preview-audio {
  width: 100%;
  max-width: 400px;
}

.preview-pdf-wrapper {
  flex: 1;
  overflow: hidden;
}

.preview-pdf {
  width: 100%;
  height: 100%;
  border: none;
}

.preview-ppt-wrapper {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  background: #fff;
  min-height: 600px;
  height: 100%;

  .ppt-preview-container {
    flex: 1;
    min-height: 600px;
  }
}

.preview-html-wrapper {
  flex: 1;
  overflow: hidden;
  background: #fff;
}

.preview-html-frame {
  width: 100%;
  height: 100%;
  border: none;
  background: #fff;
}

.preview-markdown-wrapper {
  flex: 1;
  overflow: auto;
  padding: 24px;
  background: #fff;
}

.preview-office-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px;

  .office-notice {
    text-align: center;
    color: #6b7280;

    i {
      font-size: 64px;
      color: #d1d5db;
      margin-bottom: 16px;
    }

    p {
      margin-bottom: 8px;
    }

    .file-name {
      font-size: 16px;
      font-weight: 500;
      color: #374151;
    }

    .notice-text {
      color: #9ca3af;
      margin-bottom: 20px;
    }
  }
}

.preview-text-wrapper {
  flex: 1;
  overflow: auto;
  padding: 20px;
  background: #1e1e1e;
}

.preview-text {
  margin: 0;
  font-family:
    'JetBrains Mono', 'SF Mono', Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 13px;
  line-height: 1.6;
  color: #d4d4d4;
  white-space: pre-wrap;
  word-break: break-all;

  code {
    font-family: inherit;
    background: transparent;
  }
}

.preview-unsupported {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
  text-align: center;
  color: #6b7280;

  i {
    font-size: 64px;
    color: #d1d5db;
    margin-bottom: 16px;
  }

  .file-name {
    font-size: 16px;
    font-weight: 500;
    color: #374151;
    margin-bottom: 8px;
  }

  .notice-text {
    color: #9ca3af;
    margin-bottom: 20px;
  }
}

.preview-toolbar {
  display: flex;
  justify-content: center;
  gap: 12px;
  padding: 16px;
  border-top: 1px solid #e5e7eb;
  background: #fafafa;

  .el-button {
    display: flex;
    align-items: center;
    gap: 4px;

    i {
      font-size: 14px;
    }
  }
}
</style>
