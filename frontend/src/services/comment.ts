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

export type PostCommentResp = {
  id: number
  author_id: number
  username: string
  avatar_url: string
  content: string
  parent_id: number
  root_id: number
  created_at: string
  reply_count: number
}

export const postComment2 = (
  videoId: number,
  content: string,
  rootId: number,
  parentId: number
) =>
  api.post<ApiResponse<PostCommentResp>>(`/videos/${videoId}/comments`, {
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
