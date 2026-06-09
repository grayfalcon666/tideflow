import api from './api'
import type { ApiResponse, VideoItem } from '../types'

export const getHistory = (cursor?: string, limit = 20) =>
  api.get<ApiResponse<{ items: VideoItem[]; next_cursor: string | null; has_more: boolean }>>(
    '/history', { params: { cursor, limit } }
  )

export const deleteHistoryItem = (videoId: number) =>
  api.delete<ApiResponse<void>>(`/history/${videoId}`)

export const clearHistory = () =>
  api.delete<ApiResponse<void>>('/history')

export const recordHistory = (videoId: number) =>
  api.post<ApiResponse<void>>('/history/record', { video_id: videoId })
