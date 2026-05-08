import api from './api'
import type { ApiResponse, VideoDetail, VideoItem } from '../types'

export const getVideo = (id: number) =>
  api.get<ApiResponse<VideoDetail>>(`/videos/${id}`)

export const getVideosByTag = (tag: string, cursor?: string, limit = 10) =>
  api.get<ApiResponse<{ items: VideoItem[]; next_cursor: string | null; has_more: boolean }>>(
    '/feed/tag',
    { params: { tag, cursor, limit } }
  )

export const likeVideo = (id: number) =>
  api.post<ApiResponse<{ likes_count: number }>>(`/videos/${id}/like`)

export const unlikeVideo = (id: number) =>
  api.delete<ApiResponse<{ likes_count: number }>>(`/videos/${id}/like`)

export const getMyLiked = (cursor?: string, limit = 10) =>
  api.get<ApiResponse<{ items: VideoItem[] & { id: number }[]; next_cursor: string | null; has_more: boolean }>>(
    '/likes/mine',
    { params: { cursor, limit } }
  )

export const uploadVideo = (formData: FormData) =>
  api.post<ApiResponse<{ play_url: string }>>('/videos/upload', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

export const uploadCover = (formData: FormData) =>
  api.post<ApiResponse<{ cover_url: string }>>('/videos/cover', formData, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })

export const publishVideo = (data: {
  title: string
  description?: string
  play_url: string
  cover_url?: string
  tags?: string[]
}) =>
  api.post<ApiResponse<{ video_id: number }>>('/videos', data)

export const recordView = (playToken: string) =>
  api.post<ApiResponse<{ user_id: number; client_ip: string; video_id: number }>>('/metrics/view', {
    play_token: playToken,
  })
