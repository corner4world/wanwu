<template>
  <CommonLayout
    :isButton="false"
    :showAside="false"
    class="right-page-content-body"
  >
    <template #main-content>
      <div class="app-content">
        <Chat :editForm="editForm" :chatType="'chat'" />
      </div>
    </template>
  </CommonLayout>
</template>
<script>
import { mapActions } from 'vuex';
import CommonLayout from '@/components/exploreContainer.vue';
import Chat from './components/chat.vue';
import { getRagPublishedInfo } from '@/api/rag';
export default {
  components: { CommonLayout, Chat },
  data() {
    return {
      editForm: {
        appId: '',
        avatar: {},
        name: '',
        desc: '',
        knowledgeBaseConfig: { config: {}, knowledgebases: [] },
        qaKnowledgeBaseConfig: { config: {}, knowledgebases: [] },
        recommendQuestion: [],
      },
    };
  },
  created() {
    if (this.$route.query.id) {
      this.editForm.appId = this.$route.query.id;
      this.getDetail();
    }
  },
  beforeDestroy() {
    this.clearMaxPicNum();
  },
  methods: {
    ...mapActions('app', ['setMaxPicNum', 'clearMaxPicNum']),
    getDetail() {
      getRagPublishedInfo({ ragId: this.editForm.appId }).then(res => {
        if (res.code === 0) {
          this.editForm.avatar = res.data.avatar;
          this.editForm.name = res.data.name;
          this.editForm.desc = res.data.desc;
          if (res.data.knowledgeBaseConfig) {
            this.editForm.knowledgeBaseConfig = res.data.knowledgeBaseConfig;
          }
          if (res.data.qaKnowledgeBaseConfig) {
            this.editForm.qaKnowledgeBaseConfig =
              res.data.qaKnowledgeBaseConfig;
          }
          this.editForm.recommendQuestion = res.data.recommendQuestion?.map(
            item => ({
              value: item,
            }),
          );
          // 临时隐藏visionConfig的配置，全都设置为1
          // this.setMaxPicNum(res.data.visionConfig.picNum);
          this.setMaxPicNum(1);
        }
      });
    },
    goBack() {
      this.$router.go(-1);
    },
  },
};
</script>
<style lang="scss" scoped>
::v-deep {
  .apikeyBtn {
    padding: 11px 10px;
    border: 1px solid $btn_bg;
    color: $btn_bg;
    display: flex;
    align-items: center;
    img {
      height: 14px;
    }
  }
}
.app-content {
  width: 100%;
  height: 100%;
  position: relative;
  .app-header-api {
    width: 100%;
    padding: 10px;
    position: absolute;
    z-index: 999;
    top: 0;
    left: 0;
    border-bottom: 1px solid #eaeaea;
    display: flex;
    justify-content: space-between;
    align-content: center;
    .app_name {
      font-size: 18px;
      font-weight: bold;
      color: $color_title;
      display: flex;
      align-items: center;
      .goBack {
        font-weight: bold;
        font-size: 16px;
        cursor: pointer;
        margin-right: 15px;
        color: #333;
      }
    }
    .header-api-box {
      display: flex;
      .header-api-url {
        padding: 6px 10px;
        background: #fff;
        margin: 0 10px;
        border-radius: 6px;
        .root-url {
          background-color: #eceefe;
          color: $color;
          border: none;
        }
      }
    }
  }
}
</style>
