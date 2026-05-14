import api from './api'
import type { VocabList, LearnWord, Caption, ApiResponse, ListResponse } from '../types'

export const getVocabLists = () =>
  api.get<ApiResponse<VocabList[]>>('/learn/lists')

export const getVideoLearnWords = (videoId: number, listId: number) =>
  api.get<ApiResponse<{ list_name: string; total: number; words: LearnWord[] }>>(
    `/videos/${videoId}/learn/words`,
    { params: { list_id: listId } }
  )

export const getWordCaptions = (videoId: number, word: string) =>
  api.get<ApiResponse<{ word: string; captions: Caption[] }>>(
    `/videos/${videoId}/learn/word/${encodeURIComponent(word)}/captions`
  )