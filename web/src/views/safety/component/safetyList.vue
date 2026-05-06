<template>
  <div class="app-card-container scroll-card-container">
    <div class="app-card scroll-card-pr">
      <div class="smart rl smart-create">
        <div class="app-card-create" @click="showCreate">
          <div class="create-img-wrap">
            <img
              class="create-img"
              src="@/assets/imgs/card_create_icon_safety.svg"
              alt=""
            />
          </div>
          <span>{{ $t('safety.safetyList.create') }}</span>
        </div>
      </div>
      <template v-if="listData && listData.length">
        <div
          class="smart rl"
          v-for="(n, i) in listData"
          :key="`${i}sm`"
          @click.stop="toWordList(n)"
        >
          <div class="smart-card-header">
            <img class="logo" :src="require('@/assets/imgs/safety.png')" />
            <p class="name" :title="n.tableName">
              {{ n.tableName }}
            </p>
          </div>

          <el-tooltip
            v-if="n.remark"
            popper-class="instr-tooltip tooltip-cover-arrow"
            effect="dark"
            :content="n.remark"
            placement="bottom-start"
          >
            <p class="desc">{{ n.remark }}</p>
          </el-tooltip>

          <div class="smart-card-footer">
            <div class="tags">
              <span :class="['smartDate', 'tagList']">{{ n.createdAt }}</span>
            </div>
            <div class="smart-card-footer-right">
              <div class="editor">
                <el-dropdown @command="handleClick($event, n)" placement="top">
                  <span class="el-dropdown-link">
                    <i class="el-icon-more icon edit-icon" @click.stop></i>
                  </span>
                  <el-dropdown-menu slot="dropdown">
                    <el-dropdown-item command="edit">
                      {{ $t('common.button.edit') }}
                    </el-dropdown-item>
                    <el-dropdown-item command="delete">
                      {{ $t('common.button.delete') }}
                    </el-dropdown-item>
                  </el-dropdown-menu>
                </el-dropdown>
              </div>
            </div>
          </div>
        </div>
      </template>
    </div>
    <el-empty
      class="noData"
      v-if="!(listData && listData.length)"
      :description="$t('common.noData')"
    ></el-empty>
  </div>
</template>

<script>
import { delSensitive } from '@/api/safety';
export default {
  props: {
    appData: {
      type: Array,
      required: true,
      default: [],
    },
  },
  watch: {
    appData: {
      handler: function (val) {
        this.listData = val;
      },
      immediate: true,
      deep: true,
    },
  },
  data() {
    return {
      basePath: this.$basePath,
      listData: [],
    };
  },
  methods: {
    showCreate() {
      this.$parent.showCreate();
    },
    handleClick(command, n) {
      switch (command) {
        case 'edit':
          this.editItem(n);
          break;
        case 'delete':
          this.deleteItem(n.tableId);
          break;
      }
    },
    editItem(row) {
      this.$emit('editItem', row);
    },
    reloadData() {
      this.$emit('reloadData');
    },
    deleteItem(tableId) {
      this.$confirm(
        this.$t('safety.safetyList.delTip'),
        this.$t('knowledgeManage.tip'),
        {
          confirmButtonText: this.$t('common.confirm.confirm'),
          cancelButtonText: this.$t('common.confirm.cancel'),
          type: 'warning',
          beforeClose: (action, instance, done) => {
            if (action === 'confirm') {
              instance.confirmButtonLoading = true;
              delSensitive({ tableId })
                .then(res => {
                  if (res.code === 0) {
                    this.$message.success(
                      this.$t('knowledgeManage.operateSuccess'),
                    );
                    this.$emit('reloadData');
                  }
                })
                .catch(() => {})
                .finally(() => {
                  done();
                  setTimeout(() => {
                    instance.confirmButtonLoading = false;
                  }, 300);
                });
            } else {
              done();
            }
          },
        },
      ).then(() => {});
    },
    toWordList(n) {
      this.$router.push({ path: `/safety/wordList/${n.tableId}` });
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/commonCard.scss';
.scroll-card-container {
  max-height: calc(100vh - 160px);
}

.app-card {
  .smart {
    height: 152px;
    .smartDate {
      padding-top: 3px;
      color: #888888;
    }
    .info {
      padding-right: 0;
      display: flex;
      justify-content: flex-start;
    }
    .tagList {
      cursor: pointer;
      .icon-tag {
        transform: rotate(-40deg);
        margin-right: 3px;
      }
    }
    .tagList:hover {
      color: $color;
    }
  }
}
</style>
