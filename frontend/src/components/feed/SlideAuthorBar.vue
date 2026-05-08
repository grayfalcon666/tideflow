<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { VideoItem } from '../../types'
import * as userService from '../../services/user'
import { useAuthStore } from '../../stores/auth'
import { useInteractionStore } from '../../stores/interaction'
import TFIcon from '../common/TFIcon.vue'

const props = defineProps<{
  item: VideoItem
}>()

const router = useRouter()
const authStore = useAuthStore()
const interactionStore = useInteractionStore()
const isFollowing = computed(() => interactionStore.isFollowing(props.item.author.id))
const isMe = computed(() => authStore.accountId === props.item.author.id)

const toggleFollow = async () => {
  if (!authStore.isLoggedIn) {
    router.push('/account?redirect=/')
    return
  }
  if (isMe.value) return
  const was = isFollowing.value
  interactionStore.toggleFollow(props.item.author.id)
  try {
    if (was) {
      await userService.unfollow(props.item.author.id)
    } else {
      await userService.follow(props.item.author.id)
    }
  } catch {
    interactionStore.toggleFollow(props.item.author.id)
  }
}

const goToUser = () => {
  router.push(`/u/${props.item.author.id}`)
}
</script>

<template>
  <div class="slide-author-bar">
    <q-avatar size="40px" class="author-avatar" @click="goToUser">
      <div
        class="author-avatar-img"
        :style="{ backgroundImage: `url('${item.author.avatar_url || '/default-avatar.svg'}')` }"
      />
    </q-avatar>
    <div class="author-info">
      <span class="author-name" @click="goToUser">
        {{ item.author.username }}
        <TFIcon v-if="item.author.is_big_v" name="verified" :size="14" color="orange" />
      </span>
      <q-btn
        v-if="!isMe"
        flat
        no-caps
        dense
        :label="isFollowing ? '已关注' : '关注'"
        :class="['follow-btn', { 'follow-btn--following': isFollowing }]"
        @click.stop="toggleFollow"
      />
    </div>
  </div>
</template>

<style scoped lang="scss">
.slide-author-bar {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}

.author-avatar {
  cursor: pointer;
  border: 2px solid rgba(255, 255, 255, 0.3);
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
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
}

.author-name {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-base);
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 2px;
}

.follow-btn {
  background: var(--accent);
  color: #000;
  font-size: 12px;
  font-weight: 700;
  border-radius: var(--radius-full);
  padding: 2px 10px;
  min-height: unset;
  min-width: unset;
  line-height: 1.2;

  &--following {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-base);
  }
}
</style>
