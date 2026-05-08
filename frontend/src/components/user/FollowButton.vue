<script setup lang="ts">
import { computed } from 'vue'
import * as userService from '../../services/user'
import { useInteractionStore } from '../../stores/interaction'

const props = defineProps<{
  userId: number
}>()

const interactionStore = useInteractionStore()
const following = computed(() => interactionStore.isFollowing(props.userId))

const toggle = async () => {
  const was = following.value
  interactionStore.toggleFollow(props.userId)
  try {
    if (was) {
      await userService.unfollow(props.userId)
    } else {
      await userService.follow(props.userId)
    }
  } catch {
    interactionStore.toggleFollow(props.userId)
  }
}
</script>

<template>
  <q-btn
    no-caps
    :label="following ? '已关注' : '关注'"
    :class="['follow-btn', { 'follow-btn--following': following }]"
    @click="toggle"
  />
</template>

<style scoped lang="scss">
.follow-btn {
  background: var(--accent);
  color: #000;
  font-size: 15px;
  font-weight: 700;
  border-radius: var(--radius-full);
  padding: 8px 28px;
  min-height: unset;

  &--following {
    background: transparent;
    border: 1px solid var(--border);
    color: var(--text-base);
  }
}
</style>
