import api from './api'
import type { ApiResponse, Comment, CommentListResp, RawComment } from '../types'

export const getComments = (videoId: number, rootId = 0, cursor?: string, limit = 20) =>
  api.get<ApiResponse<CommentListResp>>(`/videos/${videoId}/comments`, {
    params: { root_id: rootId, cursor, limit },
  })

export const postComment = (
  videoId: number,
  content: string,
  rootId: number,
  parentId: number
) =>
  api.post<ApiResponse<RawComment>>(`/videos/${videoId}/comments`, {
    content,
    root_id: rootId,
    parent_id: parentId,
  })

export const deleteComment = (videoId: number, commentId: number) =>
  api.delete<ApiResponse<void>>(`/videos/${videoId}/comments/${commentId}`)

export const likeComment = (videoId: number, commentId: number) =>
  api.post<ApiResponse<{ like_count: number }>>(`/videos/${videoId}/comments/${commentId}/like`)

export const unlikeComment = (videoId: number, commentId: number) =>
  api.delete<ApiResponse<{ like_count: number }>>(`/videos/${videoId}/comments/${commentId}/like`)
