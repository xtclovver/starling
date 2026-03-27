import { create } from 'zustand';
import type { User } from '@/types';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  avatarMediaId: string | null;
  bannerMediaId: string | null;
  initializing: boolean;
  login: (user: User, accessToken: string) => void;
  logout: () => void;
  updateUser: (user: Partial<User>) => void;
  setAvatarMediaId: (id: string | null) => void;
  setBannerMediaId: (id: string | null) => void;
  setAccessToken: (accessToken: string) => void;
  setInitializing: (v: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  avatarMediaId: null,
  bannerMediaId: null,
  initializing: true,

  login: (user, accessToken) => {
    set({ user, accessToken, isAuthenticated: true });
  },

  logout: () => {
    set({ user: null, accessToken: null, isAuthenticated: false, avatarMediaId: null, bannerMediaId: null });
  },

  updateUser: (partial) =>
    set((state) => ({
      user: state.user ? { ...state.user, ...partial } : null,
    })),

  setAvatarMediaId: (id) => set({ avatarMediaId: id }),
  setBannerMediaId: (id) => set({ bannerMediaId: id }),

  setAccessToken: (accessToken) => {
    set({ accessToken, isAuthenticated: true });
  },

  setInitializing: (v) => set({ initializing: v }),
}));
