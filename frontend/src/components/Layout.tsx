import { useEffect, useRef } from 'react';
import { Outlet, useNavigate } from 'react-router-dom';
import { useAuthStore } from '@/store/auth';
import { useWsStore } from '@/store/ws';
import { useFeedStore } from '@/store/feed';
import { getUser } from '@/api/users';
import { WSClient } from '@/lib/websocket';
import Sidebar from './Sidebar';
import RightPanel from './RightPanel';
import s from '@/styles/layout.module.css';
import type { Post } from '@/types';

export default function Layout() {
  const navigate = useNavigate();
  const { isAuthenticated, accessToken, user, login: setUser, logout } = useAuthStore();
  const { setConnected, setReconnecting } = useWsStore();
  const prependPost = useFeedStore((st) => st.prependPost);
  const wsRef = useRef<WSClient | null>(null);

  useEffect(() => {
    if (!isAuthenticated || user) return;
    const token = accessToken;
    if (!token) return;
    try {
      const payload = JSON.parse(atob(token.split('.')[1]));
      const userId = payload.sub;
      if (userId) {
        getUser(userId).then((u) => {
          const rt = localStorage.getItem('refresh_token') || '';
          setUser(u, token, rt);
        }).catch(() => { logout(); navigate('/login'); });
      }
    } catch { logout(); navigate('/login'); }
  }, [isAuthenticated, user, accessToken, setUser, logout, navigate]);

  useEffect(() => {
    if (!isAuthenticated || !accessToken) return;
    wsRef.current = new WSClient(accessToken, (connected, reconnecting) => {
      setConnected(connected);
      setReconnecting(reconnecting);
    });
    const unsub = wsRef.current.on('new_post', (data) => prependPost(data as Post));
    return () => { unsub(); wsRef.current?.disconnect(); };
  }, [isAuthenticated, accessToken, setConnected, setReconnecting, prependPost]);

  return (
    <div className={s.shell}>
      <div className={s.shellInner}>
        <div className={s.sidebarCol}><Sidebar /></div>
        <main className={s.mainCol}><Outlet /></main>
        <div className={s.rightCol}><RightPanel /></div>
      </div>
    </div>
  );
}
