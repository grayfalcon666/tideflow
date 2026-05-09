<script setup lang="ts">
import { ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import type { UserInfo } from '../../types'
import * as userService from '../../services/user'
import * as messageService from '../../services/message'
import { useAuthStore } from '../../stores/auth'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  videoId: number
  videoTitle: string
  videoCover: string
  authorName: string
}>()

const modelValue = defineModel<boolean>({ default: false })
const emit = defineEmits<{ sent: [] }>()

const $q = useQuasar()
const authStore = useAuthStore()

const friends = ref<UserInfo[]>([])
const selectedIds = ref<Set<number>>(new Set())
const loading = ref(false)
const sending = ref(false)

const loadFriends = async () => {
  if (!authStore.accountId) return
  loading.value = true
  try {
    const resp = await userService.getFollowing(authStore.accountId, undefined, 200)
    const d = resp.data.data
    if (d) friends.value = d.items ?? []
  } catch {
    friends.value = []
  } finally {
    loading.value = false
  }
}

const toggleSelect = (id: number) => {
  const next = new Set(selectedIds.value)
  if (next.has(id)) {
    next.delete(id)
  } else {
    next.add(id)
  }
  selectedIds.value = next
}

const send = async () => {
  if (selectedIds.value.size === 0 || sending.value) return
  sending.value = true
  try {
    await messageService.shareVideo(
      Array.from(selectedIds.value),
      props.videoId,
      props.videoTitle,
      props.videoCover,
      props.authorName,
    )
    $q.notify({ color: 'green-8', textColor: 'white', message: `已发送给 ${selectedIds.value.size} 位好友`, position: 'top' })
    modelValue.value = false
    emit('sent')
  } catch {
    $q.notify({ color: 'red-8', textColor: 'white', message: '发送失败，请重试', position: 'top' })
  } finally {
    sending.value = false
  }
}

watch(modelValue, (open) => {
  if (open) {
    selectedIds.value = new Set()
    loadFriends()
  }
})
</script>

<template>
  <q-dialog v-model="modelValue" position="bottom" full-width class="share-dialog">
    <div class="share-panel">
      <div class="panel-header">
        <div class="drag-bar" />
        <h3 class="panel-title">分享给好友</h3>
        <button class="close-btn" @click="modelValue = false">
          <TFIcon name="close" :size="20" />
        </button>
      </div>

      <div class="panel-body">
        <div v-if="loading" class="loading-state">
          <q-spinner color="accent" size="32px" />
        </div>

        <div v-else-if="!friends.length" class="empty-state">
          <TFIcon name="people" :size="48" color="var(--text-secondary)" />
          <p>暂无关注的好友</p>
        </div>

        <div v-else class="friends-grid">
          <div
            v-for="f in friends"
            :key="f.id"
            class="friend-card"
            :class="{ selected: selectedIds.has(f.id) }"
            @click="toggleSelect(f.id)"
          >
            <div class="friend-avatar">
              <div
                class="friend-avatar-img"
                :style="{ backgroundImage: `url('${f.avatar_url || '/default-avatar.svg'}')` }"
              />
              <div v-if="selectedIds.has(f.id)" class="friend-check">
                <TFIcon name="check" :size="14" color="#fff" />
              </div>
            </div>
            <span class="friend-name">{{ f.username }}</span>
          </div>
        </div>
      </div>

      <div class="panel-footer">
        <span v-if="selectedIds.size > 0" class="selected-count">已选 {{ selectedIds.size }} 人</span>
        <button
          class="send-btn"
          :class="{ disabled: selectedIds.size === 0 || sending }"
          :disabled="selectedIds.size === 0 || sending"
          @click="send"
        >
          <TFIcon name="send" :size="18" />
          <span>发送</span>
        </button>
      </div>
    </div>
  </q-dialog>
</template>

<style scoped lang="scss">
.share-dialog {
  :deep(.q-dialog__inner--bottom) {
    padding: 0;
  }
}

.share-panel {
  background: var(--bg-surface);
  border-radius: 16px 16px 0 0;
  max-height: 70svh;
  display: flex;
  flex-direction: column;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border);
  position: relative;
}

.drag-bar {
  position: absolute;
  top: 6px;
  left: 50%;
  transform: translateX(-50%);
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--border);
}

.panel-title {
  font-size: 17px;
  font-weight: 700;
  color: var(--text-base);
  margin: 0;
}

.close-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4);
  min-height: 200px;
}

.loading-state {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 160px;
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  min-height: 160px;
  gap: var(--space-3);
  color: var(--text-secondary);
  font-size: 15px;
}

.friends-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-4);
}

.friend-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 6px;
  cursor: pointer;
  padding: var(--space-2);
  border-radius: var(--radius-lg);
  transition: background var(--transition-fast);

  &:hover {
    background: rgba(255, 255, 255, 0.05);
  }
}

.friend-avatar {
  position: relative;
  width: 56px;
  height: 56px;
}

.friend-avatar-img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  border: 2px solid transparent;
  transition: border-color var(--transition-fast);

  .friend-card.selected & {
    border-color: var(--accent);
  }
}

.friend-check {
  position: absolute;
  bottom: 0;
  right: 0;
  width: 20px;
  height: 20px;
  border-radius: 50%;
  background: var(--accent);
  display: flex;
  align-items: center;
  justify-content: center;
}

.friend-name {
  font-size: 12px;
  color: var(--text-secondary);
  text-align: center;
  max-width: 64px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.panel-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--border);
}

.selected-count {
  font-size: 13px;
  color: var(--text-secondary);
}

.send-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--accent);
  color: #000;
  font-size: 15px;
  font-weight: 700;
  border-radius: var(--radius-full);
  padding: 8px 24px;
  border: none;
  cursor: pointer;
  transition: opacity var(--transition-fast);

  &:hover:not(.disabled) {
    opacity: 0.85;
  }

  &.disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
}
</style>
