import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Conversation } from '../types'
import * as messageService from '../services/message'

export const useMessageStore = defineStore('message', () => {
  const conversations = ref<Conversation[]>([])
  const totalUnread = ref(0)

  const fetchConversations = async () => {
    const resp = await messageService.getConversations()
    const d = resp.data.data
    if (!d) return
    conversations.value = d ?? []
    totalUnread.value = conversations.value.reduce((sum, c) => sum + c.unread_count, 0)
  }

  return { conversations, totalUnread, fetchConversations }
})
