import { create } from 'zustand';
import type { Post } from '@/types';

interface FeedState {
  posts: Post[];
  cursor: string;
  hasMore: boolean;
  loading: boolean;
  pendingPosts: Post[];
  setPosts: (posts: Post[], cursor: string, hasMore: boolean) => void;
  appendPosts: (posts: Post[], cursor: string, hasMore: boolean) => void;
  prependPost: (post: Post) => void;
  addPendingPost: (post: Post) => void;
  flushPendingPosts: () => void;
  removePost: (postId: string) => void;
  updatePost: (postId: string, updates: Partial<Post>) => void;
  setLoading: (loading: boolean) => void;
  reset: () => void;
}

export const useFeedStore = create<FeedState>((set) => ({
  posts: [],
  cursor: '',
  hasMore: true,
  loading: false,
  pendingPosts: [],

  setPosts: (posts, cursor, hasMore) => set({ posts, cursor, hasMore }),
  appendPosts: (posts, cursor, hasMore) =>
    set((state) => ({ posts: [...state.posts, ...posts], cursor, hasMore })),
  prependPost: (post) =>
    set((state) => ({ posts: [post, ...state.posts] })),
  addPendingPost: (post) =>
    set((state) => {
      if (state.posts.some((p) => p.id === post.id) || state.pendingPosts.some((p) => p.id === post.id)) return state;
      return { pendingPosts: [post, ...state.pendingPosts] };
    }),
  flushPendingPosts: () =>
    set((state) => ({ posts: [...state.pendingPosts, ...state.posts], pendingPosts: [] })),
  removePost: (postId) =>
    set((state) => ({ posts: state.posts.filter((p) => p.id !== postId) })),
  updatePost: (postId, updates) =>
    set((state) => ({
      posts: state.posts.map((p) => (p.id === postId ? { ...p, ...updates } : p)),
    })),
  setLoading: (loading) => set({ loading }),
  reset: () => set({ posts: [], cursor: '', hasMore: true, loading: false, pendingPosts: [] }),
}));
