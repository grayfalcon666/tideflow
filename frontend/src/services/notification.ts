import api from './api'
import type { ApiResponse, Notification } from '../types'

export const getNotifications = (cursor?: string, limit = 20) =>
  api.get<ApiResponse<{ items: Notification[]; next_cursor: string | null; has_more: boolean }>>(
    '/notifications',
    { params: { cursor, limit } }
  )

export const markRead = (ids: number[]) =>
  api.put<ApiResponse<void>>('/notifications/read', { ids })

export const markAllRead = () =>
  api.put<ApiResponse<void>>('/notifications/read', {})
