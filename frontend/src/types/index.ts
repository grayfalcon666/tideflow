// TideFlow TypeScript type definitions

// ============ Common ============
export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface ListResponse<T> {
  items: T[]
  next_cursor: string | null
  has_more: boolean
}

// ============ Auth ============
export interface LoginReq {
  username: string
  password: string
}

export interface LoginResp {
  access_token: string
  refresh_token: string
}

export interface RegisterReq {
  username: string
  password: string
}

// ============ User ============
export interface UserInfo {
  id: number
  username: string
  avatar_url: string
  bio?: string
  follower_count: number
  following_count: number
  video_count: number
  is_big_v: boolean
  likes_public?: boolean
  created_at: number
}

export interface UserProfile extends UserInfo {
  is_following: boolean
  is_following_me: boolean
  is_me: boolean
}

// ============ Video ============
export interface VideoAuthor {
  id: number
  username: string
  avatar_url: string
  follower_count: number
  is_big_v: boolean
}

export interface VideoItem {
  id?: number
  video_id: number
  author_id?: number
  username?: string
  play_url: string
  cover_url: string
  title: string
  description?: string
  create_time: number
  likes_count: number
  comment_count: number
  view_count?: number
  popularity: number
  is_liked: boolean
  author: VideoAuthor
  tags?: string[]
  width?: number
  height?: number
  duration?: number
  play_token?: string
  wordbank_status?: WordbankStatus
}

// ============ Learn ============
export type WordbankStatus = 'none' | 'pending' | 'ready' | 'failed'

export interface VocabList {
  id: number
  name: string
  slug: string
  language: string
  total: number
}

export interface LearnWord {
  value: string
  usphone: string
  ukphone: string
  definition: string
  translation: string
  pos: string
  first_caption_start: string
}

export interface Caption {
  start: string
  end: string
  content: string
}

export interface CommitLearningReq {
  word: string
  result: 'correct' | 'wrong'
}

export interface CommitLearningResp {
  new_status: number
  daily_words_today: number
  batch_remaining: number
}

export interface LearnWordsResp {
  words: LearnWord[]
  batch_total: number
  batch_remaining: number
  daily_remaining: number | null
}

export type LearnMode = 'spell' | 'type'

export interface HabitStatsResp {
  heatmap: DailyHeatmapEntry[]
  total_days: number
  current_streak: number
  today_words: number
  today_videos: number
}

export interface DailyHeatmapEntry {
  date: string
  words_count: number
}

// ============ Video (extend) ============
export interface VideoDetail extends VideoItem {
  is_following_author: boolean
  view_count: number
  play_token?: string
  wordbank_status?: WordbankStatus
}

// ============ Feed ============
export type FeedType = 'latest' | 'following'

export interface FeedReq {
  cursor?: string
  limit?: number
}

export interface PopularReq {
  window: '1m' | '5m' | '15m' | '1h' | '6h'
  cursor: string
  limit?: number
}

// ============ Comment ============
export type RawComment = {
  id: number
  author_id: number
  username: string
  avatar_url?: string
  content: string
  created_at: string
  like_count: number
  reply_count: number
  is_liked: boolean
  is_mine: boolean
  root_id: number
  parent_id: number
  replies?: RawComment[]
}

export interface Comment {
  comment_id: number
  user_id: number
  author_id: number
  username: string
  avatar_url: string
  content: string
  created_at: number
  like_count: number
  reply_count: number
  is_liked: boolean
  is_mine: boolean
  root_id: number
  parent_id: number
  replies?: Comment[]
  showReplies?: boolean
}

export const normalizeComment = (c: RawComment): Comment => ({
  comment_id: c.id,
  user_id: c.author_id,
  author_id: c.author_id,
  username: c.username,
  avatar_url: c.avatar_url ?? '',
  content: c.content,
  created_at: new Date(c.created_at).getTime() / 1000,
  like_count: c.like_count,
  reply_count: c.reply_count,
  is_liked: c.is_liked,
  is_mine: c.is_mine,
  root_id: c.root_id,
  parent_id: c.parent_id,
  replies: c.replies?.map(normalizeComment),
  showReplies: false,
})

export type CommentListResp = {
  items: RawComment[]
  next_cursor: string | null
  has_more: boolean
}

// ============ Like ============
export interface LikeResp {
  likes_count: number
  is_liked: boolean
}

// ============ Follow ============
export interface FollowResp {
  follower_count: number
  is_following: boolean
}

// ============ Notification ============
export interface Notification {
  id: number
  type: 'like' | 'comment' | 'follow'
  sender: {
    id: number
    username: string
    avatar_url: string
  }
  target_id: number
  target_title?: string
  target_cover?: string
  content: string
  created_at: string
  is_read: boolean
}

// ============ Message ============
export interface Conversation {
  peer_id: number
  peer_username: string
  peer_avatar: string
  last_message: Message
  unread_count: number
}

export interface Message {
  id: number
  from_id: number
  to_id: number
  content: string
  msg_type: string
  created_at: number
  is_read: boolean
}

export interface VideoSharePayload {
  video_id: number
  title: string
  cover_url: string
  author_name: string
}

// ============ Tag ============
export interface Tag {
  name: string
  video_count: number
}

export interface TagListResp {
  items: Tag[]
  next_cursor: string | null
  has_more: boolean
}

// ============ SSE ============
export interface SSENotification {
  type: 'like' | 'comment' | 'follow'
  sender_id: number
  target_id: number
  content: string
  occurred_at: number
}

// ============ Note ============
export interface RawNote {
  id: number
  video_id: number
  author_id: number
  username: string
  timestamp: number
  content: string
  created_at: string
}

export interface Note {
  note_id: number
  video_id: number
  author_id: number
  username: string
  timestamp: number
  content: string
  created_at: number
  is_mine: boolean
  formatted_time: string
}

export function formatTimestamp(seconds: number): string {
  const m = Math.floor(seconds / 60)
  const s = Math.floor(seconds % 60)
  return `${String(m).padStart(2, '0')}:${String(s).padStart(2, '0')}`
}

export function normalizeNote(raw: RawNote, currentUserId?: number): Note {
  return {
    note_id: raw.id,
    video_id: raw.video_id,
    author_id: raw.author_id,
    username: raw.username,
    timestamp: raw.timestamp,
    content: raw.content,
    created_at: new Date(raw.created_at).getTime() / 1000,
    is_mine: currentUserId != null ? raw.author_id === currentUserId : false,
    formatted_time: formatTimestamp(raw.timestamp),
  }
}
