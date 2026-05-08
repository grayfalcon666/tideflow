<script setup lang="ts">
import { ref } from 'vue'
import type { UserProfile, UserInfo } from '../../types'
import { useRouter } from 'vue-router'
import * as userService from '../../services/user'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  user: UserProfile
  isMe: boolean
}>()

const router = useRouter()
const followingMenuOpen = ref(false)
const followingUsers = ref<UserInfo[]>([])
const followingLoading = ref(false)
let followingLoaded = false

const followersMenuOpen = ref(false)
const followersUsers = ref<UserInfo[]>([])
const followersLoading = ref(false)
let followersLoaded = false

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
      <q-avatar size="80px" class="avatar" @click="router.push(`/u/${user.id}`)">
        <div
          class="avatar-img"
          :style="{ backgroundImage: `url('${user.avatar_url || '/default-avatar.svg'}')` }"
        />
      </q-avatar>
      <div class="user-info">
        <div class="username-row">
          <h2 class="username">{{ user.username }}</h2>
          <TFIcon v-if="user.is_big_v" name="verified" :size="20" color="orange" />
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
          <span><strong>{{ formatCount(user.video_count) }}</strong> 作品</span>
        </div>
      </div>
    </div>
    <q-btn
      v-if="props.isMe"
      flat
      no-caps
      label="编辑资料"
      class="edit-btn"
    />
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
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  font-size: 13px;
  color: var(--text-base);
  padding: 6px 16px;
}
</style>
