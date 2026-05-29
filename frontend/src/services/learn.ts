import api from './api'
import type {
  VocabList, LearnWord, Caption, ApiResponse,
  CommitLearningReq, CommitLearningResp, LearnWordsResp,
  LearnMode, HabitStatsResp
} from '../types'

export const getVocabLists = () =>
  api.get<ApiResponse<VocabList[]>>('/learn/lists')

export const getVideoLearnWords = (
  videoId: number,
  listId: number,
  chunkSize?: number,
  mode?: LearnMode
) =>
  api.get<ApiResponse<LearnWordsResp>>(
    `/videos/${videoId}/learn/words`,
    { params: { list_id: listId, chunk_size: chunkSize ?? 15, mode: mode ?? 'spell' } }
  )

export const getWordCaptions = (videoId: number, word: string) =>
  api.get<ApiResponse<{ word: string; captions: Caption[] }>>(
    `/videos/${videoId}/learn/word/${encodeURIComponent(word)}/captions`
  )

export const commitLearning = (payload: CommitLearningReq) =>
  api.post<ApiResponse<CommitLearningResp>>('/learn/commit', payload)

export const abortBatch = () =>
  api.post<ApiResponse<null>>('/learn/batch/abort')

export const getHabitStats = (year?: number) =>
  api.get<ApiResponse<HabitStatsResp>>('/learn/habit/stats', { params: { year } })

export const getTodayWords = () =>
  api.get<ApiResponse<{ word: string; status: number }[]>>('/learn/today/words')
