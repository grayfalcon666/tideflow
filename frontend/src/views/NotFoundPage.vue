<script setup lang="ts">
import { useRouter } from 'vue-router'
import TFIcon from '../components/common/TFIcon.vue'

const router = useRouter()
</script>

<template>
  <div class="not-found-page">
    <div class="content">
      <TFIcon name="search_off" :size="80" color="var(--text-secondary)" />
      <h1>404</h1>
      <p>页面不存在</p>
      <q-btn
        no-caps
        label="返回首页"
        color="primary"
        @click="router.push('/')"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.not-found-page {
  min-height: 100svh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--bg-base);
}

.content {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  color: var(--text-secondary);

  h1 {
    font-size: 64px;
    font-weight: 700;
    color: var(--text-base);
  }

  p {
    font-size: 18px;
  }
}
</style>
