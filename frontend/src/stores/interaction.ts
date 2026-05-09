import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useInteractionStore = defineStore('interaction', () => {
  const likedVideoIds = ref<Set<number>>(new Set())
  const followingUserIds = ref<Set<number>>(new Set())

  const isLiked = (videoId: number) => likedVideoIds.value.has(videoId)
  const isFollowing = (userId: number) => followingUserIds.value.has(userId)

  const toggleLike = (videoId: number) => {
    if (likedVideoIds.value.has(videoId)) {
      likedVideoIds.value.delete(videoId)
    } else {
      likedVideoIds.value.add(videoId)
    }
  }

  const toggleFollow = (userId: number) => {
    if (followingUserIds.value.has(userId)) {
      followingUserIds.value.delete(userId)
    } else {
      followingUserIds.value.add(userId)
    }
  }

  const reset = () => {
    likedVideoIds.value = new Set()
    followingUserIds.value = new Set()
    localStorage.removeItem('tideflow-interaction')
  }

  const syncLike = (videoId: number) => {
    likedVideoIds.value.add(videoId)
  }

  const syncFollow = (userId: number) => {
    followingUserIds.value.add(userId)
  }

  const syncAll = (videoIds: number[], userIds: number[]) => {
    videoIds.forEach(id => likedVideoIds.value.add(id))
    userIds.forEach(id => followingUserIds.value.add(id))
  }

  const removeLike = (videoId: number) => {
    likedVideoIds.value.delete(videoId)
  }

  const replaceLikes = (ids: number[]) => {
    likedVideoIds.value = new Set(ids)
  }

  return {
    likedVideoIds, followingUserIds,
    isLiked, isFollowing, toggleLike, toggleFollow, reset, syncLike, syncFollow, syncAll,
    removeLike, replaceLikes,
  }
}, {
  persist: {
    key: 'tideflow-interaction',
    storage: localStorage,
    serializer: {
      deserialize: (value: string) => {
        const parsed = JSON.parse(value)
        return {
          likedVideoIds: new Set(parsed.likedVideoIds),
          followingUserIds: new Set(parsed.followingUserIds),
        }
      },
      serialize: (value) => JSON.stringify({
        likedVideoIds: [...value.likedVideoIds],
        followingUserIds: [...value.followingUserIds],
      }),
    },
  },
})
