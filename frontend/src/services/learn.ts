import api from './api'
import type { VocabList, LearnWord, Caption, ApiResponse, ListResponse, CommitLearningReq, HabitStatsResp } from '../types'

export const getVocabLists = () =>
  api.get<ApiResponse<VocabList[]>>('/learn/lists')

export const getVideoLearnWords = (
  videoId: number,
  listId: number,
  chunkSize?: number
) =>
  api.get<ApiResponse<{ list_name: string; total: number; learned_count: number; unmastered_total: number; words: LearnWord[] }>>(
    `/videos/${videoId}/learn/words`,
    { params: { list_id: listId, chunk_size: chunkSize ?? 0 } }
  )

export const getWordCaptions = (videoId: number, word: string) =>
  api.get<ApiResponse<{ word: string; captions: Caption[] }>>(
    `/videos/${videoId}/learn/word/${encodeURIComponent(word)}/captions`
  )

export const commitLearning = (payload: CommitLearningReq) =>
  api.post<ApiResponse<null>>('/learn/commit', payload)

export const getHabitStats = (year?: number) =>
  api.get<ApiResponse<HabitStatsResp>>('/learn/habit/stats', { params: { year } })

export const resetProgress = (videoId: number, listId: number) =>
  api.post<ApiResponse<null>>('/learn/progress/reset', { video_id: videoId, list_id: listId })

export const getTodayWords = () =>
  api.get<ApiResponse<{ word: string; status: number }[]>>('/learn/today/words')