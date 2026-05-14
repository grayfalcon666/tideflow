<script setup lang="ts">
import { ref, onMounted } from 'vue'
import type { VocabList } from '../../types'
import * as learnService from '../../services/learn'
import TFIcon from '../common/TFIcon.vue'

const emit = defineEmits<{
  select: [list: VocabList]
}>()

const lists = ref<VocabList[]>([])
const loading = ref(true)
const error = ref('')

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const resp = await learnService.getVocabLists()
    lists.value = resp.data.data ?? []
  } catch {
    error.value = '加载词表失败，请稍后重试'
  } finally {
    loading.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="select-list">
    <div class="step-header">
      <TFIcon name="school" :size="20" color="#1ed760" />
      <span>选择考纲词表</span>
    </div>

    <div v-if="loading" class="skeleton-list">
      <div v-for="i in 4" :key="i" class="skeleton-item">
        <q-skeleton type="text" width="60%" />
        <q-skeleton type="text" width="30%" />
      </div>
    </div>

    <div v-else-if="error" class="state-error">
      <TFIcon name="error_outline" :size="32" color="#f3727f" />
      <span>{{ error }}</span>
      <button class="retry-btn" @click="load">重试</button>
    </div>

    <div v-else-if="lists.length === 0" class="state-empty">
      <span>暂无可用考纲词表</span>
    </div>

    <q-list v-else class="vocab-list" sparse>
      <q-item
        v-for="list in lists"
        :key="list.id"
        clickable
        class="vocab-item"
        @click="emit('select', list)"
      >
        <q-item-section>
          <q-item-label class="item-name">{{ list.name }}</q-item-label>
          <q-item-label caption class="item-total">{{ list.total }} 词</q-item-label>
        </q-item-section>
        <q-item-section side>
          <TFIcon name="chevron_right" :size="20" color="#b3b3b3" />
        </q-item-section>
      </q-item>
    </q-list>
  </div>
</template>

<style scoped lang="scss">
.select-list {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.step-header {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 16px;
  font-weight: 600;
  color: #fff;
  padding: 0 4px;
}

.skeleton-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.skeleton-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px;
  background: #181818;
  border-radius: 8px;
}

.state-error,
.state-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  padding: 48px 16px;
  color: #b3b3b3;
  font-size: 14px;
}

.retry-btn {
  background: #1f1f1f;
  color: #fff;
  border: none;
  border-radius: 9999px;
  padding: 8px 20px;
  font-size: 13px;
  font-weight: 600;
  letter-spacing: 1px;
  text-transform: uppercase;
  cursor: pointer;
  &:hover { background: #2a2a2a; }
}

.vocab-list {
  background: transparent !important;
}

.vocab-item {
  background: #181818;
  border-radius: 8px;
  margin-bottom: 8px;
  padding: 14px 16px;
  transition: background 0.15s;

  &:hover {
    background: #252525;
  }

  &:last-child { margin-bottom: 0; }
}

.item-name {
  font-size: 15px;
  font-weight: 600;
  color: #fff;
}

.item-total {
  font-size: 12px;
  color: #b3b3b3;
}
</style>