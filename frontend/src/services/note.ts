import api from './api'
import type { ApiResponse, ListResponse, RawNote } from '../types'

export const getNotes = (videoId: number, cursor?: string, limit = 50) =>
  api.get<ApiResponse<ListResponse<RawNote>>>(`/videos/${videoId}/notes`, {
    params: { cursor, limit },
  })

export const createNote = (videoId: number, timestamp: number, content: string) =>
  api.post<ApiResponse<RawNote>>(`/videos/${videoId}/notes`, { timestamp, content })

export const deleteNote = (videoId: number, noteId: number) =>
  api.delete<ApiResponse<void>>(`/videos/${videoId}/notes/${noteId}`)
