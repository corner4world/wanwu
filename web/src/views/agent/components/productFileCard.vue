<template>
  <div class="productFileCard">
    <div class="card-title">
      <FileIcon :type="info.fileType" size="30px" />
      <div class="card-info">
        <span class="card-name">{{ info.name }}</span>
      </div>
    </div>
    <div v-if="info.metadata.desc" class="card-des">
      {{ info.metadata.desc }}
    </div>
    <div class="card-footer">
      <div class="card-footer-left">
        <span v-if="this.info.size >= 0" class="card-size">{{ fileSize }}</span>
      </div>
      <div class="card-footer-right">
        <el-tooltip :content="$t('tempSquare.download')" placement="top">
          <i class="el-icon-download" @click.stop="handleDownload"></i>
        </el-tooltip>
      </div>
    </div>
  </div>
</template>

<script>
import FileIcon from '@/components/FileIcon.vue';
import { filterSize, directDownload } from '@/utils/util';

export default {
  name: 'ProductFileCard',
  components: { FileIcon },
  props: {
    info: {
      type: Object,
      default: () => ({}),
    },
  },
  computed: {
    fileSize() {
      return this.info.size === 0 ? '0 KB' : filterSize(this.info.size);
    },
  },
  methods: {
    handleDownload() {
      const { fileUrl } = this.info;
      directDownload(fileUrl);
    },
  },
};
</script>

<style lang="scss" scoped>
.productFileCard {
  position: relative;
  min-width: 240px;
  padding: 20px 16px;
  border-radius: 12px;
  background: #fff url('@/assets/imgs/card_bg.png');
  background-size: 100% 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  width: calc((100% / 4) - 20px);
  box-shadow: 0 1px 4px 0 rgba(0, 0, 0, 0.15);
  border: 1px solid rgba(0, 0, 0, 0);
  &:hover {
    cursor: pointer;
    box-shadow:
      0 2px 8px #171a220d,
      0 4px 16px #0000000f;
    border: 1px solid $border_color;

    .action-icon {
      display: block;
    }
  }
  .card-title {
    display: flex;
    width: 100%;
    align-items: center;
    gap: 8px;
    .card-info {
      width: calc(100% - 70px);
      display: flex;
      flex-direction: column;
      justify-content: space-between;
      .card-name {
        display: block;
        font-size: 14px;
        font-weight: 700;
        overflow: hidden;
        white-space: nowrap;
        text-overflow: ellipsis;
        color: $create_card_text_color;
      }
    }
  }
  .card-des {
    width: 100%;
    display: -webkit-box;
    text-overflow: ellipsis;
    color: #5d5d5d;
    font-weight: 400;
    overflow: hidden;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    font-size: 13px;
    height: 36px;
    word-wrap: break-word;
  }
  .card-footer {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: auto;
    .card-footer-left {
      color: #888;
    }
    .card-footer-right {
      i {
        margin-left: 5px;
        cursor: pointer;
      }
    }
  }
}

.card-footer-right {
  display: flex;
  align-items: center;
  gap: 10px;
}
</style>
