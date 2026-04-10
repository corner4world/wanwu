<template>
  <div
    class="ppt-preview-container"
    tabindex="0"
    @keydown="handleKeydown"
    ref="container"
  >
    <!-- 加载状态 -->
    <div v-if="loading" class="ppt-loading">
      <div class="loading-spinner"></div>
      <span>{{ $t('generalAgent.pptPreview.loading') }}</span>
    </div>

    <!-- 错误状态 -->
    <div v-else-if="error" class="ppt-error">
      <i class="el-icon-warning-outline"></i>
      <span>{{ error }}</span>
      <el-button size="small" @click="loadPpt">
        {{ $t('common.button.retry') }}
      </el-button>
      <el-button size="small" @click="handleDownload">
        <i class="el-icon-download"></i>
        {{ $t('generalAgent.pptPreview.downloadFile') }}
      </el-button>
    </div>

    <!-- PPT 预览内容 -->
    <div v-else-if="slides.length > 0" class="ppt-content">
      <!-- 幻灯片导航 -->
      <div class="ppt-toolbar">
        <button
          class="nav-btn"
          :disabled="currentSlide === 0"
          @click="currentSlide = 0"
          title="第一页 (Home)"
        >
          <i class="el-icon-d-arrow-left"></i>
        </button>
        <button
          class="nav-btn"
          :disabled="currentSlide === 0"
          @click="currentSlide--"
          title="上一页 (← / ↑ / PageUp)"
        >
          <i class="el-icon-arrow-left"></i>
        </button>
        <span class="slide-info">
          {{ currentSlide + 1 }} / {{ slides.length }}
        </span>
        <button
          class="nav-btn"
          :disabled="currentSlide >= slides.length - 1"
          @click="currentSlide++"
          title="下一页 (→ / ↓ / PageDown / 空格)"
        >
          <i class="el-icon-arrow-right"></i>
        </button>
        <button
          class="nav-btn"
          :disabled="currentSlide >= slides.length - 1"
          @click="currentSlide = slides.length - 1"
          title="最后一页 (End)"
        >
          <i class="el-icon-d-arrow-right"></i>
        </button>
        <span class="slide-name">{{ fileName }}</span>
        <span class="keyboard-hint">
          {{ $t('generalAgent.pptPreview.keyboardHint') }}
        </span>
      </div>

      <!-- 幻灯片显示区域 -->
      <div class="ppt-slide-wrapper" ref="slideWrapper" @wheel="handleWheel">
        <div class="ppt-slide" :style="slideStyle">
          <!-- 背景 -->
          <div
            v-if="currentSlideData && currentSlideData.fill"
            class="slide-background"
            :style="getBackgroundStyle(currentSlideData.fill)"
          ></div>

          <!-- 元素 -->
          <div
            v-for="(element, index) in currentSlideElements"
            :key="index"
            class="slide-element"
            :style="getElementStyle(element)"
          >
            <!-- 文本 -->
            <div
              v-if="element.type === 'text'"
              class="element-text"
              :style="getTextStyle(element)"
              v-html="processContent(element.content)"
            ></div>

            <!-- 图片 -->
            <img
              v-else-if="element.type === 'image'"
              class="element-image"
              :src="element.base64 || element.blob"
              :style="getImageStyle(element)"
            />

            <!-- 形状 -->
            <div
              v-else-if="element.type === 'shape'"
              class="element-shape"
              :style="getShapeStyle(element)"
            >
              <div
                v-if="element.content"
                class="shape-content"
                v-html="processContent(element.content)"
              ></div>
            </div>

            <!-- 表格 -->
            <table
              v-else-if="element.type === 'table'"
              class="element-table"
              :style="getTableStyle(element)"
            >
              <tr v-for="(row, rowIndex) in element.data" :key="rowIndex">
                <td
                  v-for="(cell, cellIndex) in row"
                  :key="cellIndex"
                  :style="getCellStyle(cell, element)"
                  v-html="cell && cell.content ? cell.content : ''"
                ></td>
              </tr>
            </table>

            <!-- 视频 -->
            <video
              v-else-if="element.type === 'video'"
              class="element-video"
              :src="element.blob"
              controls
              :style="getMediaStyle(element)"
            ></video>

            <!-- 音频 -->
            <div v-else-if="element.type === 'audio'" class="element-audio">
              <audio :src="element.blob" controls></audio>
            </div>

            <!-- 其他类型：显示占位符 -->
            <div v-else class="element-unknown">
              <i class="el-icon-document"></i>
              <span>
                {{ element.type || $t('generalAgent.pptPreview.element') }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- 幻灯片缩略图 -->
      <div class="ppt-thumbnails">
        <div
          v-for="(slide, index) in slides"
          :key="index"
          :class="['thumbnail', { active: index === currentSlide }]"
          :style="getThumbnailStyle(slide)"
          @click="currentSlide = index"
        >
          <span class="thumbnail-number">{{ index + 1 }}</span>
        </div>
      </div>
    </div>

    <!-- 空状态 -->
    <div v-else class="ppt-empty">
      <i class="el-icon-document"></i>
      <p>{{ $t('generalAgent.pptPreview.parseFailed') }}</p>
      <el-button type="primary" @click="handleDownload">
        <i class="el-icon-download"></i>
        {{ $t('generalAgent.pptPreview.downloadFile') }}
      </el-button>
    </div>
  </div>
</template>

<script>
import { parse } from 'pptxtojson';

export default {
  name: 'PptPreview',
  props: {
    src: {
      type: [String, Blob],
      default: null,
    },
    fileName: {
      type: String,
      default: '',
    },
  },
  data() {
    return {
      loading: false,
      error: null,
      slides: [],
      slideSize: { width: 960, height: 540 },
      themeColors: [],
      currentSlide: 0,
      currentScale: 1,
    };
  },
  computed: {
    currentSlideData() {
      return this.slides[this.currentSlide] || null;
    },
    currentSlideElements() {
      const slide = this.currentSlideData;
      if (!slide) return [];
      // 合并元素和布局元素
      return [...(slide.elements || []), ...(slide.layoutElements || [])];
    },
    slideStyle() {
      // pptxtojson 输出的尺寸单位是 pt
      // 1pt ≈ 1.333px (96 DPI)
      const ptToPx = 1.333;
      const originalWidthPx = this.slideSize.width * ptToPx;
      const originalHeightPx = this.slideSize.height * ptToPx;

      // 固定高度，宽度按比例
      const height = 450;
      const width = height * (originalWidthPx / originalHeightPx);

      // 计算缩放比例，用于元素定位
      this.currentScale = height / originalHeightPx;

      return {
        width: `${width}px`,
        height: `${height}px`,
      };
    },
  },
  watch: {
    src: {
      immediate: true,
      handler(newSrc) {
        if (newSrc) {
          this.$nextTick(() => this.loadPpt());
        }
      },
    },
  },
  mounted() {
    // 添加全局键盘事件监听，确保无论焦点在哪里都能响应翻页
    window.addEventListener('keydown', this.handleKeydown);
  },
  beforeDestroy() {
    // 移除全局键盘事件监听
    window.removeEventListener('keydown', this.handleKeydown);
  },
  methods: {
    async loadPpt() {
      this.loading = true;
      this.error = null;
      this.slides = [];
      this.currentSlide = 0;

      try {
        // 获取 ArrayBuffer
        let arrayBuffer;
        if (this.src instanceof Blob) {
          arrayBuffer = await this.src.arrayBuffer();
        } else if (typeof this.src === 'string') {
          // 从 URL 获取
          const response = await fetch(this.src);
          if (!response.ok) {
            throw new Error(`HTTP ${response.status}`);
          }
          arrayBuffer = await response.arrayBuffer();
        } else {
          throw new Error('无效的文件源');
        }

        // 使用 pptxtojson 解析
        const result = await parse(arrayBuffer, {
          imageMode: 'base64',
          videoMode: 'none',
          audioMode: 'none',
        });

        if (result && result.slides && result.slides.length > 0) {
          this.slides = result.slides;
          this.slideSize = result.size || { width: 960, height: 540 };
          this.themeColors = result.themeColors || [];
        } else {
          throw new Error('PPT 文件解析结果为空');
        }

        this.loading = false;
      } catch (err) {
        console.error('[PptPreview] 加载失败:', err);
        this.error = `加载失败: ${err.message}`;
        this.loading = false;
      }
    },

    // 元素定位样式
    getElementStyle(element) {
      // pptxtojson 输出的单位是 pt，需要转换为 px
      // 1pt ≈ 1.333px (在 96 DPI 标准下)
      const ptToPx = 1.333;
      const scale = this.currentScale || 1;

      const style = {
        position: 'absolute',
        left: `${element.left * ptToPx * scale}px`,
        top: `${element.top * ptToPx * scale}px`,
        width: `${element.width * ptToPx * scale}px`,
        height: `${element.height * ptToPx * scale}px`,
      };
      if (element.rotate) {
        style.transform = `rotate(${element.rotate}deg)`;
      }
      // 处理翻转
      if (element.isFlipV || element.isFlipH) {
        const flipScaleX = element.isFlipH ? -1 : 1;
        const flipScaleY = element.isFlipV ? -1 : 1;
        const rotate = element.rotate || 0;
        style.transform = `rotate(${rotate}deg) scale(${flipScaleX}, ${flipScaleY})`;
      }
      return style;
    },

    // 解析颜色（处理带透明度的颜色）
    parseColor(color) {
      if (!color) return null;
      // 处理 #RRGGBBAA 格式
      if (color.startsWith('#') && color.length === 9) {
        const r = color.substring(1, 3);
        const g = color.substring(3, 5);
        const b = color.substring(5, 7);
        const a = parseInt(color.substring(7, 9), 16) / 255;
        return `rgba(${parseInt(r, 16)}, ${parseInt(g, 16)}, ${parseInt(b, 16)}, ${a.toFixed(2)})`;
      }
      return color;
    },

    // 背景样式
    getBackgroundStyle(fill) {
      if (!fill) return {};
      switch (fill.type) {
        case 'color':
          return { backgroundColor: this.parseColor(fill.value) };
        case 'image':
          return {
            backgroundImage: `url(${fill.base64 || fill.blob})`,
            backgroundSize: 'cover',
            backgroundPosition: 'center',
          };
        case 'gradient':
          if (fill.colors && fill.colors.length > 0) {
            return {
              background: `linear-gradient(${fill.angle || 0}deg, ${fill.colors.map(c => c.color).join(', ')})`,
            };
          }
          return {};
        default:
          return {};
      }
    },

    // 文本样式
    getTextStyle(element) {
      const scale = this.currentScale || 1;
      const style = {
        width: '100%',
        height: '100%',
        display: 'flex',
        flexDirection: 'column',
        justifyContent: this.getVAlign(element.vAlign),
        alignItems: this.getHAlign(element.align),
        overflow: 'hidden',
        lineHeight: 1.2,
      };

      // 处理段落间距
      if (element.lineHeight) {
        style.lineHeight = element.lineHeight;
      }

      // 处理文本填充颜色
      if (element.fill && element.fill.type === 'color') {
        style.color = this.parseColor(element.fill.value);
      }

      return style;
    },

    getVAlign(align) {
      switch (align) {
        case 'top':
          return 'flex-start';
        case 'mid':
        case 'middle':
          return 'center';
        case 'bottom':
          return 'flex-end';
        default:
          return 'center';
      }
    },

    getHAlign(align) {
      switch (align) {
        case 'left':
          return 'flex-start';
        case 'center':
        case 'centered':
          return 'center';
        case 'right':
          return 'flex-end';
        default:
          return 'flex-start';
      }
    },

    // 图片样式
    getImageStyle(element) {
      return {
        width: '100%',
        height: '100%',
        objectFit: 'contain',
      };
    },

    // 形状样式
    getShapeStyle(element) {
      const style = {
        width: '100%',
        height: '100%',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
      };

      // 填充 - 只有在有颜色填充时才设置背景
      if (element.fill && element.fill.type === 'color') {
        style.backgroundColor = this.parseColor(element.fill.value);
      } else if (
        element.fill &&
        element.fill.type === 'image' &&
        element.fill.base64
      ) {
        style.backgroundImage = `url(${element.fill.base64})`;
        style.backgroundSize = 'cover';
      }
      // fill 为 null 或 type 为 noFill/none 时，背景透明（不设置 backgroundColor）

      // 边框 - 只在有边框宽度时显示
      if (
        element.borderColor &&
        element.borderWidth &&
        element.borderWidth > 0
      ) {
        style.border = `${element.borderWidth}px ${element.borderType || 'solid'} ${element.borderColor}`;
      }

      // 形状类型
      if (element.shapType === 'ellipse') {
        style.borderRadius = '50%';
      } else if (element.shapType === 'roundRect' && element.keypoints) {
        style.borderRadius = '10px';
      }

      return style;
    },

    // 表格样式
    getTableStyle(element) {
      return {
        width: '100%',
        height: '100%',
        borderCollapse: 'collapse',
      };
    },

    // 表格单元格样式
    getCellStyle(cell, element) {
      const style = {
        border: '1px solid #ccc',
        padding: '4px',
        verticalAlign: 'middle',
      };
      if (cell && cell.fill) {
        style.backgroundColor = cell.fill.value || cell.fill;
      }
      return style;
    },

    // 媒体样式
    getMediaStyle(element) {
      return {
        width: '100%',
        height: '100%',
        objectFit: 'contain',
      };
    },

    // 缩略图样式
    getThumbnailStyle(slide) {
      if (slide && slide.fill && slide.fill.type === 'color') {
        return { backgroundColor: slide.fill.value };
      }
      return {};
    },

    handleDownload() {
      this.$emit('download');
    },

    // 处理内容，修复字体等问题
    processContent(content) {
      if (!content) return '';
      const scale = this.currentScale || 1;
      // 替换不常见字体为系统字体，并缩放字体大小
      return content
        .replace(
          /font-family:\s*["']?Arial Black["']?/gi,
          'font-family: Arial Black, Arial, sans-serif',
        )
        .replace(
          /font-family:\s*["']?Arial["']?/gi,
          'font-family: Arial, Helvetica, sans-serif',
        )
        .replace(
          /font-family:\s*["']?Calibri["']?/gi,
          'font-family: Calibri, Arial, sans-serif',
        )
        .replace(
          /font-family:\s*["']?Times New Roman["']?/gi,
          'font-family: Times New Roman, Times, serif',
        )
        .replace(
          /font-family:\s*["']?Courier New["']?/gi,
          'font-family: Courier New, Courier, monospace',
        )
        .replace(
          /font-family:\s*["']?Verdana["']?/gi,
          'font-family: Verdana, Geneva, sans-serif',
        )
        .replace(
          /font-family:\s*["']?Georgia["']?/gi,
          'font-family: Georgia, serif',
        )
        .replace(
          /font-family:\s*["']?微软雅黑["']?/gi,
          'font-family: "Microsoft YaHei", Arial, sans-serif',
        )
        .replace(
          /font-family:\s*["']?宋体["']?/gi,
          'font-family: SimSun, serif',
        )
        .replace(
          /font-family:\s*["']?黑体["']?/gi,
          'font-family: SimHei, sans-serif',
        )
        .replace(/font-size:\s*(\d+(?:\.\d+)?)pt/gi, (match, size) => {
          // pt 转 px 并按缩放比例调整
          const pxSize = parseFloat(size) * 1.333 * scale;
          return `font-size: ${pxSize.toFixed(1)}px`;
        });
    },

    // 鼠标滚轮翻页
    handleWheel(event) {
      // 防止在滚动条位置时触发
      const wrapper = this.$refs.slideWrapper;
      if (!wrapper) return;

      // 检查是否有滚动条且不在顶部/底部
      const hasScroll = wrapper.scrollHeight > wrapper.clientHeight;
      if (hasScroll) {
        const atTop = wrapper.scrollTop === 0;
        const atBottom =
          wrapper.scrollTop + wrapper.clientHeight >= wrapper.scrollHeight - 1;

        // 向下滚动且不在底部，或向上滚动且不在顶部时，不翻页
        if ((event.deltaY > 0 && !atBottom) || (event.deltaY < 0 && !atTop)) {
          return;
        }
      }

      // 防抖处理
      if (this._wheelTimeout) return;

      this._wheelTimeout = setTimeout(() => {
        this._wheelTimeout = null;
      }, 300);

      // 向下滚动 -> 下一页
      if (event.deltaY > 0 && this.currentSlide < this.slides.length - 1) {
        this.currentSlide++;
        event.preventDefault();
      }
      // 向上滚动 -> 上一页
      else if (event.deltaY < 0 && this.currentSlide > 0) {
        this.currentSlide--;
        event.preventDefault();
      }
    },

    // 键盘翻页（全局监听）
    handleKeydown(event) {
      // 如果没有幻灯片，不处理
      if (!this.slides || this.slides.length === 0) return;

      // 如果焦点在输入框、文本域等可编辑元素上，不处理
      const activeElement = document.activeElement;
      const editableTags = ['INPUT', 'TEXTAREA', 'SELECT'];
      const isEditable =
        activeElement &&
        (editableTags.includes(activeElement.tagName) ||
          activeElement.isContentEditable ||
          activeElement.getAttribute('contenteditable') === 'true');
      if (isEditable) return;

      // 左箭头 或 上箭头 或 PageUp -> 上一页
      if (
        event.key === 'ArrowLeft' ||
        event.key === 'ArrowUp' ||
        event.key === 'PageUp'
      ) {
        if (this.currentSlide > 0) {
          this.currentSlide--;
          event.preventDefault();
        }
      }
      // 右箭头 或 下箭头 或 PageDown 或 空格 -> 下一页
      else if (
        event.key === 'ArrowRight' ||
        event.key === 'ArrowDown' ||
        event.key === 'PageDown' ||
        event.key === ' '
      ) {
        if (this.currentSlide < this.slides.length - 1) {
          this.currentSlide++;
          event.preventDefault();
        }
      }
      // Home -> 第一页
      else if (event.key === 'Home') {
        this.currentSlide = 0;
        event.preventDefault();
      }
      // End -> 最后一页
      else if (event.key === 'End') {
        this.currentSlide = this.slides.length - 1;
        event.preventDefault();
      }
      // Escape -> 关闭预览（通知父组件）
      else if (event.key === 'Escape') {
        this.$emit('close');
        event.preventDefault();
      }
    },
  },
};
</script>

<style lang="scss" scoped>
.ppt-preview-container {
  width: 100%;
  height: 100%;
  min-height: 600px;
  background: #fff;
  display: flex;
  flex-direction: column;
  outline: none; // 移除默认焦点边框

  &:focus {
    outline: none;
  }
}

.ppt-loading {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 400px;
  color: #666;

  .loading-spinner {
    width: 40px;
    height: 40px;
    border: 3px solid #e0e0e0;
    border-top-color: #f59e0b;
    border-radius: 50%;
    animation: spin 1s linear infinite;
    margin-bottom: 16px;
  }
}

@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}

.ppt-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 400px;
  color: #f56c6c;

  i {
    font-size: 48px;
    margin-bottom: 16px;
  }

  span {
    margin-bottom: 16px;
  }

  .el-button {
    margin: 8px;
  }
}

.ppt-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ppt-toolbar {
  display: flex;
  align-items: center;
  padding: 12px 16px;
  background: #f5f5f5;
  border-bottom: 1px solid #e0e0e0;
  gap: 8px;

  .nav-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 32px;
    height: 32px;
    border: 1px solid #dcdfe6;
    background: #fff;
    border-radius: 4px;
    cursor: pointer;
    color: #606266;
    transition: all 0.2s;

    &:hover:not(:disabled) {
      border-color: #409eff;
      color: #409eff;
    }

    &:disabled {
      cursor: not-allowed;
      opacity: 0.5;
    }
  }

  .slide-info {
    font-size: 14px;
    color: #333;
    min-width: 60px;
    text-align: center;
  }

  .slide-name {
    margin-left: auto;
    font-size: 13px;
    color: #666;
    max-width: 200px;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .keyboard-hint {
    font-size: 12px;
    color: #999;
    margin-left: 8px;
    padding: 2px 8px;
    background: #f0f0f0;
    border-radius: 4px;
  }
}

.ppt-slide-wrapper {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 20px;
  background: #e8e8e8;
  overflow: auto;
}

.ppt-slide {
  position: relative;
  background: #fff;
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.15);
  overflow: hidden;

  .slide-background {
    position: absolute;
    top: 0;
    left: 0;
    right: 0;
    bottom: 0;
    z-index: 0;
  }

  .slide-element {
    z-index: 1;
    box-sizing: border-box;
  }
}

.element-text {
  width: 100%;
  height: 100%;
  box-sizing: border-box;
  overflow: hidden;
  line-height: 1.2;

  :deep(p) {
    margin: 0;
    padding: 0;
    width: 100%;
    box-sizing: border-box;
    line-height: inherit;
  }

  :deep(span) {
    display: inline;
    word-wrap: break-word;
    white-space: pre-wrap;
    line-height: inherit;
  }

  :deep(b),
  :deep(strong) {
    font-weight: bold;
  }

  :deep(i),
  :deep(em) {
    font-style: italic;
  }

  :deep(u) {
    text-decoration: underline;
  }

  // 处理 font-family，提供备选字体
  :deep([style*='font-family']) {
    font-family: inherit;
  }
}

.element-image {
  display: block;
}

.element-shape {
  box-sizing: border-box;

  .shape-content {
    width: 100%;
    height: 100%;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
    box-sizing: border-box;
    overflow: hidden;
    line-height: 1.2;

    :deep(p) {
      margin: 0;
      padding: 0;
      width: 100%;
      box-sizing: border-box;
      line-height: inherit;
    }

    :deep(span) {
      display: inline;
      word-wrap: break-word;
      white-space: pre-wrap;
      line-height: inherit;
    }

    :deep(b),
    :deep(strong) {
      font-weight: bold;
    }

    :deep(i),
    :deep(em) {
      font-style: italic;
    }

    :deep(u) {
      text-decoration: underline;
    }
  }
}

.element-table {
  font-size: 12px;

  td {
    padding: 4px 8px;
  }
}

.element-video {
  background: #000;
}

.element-audio {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  background: #f0f0f0;
}

.element-unknown {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  background: #f5f5f5;
  color: #999;
  font-size: 12px;

  i {
    font-size: 24px;
    margin-bottom: 4px;
  }
}

.ppt-thumbnails {
  display: flex;
  gap: 8px;
  padding: 12px 16px;
  background: #f5f5f5;
  border-top: 1px solid #e0e0e0;
  overflow-x: auto;

  .thumbnail {
    flex-shrink: 0;
    width: 50px;
    height: 38px;
    background: #fff;
    border: 2px solid #dcdfe6;
    border-radius: 4px;
    display: flex;
    align-items: center;
    justify-content: center;
    cursor: pointer;
    transition: all 0.2s;

    &:hover {
      border-color: #409eff;
    }

    &.active {
      border-color: #409eff;
      background: #ecf5ff;
    }

    .thumbnail-number {
      font-size: 12px;
      color: #666;
      font-weight: 500;
    }
  }
}

.ppt-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  flex: 1;
  min-height: 400px;
  color: #999;

  i {
    font-size: 64px;
    margin-bottom: 16px;
    color: #dcdfe6;
  }

  p {
    margin-bottom: 20px;
    color: #666;
  }
}
</style>
