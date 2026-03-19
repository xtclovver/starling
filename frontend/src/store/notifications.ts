import { create } from 'zustand';
import type { Notification } from '@/types';

interface NotificationState {
  notifications: Notification[];
  unreadCount: number;
  cursor: string;
  hasMore: boolean;
  loading: boolean;
  setNotifications: (notifications: Notification[], cursor: string, hasMore: boolean) => void;
  appendNotifications: (notifications: Notification[], cursor: string, hasMore: boolean) => void;
  prependNotification: (notification: Notification) => void;
  setUnreadCount: (count: number) => void;
  incrementUnread: () => void;
  markAsRead: (id: string) => void;
  markAllAsRead: () => void;
  setLoading: (loading: boolean) => void;
  reset: () => void;
}

export const useNotificationStore = create<NotificationState>((set) => ({
  notifications: [],
  unreadCount: 0,
  cursor: '',
  hasMore: true,
  loading: false,

  setNotifications: (notifications, cursor, hasMore) => set({ notifications, cursor, hasMore }),
  appendNotifications: (notifications, cursor, hasMore) =>
    set((state) => ({ notifications: [...state.notifications, ...notifications], cursor, hasMore })),
  prependNotification: (notification) =>
    set((state) => ({ notifications: [notification, ...state.notifications] })),
  setUnreadCount: (count) => set({ unreadCount: count }),
  incrementUnread: () => set((state) => ({ unreadCount: state.unreadCount + 1 })),
  markAsRead: (id) =>
    set((state) => ({
      notifications: state.notifications.map((n) => (n.id === id ? { ...n, read: true } : n)),
      unreadCount: Math.max(0, state.unreadCount - 1),
    })),
  markAllAsRead: () =>
    set((state) => ({
      notifications: state.notifications.map((n) => ({ ...n, read: true })),
      unreadCount: 0,
    })),
  setLoading: (loading) => set({ loading }),
  reset: () => set({ notifications: [], unreadCount: 0, cursor: '', hasMore: true, loading: false }),
}));
