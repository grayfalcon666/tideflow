<script setup lang="ts">
import type { UserProfile } from '../../types'
import { useRouter } from 'vue-router'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  user: UserProfile
  isMe: boolean
}>()

const router = useRouter()

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
          <span><strong>{{ formatCount(user.following_count) }}</strong> 关注</span>
          <span><strong>{{ formatCount(user.follower_count) }}</strong> 粉丝</span>
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

.edit-btn {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  border-radius: var(--radius-full);
  font-size: 13px;
  color: var(--text-base);
  padding: 6px 16px;
}
</style>
