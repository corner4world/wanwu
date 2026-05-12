<template>
  <el-image-viewer
    v-show="showImageViewer"
    :initial-index="currentImageIndex"
    :on-close="handleClose"
    :url-list="imageList"
    :z-index="zIndex"
  />
</template>

<script>
import ElImageViewer from 'element-ui/packages/image/src/image-viewer';

export default {
  name: 'ImagePreview',
  components: {
    ElImageViewer,
  },
  props: {
    // z-index层级
    zIndex: {
      type: Number,
      default: 9999,
    },
  },
  data() {
    return {
      imageList: [], // 存储所有图片URL用于预览
      showImageViewer: false, // 是否显示图片预览
      currentImageIndex: 0, // 当前预览的图片索引
    };
  },
  methods: {
    /**
     * 处理图片点击事件
     * @param {Event} event - 点击事件对象
     */
    handleImageClick(event) {
      // 检查点击的是否是图片元素
      if (event.target.tagName !== 'IMG') {
        return;
      }

      const clickedSrc = event.target.getAttribute('src');

      // 从触发事件的容器中查找所有图片
      const container = event.currentTarget;
      const images = container.querySelectorAll('img');
      this.imageList = Array.from(images).map(img => img.src);
      const index = this.imageList.indexOf(clickedSrc);

      // 如果找到对应的图片，显示预览
      if (index !== -1) {
        this.currentImageIndex = index;
        this.showImageViewer = true;
      }
    },

    /**
     * 关闭图片预览
     */
    handleClose() {
      this.showImageViewer = false;
      this.$emit('close');
    },

    /**
     * 手动打开图片预览（适用于需要自定义图片列表的场景）
     * @param {Array} images - 图片URL数组
     * @param {number} index - 初始显示的图片索引
     */
    open(images, index = 0) {
      if (images && images.length > 0) {
        this.imageList = images;
        this.currentImageIndex = index;
        this.showImageViewer = true;
      }
    },

    /**
     * 手动关闭图片预览
     */
    close() {
      this.showImageViewer = false;
    },
  },
};
</script>
