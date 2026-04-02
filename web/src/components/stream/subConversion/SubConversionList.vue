<template>
  <div class="sub-conversion-list">
    <sub-conversion
      v-for="item in displayList"
      :key="item.id"
      :conversion="item"
      :parents-index="parentsIndex"
      :all-sub-conversions="allSubConversions"
      @toggle-conversion="$emit('toggle-conversion', $event)"
      @collapse-click="
        $emit('collapse-click', arguments[0], arguments[1], arguments[2])
      "
    />
  </div>
</template>

<script>
export default {
  name: 'SubConversionList',
  components: {
    // 异步加载以支持循环递归引用
    SubConversion: () => import('./index.vue'),
  },
  props: {
    parentId: {
      type: String,
      default: '',
    },
    allSubConversions: {
      type: Array,
      default: () => [],
    },
    parentsIndex: {
      type: Number,
      required: true,
    },
  },
  computed: {
    displayList() {
      return (this.allSubConversions || [])
        .filter(item => (item.parentId || '') === (this.parentId || ''))
        .sort((a, b) => (a.innerOrder || 0) - (b.innerOrder || 0));
    },
  },
};
</script>

<style scoped lang="scss">
.sub-conversion-list {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 10px;
  margin-top: 10px;
  background: #fff;
  border-radius: 6px;
}
</style>
