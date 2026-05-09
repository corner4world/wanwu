<template>
  <div class="auth">
    <div class="overview">
      <img :src="backgroundSrc" alt="" />
    </div>
    <div class="auth-modal">
      <div class="header__left">
        <img
          v-if="commonInfo.login.logo && commonInfo.login.logo.path"
          style="max-height: 60px; max-width: 220px; margin: 0 15px 0 22px"
          :src="avatarSrc(commonInfo.login.logo.path)"
          alt=""
        />
        <!--<span style="font-size: 16px;">{{commonInfo.home.title || ''}}</span>-->
        <!--<div style="margin-left: 10px">
          <ChangeLang :isLogin="true" />
        </div>-->
      </div>
      <!--      <div class="container__left">-->
      <!--        {{ commonInfo.login.welcomeText }}-->
      <!--      </div>-->

      <slot :commonInfo="commonInfo" />
    </div>
  </div>
</template>

<script>
import { mapState, mapActions } from 'vuex';
import ChangeLang from '@/components/changeLang.vue';
import { replaceTitle, replaceIcon, avatarSrc } from '@/utils/util';
import { getCommonInfo } from '@/api/user';

export default {
  components: { ChangeLang },
  computed: {
    ...mapState('login', ['commonInfo']),
    ...mapState('user', ['lang']),
    backgroundSrc() {
      return avatarSrc(
        this.commonInfo?.login?.background?.path || '',
        require('@/assets/imgs/auth_bg.png'),
      );
    },
  },
  watch: {
    lang: {
      handler(val) {
        if (val) {
          /*this.getImgCode()
          this.getLogoInfo()*/
        }
      },
      immediate: true,
    },
  },
  created() {
    this.getCommonInfo().then(() => {
      replaceTitle(this.commonInfo?.tab?.title || '');
      replaceIcon(this.commonInfo?.tab?.logo?.path || '');
      this.$emit('getCommonInfo', this.commonInfo);
    });
  },
  methods: {
    avatarSrc,
    ...mapActions('login', ['getCommonInfo']),
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/auth.scss';
.overview {
  position: relative;
  height: 100%;
  overflow: hidden;
  //background-color: #000;
  z-index: 10;

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
    background-size: 100% 100%;
  }

  .overview-desc {
    width: 800px;
    position: absolute;
    bottom: 56px;
    left: 56px;
    color: #fff;
    text-align: center;
    opacity: 0.8;
    letter-spacing: 1px;

    .desc {
      font-size: 30px;
      text-align: left;

      p:nth-child(1) {
        font-size: 22px;
      }

      p:nth-child(2) {
        font-size: 30px;
        margin: 10px 0;
      }

      p:nth-child(3) {
        font-size: 18px;
      }
    }
  }
}

.auth {
  height: 100%;
}

.auth-modal {
  position: fixed;
  top: 0;
  bottom: 0;
  left: 0;
  right: 0;
  width: 100%;
  height: 100%;
  z-index: 1000;

  .header__left {
    position: relative;
    width: 100%;
    min-width: 500px;
    color: #fff;
    font-weight: bold;
    display: flex;
    align-items: center;
    margin-top: 16px;
    margin-left: 10px;
    height: 60px;
  }

  .container__left {
    display: flex;
    align-items: center;
    height: calc(80% - 60px);
    font-size: 35px;
    width: calc(100% - 13% - 400px);
    justify-content: center;
    color: #fff;
    text-shadow: 2px 2px 4px rgba(0, 0, 0, 0.6);
  }
}
</style>
