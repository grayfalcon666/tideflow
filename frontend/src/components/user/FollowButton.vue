<script setup lang="ts">
import { ref } from 'vue'
import * as userService from '../../services/user'

const props = defineProps<{
  userId: number
  initialFollowing: boolean
}>()

const following = ref(props.initialFollowing)

const toggle = async () => {
  const was = following.value
  following.value = !was
  try {
    if (was) {
      await userService.unfollow(props.userId)
    } else {
      await userService.follow(props.userId)
    }
  } catch {
    following.value = was
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
