<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { useQuasar } from 'quasar'
import UserInfoCard from '../components/user/UserInfoCard.vue'
import FollowButton from '../components/user/FollowButton.vue'
import TFIcon from '../components/common/TFIcon.vue'
import UserVideoTab from '../components/user/UserVideoTab.vue'
import UserLikedTab from '../components/user/UserLikedTab.vue'
import UserSettingsTab from '../components/user/UserSettingsTab.vue'
import UserHabitTab from '../components/user/UserHabitTab.vue'
import * as userService from '../services/user'
import type { UserProfile, VideoItem } from '../types'
import { useAuthStore } from '../stores/auth'
import { useInteractionStore } from '../stores/interaction'

const route = useRoute()
const router = useRouter()
const authStore = useAuthStore()
const interactionStore = useInteractionStore()
const $q = useQuasar()

// 安全获取用户 ID，无效则重定向
const userId = computed(() => {
  const idParam = route.params.id
  if (typeof idParam === 'string') {
    const num = Number(idParam)
    if (!isNaN(num) && num > 0) return num
  }
  return null
})

const profile = ref<UserProfile | null>(null)
const activeTab = ref<'videos' | 'liked' | 'settings' | 'habit'>('videos')
const loading = ref(true)

const isMe = computed(() => authStore.accountId === userId.value)

const fetchProfile = async () => {
  const id = userId.value
  if (!id) {
    $q.notify({ color: 'red-8', textColor: 'white', message: '无效的用户ID', position: 'top' })
    router.push('/')
    return
  }

  loading.value = true
  try {
    const resp = await userService.getUser(id)
    const d = resp.data.data
    if (d) {
      profile.value = d
      // 同步关注状态到 interaction store
      if (d.is_following) {
        interactionStore.syncFollow(d.id)
      }
    } else {
      throw new Error('用户不存在')
    }
  } catch (error) {
    console.error('获取用户资料失败', error)
    $q.notify({ color: 'red-8', textColor: 'white', message: '用户不存在或网络错误', position: 'top' })
    router.push('/')
  } finally {
    loading.value = false
  }
}

onMounted(() => fetchProfile())
</script>

<template>
  <div class="user-profile-page">
    <div v-if="loading" class="loading-state">
      <q-spinner color="accent" size="48px" />
    </div>

    <template v-else-if="profile">
      <UserInfoCard
        :user="profile"
        :isMe="isMe"
        @updated="() => authStore.fetchMe()"
      />

      <div v-if="!isMe" class="profile-follow-btn">
        <FollowButton
          v-if="userId !== null"
          :userId="userId"
          :isFollowingMe="profile?.is_following_me ?? false"
        />
        <button class="message-btn" @click="router.push(`/messages/${userId}`)">
          <TFIcon name="chat_bubble" :size="18" />
          <span>私信</span>
        </button>
      </div>

      <div class="user-tabs">
        <button class="tab-btn" :class="{ active: activeTab === 'videos' }" @click="activeTab = 'videos'">
          作品
        </button>
        <button v-if="isMe || profile.likes_public" class="tab-btn" :class="{ active: activeTab === 'liked' }" @click="activeTab = 'liked'">
          {{ isMe ? '我赞过的' : '赞过的' }}
        </button>
        <button v-if="isMe" class="tab-btn" :class="{ active: activeTab === 'habit' }" @click="activeTab = 'habit'">
          学习习惯
        </button>
        <button v-if="isMe" class="tab-btn" :class="{ active: activeTab === 'settings' }" @click="activeTab = 'settings'">
          设置
        </button>
      </div>

      <UserVideoTab v-if="activeTab === 'videos' && userId !== null" :userId="userId" />
      <UserLikedTab v-if="activeTab === 'liked'" :userId="userId ?? undefined" />
      <UserHabitTab v-if="activeTab === 'habit'" />
      <UserSettingsTab v-if="activeTab === 'settings'" />
    </template>
  </div>
</template>

<style scoped lang="scss">
.user-profile-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-4);
  padding-bottom: 80px;
}

.loading-state {
  display: flex;
  justify-content: center;
  padding: 60px;
}

.profile-follow-btn {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: var(--space-3);
  margin: var(--space-4) 0;
}

.message-btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  background: var(--accent);
  color: #000;
  font-size: 15px;
  font-weight: 700;
  border-radius: var(--radius-full);
  padding: 8px 28px;
  border: none;
  cursor: pointer;
  transition: opacity var(--transition-fast);

  &:hover { opacity: 0.85; }
}

.user-tabs {
  display: flex;
  gap: var(--space-6);
  border-bottom: 1px solid var(--border);
  margin-bottom: var(--space-4);
}

.tab-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 16px;
  font-weight: 600;
  padding: var(--space-2) 0;
  cursor: pointer;
  border-bottom: 2px solid transparent;
  transition: all var(--transition-fast);

  &.active {
    color: var(--text-base);
    border-bottom-color: var(--accent);
  }
}
</style>
