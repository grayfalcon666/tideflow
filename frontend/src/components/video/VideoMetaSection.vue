<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import type { VideoDetail } from '../../types'
import * as videoService from '../../services/video'
import * as userService from '../../services/user'
import { useAuthStore } from '../../stores/auth'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  video: VideoDetail
}>()

const router = useRouter()
const authStore = useAuthStore()
const liked = ref(props.video.is_liked)
const likesCount = ref(props.video.likes_count)
const following = ref(props.video.is_following_author)

const isMe = () => authStore.accountId === props.video.author.id

const toggleLike = async () => {
  if (!authStore.isLoggedIn) {
    router.push('/account')
    return
  }
  const was = liked.value
  liked.value = !was
  likesCount.value += was ? -1 : 1
  try {
    if (was) {
      const r = await videoService.unlikeVideo(props.video.video_id)
      likesCount.value = r.data.data!.likes_count
    } else {
      const r = await videoService.likeVideo(props.video.video_id)
      likesCount.value = r.data.data!.likes_count
    }
  } catch {
    liked.value = was
    likesCount.value += was ? 1 : -1
  }
}

const toggleFollow = async () => {
  if (!authStore.isLoggedIn) {
    router.push('/account')
    return
  }
  if (isMe()) return
  const was = following.value
  following.value = !was
  try {
    if (was) {
      await userService.unfollow(props.video.author.id)
    } else {
      await userService.follow(props.video.author.id)
    }
  } catch {
    following.value = was
  }
}

const goToTag = (tag: string) => router.push(`/tag/${encodeURIComponent(tag)}`)

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

const share = async () => {
  const url = window.location.href
  if (navigator.share) {
    await navigator.share({ url, title: props.video.title })
  } else {
    await navigator.clipboard.writeText(url)
  }
}
</script>

<template>
  <div class="video-meta-section">
    <div class="author-row">
      <q-avatar size="44px" @click="router.push(`/u/${video.author.id}`)" class="author-avatar">
        <div
          class="author-avatar-img"
          :style="{ backgroundImage: `url('${video.author.avatar_url || '/default-avatar.svg'}')` }"
        />
      </q-avatar>
      <div class="author-info" @click="router.push(`/u/${video.author.id}`)">
        <span class="author-name">
          {{ video.author.username }}
          <TFIcon v-if="video.author.is_big_v" name="verified" :size="16" color="orange" />
        </span>
        <span class="author-followers">{{ formatCount(video.author.follower_count) }} 粉丝</span>
      </div>
      <q-btn
        v-if="!isMe()"
        flat
        no-caps
        dense
        :label="following ? '已关注' : '关注'"
        :class="['follow-btn', { 'follow-btn--following': following }]"
        @click.stop="toggleFollow"
      />
    </div>

    <h3 class="video-title">{{ video.title }}</h3>
    <p v-if="video.description" class="video-desc">{{ video.description }}</p>

    <div v-if="video.tags?.length" class="tags-row">
      <span
        v-for="tag in video.tags"
        :key="tag"
        class="tag-chip"
        @click="goToTag(tag)"
      ># {{ tag }}</span>
    </div>

    <div class="stats-row">
      <span><TFIcon name="favorite" :size="16" /> {{ formatCount(likesCount) }}</span>
      <span><TFIcon name="chat_bubble_outline" :size="16" /> {{ formatCount(video.comment_count) }}</span>
      <span><TFIcon name="visibility" :size="16" /> {{ formatCount(video.views_count) }}</span>
    </div>

    <div class="action-row">
      <div class="action-btn" @click="toggleLike">
        <TFIcon :name="liked ? 'favorite' : 'favorite_border'" :size="24" :color="liked ? 'var(--accent)' : 'currentColor'" />
        <span>{{ liked ? '已赞' : '赞' }}</span>
      </div>
      <div class="action-btn" @click="share">
        <TFIcon name="share" :size="24" />
        <span>分享</span>
      </div>
    </div>
  </div>
</template>

<style scoped lang="scss">
.video-meta-section {
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.author-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.author-avatar {
  cursor: pointer;
  flex-shrink: 0;
  background: transparent !important;
  overflow: hidden;

  :deep(.q-avatar__content) {
    padding: 0 !important;
  }
}

.author-avatar-img {
  width: 100%;
  height: 100%;
  background-size: cover;
  background-position: center;
  background-repeat: no-repeat;
  display: block;
}

.author-info {
  flex: 1;
  cursor: pointer;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.author-name {
  font-size: 15px;
  font-weight: 700;
  color: var(--text-base);
  display: flex;
  align-items: center;
  gap: 2px;
}

.author-followers {
  font-size: 12px;
  color: var(--text-secondary);
}

.follow-btn {
  background: var(--accent);
  color: #000;
  font-size: 13px;
  font-weight: 700;
  border-radius: var(--radius-full);
  padding: 4px 14px;
  min-height: unset;

  &--following {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-base);
  }
}

.video-title {
  font-size: 18px;
  font-weight: 700;
  color: var(--text-base);
  line-height: 1.3;
  margin: 0;
}

.video-desc {
  font-size: 14px;
  color: var(--text-secondary);
  margin: 0;
}

.tags-row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.tag-chip {
  font-size: 13px;
  color: var(--accent);
  cursor: pointer;
}

.stats-row {
  display: flex;
  gap: var(--space-4);
  font-size: 14px;
  color: var(--text-secondary);

  span {
    display: flex;
    align-items: center;
    gap: 4px;
  }
}

.action-row {
  display: flex;
  gap: var(--space-6);
  padding-top: var(--space-2);
}

.action-btn {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  cursor: pointer;
  color: var(--text-secondary);
  font-size: 12px;
  transition: color var(--transition-fast);

  &:hover { color: var(--text-base); }
}
</style>
