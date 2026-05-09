<script setup lang="ts">
import { useRouter } from 'vue-router'
import type { VideoItem } from '../../types'

const props = defineProps<{
  item: VideoItem
}>()

const router = useRouter()

const goToUser = () => {
  router.push(`/u/${props.item.author?.id}`)
}
</script>

<template>
  <div class="slide-author-bar">
    <q-avatar size="44px" class="author-avatar" @click="goToUser">
      <div
        class="author-avatar-img"
        :style="{ backgroundImage: `url('${item.author?.avatar_url || '/default-avatar.svg'}')` }"
      />
    </q-avatar>
  </div>
</template>

<style scoped lang="scss">
.slide-author-bar {
  display: flex;
  flex-direction: column;
  align-items: center;
}

.author-avatar {
  cursor: pointer;
  border: 2px solid rgba(255, 255, 255, 0.4);
  border-radius: 50%;
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
</style>
