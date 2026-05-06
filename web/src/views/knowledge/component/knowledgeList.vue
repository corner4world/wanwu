<template>
  <div class="app-card-container scroll-card-container">
    <div class="app-card scroll-card-pr">
      <div class="smart rl smart-create">
        <div class="app-card-create" @click="showCreate">
          <div class="create-img-wrap">
            <img
              class="create-img"
              :src="categoryLabelMap.createIcon[category]"
              alt=""
            />
          </div>
          <span>
            {{ categoryLabelMap.createButton[category] }}
          </span>
        </div>
      </div>
      <template v-if="listData && listData.length">
        <div
          class="smart rl"
          v-for="(n, i) in listData"
          :key="`${i}sm`"
          @click.stop="toDocList(n)"
        >
          <div
            v-if="category === KNOWLEDGE"
            :class="['ribbon', n.external === INTERNAL ? 'blue' : 'gold']"
          >
            <span>
              {{
                n.external === INTERNAL
                  ? $t('knowledgeManage.ribbon.internal')
                  : $t('knowledgeManage.ribbon.external')
              }}
            </span>
          </div>
          <div class="card-content">
            <div class="card-header">
              <img
                class="common-card-logo ml-10"
                :src="
                  avatarSrc(
                    n.avatar?.path,
                    require('@/assets/imgs/knowledgeIcon.png'),
                  )
                "
              />
              <div class="header-info">
                <div style="display: flex; align-items: center">
                  <p class="name" :title="n.name">
                    {{ n.name }}
                  </p>
                </div>
                <p class="desc">
                  <label class="desc-item">
                    {{ n.docCount || 0 }}
                    {{ categoryLabelMap.countUnit[category] }}
                  </label>
                  <label v-if="n.category === MULTIMODAL" class="desc-item">
                    {{ $t('knowledgeManage.multiKnowledgeDatabase.label') }}
                  </label>
                </p>
              </div>
            </div>
            <div class="card-description">
              <p class="card-desc">
                {{ n.description }}
              </p>
            </div>
          </div>
          <div class="tags">
            <span v-if="category === KNOWLEDGE">
              <span
                :class="['smartDate', 'tagList']"
                v-if="formattedTagNames(n.knowledgeTagList).length === 0"
                @click.stop="addTag(n.knowledgeId, n)"
              >
                <span class="el-icon-price-tag icon-tag"></span>
                {{ $t('knowledgeManage.addTag') }}
              </span>
              <span v-else @click.stop="addTag(n.knowledgeId, n)">
                {{ formattedTagNames(n.knowledgeTagList) }}
              </span>
            </span>
          </div>
          <div class="editor">
            <el-tooltip
              class="item"
              effect="dark"
              :content="n.orgName"
              placement="right-start"
            >
              <span style="margin-right: 52px; color: #999; font-size: 12px">
                {{
                  n.orgName.length > 10
                    ? n.orgName.substring(0, 10) + '...'
                    : n.orgName
                }}
              </span>
            </el-tooltip>
            <div v-if="n.share" class="publishType" style="right: 22px">
              <span v-if="n.share" class="publishType-tag">
                <span class="el-icon-unlock"></span>
                {{ $t('knowledgeManage.public') }}
              </span>
              <span v-else class="publishType-tag">
                <span class="el-icon-lock"></span>
                {{ $t('knowledgeManage.private') }}
              </span>
            </div>
            <el-dropdown @command="handleClick($event, n)" placement="top">
              <span class="el-dropdown-link">
                <i class="el-icon-more icon edit-icon" @click.stop></i>
              </span>
              <el-dropdown-menu slot="dropdown">
                <el-dropdown-item
                  command="edit"
                  v-if="[POWER_TYPE_SYSTEM_ADMIN].includes(n.permissionType)"
                >
                  {{ $t('common.button.edit') }}
                </el-dropdown-item>
                <el-dropdown-item
                  command="delete"
                  v-if="[POWER_TYPE_SYSTEM_ADMIN].includes(n.permissionType)"
                >
                  {{ $t('common.button.delete') }}
                </el-dropdown-item>
                <el-dropdown-item
                  command="export"
                  v-if="
                    n.external === INTERNAL &&
                    [
                      POWER_TYPE_EDIT,
                      POWER_TYPE_ADMIN,
                      POWER_TYPE_SYSTEM_ADMIN,
                    ].includes(n.permissionType)
                  "
                >
                  {{ $t('common.button.export') }}
                </el-dropdown-item>
                <el-dropdown-item
                  command="exportRecord"
                  v-if="
                    n.external === INTERNAL &&
                    [
                      POWER_TYPE_EDIT,
                      POWER_TYPE_ADMIN,
                      POWER_TYPE_SYSTEM_ADMIN,
                    ].includes(n.permissionType)
                  "
                >
                  {{ $t('knowledgeManage.qaDatabase.exportRecord') }}
                </el-dropdown-item>
                <el-dropdown-item command="power">
                  {{ $t('knowledgeSelect.power') }}
                </el-dropdown-item>
              </el-dropdown-menu>
            </el-dropdown>
          </div>
        </div>
      </template>
    </div>
    <el-empty
      class="noData"
      v-if="!(listData && listData.length)"
      :description="$t('common.noData')"
    ></el-empty>
    <tagDialog
      ref="tagDialog"
      @reloadData="reloadData"
      type="knowledge"
      :title="title"
    />
    <PowerManagement ref="powerManagement" />
    <exportRecord ref="exportRecord" />
  </div>
</template>

<script>
import { delKnowledgeItem } from '@/api/knowledge';
import { AppType } from '@/utils/commonSet';
import tagDialog from './tagDialog.vue';
import PowerManagement from './power/index.vue';
import exportRecord from '@/views/knowledge/qaDatabase/exportRecord.vue';
import { mapActions } from 'vuex';
import {
  INITIAL,
  POWER_TYPE_READ,
  POWER_TYPE_EDIT,
  POWER_TYPE_ADMIN,
  POWER_TYPE_SYSTEM_ADMIN,
  INTERNAL,
  EXTERNAL,
  KNOWLEDGE,
  QA,
  MULTIMODAL,
  DB,
} from '@/views/knowledge/constants';
import { avatarSrc } from '@/utils/util';

export default {
  components: { tagDialog, PowerManagement, exportRecord },
  props: {
    appData: {
      type: Array,
      required: true,
      default: [],
    },
    category: {
      type: Number,
      required: true,
      default: 0,
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
      apptype: AppType,
      basePath: this.$basePath,
      listData: [],
      title: this.$t('knowledgeManage.createTag'),
      INITIAL,
      POWER_TYPE_READ,
      POWER_TYPE_EDIT,
      POWER_TYPE_ADMIN,
      POWER_TYPE_SYSTEM_ADMIN,
      INTERNAL,
      EXTERNAL,
      KNOWLEDGE,
      QA,
      MULTIMODAL,
      DB,
      categoryLabelMap: {
        createButton: {
          [KNOWLEDGE]: this.$t('knowledgeManage.createKnowledge'),
          [QA]: this.$t('knowledgeManage.createQaDatabase'),
          [DB]: this.$t('knowledgeManage.createDatabase'),
        },
        createIcon: {
          [KNOWLEDGE]: require('@/assets/imgs/card_create_icon_knowledge.svg'),
          [QA]: require('@/assets/imgs/card_create_icon_rag.svg'),
          [DB]: require('@/assets/imgs/card_create_icon_rag.svg'),
        },
        countUnit: {
          [KNOWLEDGE]: this.$t('knowledgeManage.docCountUnit'),
          [QA]: this.$t('knowledgeManage.qaCountUnit'),
          [DB]: this.$t('knowledgeManage.dbCountUnit'),
        },
      },
    };
  },

  methods: {
    avatarSrc,
    ...mapActions('app', ['setPermissionType', 'clearPermissionType']),
    formattedTagNames(data) {
      if (data.length === 0) {
        return [];
      }
      const tags = data
        .filter(item => item.selected)
        .map(item => item.tagName)
        .join(', ');
      if (tags.length > 30) {
        return tags.slice(0, 30) + '...';
      }
      return tags;
    },
    addTag(id, n) {
      if ([POWER_TYPE_READ].includes(n.permissionType)) {
        this.$message.warning(this.$t('knowledgeSelect.noOperationPermission'));
        return;
      }
      this.$nextTick(() => {
        this.$refs.tagDialog.showDialog(id);
      });
    },
    showCreate() {
      this.$parent.showCreate();
    },
    handleClick(command, n) {
      switch (command) {
        case 'edit':
          this.editItem(n);
          break;
        case 'delete':
          this.deleteItem(n.knowledgeId);
          break;
        case 'export':
          this.exportItem(n);
          break;
        case 'exportRecord':
          this.exportRecord(n.knowledgeId);
          break;
        case 'power':
          this.showPowerManagement(n);
          break;
      }
    },
    exportItem(row) {
      this.$emit('exportItem', row);
    },
    exportRecord(knowledgeId) {
      this.$refs.exportRecord.showDialog(knowledgeId);
    },
    editItem(row) {
      this.$emit('editItem', row);
    },
    reloadData() {
      this.$emit('reloadData');
    },
    deleteItem(knowledgeId) {
      this.$confirm(
        this.$t('knowledgeManage.delKnowledgeTips'),
        this.$t('knowledgeManage.tip'),
        {
          confirmButtonText: this.$t('common.confirm.confirm'),
          cancelButtonText: this.$t('common.confirm.cancel'),
          type: 'warning',
          beforeClose: (action, instance, done) => {
            if (action === 'confirm') {
              instance.confirmButtonLoading = true;
              delKnowledgeItem({ knowledgeId })
                .then(res => {
                  if (res.code === 0) {
                    this.$message.success(
                      this.$t('knowledgeManage.operateSuccess'),
                    );
                    this.$emit('reloadData', this.category);
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
    toDocList(n) {
      if (n.external === EXTERNAL) {
        this.$router.push(
          `/knowledge/hitTest?knowledgeId=${n.knowledgeId}&external=${n.external}`,
        );
        return;
      }
      if (this.category === KNOWLEDGE) {
        this.$router.push({ path: `/knowledge/doclist/${n.knowledgeId}` });
      } else if (this.category === QA) {
        this.$router.push({ path: `/knowledge/qa/docList/${n.knowledgeId}` });
      } else if (this.category === DB) {
        this.$router.push({ path: `/knowledge/db/docList/${n.knowledgeId}` });
      }

      this.setPermissionType(n.permissionType);
    },
    showPowerManagement(knowledgeItem) {
      this.$refs.powerManagement.knowledgeId = knowledgeItem.knowledgeId;
      this.$refs.powerManagement.knowledgeName = knowledgeItem.knowledgeName;
      this.$refs.powerManagement.permissionType = knowledgeItem.permissionType;
      this.$refs.powerManagement.showDialog();
    },
  },
};
</script>

<style lang="scss" scoped>
@import '@/style/appCard.scss';

.app-card {
  .smart {
    height: 152px;

    .card-content {
      display: flex;
      flex-direction: column;
      flex: 1;
      min-width: 0;
    }

    .card-header {
      display: flex;
      align-items: flex-start;
      margin-bottom: 8px;

      .common-card-logo {
        flex-shrink: 0;
      }

      .header-info {
        flex: 1;
        min-width: 0;

        .name {
          margin: 0;
        }

        .desc {
          margin: 4px 0 0 0;
          padding-top: 0;
          display: flex;
          gap: 8px;
          align-items: center;

          .desc-item {
            margin-left: 0;
            display: inline-block;
          }
        }
      }
    }

    .card-description {
      margin-bottom: 8px;
      min-height: 36px;

      .card-desc {
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
