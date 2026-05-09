<template>
  <div class="main-content docs-page-content">
    <div class="docs-page-wrapper">
      <div class="docs-page-main">
        <MdContent :content="mdContent" />
      </div>
      <div class="docs-page-outline">
        <MdOutline :content="mdContent" />
      </div>
    </div>
  </div>
</template>
<script>
import { getMarkdown } from '@/api/docs';
import { Loading } from 'element-ui';
import { DOC_FIRST_KEY } from '../constants';
import MdContent from '@/components/mdContent.vue';
import MdOutline from '@/components/mdOutline.vue';

export default {
  components: { MdContent, MdOutline },
  data() {
    return {
      mdContent: '',
    };
  },
  watch: {
    $route: {
      handler(val, oldValue) {
        if (val !== oldValue) {
          this.getMarkdown(val.params.id);
          const docsPageContent = document.querySelector('.el-main');
          if (docsPageContent) docsPageContent.scrollTo(0, 0);
        }
      },
      // 深度观察监听
      deep: true,
    },
  },
  created() {
    this.getMarkdown(this.$route.params.id);
  },
  methods: {
    docContentScrollTop() {
      const docPageMain = document.querySelector('.doc-page-main');
      if (docPageMain) docPageMain.scrollTop = 0;
    },
    getMarkdown(path) {
      if (path === DOC_FIRST_KEY) return;

      const loadingInstance = Loading.service({
        target: document.querySelector('.docs-page-content'),
      });
      getMarkdown({ path })
        .then(res => {
          this.mdContent = res.data || '';
          loadingInstance.close();
          this.docContentScrollTop();
        })
        .catch(() => {
          this.mdContent = '';
          loadingInstance.close();
        });
    },
  },
};
</script>

<style lang="scss" scoped>
.docs-page-content {
  .docs-page-wrapper {
    display: flex;
    gap: 24px;
    .docs-page-main {
      flex: 1;
      min-width: 0;
    }
    .docs-page-outline {
      width: 220px;
      flex-shrink: 0;
    }
  }
}
</style>
