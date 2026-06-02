<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import * as noteService from '../../services/note'
import type { Note, RawNote } from '../../types'
import { normalizeNote } from '../../types'
import TFIcon from '../common/TFIcon.vue'

const props = withDefaults(defineProps<{
  videoId: number
  currentTime: number
  open: boolean
  mode?: 'inline' | 'dialog'
}>(), {
  mode: 'inline',
})

const emit = defineEmits<{
  seek: [timestamp: number]
  close: []
}>()

const dialogModel = computed({
  get: () => props.open,
  set: (val: boolean) => { if (!val) emit('close') },
})

const notes = ref<Note[]>([])
const cursor = ref<string | null>(null)
const hasMore = ref(true)
const loading = ref(false)
const sending = ref(false)
const inputText = ref('')
const inputTimestamp = ref(0)

const fetchNotes = async (reset = false) => {
  if (loading.value) return
  loading.value = true
  try {
    if (reset) {
      notes.value = []
      cursor.value = null
      hasMore.value = true
    }
    const resp = await noteService.getNotes(props.videoId, cursor.value ?? undefined, 50)
    const d = resp.data.data
    if (!d) return

    const items = (d.items ?? []).map((raw: RawNote) => normalizeNote(raw))
    if (reset) {
      notes.value = items
    } else {
      notes.value.push(...items)
    }
    cursor.value = d.next_cursor
    hasMore.value = d.has_more ?? false
  } finally {
    loading.value = false
  }
}

const sendNote = async () => {
  if (!inputText.value.trim() || sending.value) return
  sending.value = true
  const content = inputText.value.trim()
  const timestamp = inputTimestamp.value
  inputText.value = ''
  try {
    const resp = await noteService.createNote(props.videoId, timestamp, content)
    const raw = resp.data.data
    if (!raw) return
    const note = normalizeNote(raw)
    const idx = notes.value.findIndex(n => n.timestamp > note.timestamp)
    if (idx === -1) {
      notes.value.push(note)
    } else {
      notes.value.splice(idx, 0, note)
    }
  } finally {
    sending.value = false
  }
}

const deleteNoteHandler = async (note: Note) => {
  try {
    await noteService.deleteNote(props.videoId, note.note_id)
    notes.value = notes.value.filter(n => n.note_id !== note.note_id)
  } catch {}
}

const handleNoteClick = (note: Note) => {
  emit('seek', note.timestamp)
}

const captureTimestamp = () => {
  inputTimestamp.value = props.currentTime
}

const noteListEl = ref<HTMLDivElement>()
const handleScroll = () => {
  if (!noteListEl.value) return
  const el = noteListEl.value
  if (el.scrollHeight - el.scrollTop - el.clientHeight < 50 && hasMore.value) {
    fetchNotes()
  }
}

watch(() => props.open, (val) => {
  if (val) {
    inputTimestamp.value = props.currentTime
    fetchNotes(true)
  } else {
    notes.value = []
    cursor.value = null
    hasMore.value = true
    inputText.value = ''
  }
})
</script>

<template>
  <q-dialog v-if="mode === 'dialog'" v-model="dialogModel" position="right" seamless class="note-dialog">
    <q-card class="note-panel note-panel-dialog" flat>
      <div class="note-panel-inner note-panel-inner-dialog">
        <div class="panel-header">
          <div class="panel-header-bar" />
          <div class="panel-header-row">
            <span class="panel-title">时间轴笔记</span>
            <span v-if="notes.length" class="note-count">{{ notes.length }}</span>
            <div class="spacer" />
            <button class="close-btn" @click="emit('close')">
              <TFIcon name="close" :size="20" color="var(--text-secondary)" />
            </button>
          </div>
        </div>
        <div class="add-note-section">
          <div class="timestamp-row">
            <span class="timestamp-label">定位时间</span>
            <button class="timestamp-btn" @click="captureTimestamp">
              <TFIcon name="access_time" :size="16" color="var(--accent)" />
              <span class="timestamp-value">{{ inputTimestamp.toFixed(0) }}s</span>
              <span class="timestamp-hint">点击设为当前播放时间</span>
            </button>
          </div>
          <div class="add-note-row">
            <textarea v-model="inputText" :maxlength="2000" class="note-textarea" placeholder="写下你的笔记…" rows="2" @keydown.enter.exact.prevent="sendNote()" />
            <button class="send-btn" :disabled="!inputText.trim() || sending" @click="sendNote">发送</button>
          </div>
          <div class="note-char-count">{{ inputText.length }}/2000</div>
        </div>
        <div ref="noteListEl" class="note-list" @scroll="handleScroll">
          <div v-if="loading && notes.length === 0" class="loading-state">
            <q-spinner color="accent" size="24px" />
          </div>
          <div v-else-if="notes.length === 0" class="empty-state">
            <TFIcon name="sticky_note_2" :size="40" color="var(--text-muted)" />
            <p>还没有时间轴笔记</p>
          </div>
          <template v-else>
            <div v-for="note in notes" :key="note.note_id" class="note-item" @click="handleNoteClick(note)">
              <div class="note-item-left">
                <div class="note-timestamp-badge">{{ note.formatted_time }}</div>
              </div>
              <div class="note-item-right">
                <div class="note-item-header">
                  <span class="note-author">{{ note.username }}</span>
                  <span v-if="note.is_mine" class="note-mine-tag">我</span>
                </div>
                <div class="note-content">{{ note.content }}</div>
              </div>
              <button v-if="note.is_mine" class="note-delete-btn" @click.stop="deleteNoteHandler(note)">
                <TFIcon name="delete" :size="16" color="var(--text-muted)" />
              </button>
            </div>
            <div v-if="loading && notes.length > 0" class="load-more">
              <q-spinner color="accent" size="20px" />
            </div>
          </template>
        </div>
      </div>
    </q-card>
  </q-dialog>

  <div v-else class="note-panel note-panel-inline">
    <div class="note-panel-inner note-panel-inner-inline">
      <div class="panel-header">
        <div class="panel-header-row">
          <span class="panel-title">时间轴笔记</span>
          <span v-if="notes.length" class="note-count">{{ notes.length }}</span>
          <div class="spacer" />
          <button class="close-btn" @click="emit('close')">
            <TFIcon name="close" :size="20" color="var(--text-secondary)" />
          </button>
        </div>
      </div>
      <div class="add-note-section">
        <div class="timestamp-row">
          <span class="timestamp-label">定位时间</span>
          <button class="timestamp-btn" @click="captureTimestamp">
            <TFIcon name="access_time" :size="16" color="var(--accent)" />
            <span class="timestamp-value">{{ inputTimestamp.toFixed(0) }}s</span>
            <span class="timestamp-hint">点击设为当前播放时间</span>
          </button>
        </div>
        <div class="add-note-row">
          <textarea v-model="inputText" :maxlength="2000" class="note-textarea" placeholder="写下你的笔记…" rows="2" @keydown.enter.exact.prevent="sendNote()" />
          <button class="send-btn" :disabled="!inputText.trim() || sending" @click="sendNote">发送</button>
        </div>
        <div class="note-char-count">{{ inputText.length }}/2000</div>
      </div>
      <div ref="noteListEl" class="note-list" @scroll="handleScroll">
        <div v-if="loading && notes.length === 0" class="loading-state">
          <q-spinner color="accent" size="24px" />
        </div>
        <div v-else-if="notes.length === 0" class="empty-state">
          <TFIcon name="sticky_note_2" :size="40" color="var(--text-muted)" />
          <p>还没有时间轴笔记</p>
        </div>
        <template v-else>
          <div v-for="note in notes" :key="note.note_id" class="note-item" @click="handleNoteClick(note)">
            <div class="note-item-left">
              <div class="note-timestamp-badge">{{ note.formatted_time }}</div>
            </div>
            <div class="note-item-right">
              <div class="note-item-header">
                <span class="note-author">{{ note.username }}</span>
                <span v-if="note.is_mine" class="note-mine-tag">我</span>
              </div>
              <div class="note-content">{{ note.content }}</div>
            </div>
            <button v-if="note.is_mine" class="note-delete-btn" @click.stop="deleteNoteHandler(note)">
              <TFIcon name="delete" :size="16" color="var(--text-muted)" />
            </button>
          </div>
          <div v-if="loading && notes.length > 0" class="load-more">
            <q-spinner color="accent" size="20px" />
          </div>
        </template>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.note-dialog {
  align-items: stretch;
}

.note-panel {
  background: var(--bg-base);

  &.note-panel-dialog {
    border-radius: 0;
    width: 400px;
    max-width: 80vw;
    height: 100svh;
  }

  &.note-panel-inline {
    height: 100%;
  }
}

.note-panel-inner {
  display: flex;
  flex-direction: column;

  &.note-panel-inner-dialog {
    height: 100svh;
  }

  &.note-panel-inner-inline {
    height: 100%;
  }
}

.panel-header {
  padding: 8px 16px 4px;
}

.panel-header-bar {
  width: 36px;
  height: 4px;
  border-radius: 2px;
  background: var(--border-color);
  margin: 0 auto 8px;
}

.panel-header-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.panel-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--text-primary);
}

.note-count {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-hover);
  padding: 2px 8px;
  border-radius: 10px;
}

.spacer {
  flex: 1;
}

.close-btn {
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  display: flex;
}

.add-note-section {
  padding: 8px 16px 12px;
  border-bottom: 1px solid var(--border-color);
}

.timestamp-row {
  margin-bottom: 8px;
}

.timestamp-label {
  font-size: 12px;
  color: var(--text-secondary);
  margin-bottom: 4px;
  display: block;
}

.timestamp-btn {
  display: flex;
  align-items: center;
  gap: 4px;
  background: var(--bg-hover);
  border: none;
  border-radius: 8px;
  padding: 4px 10px;
  cursor: pointer;
  color: var(--text-primary);
  font-size: 13px;

  &:hover {
    background: var(--bg-active);
  }
}

.timestamp-value {
  font-weight: 600;
  color: var(--accent);
}

.timestamp-hint {
  color: var(--text-muted);
  font-size: 12px;
}

.add-note-row {
  display: flex;
  gap: 8px;
  align-items: flex-end;
}

.note-textarea {
  flex: 1;
  background: var(--bg-input);
  border: 1px solid var(--border-color);
  border-radius: 8px;
  padding: 8px 12px;
  color: var(--text-primary);
  resize: none;
  font-family: inherit;
  font-size: 14px;
  line-height: 1.4;
  outline: none;

  &:focus {
    border-color: var(--accent);
  }

  &::placeholder {
    color: var(--text-muted);
  }
}

.send-btn {
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: 8px;
  padding: 8px 16px;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;

  &:disabled {
    opacity: 0.4;
    cursor: not-allowed;
  }
}

.note-char-count {
  text-align: right;
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 4px;
}

.note-list {
  flex: 1;
  overflow-y: auto;
}

.loading-state,
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 32px 16px;
  gap: 8px;
  color: var(--text-secondary);
  font-size: 14px;
}

.note-item {
  display: flex;
  gap: 10px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border-color-light, rgba(0,0,0,0.04));
  cursor: pointer;
  transition: background 0.15s;

  &:hover {
    background: var(--bg-hover);
  }
}

.note-item-left {
  flex-shrink: 0;
}

.note-timestamp-badge {
  background: var(--accent);
  color: #fff;
  font-size: 12px;
  font-weight: 600;
  padding: 2px 8px;
  border-radius: 4px;
  font-variant-numeric: tabular-nums;
}

.note-item-right {
  flex: 1;
  min-width: 0;
}

.note-item-header {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 4px;
}

.note-author {
  font-size: 13px;
  font-weight: 600;
  color: var(--text-primary);
}

.note-mine-tag {
  font-size: 10px;
  color: var(--accent);
  background: rgba(0, 163, 255, 0.1);
  padding: 1px 6px;
  border-radius: 4px;
}

.note-content {
  font-size: 14px;
  color: var(--text-primary);
  line-height: 1.5;
  word-break: break-word;
}

.note-delete-btn {
  flex-shrink: 0;
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
  align-self: flex-start;
  opacity: 0;
  transition: opacity 0.15s;

  .note-item:hover & {
    opacity: 1;
  }

  &:hover {
    color: var(--text-negative);
  }
}

.load-more {
  display: flex;
  justify-content: center;
  padding: 12px;
}
</style>
