<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import * as searchService from '../services/search'
import type { VideoItem } from '../types'
import type { SearchUserItem } from '../services/search'
import TFIcon from '../components/common/TFIcon.vue'

const route = useRoute()
const router = useRouter()

const query = ref((route.query.q as string) || '')
const activeTab = ref<'videos' | 'users'>('videos')
const videoSort = ref<'popularity' | 'create_time'>('popularity')

// Video state
const videos = ref<VideoItem[]>([])
const videoPage = ref(1)
const videoHasMore = ref(true)
const videoTotal = ref(0)
const videoLoading = ref(false)

// User state
const users = ref<SearchUserItem[]>([])
const userPage = ref(1)
const userHasMore = ref(true)
const userTotal = ref(0)
const userLoading = ref(false)

const searchVideos = async (reset = false) => {
  if (!query.value.trim()) return
  if (videoLoading.value) return
  videoLoading.value = true
  try {
    if (reset) {
      videos.value = []
      videoPage.value = 1
      videoHasMore.value = true
    }
    const resp = await searchService.searchVideos(query.value.trim(), videoSort.value, 'desc', videoPage.value, 20)
    const d = resp.data.data
    if (!d) return
    if (reset) {
      videos.value = d.items ?? []
    } else {
      videos.value.push(...(d.items ?? []))
    }
    videoTotal.value = d.total
    videoHasMore.value = d.has_more
    if (d.has_more) videoPage.value++
  } catch {} finally {
    videoLoading.value = false
  }
}

const searchUsers = async (reset = false) => {
  if (!query.value.trim()) return
  if (userLoading.value) return
  userLoading.value = true
  try {
    if (reset) {
      users.value = []
      userPage.value = 1
      userHasMore.value = true
    }
    const resp = await searchService.searchUsers(query.value.trim(), userPage.value, 20)
    const d = resp.data.data
    if (!d) return
    if (reset) {
      users.value = d.items ?? []
    } else {
      users.value.push(...(d.items ?? []))
    }
    userTotal.value = d.total
    userHasMore.value = d.has_more
    if (d.has_more) userPage.value++
  } catch {} finally {
    userLoading.value = false
  }
}

const doSearch = () => {
  const q = query.value.trim()
  if (!q) return
  router.replace({ query: { q } })
  if (activeTab.value === 'videos') {
    searchVideos(true)
  } else {
    searchUsers(true)
  }
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Enter') doSearch()
}

const switchTab = (tab: 'videos' | 'users') => {
  activeTab.value = tab
  if (tab === 'videos' && videos.value.length === 0 && query.value.trim()) {
    searchVideos(true)
  } else if (tab === 'users' && users.value.length === 0 && query.value.trim()) {
    searchUsers(true)
  }
}

const onSortChange = (sort: 'popularity' | 'create_time') => {
  videoSort.value = sort
  searchVideos(true)
}

const formatCount = (n: number) => {
  if (n >= 10000) return `${(n / 10000).toFixed(1)}w`
  if (n >= 1000) return `${(n / 1000).toFixed(1)}k`
  return String(n)
}

onMounted(() => {
  if (query.value.trim()) {
    searchVideos(true)
  }
})

// Sync query from URL
watch(() => route.query.q, (val) => {
  if (val && val !== query.value) {
    query.value = val as string
    doSearch()
  }
})
</script>

<template>
  <div class="search-page">
    <!-- Header -->
    <div class="search-header">
      <div class="back-btn" @click="router.back()">
        <TFIcon name="arrow_back" :size="24" />
      </div>
      <div class="search-input-wrap">
        <TFIcon name="search" :size="20" color="var(--text-muted)" />
        <input
          v-model="query"
          class="search-input"
          placeholder="搜索视频或用户"
          @keydown="onKeydown"
          autofocus
        />
        <button v-if="query" class="clear-btn" @click="query = ''">
          <TFIcon name="close" :size="18" />
        </button>
      </div>
      <button class="search-btn" @click="doSearch">搜索</button>
    </div>

    <!-- Tabs -->
    <div class="search-tabs">
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'videos' }"
        @click="switchTab('videos')"
      >
        视频
        <span v-if="videoTotal" class="tab-count">{{ formatCount(videoTotal) }}</span>
      </button>
      <button
        class="tab-btn"
        :class="{ active: activeTab === 'users' }"
        @click="switchTab('users')"
      >
        用户
        <span v-if="userTotal" class="tab-count">{{ formatCount(userTotal) }}</span>
      </button>
    </div>

    <!-- Video sort -->
    <div v-if="activeTab === 'videos' && videos.length > 0" class="sort-bar">
      <button
        class="sort-btn"
        :class="{ active: videoSort === 'popularity' }"
        @click="onSortChange('popularity')"
      >最热</button>
      <button
        class="sort-btn"
        :class="{ active: videoSort === 'create_time' }"
        @click="onSortChange('create_time')"
      >最新</button>
    </div>

    <!-- Video results -->
    <template v-if="activeTab === 'videos'">
      <div v-if="!videos.length && !videoLoading && query.trim()" class="empty-state">
        <TFIcon name="search_off" :size="64" color="var(--text-secondary)" />
        <p>未找到相关视频</p>
      </div>

      <div class="video-list">
        <div
          v-for="item in videos"
          :key="item.video_id"
          class="video-card"
          @click="router.push(`/video/${item.video_id}`)"
        >
          <div class="card-cover">
            <img :src="item.cover_url" loading="lazy" />
          </div>
          <div class="card-info">
            <h4 class="card-title">{{ item.title }}</h4>
            <div class="card-author">{{ item.author?.username }}</div>
            <div class="card-stats">
              <TFIcon name="favorite" :size="14" />
              {{ formatCount(item.likes_count) }}
            </div>
          </div>
        </div>
      </div>

      <div v-if="videoLoading" class="loading-state">
        <q-spinner color="accent" size="32px" />
      </div>
      <div v-if="!videoHasMore && videos.length" class="no-more">没有更多了</div>
      <div v-if="videoHasMore && videos.length && !videoLoading" class="load-more" @click="searchVideos()">
        加载更多
      </div>
    </template>

    <!-- User results -->
    <template v-if="activeTab === 'users'">
      <div v-if="!users.length && !userLoading && query.trim()" class="empty-state">
        <TFIcon name="search_off" :size="64" color="var(--text-secondary)" />
        <p>未找到相关用户</p>
      </div>

      <div class="user-list">
        <div
          v-for="user in users"
          :key="user.id"
          class="user-card"
          @click="router.push(`/u/${user.id}`)"
        >
          <div class="user-avatar">
            <img v-if="user.avatar_url" :src="user.avatar_url" />
            <div v-else class="avatar-placeholder">
              {{ user.username.charAt(0).toUpperCase() }}
            </div>
          </div>
          <div class="user-info">
            <div class="user-name">{{ user.username }}</div>
            <div class="user-bio" v-if="user.bio">{{ user.bio }}</div>
            <div class="user-followers">{{ formatCount(user.follower_count) }} 粉丝</div>
          </div>
        </div>
      </div>

      <div v-if="userLoading" class="loading-state">
        <q-spinner color="accent" size="32px" />
      </div>
      <div v-if="!userHasMore && users.length" class="no-more">没有更多了</div>
      <div v-if="userHasMore && users.length && !userLoading" class="load-more" @click="searchUsers()">
        加载更多
      </div>
    </template>
  </div>
</template>

<style scoped lang="scss">
.search-page {
  max-width: 900px;
  margin: 0 auto;
  padding: var(--space-3) var(--space-4);
  padding-bottom: 80px;
}

.search-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.back-btn {
  cursor: pointer;
  padding: var(--space-1);
  display: flex;
  align-items: center;
  color: var(--text-secondary);
  flex-shrink: 0;

  &:hover { color: var(--text-base); }
}

.search-input-wrap {
  flex: 1;
  display: flex;
  align-items: center;
  gap: var(--space-2);
  background: var(--bg-elevated);
  border-radius: var(--radius-full);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--border);
  transition: border-color var(--transition-fast);

  &:focus-within {
    border-color: var(--accent);
  }
}

.search-input {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: var(--text-base);
  font-size: 14px;

  &::placeholder {
    color: var(--text-muted);
  }
}

.clear-btn {
  background: transparent;
  border: none;
  cursor: pointer;
  color: var(--text-muted);
  display: flex;
  align-items: center;
  padding: 0;

  &:hover { color: var(--text-secondary); }
}

.search-btn {
  background: var(--accent);
  color: #fff;
  border: none;
  border-radius: var(--radius-full);
  padding: var(--space-2) var(--space-4);
  font-size: 14px;
  font-weight: 600;
  cursor: pointer;
  flex-shrink: 0;
  transition: opacity var(--transition-fast);

  &:hover { opacity: 0.9; }
}

.search-tabs {
  display: flex;
  gap: var(--space-1);
  margin-bottom: var(--space-3);
  border-bottom: 1px solid var(--border);
}

.tab-btn {
  background: transparent;
  border: none;
  color: var(--text-secondary);
  font-size: 15px;
  font-weight: 600;
  padding: var(--space-2) var(--space-4);
  cursor: pointer;
  position: relative;
  transition: color var(--transition-fast);

  &.active {
    color: var(--text-base);

    &::after {
      content: '';
      position: absolute;
      bottom: -1px;
      left: 0;
      right: 0;
      height: 2px;
      background: var(--accent);
      border-radius: 1px;
    }
  }

  &:hover:not(.active) {
    color: var(--text-base);
  }
}

.tab-count {
  font-size: 12px;
  font-weight: 400;
  color: var(--text-muted);
  margin-left: 4px;
}

.sort-bar {
  display: flex;
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.sort-btn {
  background: var(--bg-elevated);
  border: 1px solid var(--border);
  color: var(--text-secondary);
  font-size: 13px;
  padding: 4px 14px;
  border-radius: var(--radius-full);
  cursor: pointer;
  transition: all var(--transition-fast);

  &.active {
    background: var(--accent);
    color: #fff;
    border-color: var(--accent);
  }

  &:hover:not(.active) {
    color: var(--text-base);
  }
}

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 60px 0;
  gap: var(--space-4);
  color: var(--text-secondary);
}

// Video grid
.video-list {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: var(--space-3);

  @media (max-width: 600px) {
    grid-template-columns: 1fr;
  }
}

.video-card {
  background: var(--bg-card);
  border-radius: var(--radius-md);
  overflow: hidden;
  cursor: pointer;
  transition: transform var(--transition-fast);

  &:hover { transform: translateY(-2px); }
}

.card-cover {
  aspect-ratio: 16 / 9;
  background: var(--bg-elevated);

  img { width: 100%; height: 100%; object-fit: cover; }
}

.card-info {
  padding: var(--space-3);
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.card-title {
  font-size: 14px;
  font-weight: 700;
  color: var(--text-base);
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
  line-height: 1.3;
  margin: 0;
}

.card-author {
  font-size: 12px;
  color: var(--text-secondary);
}

.card-stats {
  font-size: 12px;
  color: var(--text-secondary);
  display: flex;
  align-items: center;
  gap: 2px;
}

// User list
.user-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.user-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background var(--transition-fast);

  &:hover {
    background: var(--bg-hover);
  }
}

.user-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  background: var(--bg-elevated);

  img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
}

.avatar-placeholder {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 20px;
  font-weight: 700;
  color: var(--text-secondary);
  background: var(--bg-surface);
}

.user-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.user-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--text-base);
}

.user-bio {
  font-size: 13px;
  color: var(--text-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.user-followers {
  font-size: 12px;
  color: var(--text-muted);
}

.loading-state {
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.no-more {
  text-align: center;
  padding: var(--space-4);
  color: var(--text-secondary);
  font-size: 14px;
}

.load-more {
  text-align: center;
  padding: var(--space-4);
  color: var(--accent);
  font-size: 14px;
  cursor: pointer;

  &:hover { text-decoration: underline; }
}
</style>
