<!--问答文件上传-->
<template>
  <div class="fileUpload">
    <!-- 上传触发按钮 -->
    <slot :openDialog="openDialog">
      <el-button
        class="chat-upload-btn"
        icon="el-icon-circle-plus-outline"
        circle
        plain
        @click="openDialog"
      ></el-button>
    </slot>

    <el-dialog
      custom-class="upload-dialog"
      :visible.sync="dialogVisible"
      width="800px"
      append-to-body
      :before-close="handleClose"
    >
      <div
        v-loading="loading"
        element-loading-background="rgba(255, 255, 255, 0.5)"
      >
        <div class="dialog-body">
          <p class="upload-title">{{ $t('common.fileUpload.uploadFile') }}</p>
          <el-upload
            :class="['upload-box']"
            drag
            action=""
            :show-file-list="false"
            :auto-upload="false"
            :limit="fileType === 'image/*' ? maxPicNum : 2"
            :accept="tipsArr"
            :file-list="fileList"
            :on-change="uploadOnChange"
            :on-exceed="uploadOnExceed"
          >
            <div v-if="fileUrl" class="echo-img-box">
              <div class="echo-img">
                <video
                  v-if="fileType === 'video/*'"
                  id="video"
                  muted
                  loop
                  playsinline
                >
                  <source :src="fileUrl" type="video/mp4" />
                  {{ $t('common.fileUpload.videoTips') }}
                </video>
                <audio v-if="fileType === 'audio/*'" id="audio" controls>
                  <source :src="fileUrl" type="video/mp3" />
                  <source :src="fileUrl" type="audio/ogg" />
                  <source :src="fileUrl" type="audio/mpeg" />
                  {{ $t('common.fileUpload.audioTips') }}
                </audio>
                <div v-if="fileType === 'doc/*'" class="docFile">
                  <img :src="require('@/assets/imgs/fileicon.png')" />
                </div>
                <div v-if="fileType === 'image/*'" class="type-img-container">
                  <el-button
                    v-show="canScroll"
                    icon="el-icon-arrow-left "
                    @click="prev($event)"
                    circle
                    class="scroll-btn left"
                    size="mini"
                    type="primary"
                  ></el-button>
                  <div
                    class="type-img"
                    ref="imgList"
                    :style="{ justifyContent: !canScroll ? 'center' : 'unset' }"
                  >
                    <div
                      v-for="(f, idx) in fileList"
                      :key="f.uid || idx"
                      class="type-img-item"
                    >
                      <img :src="f.fileUrl" />
                      <p class="type-img-info">
                        <el-tooltip
                          class="item"
                          effect="dark"
                          :content="f.name"
                          placement="top-start"
                        >
                          <span>
                            {{
                              f.name.length > 6
                                ? f.name.slice(0, 6) + '...'
                                : f.name
                            }}
                          </span>
                        </el-tooltip>
                        <span>
                          [
                          {{
                            f.size > 1024
                              ? (f.size / (1024 * 1024)).toFixed(2) + ' MB'
                              : f.size + ' bytes'
                          }}
                          ]
                        </span>
                      </p>
                    </div>
                  </div>
                  <el-button
                    v-show="canScroll"
                    icon="el-icon-arrow-right"
                    @click="next($event)"
                    circle
                    class="scroll-btn right"
                    size="mini"
                    type="primary"
                  ></el-button>
                </div>
                <div v-else>
                  <p>
                    {{ $t('knowledgeManage.fileName') }}:
                    {{ fileList[0]['name'] }}
                  </p>
                  <p>
                    {{ $t('knowledgeManage.fileSize') }}:
                    {{
                      fileList[0]['size'] > 1024
                        ? (fileList[0]['size'] / (1024 * 1024)).toFixed(2) +
                          ' MB'
                        : fileList[0]['size'] + ' bytes'
                    }}
                  </p>
                </div>
              </div>
              <div class="tips">
                <el-progress
                  :percentage="file.percentage"
                  v-if="file.percentage !== 100"
                  :status="file.progressStatus"
                  max="100"
                  style="width: 360px; margin: 0 auto"
                ></el-progress>
                <p
                  v-if="
                    fileTypeArr.length === 1 && fileTypeArr[0] === 'image/*'
                  "
                >
                  {{ $t('app.imgLimitOnly', { num: maxPicNum }) }}
                </p>
                <p v-else>
                  {{ $t('app.imgLimit', { num: maxPicNum }) }}
                  <span style="color: var(--color)">
                    {{ $t('common.fileUpload.click') }}
                  </span>
                  {{ $t('app.imgLimitTips') }}
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
                <p
                  v-if="type === 'agentChat'"
                  style="padding-top: 5px; color: #dc6803 !important"
                >
                  {{ $t('app.uploadModelTips') }}
                </p>
              </div>
            </div>
          </el-upload>
        </div>
        <div class="dialog-footer">
          <el-button
            type="primary"
            :disabled="!fileUrl || !(file && file.percentage === 100)"
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
import { mapGetters } from 'vuex';
import uploadChunk from '@/mixins/uploadChunk';
export default {
  props: {
    fileTypeArr: {
      type: Array,
      required: false,
      default: () => [],
    },
    type: { type: String },
    maxImageSize: {
      type: [Number, String],
      required: false,
      default: null,
    },
  },
  mixins: [uploadChunk],
  data() {
    return {
      canScroll: false,
      fileIdList: [],
      fileList: [],
      fileType: '',
      loading: false,
      dialogVisible: false,
      fileUrl: '',
      tipsArr: '',
      tipsObj: {
        'image/*': ['jpg', 'jpeg', 'png'],
        'audio/*': ['wav', 'mp3'],
        'doc/*': ['txt', 'csv', 'xlsx', 'docx', 'html', 'pptx', 'pdf'],
      },
      fileInfo: [],
      lastFileType: '',
      imgUrl: '',
    };
  },
  watch: {
    fileTypeArr: {
      handler(val, oldVal) {
        this.setFileType(val);
      },
      immediate: true,
    },
  },
  computed: {
    ...mapGetters('app', ['maxPicNum']),
    maxImageSizeMB() {
      const maxSize = Number(this.maxImageSize);
      return maxSize > 0 ? maxSize : 0;
    },
    maxImageSizeBytes() {
      return this.maxImageSizeMB ? this.maxImageSizeMB * 1024 * 1024 : 0;
    },
  },
  methods: {
    checkScrollable() {
      this.$nextTick(() => {
        const container = this.$refs.imgList;
        if (container) {
          this.canScroll = container.scrollWidth > container.clientWidth;
        }
      });
    },
    prev(e) {
      e.stopPropagation();
      this.$refs.imgList.scrollBy({
        left: -200,
        behavior: 'smooth',
      });
    },
    next(e) {
      e.stopPropagation();
      this.$refs.imgList.scrollBy({
        left: 200,
        behavior: 'smooth',
      });
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
      this.fileIdList = [];
      this.fileList = [];
      this.fileType = '';
      this.fileUrl = '';
      this.imgUrl = '';
      this.fileInfo = [];
      this.canScroll = false;
    },
    handleClose() {
      this.clearFile();
      this.dialogVisible = false;
    },
    uploadOnExceed(files) {
      if (this.fileType === 'image/*' && this.maxPicNum === 1) {
        const rawFile = files && files[0];
        if (!rawFile) return;
        const file = this.createUploadFile(rawFile);
        const validateResult = this.validateUploadFile(file);
        if (!validateResult.valid) {
          this.showUploadValidateMessage(validateResult);
          return;
        }

        this.resetReplacingFiles();
        this.uploadOnChange(file, [file]);
        return;
      }

      this.$message.warning(
        this.$t('app.uploadImgTips', { num: this.maxPicNum }),
      );
    },
    uploadOnChange(file, fileList) {
      const prevFileType = this.fileType;
      let filename = file.name;
      let fileType = filename.split('.')[filename.split('.').length - 1];
      let nextFileType = '';
      let nextImgUrl = '';

      const fileTypeLower = fileType.toLowerCase();

      if (this.tipsObj['image/*'].includes(fileTypeLower)) {
        nextFileType = 'image/*';
        if (file.url) {
          nextImgUrl = file.url;
        }
      }
      if (this.tipsObj['audio/*'].includes(fileTypeLower)) {
        nextFileType = 'audio/*';
      }
      if ([...this.tipsObj['doc/*'], 'md'].includes(fileTypeLower)) {
        nextFileType = 'doc/*';
      }

      // 格式拦截
      const validateResult = this.validateUploadFile(file, nextFileType);
      if (!validateResult.valid) {
        this.showUploadValidateMessage(validateResult);
        this.removeUploadFile(file, fileList);
        return;
      }

      if (nextFileType === 'image/*' && fileList.length > this.maxPicNum) {
        this.$message.warning(
          this.$t('app.uploadImgTips', { num: this.maxPicNum }),
        );
        this.removeUploadFile(file, fileList);
        return;
      }

      if (this.shouldResetFileInfo(nextFileType, prevFileType)) {
        this.resetReplacingFiles();
      }

      if (nextFileType === 'image/*' && !nextImgUrl && file.raw) {
        nextImgUrl = URL.createObjectURL(file.raw);
      }

      this.fileType = nextFileType;
      this.imgUrl = nextImgUrl;
      this.fileUrl = file.raw ? URL.createObjectURL(file.raw) : file.url;

      if (this.fileType === 'image/*') {
        // 图片类型可累加至maxPicNum个
        if (prevFileType && prevFileType !== this.fileType) {
          this.fileList = [];
          this.canScroll = false;
          this.fileList.push(file);
        } else {
          this.fileList = fileList;
        }
        const currentFileIndex = this.fileList.length - 1;
        if (file.raw) {
          this.fileList[currentFileIndex].fileUrl = URL.createObjectURL(
            file.raw,
          );
        }
        this.checkScrollable();
      } else {
        // 非图片类型只保留最新一个
        this.fileList = [];
        this.fileList.push(file);
      }

      if (this.fileList.length > 0) {
        this.maxSizeBytes = 0;
        this.isExpire = true;
        // 为每个文件启动上传，而不是只上传索引0的文件
        for (let i = 0; i < this.fileList.length; i++) {
          if (!this.fileList[i].uploaded) {
            // 添加标记避免重复上传
            this.startUpload(i, this.type === 'webChat');
            this.fileList[i].uploaded = true;
          }
        }
      }
    },
    removeUploadFile(file, fileList) {
      const index = fileList.indexOf(file);
      if (index > -1) {
        fileList.splice(index, 1);
      }
    },
    createUploadFile(rawFile) {
      const uid = rawFile.uid || this.$guid();
      rawFile.uid = uid;
      return {
        name: rawFile.name,
        size: rawFile.size,
        uid,
        raw: rawFile,
        percentage: 0,
        progressStatus: 'active',
      };
    },
    validateUploadFile(file, fileType) {
      const filename = (file && file.name) || '';
      const acceptedExtensions = this.tipsArr
        .split(',')
        .map(ext => ext.trim().toLowerCase())
        .filter(Boolean);
      const isAccepted = acceptedExtensions.some(ext =>
        filename.toLowerCase().endsWith(ext),
      );
      if (!isAccepted) {
        return { valid: false, type: 'fileType' };
      }

      const nextFileType = fileType || this.getFileType(filename);
      if (nextFileType === 'image/*' && this.isImageOverSize(file)) {
        return { valid: false, type: 'imageSize' };
      }

      return { valid: true, type: nextFileType };
    },
    showUploadValidateMessage(validateResult) {
      if (validateResult.type === 'imageSize') {
        this.$message.warning(
          this.$t('knowledgeManage.multiKnowledgeDatabase.imageSizeLimit', {
            maxSize: this.maxImageSizeMB,
          }),
        );
        return;
      }

      this.$message.warning(
        this.$t('common.fileUpload.typeFileTip1') +
          this.tipsArr +
          this.$t('common.fileUpload.typeFileTip'),
      );
    },
    getFileType(filename) {
      const fileTypeLower = (filename.split('.').pop() || '').toLowerCase();
      if (this.tipsObj['image/*'].includes(fileTypeLower)) return 'image/*';
      if (this.tipsObj['audio/*'].includes(fileTypeLower)) return 'audio/*';
      if ([...this.tipsObj['doc/*'], 'md'].includes(fileTypeLower)) {
        return 'doc/*';
      }
      return '';
    },
    shouldResetFileInfo(nextFileType, prevFileType) {
      const hasPreviousFile =
        (this.fileList && this.fileList.length) ||
        (this.fileInfo && this.fileInfo.length);
      if (!hasPreviousFile) return false;
      return nextFileType !== 'image/*' || prevFileType !== nextFileType;
    },
    resetReplacingFiles() {
      this.cancelAllRequests();
      this.fileList = [];
      this.fileType = '';
      this.fileUrl = '';
      this.imgUrl = '';
      this.fileInfo = [];
      this.fileIdList = [];
      this.lastFileType = '';
      this.canScroll = false;
    },
    isImageOverSize(file) {
      return (
        this.maxImageSizeBytes && file && file.size > this.maxImageSizeBytes
      );
    },
    uploadFile(fileName, oldFileName, fiePath) {
      //文件上传完之后
      if (this.lastFileType && this.lastFileType !== this.fileType) {
        this.fileInfo = [];
      }
      this.lastFileType = this.fileType;
      const fileInfoItem = {
        fileName,
        oldFileName,
        fileSize: this.fileList[this.fileIndex]['size'],
        fileUrl: fiePath,
      };
      // 如果是图片类型，添加 imgUrl 用于前端预览
      if (this.fileType === 'image/*' && this.imgUrl) {
        fileInfoItem.imgUrl = this.imgUrl;
      }
      this.fileInfo.push(fileInfoItem);
    },
    doBatchUpload() {
      this.$emit('setFileId', this.fileInfo);
      this.$emit('setFile', this.fileList);
      this.clearFile();
      this.handleClose();
    },
    getFileIdList() {
      return this.fileIdList;
    },
  },
};
</script>

<style lang="scss" scoped>
.upload-dialog {
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

    .echo-img-box {
      background-color: transparent !important;
      .echo-img {
        .type-img-container {
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
          .type-img {
            display: flex;
            gap: 10px;
            width: 100%;
            overflow-x: hidden;
            scroll-behavior: smooth;
            .type-img-item {
              width: auto !important;
              flex-shrink: 0;
              margin-bottom: 10px;
            }
            .type-img-info {
              display: flex;
              gap: 5px;
              justify-content: center;
              span {
                color: $color;
              }
            }
          }
        }
        img,
        video {
          width: auto;
          height: 80px;
          margin: 10px auto;
          border-radius: 4px;
          background-color: transparent;
        }
        audio {
          width: 300px;
          height: 54px;
          margin: 50px auto;
        }
      }
      .docFile {
        img {
          margin: 0;
          width: 60px;
          height: 100px;
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
