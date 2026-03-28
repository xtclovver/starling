export interface User {
  id: string;
  username: string;
  email: string;
  display_name: string;
  bio: string;
  avatar_url: string;
  created_at: string;
  followers_count?: number;
  following_count?: number;
  is_following?: boolean;
  banner_url?: string;
}

export interface Post {
  id: string;
  user_id: string;
  content: string;
  media_url: string;
  likes_count: number;
  comments_count: number;
  reposts_count: number;
  created_at: string;
  updated_at: string;
  edited_at?: string;
  author?: User;
  liked?: boolean;
  bookmarked?: boolean;
  reposted?: boolean;
  hashtags?: string[];
}

export interface Comment {
  id: string;
  post_id: string;
  user_id: string;
  parent_id: string;
  content: string;
  media_url?: string;
  likes_count: number;
  depth: number;
  created_at: string;
  updated_at: string;
  edited_at?: string;
  children: Comment[];
  author?: User;
  liked?: boolean;
}

export interface PaginationResponse {
  next_cursor: string;
  has_more: boolean;
}

export interface ApiResponse<T> {
  data: T;
  error: { code: number; message: string } | null;
}

export interface AuthTokens {
  user: User;
  access_token: string;
}

export interface TrendingHashtag {
  tag: string;
  post_count: number;
}

export interface Notification {
  id: string;
  user_id: string;
  actor_id: string;
  type: string;
  post_id?: string;
  comment_id?: string;
  read: boolean;
  created_at: string;
  actor?: User;
}
