import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { VideoItem, WordbankStatus } from '../types'
import * as feedService from '../services/feed'

type RawVideoItem = {
  id: number
  author_id: number
  username: string
  avatar_url?: string
  title: string
  description?: string
  play_url: string
  cover_url: string
  create_time: string
  likes_count: number
  comment_count: number
  view_count?: number
  popularity: number
  is_liked: boolean
  is_big_v?: boolean
  tags?: string[]
  width?: number
  height?: number
  duration?: number
  wordbank_status?: string
}

// Normalize flat backend video to frontend VideoItem
const normalizeVideo = (v: RawVideoItem): VideoItem => ({
  video_id: v.id,
  author_id: v.author_id,
  username: v.username,
  title: v.title,
  description: v.description,
  play_url: v.play_url,
  cover_url: v.cover_url,
  create_time: new Date(v.create_time).getTime() / 1000,
  likes_count: v.likes_count,
  comment_count: v.comment_count,
  view_count: v.view_count,
  popularity: v.popularity,
  is_liked: v.is_liked,
  author: {
    id: v.author_id,
    username: v.username,
    avatar_url: v.avatar_url ?? '',
    follower_count: 0,
    is_big_v: v.is_big_v ?? false,
  },
  tags: v.tags,
  width: v.width,
  height: v.height,
  duration: v.duration,
  wordbank_status: v.wordbank_status as WordbankStatus | undefined,
})

const normalizeVideos = (items: RawVideoItem[]): VideoItem[] =>
  (items ?? []).filter(Boolean).map(normalizeVideo)

export type FeedTab = 'latest' | 'following'

export const useFeedStore = defineStore('feed', () => {
  const latestItems = ref<VideoItem[]>([])
  const followingItems = ref<VideoItem[]>([])
  const latestCursor = ref<string | null>(null)
  const followingCursor = ref<string | null>(null)
  const latestHasMore = ref(true)
  const followingHasMore = ref(true)
  const latestLoading = ref(false)
  const followingLoading = ref(false)

  const loadLatest = async (reset = false) => {
    if (latestLoading.value) return
    latestLoading.value = true
    try {
      if (reset) {
        latestItems.value = []
        latestCursor.value = null
        latestHasMore.value = true
      }
      const resp = await feedService.getLatest(
        latestCursor.value ?? undefined,
        10
      )
      const d = resp.data.data
      if (!d) return
      const newItems = normalizeVideos(d.items as unknown as RawVideoItem[])
      if (reset) {
        latestItems.value = newItems
      } else {
        latestItems.value.push(...newItems)
      }
      latestCursor.value = d.next_cursor
      latestHasMore.value = d.has_more ?? false
    } finally {
      latestLoading.value = false
    }
  }

  const loadFollowing = async (reset = false) => {
    if (followingLoading.value) return
    followingLoading.value = true
    try {
      if (reset) {
        followingItems.value = []
        followingCursor.value = null
        followingHasMore.value = true
      }
      const resp = await feedService.getFollowing(
        followingCursor.value ?? undefined,
        10
      )
      const d = resp.data.data
      if (!d) return
      const newItems = normalizeVideos(d.items as unknown as RawVideoItem[])
      if (reset) {
        followingItems.value = newItems
      } else {
        followingItems.value.push(...newItems)
      }
      followingCursor.value = d.next_cursor
      followingHasMore.value = d.has_more ?? false
    } finally {
      followingLoading.value = false
    }
  }

  const loadMore = async (tab: FeedTab) => {
    if (tab === 'latest') {
      if (!latestHasMore.value || latestLoading.value) return
      latestLoading.value = true
      try {
        const resp = await feedService.getLatest(latestCursor.value ?? undefined, 10)
        const d = resp.data.data
        if (!d) return
        latestItems.value.push(...normalizeVideos(d.items as unknown as RawVideoItem[]))
        latestCursor.value = d.next_cursor
        latestHasMore.value = d.has_more ?? false
      } finally {
        latestLoading.value = false
      }
    } else {
      if (!followingHasMore.value || followingLoading.value) return
      followingLoading.value = true
      try {
        const resp = await feedService.getFollowing(followingCursor.value ?? undefined, 10)
        const d = resp.data.data
        if (!d) return
        followingItems.value.push(...normalizeVideos(d.items as unknown as RawVideoItem[]))
        followingCursor.value = d.next_cursor
        followingHasMore.value = d.has_more ?? false
      } finally {
        followingLoading.value = false
      }
    }
  }

  return {
    latestItems, followingItems,
    latestCursor, followingCursor,
    latestHasMore, followingHasMore,
    latestLoading, followingLoading,
    loadLatest, loadFollowing, loadMore,
  }
})