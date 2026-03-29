import { useEffect, useRef } from 'react';
import { Outlet } from 'react-router-dom';
import axios from 'axios';
import { useAuthStore } from '@/store/auth';
import { useWsStore } from '@/store/ws';
import { useFeedStore } from '@/store/feed';
import { useNotificationStore } from '@/store/notifications';
import { getUser } from '@/api/users';
import { getUnreadCount } from '@/api/notifications';
import { WSClient } from '@/lib/websocket';
import Sidebar from './Sidebar';
import RightPanel from './RightPanel';
import AuthModal from './AuthModal';
import AppSkeleton from './AppSkeleton';
import s from '@/styles/layout.module.css';
import type { Post, Notification } from '@/types';

export default function Layout() {
  const { isAuthenticated, accessToken, user, login: setUser, logout, setAccessToken, initializing, setInitializing } = useAuthStore();
  const { setConnected, setReconnecting } = useWsStore();
  const { addPendingPost } = useFeedStore();
  const { prependNotification, incrementUnread, setUnreadCount } = useNotificationStore();
  const wsRef = useRef<WSClient | null>(null);
  const triedRefresh = useRef(false);

  // Silent refresh on app load — attempt to restore session via httpOnly cookie
  useEffect(() => {
    if (accessToken || triedRefresh.current) {
      setInitializing(false);
      return;
    }
    triedRefresh.current = true;
    axios.post('/api/auth/refresh', {}, { withCredentials: true })
      .then(({ data }) => {
        const newAccess = data.data.access_token;
        setAccessToken(newAccess);
        const payload = JSON.parse(atob(newAccess.split('.')[1]));
        return Promise.all([
          getUser(payload.sub),
          getUnreadCount().catch(() => 0),
        ]).then(([u, unreadCount]) => {
          setUser(u, newAccess);
          setUnreadCount(unreadCount as number);
        });
      })
      .catch(() => { /* no valid session */ })
      .finally(() => setInitializing(false));
  }, [accessToken, setAccessToken, setUser, setInitializing, setUnreadCount]);

  // Fetch user profile when we have a token but no user object (fallback for manual login flow)
  useEffect(() => {
    if (!isAuthenticated || user || !accessToken || initializing) return;
    try {
      const payload = JSON.parse(atob(accessToken.split('.')[1]));
      const userId = payload.sub;
      if (userId) {
        getUser(userId).then((u) => setUser(u, accessToken)).catch(() => logout());
      }
    } catch { logout(); }
  }, [isAuthenticated, user, accessToken, initializing, setUser, logout]);

  // WebSocket connection — only re-create when auth state changes, not on every token refresh
  useEffect(() => {
    if (!isAuthenticated) return;
    wsRef.current = new WSClient(
      () => useAuthStore.getState().accessToken ?? '',
      (connected, reconnecting) => {
        setConnected(connected);
        setReconnecting(reconnecting);
      }
    );
    const unsubPost = wsRef.current.on('new_post', (data) => addPendingPost(data as Post));
    const unsubNotif = wsRef.current.on('notification', (data) => {
      prependNotification(data as Notification);
      incrementUnread();
    });
    return () => { unsubPost(); unsubNotif(); wsRef.current?.disconnect(); };
  }, [isAuthenticated, setConnected, setReconnecting, addPendingPost, prependNotification, incrementUnread]);

  if (initializing) {
    return <AppSkeleton />;
  }

  return (
    <div className={s.shell}>
      <div className={s.shellInner}>
        <div className={s.sidebarCol}><Sidebar /></div>
        <main className={s.mainCol}><Outlet /></main>
        <div className={s.rightCol}><RightPanel /></div>
      </div>
      <AuthModal />
    </div>
  );
}
