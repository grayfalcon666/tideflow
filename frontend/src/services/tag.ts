import api from './api'
import type { ApiResponse, Tag } from '../types'

export const getHotTags = () =>
  api.get<ApiResponse<Tag[]>>('/tags/hot')
