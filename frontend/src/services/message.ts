import api from './api'
import type { ApiResponse, Conversation, Message } from '../types'

export const getConversations = () =>
  api.get<ApiResponse<Conversation[]>>('/messages/conversations')

export const getMessages = (peerId: number, cursor?: string, limit = 20) =>
  api.get<ApiResponse<{ items: Message[]; next_cursor: string | null; has_more: boolean }>>(
    `/messages/conversations/${peerId}`,
    { params: { cursor, limit } }
  )

export const sendMessage = (toId: number, content: string) =>
  api.post<ApiResponse<Message>>('/messages', { to_id: toId, content })

export const markRead = (peerId: number) =>
  api.put<ApiResponse<void>>(`/messages/conversations/${peerId}/read`)
