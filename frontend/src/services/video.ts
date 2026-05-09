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

export const getUserLikedVideos = (userId: number, cursor?: string, limit = 10) =>
  api.get<ApiResponse<{ items: VideoItem[] & { id: number }[]; next_cursor: string | null; has_more: boolean }>>(
    `/users/${userId}/liked-videos`,
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

export const uploadAvatar = (formData: FormData) =>
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

export const updateVideo = (id: number, data: {
  title?: string
  description?: string
  cover_url?: string
  tags?: string[]
}) =>
  api.put<ApiResponse<null>>(`/videos/${id}`, data)

export const deleteVideo = (id: number) =>
  api.delete<ApiResponse<null>>(`/videos/${id}`)

// 切片上传：初始化上传会话
export const initChunkedUpload = (filename: string, fileSize: number, chunkSize = 5 * 1024 * 1024) =>
  api.post<ApiResponse<{ upload_id: string }>>('/videos/upload/init', {
    filename, file_size: fileSize, chunk_size: chunkSize,
  })

// 切片上传：上传单个分片
export const uploadChunk = (uploadId: string, chunkIndex: number, chunk: Blob) => {
  const fd = new FormData()
  fd.append('upload_id', uploadId)
  fd.append('chunk_index', String(chunkIndex))
  fd.append('file', chunk)
  return api.post<ApiResponse<{ chunk_index: number }>>('/videos/upload/chunk', fd, {
    headers: { 'Content-Type': 'multipart/form-data' },
  })
}

// 切片上传：查询上传进度（断点续传）
export const getUploadStatus = (uploadId: string) =>
  api.get<ApiResponse<{
    upload_id: string
    total_chunks: number
    uploaded_chunks: number[]
    chunk_size: number
    file_size: number
    filename: string
  }>>(`/videos/upload/status/${uploadId}`)

// 切片上传：合并分片，完成上传
export const completeChunkedUpload = (uploadId: string) =>
  api.post<ApiResponse<{
    play_url: string
    duration: number
    width: number
    height: number
    is_vertical: boolean
  }>>('/videos/upload/complete', { upload_id: uploadId })

export const recordView = (playToken: string) =>
  api.post<ApiResponse<{ user_id: number; client_ip: string; video_id: number }>>('/metrics/view', {
    play_token: playToken,
  })
