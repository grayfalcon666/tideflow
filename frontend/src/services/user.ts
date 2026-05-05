import api from './api'
import type { ApiResponse, UserInfo, UserProfile } from '../types'

export const getMe = () =>
  api.get<ApiResponse<UserInfo>>('/users/me')

export const getUser = (id: number) =>
  api.get<ApiResponse<UserProfile>>(`/users/${id}`)

export const getUserVideos = (id: number, cursor?: string, limit = 10) =>
  api.get<ApiResponse<{ items: any[]; next_cursor: string | null; has_more: boolean }>>(
    `/users/${id}/videos`,
    { params: { cursor, limit } }
  )

export const follow = (id: number) =>
  api.post<ApiResponse<{ follower_count: number }>>(`/users/${id}/follow`)

export const unfollow = (id: number) =>
  api.delete<ApiResponse<void>>(`/users/${id}/follow`)

export const getFollowing = (id: number, cursor?: string, limit = 20) =>
  api.get<ApiResponse<{ items: UserInfo[]; next_cursor: string | null; has_more: boolean }>>(
    `/users/${id}/following`,
    { params: { cursor, limit } }
  )

export const getFollowers = (id: number, cursor?: string, limit = 20) =>
  api.get<ApiResponse<{ items: UserInfo[]; next_cursor: string | null; has_more: boolean }>>(
    `/users/${id}/followers`,
    { params: { cursor, limit } }
  )
