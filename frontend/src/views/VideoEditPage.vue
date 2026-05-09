<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import { useAuthStore } from '../stores/auth'
import * as videoService from '../services/video'
import type { VideoDetail } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()
const $q = useQuasar()
const authStore = useAuthStore()

const videoId = Number(route.params.id)
const video = ref<VideoDetail | null>(null)
const loading = ref(true)
const saving = ref(false)
const deleting = ref(false)

const title = ref('')
const description = ref('')
const coverUrl = ref('')
const tagsStr = ref('')
const titleError = ref('')

const fetchVideo = async () => {
  loading.value = true
  try {
    const resp = await videoService.getVideo(videoId)
    const d = resp.data.data
    if (!d) throw new Error('视频不存在')
    // Validate ownership
    if (d.author.id !== authStore.accountId) {
      $q.notify({ color: 'red-8', textColor: 'white', message: '无权编辑此视频', position: 'top' })
      router.push(`/video/${videoId}`)
      return
    }
    video.value = d
    title.value = d.title
    description.value = d.description ?? ''
    coverUrl.value = d.cover_url
    tagsStr.value = (d.tags ?? []).join(', ')
  } catch {
    $q.notify({ color: 'red-8', textColor: 'white', message: '加载视频信息失败', position: 'top' })
    router.push(`/video/${videoId}`)
  } finally {
    loading.value = false
  }
}

const selectCover = async (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 10 * 1024 * 1024) {
    $q.notify({ color: 'red-8', textColor: 'white', message: '封面最大 10MB', position: 'top' })
    return
  }
  const fd = new FormData()
  fd.append('file', file)
  try {
    const resp = await videoService.uploadCover(fd)
    const d = resp.data.data
    if (d) coverUrl.value = d.cover_url
  } catch {
    $q.notify({ color: 'red-8', textColor: 'white', message: '封面上传失败', position: 'top' })
  }
}

const validate = (): boolean => {
  titleError.value = ''
  if (!title.value.trim()) {
    titleError.value = '标题不能为空'
    return false
  }
  if (title.value.trim().length > 255) {
    titleError.value = '标题不能超过 255 字'
    return false
  }
  return true
}

const save = async () => {
  if (!validate() || saving.value) return
  saving.value = true
  try {
    const tags = tagsStr.value
      .split(/[,，]/)
      .map((t) => t.trim())
      .filter(Boolean)
    await videoService.updateVideo(videoId, {
      title: title.value.trim(),
      description: description.value.trim() || undefined,
      cover_url: coverUrl.value || undefined,
      tags: tags.length > 0 ? tags : undefined,
    })
    $q.notify({ color: 'green-8', textColor: 'white', message: '保存成功', position: 'top' })
    router.push(`/video/${videoId}`)
  } catch {
    $q.notify({ color: 'red-8', textColor: 'white', message: '保存失败', position: 'top' })
  } finally {
    saving.value = false
  }
}

const confirmDelete = () => {
  $q.dialog({
    title: '确认删除',
    message: '删除后无法恢复，确定删除该视频？',
    cancel: true,
    persistent: true,
    dark: true,
  }).onOk(async () => {
    deleting.value = true
    try {
      await videoService.deleteVideo(videoId)
      $q.notify({ color: 'green-8', textColor: 'white', message: '已删除', position: 'top' })
      router.push(`/u/${authStore.accountId}`)
    } catch {
      $q.notify({ color: 'red-8', textColor: 'white', message: '删除失败', position: 'top' })
    } finally {
      deleting.value = false
    }
  })
}

onMounted(fetchVideo)
</script>

<template>
  <div class="edit-page">
    <div class="edit-header">
      <button class="back-btn" @click="router.back()">
        <TFIcon name="arrow_back" :size="24" />
      </button>
      <h3>编辑视频</h3>
    </div>

    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="32px" />
    </div>

    <div v-else-if="video" class="edit-form">
      <div class="cover-section">
        <div
          class="cover-preview"
          :style="{ backgroundImage: `url('${coverUrl}')` }"
          @click="($refs.coverInput as HTMLInputElement).click()"
        >
          <div class="cover-overlay">
            <TFIcon name="image" :size="24" />
            <span>更换封面</span>
          </div>
        </div>
        <input ref="coverInput" type="file" accept="image/*" hidden @change="selectCover" />
      </div>

      <div class="form-group">
        <label class="form-label">标题</label>
        <q-input
          v-model="title"
          dark
          outlined
          dense
          placeholder="视频标题"
          :error="!!titleError"
          :error-message="titleError"
          class="form-input"
          maxlength="255"
        />
      </div>

      <div class="form-group">
        <label class="form-label">描述</label>
        <q-input
          v-model="description"
          dark
          outlined
          dense
          type="textarea"
          placeholder="视频描述（可选）"
          class="form-input"
          autogrow
          maxlength="1000"
        />
      </div>

      <div class="form-group">
        <label class="form-label">标签</label>
        <q-input
          v-model="tagsStr"
          dark
          outlined
          dense
          placeholder="用逗号分隔，如：搞笑, 音乐, 舞蹈"
          class="form-input"
        />
      </div>

      <div class="edit-actions">
        <button class="save-btn" :disabled="saving" @click="save">
          <TFIcon name="check" :size="18" />
          <span>{{ saving ? '保存中...' : '保存' }}</span>
        </button>

        <button class="delete-btn" :disabled="deleting" @click="confirmDelete">
          <TFIcon name="delete" :size="18" />
          <span>{{ deleting ? '删除中...' : '删除视频' }}</span>
        </button>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.edit-page {
  min-height: 100svh;
  background: var(--bg-base);
  display: flex;
  flex-direction: column;
}

.edit-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--bg-surface);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;

  h3 {
    margin: 0;
    font-size: 18px;
    font-weight: 700;
    color: var(--text-base);
  }
}

.back-btn {
  background: none;
  border: none;
  color: var(--text-secondary);
  cursor: pointer;
  padding: 4px;
  display: flex;
  align-items: center;

  &:hover { color: var(--text-base); }
}

.loading-state {
  flex: 1;
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 200px;
}

.edit-form {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  max-width: 480px;
  margin: 0 auto;
  width: 100%;
}

.cover-section {
  display: flex;
  justify-content: center;
}

.cover-preview {
  width: 160px;
  height: 210px;
  background-size: cover;
  background-position: center;
  border-radius: var(--radius-md);
  cursor: pointer;
  position: relative;
  overflow: hidden;
}

.cover-overlay {
  position: absolute;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 4px;
  color: #fff;
  font-size: 13px;
  opacity: 0;
  transition: opacity var(--transition-fast);

  .cover-preview:hover & {
    opacity: 1;
  }
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.form-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}

.form-input {
  :deep(.q-field__control) {
    background: var(--bg-elevated);
    border-radius: var(--radius-md);
  }
}

.edit-actions {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--border);
}

.save-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: var(--accent);
  color: #000;
  font-size: 16px;
  font-weight: 700;
  border-radius: var(--radius-full);
  padding: 12px 24px;
  border: none;
  cursor: pointer;
  transition: opacity var(--transition-fast);

  &:hover:not(:disabled) { opacity: 0.85; }
  &:disabled { opacity: 0.4; cursor: not-allowed; }
}

.delete-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  background: transparent;
  color: #ef4444;
  font-size: 14px;
  font-weight: 600;
  border-radius: var(--radius-full);
  padding: 10px 24px;
  border: 1px solid #ef4444;
  cursor: pointer;
  transition: background var(--transition-fast), color var(--transition-fast);

  &:hover:not(:disabled) {
    background: #ef4444;
    color: #fff;
  }
  &:disabled { opacity: 0.4; cursor: not-allowed; }
}
</style>
