<script setup lang="ts">
import { ref, watch } from 'vue'
import type { UserProfile, UserInfo } from '../../types'
import { useRouter } from 'vue-router'
import * as userService from '../../services/user'
import * as videoService from '../../services/video'
import { useAuthStore } from '../../stores/auth'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  user: UserProfile
  isMe: boolean
}>()
const emit = defineEmits<{
  (e: 'updated', user: UserInfo): void
}>()

const router = useRouter()
const authStore = useAuthStore()

const followingMenuOpen = ref(false)
const followingUsers = ref<UserInfo[]>([])
const followingLoading = ref(false)
let followingLoaded = false

const followersMenuOpen = ref(false)
const followersUsers = ref<UserInfo[]>([])
const followersLoading = ref(false)
let followersLoaded = false

// Edit dialog
const editDialogOpen = ref(false)
const editUsername = ref('')
const editBio = ref('')
const editSaving = ref(false)
const avatarInputRef = ref<HTMLInputElement>()
const avatarUploading = ref(false)

const triggerAvatarUpload = () => {
  avatarInputRef.value?.click()
}

const onAvatarFileChange = async (e: Event) => {
  const file = (e.target as HTMLInputElement).files?.[0]
  if (!file) return
  avatarUploading.value = true
  try {
    const fd = new FormData()
    fd.append('file', file)
    const resp = await videoService.uploadAvatar(fd)
    const url = resp.data.data?.cover_url
    if (url) {
      await userService.updateMe({ avatar_url: url })
      authStore.fetchMe()
      emit('updated', { ...props.user, avatar_url: url } as any)
    }
  } catch (err: any) {
    console.error('[Avatar] 上传失败:', err?.response?.data ?? err.message)
  } finally {
    avatarUploading.value = false
    if (avatarInputRef.value) avatarInputRef.value.value = ''
  }
}

const openEditDialog = () => {
  editUsername.value = props.user.username ?? ''
  editBio.value = props.user.bio ?? ''
  editDialogOpen.value = true
}

watch(() => props.user.username, (v) => { editUsername.value = v ?? '' })
watch(() => props.user.bio, (v) => { editBio.value = v ?? '' })

const saveEdit = async () => {
  if (editSaving.value) return
  editSaving.value = true
  try {
    if (editBio.value !== props.user.bio) {
      await userService.updateMe({ bio: editBio.value || undefined })
    }
    if (editUsername.value !== props.user.username) {
      await userService.updateUsername(editUsername.value)
    }
    authStore.fetchMe()
    editDialogOpen.value = false
  } catch (err: any) {
    console.error('[EditDialog] 请求失败:', err?.response?.data ?? err.message)
  } finally {
    editSaving.value = false
  }
}

const loadFollowing = async () => {
  if (followingLoaded) return
  followingLoaded = true
  followingLoading.value = true
  try {
    const resp = await userService.getFollowing(props.user.id)
    const d = resp.data.data
    if (d) followingUsers.value = d.items ?? []
  } catch {
    followingLoaded = false
  } finally {
    followingLoading.value = false
  }
}

const loadFollowers = async () => {
  if (followersLoaded) return
  followersLoaded = true
  followersLoading.value = true
  try {
    const resp = await userService.getFollowers(props.user.id)
    const d = resp.data.data
    if (d) followersUsers.value = d.items ?? []
  } catch {
    followersLoaded = false
  } finally {
    followersLoading.value = false
  }
}

const goToUser = (id: number) => {
  router.push(`/u/${id}`)
}

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}
</script>

<template>
  <div class="user-info-card">
    <div class="card-main">
      <q-avatar size="min(80px, 20vw)" class="avatar" @click="props.isMe ? triggerAvatarUpload() : router.push(`/u/${user.id}`)">
        <div
          class="avatar-img"
          :style="{ backgroundImage: `url('${user.avatar_url || '/default-avatar.svg'}')` }"
        />
        <input
          ref="avatarInputRef"
          type="file"
          accept="image/*"
          class="hidden-input"
          @change="onAvatarFileChange"
        />
      </q-avatar>
      <div class="user-info">
        <div class="username-row">
          <h2 class="username">{{ user.username }}</h2>
          <TFIcon v-if="user.is_big_v" name="verified" :size="20" color="orange" />
          <q-btn
            v-if="props.isMe"
            flat
            dense
            class="edit-btn"
            @click="openEditDialog"
          >
            <TFIcon name="edit" :size="18" />
          </q-btn>
        </div>
        <p v-if="user.bio" class="user-bio">{{ user.bio }}</p>
        <div class="stats-row">
          <q-btn-dropdown
            flat
            dense
            no-caps
            class="stats-btn"
            v-model="followingMenuOpen"
            @update:model-value="(open: boolean) => open && loadFollowing()"
            dropdown-icon=""
          >
            <template #label>
              <span><strong>{{ formatCount(user.following_count) }}</strong> 关注</span>
              <TFIcon name="expand_more" :size="14" />
            </template>
            <q-list class="following-dropdown">
              <div class="dropdown-header">关注列表</div>
              <div v-if="followingLoading" class="dropdown-loading">
                <q-spinner color="accent" size="24px" />
              </div>
              <div v-else-if="!followingUsers.length" class="dropdown-empty">暂无关注</div>
              <q-item
                v-for="u in followingUsers"
                :key="u.id"
                clickable
                v-close-popup
                @click="goToUser(u.id)"
                class="following-item"
              >
                <q-item-section avatar>
                  <q-avatar size="36px">
                    <div
                      class="avatar-sm"
                      :style="{ backgroundImage: `url('${u.avatar_url || '/default-avatar.svg'}')` }"
                    />
                  </q-avatar>
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ u.username }}</q-item-label>
                </q-item-section>
              </q-item>
            </q-list>
          </q-btn-dropdown>
          <q-btn-dropdown
            flat
            dense
            no-caps
            class="stats-btn"
            v-model="followersMenuOpen"
            @update:model-value="(open: boolean) => open && loadFollowers()"
            dropdown-icon=""
          >
            <template #label>
              <span><strong>{{ formatCount(user.follower_count) }}</strong> 粉丝</span>
              <TFIcon name="expand_more" :size="14" />
            </template>
            <q-list class="following-dropdown">
              <div class="dropdown-header">粉丝列表</div>
              <div v-if="followersLoading" class="dropdown-loading">
                <q-spinner color="accent" size="24px" />
              </div>
              <div v-else-if="!followersUsers.length" class="dropdown-empty">暂无粉丝</div>
              <q-item
                v-for="u in followersUsers"
                :key="u.id"
                clickable
                v-close-popup
                @click="goToUser(u.id)"
                class="following-item"
              >
                <q-item-section avatar>
                  <q-avatar size="36px">
                    <div
                      class="avatar-sm"
                      :style="{ backgroundImage: `url('${u.avatar_url || '/default-avatar.svg'}')` }"
                    />
                  </q-avatar>
                </q-item-section>
                <q-item-section>
                  <q-item-label>{{ u.username }}</q-item-label>
                </q-item-section>
              </q-item>
            </q-list>
          </q-btn-dropdown>
        </div>
      </div>
    </div>

    <!-- Edit Profile Dialog -->
    <q-dialog v-model="editDialogOpen" class="edit-dialog">
      <div class="edit-dialog-inner">
        <div class="edit-dialog-header">
          <span>编辑资料</span>
          <button class="close-btn" @click="editDialogOpen = false">
            <TFIcon name="close" :size="20" />
          </button>
        </div>
        <div class="edit-dialog-body">
          <div class="field-group">
            <label>用户名</label>
            <input v-model="editUsername" placeholder="输入用户名" class="edit-input" />
          </div>
          <div class="field-group">
            <label>个人简介</label>
            <textarea v-model="editBio" placeholder="输入简介" class="edit-textarea" rows="3" />
          </div>
        </div>
        <div class="edit-dialog-footer">
          <button class="cancel-btn" @click="editDialogOpen = false">取消</button>
          <button class="save-btn" :disabled="editSaving" @click="saveEdit">
            {{ editSaving ? '保存中...' : '保存' }}
          </button>
        </div>
      </div>
    </q-dialog>
  </div>
</template>

<style scoped lang="scss">
.user-info-card {
  background: var(--bg-card);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.card-main {
  display: flex;
  gap: var(--space-4);
  align-items: center;
}

.avatar {
  cursor: pointer;
  flex-shrink: 0;
  border: 2px solid var(--border);
  background: transparent !important;
  overflow: hidden;

  :deep(.q-avatar__content) {
    padding: 0 !important;
  }
}

.avatar-img {
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  display: block;
}

.user-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.username-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.username {
  font-size: 20px;
  font-weight: 700;
  margin: 0;
}

.user-bio {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.stats-row {
  display: flex;
  gap: var(--space-4);
  font-size: 14px;
  color: var(--text-secondary);

  strong {
    color: var(--text-base);
    font-weight: 700;
  }
}

.stats-btn {
  padding: 0;
  min-height: unset;
  font-size: 14px;
  color: var(--text-secondary);

  :deep(.q-btn__content) {
    gap: 2px;
  }

  :deep(.q-btn-dropdown__arrow) {
    display: none;
  }
}

.following-dropdown {
  min-width: 220px;
  background: var(--bg-elevated);
}

.dropdown-header {
  padding: var(--space-3) var(--space-4);
  font-size: 13px;
  font-weight: 600;
  color: var(--text-secondary);
  border-bottom: 1px solid var(--border);
}

.dropdown-loading,
.dropdown-empty {
  padding: var(--space-4);
  display: flex;
  justify-content: center;
  color: var(--text-secondary);
  font-size: 13px;
}

.following-item {
  min-height: 48px;

  .avatar-sm {
    width: 100%;
    height: 100%;
    background-size: cover;
    background-position: center;
    border-radius: 50%;
  }
}

.edit-btn {
  color: var(--text-secondary);
  padding: 4px;
}

.edit-dialog {
  align-items: flex-end;
  justify-content: center;

  :deep(.q-dialog__inner) {
    padding: 0;
  }
}

.edit-dialog-inner {
  width: 100%;
  max-width: 480px;
  background: var(--bg-surface);
  border-radius: var(--radius-lg) var(--radius-lg) 0 0;
  padding: var(--space-4);
  box-sizing: border-box;
}

.edit-dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
  font-size: 16px;
  font-weight: 700;
  color: var(--text-base);

  .close-btn {
    background: none;
    border: none;
    cursor: pointer;
    color: var(--text-secondary);
    padding: 4px;
  }
}

.edit-dialog-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.field-group {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);

  label {
    font-size: 13px;
    color: var(--text-secondary);
    font-weight: 600;
  }
}

.edit-input,
.edit-textarea {
  width: 100%;
  padding: var(--space-3);
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  color: var(--text-base);
  font-size: 14px;
  box-sizing: border-box;
  outline: none;
  resize: none;

  &:focus {
    border-color: var(--accent);
  }
}

.edit-dialog-footer {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-4);

  .cancel-btn,
  .save-btn {
    flex: 1;
    padding: var(--space-3);
    border-radius: var(--radius-md);
    font-size: 14px;
    font-weight: 600;
    cursor: pointer;
    border: none;
  }

  .cancel-btn {
    background: var(--bg-elevated);
    color: var(--text-secondary);
  }

  .save-btn {
    background: var(--accent);
    color: #000;

    &:disabled {
      opacity: 0.5;
      cursor: not-allowed;
    }
  }
}
</style>
