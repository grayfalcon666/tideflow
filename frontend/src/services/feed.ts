import api from './api'
import type { ApiResponse, VideoItem } from '../types'

// Time-cursor pagination (latest / following / tag)
export const getLatest = (cursor?: string, limit = 10) =>
  api.get<ApiResponse<{ items: VideoItem[]; next_cursor: string | null; has_more: boolean }>>(
    '/feed/latest',
    { params: { cursor, limit } }
  )

export const getFollowing = (cursor?: string, limit = 10) =>
  api.get<ApiResponse<{ items: VideoItem[]; next_cursor: string | null; has_more: boolean }>>(
    '/feed/following',
    { params: { cursor, limit } }
  )

// Offset-cursor pagination (popular)
export const getPopular = (window: string, cursor = '0', limit = 20) =>
  api.get<ApiResponse<{ items: VideoItem[]; next_cursor: string | null; has_more: boolean }>>(
    '/feed/popular',
    { params: { window, cursor, limit } }
  )
