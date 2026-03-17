import { create } from 'zustand';
import type { Post } from '@/types';

interface FeedState {
  posts: Post[];
  cursor: string;
  hasMore: boolean;
  loading: boolean;
  setPosts: (posts: Post[], cursor: string, hasMore: boolean) => void;
  appendPosts: (posts: Post[], cursor: string, hasMore: boolean) => void;
  prependPost: (post: Post) => void;
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

  setPosts: (posts, cursor, hasMore) => set({ posts, cursor, hasMore }),
  appendPosts: (posts, cursor, hasMore) =>
    set((state) => ({ posts: [...state.posts, ...posts], cursor, hasMore })),
  prependPost: (post) =>
    set((state) => ({ posts: [post, ...state.posts] })),
  removePost: (postId) =>
    set((state) => ({ posts: state.posts.filter((p) => p.id !== postId) })),
  updatePost: (postId, updates) =>
    set((state) => ({
      posts: state.posts.map((p) => (p.id === postId ? { ...p, ...updates } : p)),
    })),
  setLoading: (loading) => set({ loading }),
  reset: () => set({ posts: [], cursor: '', hasMore: true, loading: false }),
}));
