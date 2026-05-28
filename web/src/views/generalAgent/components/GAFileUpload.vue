<!-- GeneralAgent 专用文件上传组件 - 支持混合类型多文件上传 -->
<template>
  <div class="ga-file-upload">
    <!-- 上传触发按钮 -->
    <slot :openDialog="openDialog">
      <el-button
        circle
        class="chat-upload-btn"
        icon="el-icon-circle-plus-outline"
        plain
        @click="openDialog"
      ></el-button>
    </slot>

    <el-dialog
      :before-close="handleClose"
      :visible.sync="dialogVisible"
      append-to-body
      custom-class="ga-upload-dialog"
      width="800px"
    >
      <div>
        <div class="dialog-body">
          <p class="upload-title">{{ $t('common.fileUpload.uploadFile') }}</p>
          <el-upload
            :accept="tipsArr"
            :auto-upload="false"
            :file-list="fileList"
            :limit="10"
            :on-change="uploadOnChange"
            :show-file-list="false"
            action=""
            class="upload-box"
            drag
            multiple
          >
            <div v-if="fileList.length > 0" class="file-preview-area">
              <div ref="fileListContainer" class="file-list-container">
                <el-button
                  v-show="canScroll"
                  circle
                  class="scroll-btn left"
                  icon="el-icon-arrow-left"
                  size="mini"
                  type="primary"
                  @click="prev($event)"
                ></el-button>
                <div
                  ref="fileItems"
                  :style="{ justifyContent: !canScroll ? 'center' : 'unset' }"
                  class="file-items"
                >
                  <div
                    v-for="(f, idx) in fileList"
                    :key="f.uid || idx"
                    class="file-item"
                  >
                    <!-- 删除按钮 -->
                    <div class="delete-btn" @click.stop="removeFile(idx)">
                      <i class="el-icon-close"></i>
                    </div>
                    <!-- 图片预览 -->
                    <img
                      v-if="isImageFile(f.name)"
                      :src="f.fileUrl || getFilePreviewUrl(f)"
                      class="file-preview"
                    />
                    <!-- 文档图标 -->
                    <div v-else class="doc-icon">
                      <img :src="require('@/assets/imgs/fileicon.png')" />
                    </div>
                    <!-- 文件信息 -->
                    <p class="file-info">
                      <el-tooltip
                        :content="f.name"
                        class="item"
                        effect="dark"
                        placement="top-start"
                      >
                        <span>
                          {{
                            f.name.length > 8
                              ? f.name.slice(0, 8) + '...'
                              : f.name
                          }}
                        </span>
                      </el-tooltip>
                      <span class="file-size">
                        {{ formatFileSize(f.size) }}
                      </span>
                    </p>
                    <!-- 上传进度 -->
                    <el-progress
                      v-if="f.percentage !== undefined && f.percentage < 100"
                      :percentage="f.percentage"
                      :status="f.progressStatus"
                      :stroke-width="4"
                      class="upload-progress"
                    ></el-progress>
                  </div>
                </div>
                <el-button
                  v-show="canScroll"
                  circle
                  class="scroll-btn right"
                  icon="el-icon-arrow-right"
                  size="mini"
                  type="primary"
                  @click="next($event)"
                ></el-button>
              </div>
              <div class="tips">
                <p>
                  最多上传10个文件，支持图片、文档混合上传
                  <span style="color: var(--color)">
                    {{ $t('common.fileUpload.click') }}
                  </span>
                  可继续添加文件
                </p>
              </div>
            </div>
            <div v-else>
              <i class="el-icon-upload"></i>
              <p>
                {{
                  $t('common.fileUpload.uploadText') +
                  $t('common.fileUpload.uploadClick')
                }}
              </p>
              <div class="tips">
                <p>
                  {{ $t('common.fileUpload.typeFileTip1') }}
                  <span>{{ tipsArr }}</span>
                  {{ $t('common.fileUpload.typeFileTip') }}
                </p>
              </div>
            </div>
          </el-upload>
        </div>
        <div class="dialog-footer">
          <el-button
            :disabled="fileList.length === 0 || !allFilesUploaded"
            type="primary"
            @click="doBatchUpload"
          >
            {{ $t('common.fileUpload.submitBtn') }}
          </el-button>
        </div>
      </div>
    </el-dialog>
  </div>
</template>

<script>
import uploadChunk from '@/mixins/uploadChunk';

export default {
  name: 'GAFileUpload',
  mixins: [uploadChunk],
  props: {
    fileTypeArr: {
      type: Array,
      default: () => ['doc/*', 'image/*'],
    },
    type: {
      type: String,
      default: 'wga',
    },
  },
  data() {
    return {
      canScroll: false,
      fileList: [],
      loading: false,
      dialogVisible: false,
      tipsArr: '',
      tipsObj: {
        'image/*': ['jpg', 'jpeg', 'png'],
        'audio/*': ['wav', 'mp3'],
        'doc/*': ['txt', 'csv', 'xlsx', 'docx', 'html', 'pptx', 'pdf', 'md'],
      },
    };
  },
  computed: {
    allFilesUploaded() {
      if (this.fileList.length === 0) return false;
      return this.fileList.every(
        file => file.percentage === 100 || file.uploaded,
      );
    },
  },
  watch: {
    fileTypeArr: {
      handler(val) {
        this.setFileType(val);
      },
      immediate: true,
    },
  },
  methods: {
    isImageFile(fileName) {
      const ext = fileName.split('.').pop().toLowerCase();
      return ['jpg', 'jpeg', 'png', 'gif', 'bmp', 'webp'].includes(ext);
    },

    formatFileSize(size) {
      if (size > 1024 * 1024) {
        return (size / (1024 * 1024)).toFixed(2) + ' MB';
      } else if (size > 1024) {
        return (size / 1024).toFixed(2) + ' KB';
      }
      return size + ' B';
    },

    getFilePreviewUrl(file) {
      if (file.raw) {
        return URL.createObjectURL(file.raw);
      }
      return '';
    },

    setFileType(fileTypeArr) {
      if (fileTypeArr.length) {
        this.tipsArr = '';
        let tips_arr = [];
        fileTypeArr.forEach(item => {
          const extensions = (this.tipsObj[item] || [item]).map(
            ext => '.' + ext,
          );
          tips_arr = tips_arr.concat(extensions);
        });
        this.tipsArr = tips_arr.join(', ');
      }
    },

    openDialog() {
      this.dialogVisible = true;
    },

    clearFile() {
      this.fileList = [];
      this.canScroll = false;
    },

    handleClose() {
      this.clearFile();
      this.dialogVisible = false;
    },

    checkScrollable() {
      this.$nextTick(() => {
        const container = this.$refs.fileItems;
        if (container) {
          this.canScroll = container.scrollWidth > container.clientWidth;
        }
      });
    },

    prev(e) {
      e.stopPropagation();
      this.$refs.fileItems.scrollBy({
        left: -200,
        behavior: 'smooth',
      });
    },

    next(e) {
      e.stopPropagation();
      this.$refs.fileItems.scrollBy({
        left: 200,
        behavior: 'smooth',
      });
    },

    removeFile(index) {
      const file = this.fileList[index];

      if (file.fileUrl && file.fileUrl.startsWith('blob:')) {
        URL.revokeObjectURL(file.fileUrl);
      }

      this.fileList.splice(index, 1);
      this.checkScrollable();
    },

    uploadOnChange(file, fileList) {
      const filename = file.name;

      const acceptedExtensions = this.tipsArr
        .split(',')
        .map(ext => ext.trim().toLowerCase());
      const isAccepted = acceptedExtensions.some(ext =>
        filename.toLowerCase().endsWith(ext),
      );

      if (!isAccepted) {
        this.$message.warning(
          this.$t('common.fileUpload.typeFileTip1') +
            this.tipsArr +
            this.$t('common.fileUpload.typeFileTip'),
        );
        const index = fileList.indexOf(file);
        if (index > -1) {
          fileList.splice(index, 1);
        }
        return;
      }

      if (fileList.length > 10) {
        this.$message.warning('最多上传10个文件');
        const index = fileList.indexOf(file);
        if (index > -1) {
          fileList.splice(index, 1);
        }
        return;
      }

      this.fileList = fileList;

      this.fileList.forEach((f, index) => {
        if (!f.fileUrl && f.raw) {
          this.$set(this.fileList, index, {
            ...f,
            fileUrl: URL.createObjectURL(f.raw),
          });
        }
      });

      this.checkScrollable();

      if (this.fileList.length > 0 && !this.loading) {
        this.autoUploadAllFiles();
      }
    },

    async autoUploadAllFiles() {
      this.loading = true;
      this.maxSizeBytes = 0;
      this.isExpire = true;

      for (let i = 0; i < this.fileList.length; i++) {
        const file = this.fileList[i];

        if (file.percentage === 100 || file.progressStatus === 'success')
          continue;

        await new Promise(resolve => {
          let isResolved = false;

          const checkComplete = setInterval(() => {
            if (isResolved) return;

            if (
              file.percentage === 100 ||
              file.progressStatus === 'success' ||
              file.progressStatus === 'exception'
            ) {
              isResolved = true;
              clearInterval(checkComplete);
              resolve();
            }
          }, 100);

          this.startUpload(i, this.type === 'webChat');
        });
      }

      this.loading = false;
    },
    doBatchUpload() {
      const hasFailed = this.fileList.some(
        file => file.progressStatus === 'exception',
      );

      if (hasFailed) {
        this.$message.warning('存在上传失败的文件，请删除后重新上传');
        return;
      }

      const hasUnuploaded = this.fileList.some(
        file => file.percentage !== 100 && file.progressStatus !== 'success',
      );

      if (hasUnuploaded) {
        this.$message.warning('请等待所有文件上传完成');
        return;
      }

      const fileInfo = this.fileList.map(file => ({
        fileName: file.name,
        oldFileName: file.name,
        fileSize: file.size,
        fileUrl: file.filePath || file.url,
        imgUrl: this.isImageFile(file.name) ? file.fileUrl : null,
      }));

      this.$emit('setFileId', fileInfo);
      this.$emit('setFile', this.fileList);
      this.clearFile();
      this.handleClose();
    },

    uploadFile(fileName, oldFileName, filePath) {
      const currentFile = this.fileList[this.fileIndex];
      if (currentFile) {
        this.$set(currentFile, 'filePath', filePath);
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.ga-upload-dialog {
  .dialog-body {
    padding: 0 20px;
    .upload-title {
      text-align: center;
      font-size: 18px;
      margin-bottom: 20px;
    }
    .upload-box {
      height: 190px;
      width: 100% !important;
      background-color: #fff;
      .el-upload-dragger {
        .el-icon-upload {
          margin: 46px 0 10px 0 !important;
          font-size: 32px !important;
          line-height: 36px !important;
          color: $color;
        }
        .el-upload__text {
          margin-top: -10px;
        }
      }
    }

    .file-preview-area {
      background-color: transparent !important;
      .file-list-container {
        width: 100%;
        position: relative;
        .scroll-btn {
          position: absolute;
          top: 50%;
          transform: translateY(-32px);
          &.left {
            left: 5px;
          }
          &.right {
            right: 5px;
          }
        }
        .file-items {
          display: flex;
          gap: 10px;
          width: 100%;
          overflow-x: auto;
          scroll-behavior: smooth;
          padding: 10px 0;
          .file-item {
            flex-shrink: 0;
            width: 120px;
            text-align: center;
            position: relative;
            .delete-btn {
              position: absolute;
              top: -6px;
              right: -6px;
              width: 18px;
              height: 18px;
              background: #ef4444;
              color: #fff;
              border-radius: 50%;
              display: flex;
              align-items: center;
              justify-content: center;
              cursor: pointer;
              font-size: 12px;
              transition: transform 0.2s;
              z-index: 10;

              &:hover {
                transform: scale(1.1);
              }
            }
            .file-preview {
              width: 80px;
              height: 80px;
              object-fit: cover;
              border-radius: 4px;
              margin: 0 auto 5px;
            }
            .doc-icon {
              width: 80px;
              height: 80px;
              margin: 0 auto 5px;
              img {
                width: 60px;
                height: 80px;
              }
            }
            .file-info {
              display: flex;
              flex-direction: column;
              gap: 2px;
              align-items: center;
              font-size: 12px;
              span {
                color: #666;
                max-width: 100%;
                overflow: hidden;
                text-overflow: ellipsis;
                white-space: nowrap;
              }
              .file-size {
                color: $color;
                font-size: 11px;
              }
            }
            .upload-progress {
              margin-top: 5px;
            }
          }
        }
      }
      .tips {
        position: absolute;
        bottom: 16px;
        left: 0;
        right: 0;
        p {
          color: #9d8d8d !important;
          text-align: center;
        }
      }
    }
  }
  .dialog-footer {
    text-align: center;
    margin: 30px 0 20px 0;
  }
}

.chat-upload-btn {
  padding: 8px;
  color: rgba(15, 21, 40, 0.82);
  border: none;
  &:hover {
    background-color: rgba(87, 104, 161, 0.08) !important;
    color: rgba(15, 21, 40, 0.82);
  }
  ::v-deep i {
    font-size: 16px;
  }
}
</style>
