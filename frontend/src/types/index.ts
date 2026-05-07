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
  created_at: number
}

export interface UserProfile extends UserInfo {
  is_following: boolean
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
  popularity: number
  is_liked: boolean
  author: VideoAuthor
  tags?: string[]
}

export interface VideoDetail extends VideoItem {
  is_following_author: boolean
  views_count: number
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
export interface Comment {
  comment_id: number
  user_id: number
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
}

export interface CommentListResp {
  items: Comment[]
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
  created_at: number
  is_read: boolean
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
