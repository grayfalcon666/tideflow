<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import * as videoService from '../services/video'
import * as tagService from '../services/tag'
import type { Tag } from '../types'
import TFIcon from '../components/common/TFIcon.vue'

const router = useRouter()
const step = ref(1)
const loading = ref(false)
const error = ref('')

// Step 1: video upload
const videoFile = ref<File>()
const videoUrl = ref('')
const videoProgress = ref(0)

// Step 2: cover upload
const coverFile = ref<File>()
const coverUrl = ref('')

// Step 3: info
const title = ref('')
const description = ref('')
const selectedTags = ref<string[]>([])
const availableTags = ref<Tag[]>([])
const titleError = ref('')

// Step 4: preview
const publishLoading = ref(false)

const selectVideo = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  if (file.size > 500 * 1024 * 1024) {
    error.value = '视频最大 500MB'
    return
  }
  videoFile.value = file
  error.value = ''
}

const uploadVideo = async () => {
  if (!videoFile.value) return
  loading.value = true
  error.value = ''
  try {
    const fd = new FormData()
    fd.append('file', videoFile.value)
    // Simulate progress via axios interceptors isn't easy without ref
    // We'll use a simple progress tracking approach
    const resp = await videoService.uploadVideo(fd)
    const d = resp.data.data
    if (d) videoUrl.value = d.play_url
    step.value = 2
  } catch (e: any) {
    error.value = e.message || '上传失败'
  } finally {
    loading.value = false
  }
}

const selectCover = (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (file) {
    coverFile.value = file
    coverUrl.value = URL.createObjectURL(file)
  }
}

const skipCover = () => {
  step.value = 3
  loadTags()
}

const uploadCover = async () => {
  if (!coverFile.value) return
  loading.value = true
  try {
    const fd = new FormData()
    fd.append('file', coverFile.value)
    const resp = await videoService.uploadCover(fd)
    const d = resp.data.data
    if (d) coverUrl.value = d.cover_url
  } catch (e: any) {
    error.value = e.message || '上传失败'
  } finally {
    loading.value = false
  }
  step.value = 3
  loadTags()
}

const loadTags = async () => {
  try {
    const resp = await tagService.getHotTags()
    const d = resp.data.data
    if (d) availableTags.value = d
  } catch {}
}

const goToStep3 = async () => {
  if (coverFile.value) {
    await uploadCover()
  } else {
    step.value = 3
    loadTags()
  }
}

const toggleTag = (tag: string) => {
  const idx = selectedTags.value.indexOf(tag)
  if (idx >= 0) {
    selectedTags.value.splice(idx, 1)
  } else if (selectedTags.value.length < 10) {
    selectedTags.value.push(tag)
  }
}

const validateTitle = () => {
  if (!title.value.trim()) {
    titleError.value = '标题不能为空'
  } else if (title.value.length > 255) {
    titleError.value = '标题最长255字符'
  } else {
    titleError.value = ''
  }
  return !titleError.value
}

const goToStep4 = () => {
  if (!validateTitle()) return
  step.value = 4
}

const publish = async () => {
  publishLoading.value = true
  try {
    const resp = await videoService.publishVideo({
      title: title.value.trim(),
      description: description.value.trim() || undefined,
      play_url: videoUrl.value,
      cover_url: coverUrl.value || undefined,
      tags: selectedTags.value.length ? selectedTags.value : undefined,
    })
    const d = resp.data.data
    if (d) router.push(`/video/${d.video_id}`)
  } catch (e: any) {
    error.value = e.message || '发布失败'
  } finally {
    publishLoading.value = false
  }
}
</script>

<template>
  <div class="publish-page">
    <!-- Step indicator -->
    <div class="steps-bar">
      <div v-for="n in 4" :key="n" class="step-dot" :class="{ active: step >= n, current: step === n }">
        <span>{{ n }}</span>
      </div>
    </div>

    <!-- Step 1: Upload video -->
    <div v-if="step === 1" class="step-content">
      <h2>上传视频</h2>
      <div class="upload-zone" :class="{ 'has-file': videoFile }">
        <input type="file" accept=".mp4" @change="selectVideo" />
        <div v-if="!videoFile" class="upload-hint">
          <TFIcon name="videocam" :size="48" color="var(--text-secondary)" />
          <p>点击或拖拽上传 .mp4 视频，最大 500MB</p>
        </div>
        <div v-else class="file-selected">
          <TFIcon name="check_circle" :size="48" color="var(--accent)" />
          <p>{{ videoFile.name }}</p>
        </div>
      </div>
      <p v-if="error" class="error-msg">{{ error }}</p>
      <q-btn v-if="videoFile" class="next-btn" no-caps label="下一步" color="primary" @click="uploadVideo" :loading="loading" />
    </div>

    <!-- Step 2: Upload cover -->
    <div v-if="step === 2" class="step-content">
      <h2>上传封面 <span class="optional">(可跳过)</span></h2>
      <div class="upload-zone" :class="{ 'has-file': coverFile }">
        <input type="file" accept="image/*" @change="selectCover" />
        <div v-if="!coverFile" class="upload-hint">
          <TFIcon name="image" :size="48" color="var(--text-secondary)" />
          <p>点击上传封面图片</p>
        </div>
        <img v-else :src="coverUrl" class="cover-preview" />
      </div>
      <div class="step-btns">
        <q-btn flat no-caps label="跳过" @click="skipCover" />
        <q-btn no-caps label="下一步" color="primary" :disabled="!coverFile" @click="goToStep3" />
      </div>
    </div>

    <!-- Step 3: Info -->
    <div v-if="step === 3" class="step-content">
      <h2>填写信息</h2>
      <q-input
        v-model="title"
        label="标题 *"
        outlined
        dark
        :error="!!titleError"
        :error-message="titleError"
        maxlength="255"
        @blur="validateTitle"
        class="form-input"
      />
      <q-input
        v-model="description"
        label="描述"
        outlined
        dark
        type="textarea"
        maxlength="500"
        autogrow
        class="form-input"
      />
      <div class="tags-section">
        <p class="tags-label">标签（最多10个）</p>
        <div class="tags-grid">
          <span
            v-for="tag in availableTags"
            :key="tag.name"
            class="tag-item"
            :class="{ selected: selectedTags.includes(tag.name) }"
            @click="toggleTag(tag.name)"
          ># {{ tag.name }}</span>
        </div>
      </div>
      <q-btn class="next-btn" no-caps label="下一步" color="primary" @click="goToStep4" />
    </div>

    <!-- Step 4: Preview -->
    <div v-if="step === 4" class="step-content">
      <h2>预览与发布</h2>
      <div class="preview-card">
        <div class="preview-cover">
          <img v-if="coverUrl" :src="coverUrl" />
          <div v-else class="cover-placeholder" />
        </div>
        <div class="preview-info">
          <h3>{{ title }}</h3>
          <p v-if="description">{{ description }}</p>
          <div v-if="selectedTags.length" class="preview-tags">
            <span v-for="t in selectedTags" :key="t" class="tag-chip"># {{ t }}</span>
          </div>
        </div>
      </div>
      <p v-if="error" class="error-msg">{{ error }}</p>
      <div class="step-btns">
        <q-btn flat no-caps label="上一步" @click="step = 3" />
        <q-btn no-caps label="发布" color="primary" @click="publish" :loading="publishLoading" />
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.publish-page {
  max-width: 600px;
  margin: 0 auto;
  padding: var(--space-6);
  padding-bottom: 100px;
}

.steps-bar {
  display: flex;
  justify-content: center;
  gap: var(--space-4);
  margin-bottom: var(--space-8);
}

.step-dot {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--bg-elevated);
  border: 2px solid var(--border);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  font-weight: 700;
  color: var(--text-secondary);
  transition: all var(--transition-fast);

  &.active {
    background: var(--accent);
    border-color: var(--accent);
    color: #000;
  }
}

.step-content {
  h2 {
    margin-bottom: var(--space-6);
    font-size: 20px;

    .optional {
      font-size: 14px;
      font-weight: 400;
      color: var(--text-secondary);
    }
  }
}

.upload-zone {
  position: relative;
  border: 2px dashed var(--border);
  border-radius: var(--radius-md);
  padding: var(--space-8);
  text-align: center;
  cursor: pointer;
  transition: border-color var(--transition-fast);
  margin-bottom: var(--space-4);

  &:hover, &.has-file {
    border-color: var(--accent);
  }

  input[type="file"] {
    position: absolute;
    inset: 0;
    opacity: 0;
    cursor: pointer;
  }
}

.upload-hint, .file-selected {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-secondary);
  pointer-events: none;
}

.cover-preview {
  max-width: 100%;
  max-height: 200px;
  border-radius: var(--radius-md);
  object-fit: cover;
}

.form-input {
  margin-bottom: var(--space-4);
}

.tags-section {
  margin-bottom: var(--space-4);
}

.tags-label {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: var(--space-2);
}

.tags-grid {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.tag-item {
  font-size: 13px;
  padding: 4px 10px;
  border-radius: var(--radius-full);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);

  &.selected {
    background: var(--accent-dim);
    border-color: var(--accent);
    color: var(--accent);
  }
}

.step-btns {
  display: flex;
  gap: var(--space-3);
  justify-content: center;
  margin-top: var(--space-4);
}

.next-btn {
  width: 100%;
  margin-top: var(--space-4);
  border-radius: var(--radius-pill);
}

.error-msg {
  color: var(--text-negative);
  font-size: 14px;
  text-align: center;
}

.preview-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  overflow: hidden;
  margin-bottom: var(--space-4);
}

.preview-cover {
  aspect-ratio: 16 / 9;
  background: var(--bg-elevated);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.cover-placeholder {
  width: 100%;
  height: 100%;
  background: var(--bg-elevated);
}

.preview-info {
  padding: var(--space-4);

  h3 { margin-bottom: var(--space-2); }
  p { color: var(--text-secondary); font-size: 14px; margin-bottom: var(--space-2); }
}

.preview-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);

  .tag-chip {
    font-size: 12px;
    padding: 2px 8px;
    background: rgba(255,255,255,0.1);
    border-radius: var(--radius-full);
    color: var(--text-base);
  }
}
</style>
