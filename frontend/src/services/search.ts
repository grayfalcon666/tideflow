import api from './api'
import type { ApiResponse, VideoItem } from '../types'

export interface SearchUserItem {
  id: number
  username: string
  avatar_url: string
  bio: string
  follower_count: number
}

export interface SearchVideoResp {
  items: VideoItem[]
  total: number
  page: number
  size: number
  has_more: boolean
  next_cursor: string | null
}

export interface SearchUserResp {
  items: SearchUserItem[]
  total: number
  page: number
  size: number
  has_more: boolean
  next_cursor: string | null
}

export const searchVideos = (q: string, sortBy = 'popularity', order = 'desc', page = 1, size = 20) =>
  api.get<ApiResponse<SearchVideoResp>>('/search/videos', {
    params: { q, sort_by: sortBy, order, page, size },
  })

export const searchUsers = (q: string, page = 1, size = 20) =>
  api.get<ApiResponse<SearchUserResp>>('/search/users', {
    params: { q, page, size },
  })
