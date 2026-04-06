import { useState, useEffect, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Shield, Ban, ChevronDown, ChevronUp } from 'lucide-react';
import { listUsers, setAdmin, banUser, getLoginHistory } from '@/api/admin';
import { useAuthStore } from '@/store/auth';
import Avatar from '@/components/Avatar';
import type { User, LoginHistoryEntry } from '@/types';
import l from '@/styles/layout.module.css';
import s from '@/styles/admin.module.css';

export default function AdminPanel() {
  const navigate = useNavigate();
  const currentUser = useAuthStore((st) => st.user);

  const [users, setUsers] = useState<User[]>([]);
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [actionLoading, setActionLoading] = useState<string | null>(null);
  const [expandedUser, setExpandedUser] = useState<string | null>(null);
  const [loginHistory, setLoginHistory] = useState<Record<string, LoginHistoryEntry[]>>({});
  const [historyLoading, setHistoryLoading] = useState<string | null>(null);

  const fetchUsers = useCallback(async (c?: string) => {
    const data = await listUsers(c);
    return data;
  }, []);

  useEffect(() => {
    fetchUsers().then((data) => {
      setUsers(data.users);
      setCursor(data.pagination.next_cursor);
      setHasMore(data.pagination.has_more);
    }).finally(() => setLoading(false));
  }, [fetchUsers]);

  const loadMore = async () => {
    if (!hasMore || loadingMore) return;
    setLoadingMore(true);
    try {
      const data = await fetchUsers(cursor);
      setUsers((prev) => [...prev, ...data.users]);
      setCursor(data.pagination.next_cursor);
      setHasMore(data.pagination.has_more);
    } finally {
      setLoadingMore(false);
    }
  };

  const handleSetAdmin = async (userId: string, isAdmin: boolean) => {
    setActionLoading(userId);
    try {
      const updated = await setAdmin(userId, isAdmin);
      setUsers((prev) => prev.map((u) => (u.id === userId ? { ...u, ...updated } : u)));
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      alert(msg || 'Ошибка');
    } finally {
      setActionLoading(null);
    }
  };

  const handleBan = async (userId: string, isBanned: boolean) => {
    setActionLoading(userId);
    try {
      const updated = await banUser(userId, isBanned);
      setUsers((prev) => prev.map((u) => (u.id === userId ? { ...u, ...updated } : u)));
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      alert(msg || 'Ошибка');
    } finally {
      setActionLoading(null);
    }
  };

  const handleToggleExpand = async (userId: string) => {
    if (expandedUser === userId) {
      setExpandedUser(null);
      return;
    }
    setExpandedUser(userId);
    if (!loginHistory[userId]) {
      setHistoryLoading(userId);
      try {
        const entries = await getLoginHistory(userId);
        setLoginHistory((prev) => ({ ...prev, [userId]: entries }));
      } catch { /* ignore */ } finally {
        setHistoryLoading(null);
      }
    }
  };

  const isSelf = (userId: string) => currentUser?.id === userId;

  return (
    <div>
      <header className={l.pageHeader}>
        <button onClick={() => navigate(-1)} className={l.backBtn}><ArrowLeft size={18} /></button>
        <h1 className={l.pageHeaderTitle}>Админ-панель</h1>
      </header>

      {loading ? (
        <div className={s.emptyState}>Загрузка...</div>
      ) : users.length === 0 ? (
        <div className={s.emptyState}>Нет пользователей</div>
      ) : (
        <div className={s.userList}>
          {users.map((user) => (
            <div key={user.id}>
              <div
                className={s.userRow}
                style={{ cursor: 'pointer' }}
                onClick={() => handleToggleExpand(user.id)}
              >
                <Avatar url={user.avatar_url} name={user.display_name || user.username} size="md" />
                <div className={s.userInfo}>
                  <div className={s.userName}>
                    {user.display_name || user.username}
                    <span className={s.userEmail}>@{user.username}</span>
                  </div>
                  <div className={s.userMeta}>{user.email}</div>
                </div>
                <div className={s.badges}>
                  {user.is_admin && <span className={`${s.badge} ${s.badgeAdmin}`}><Shield size={10} /> Admin</span>}
                  {user.is_banned && <span className={`${s.badge} ${s.badgeBanned}`}><Ban size={10} /> Banned</span>}
                </div>
                <div className={s.actions} onClick={(e) => e.stopPropagation()}>
                  {!isSelf(user.id) && (
                    <>
                      <button
                        className={`${s.actionBtn} ${user.is_admin ? s.actionBtnDanger : ''}`}
                        disabled={actionLoading === user.id}
                        onClick={() => handleSetAdmin(user.id, !user.is_admin)}
                      >
                        {user.is_admin ? 'Снять админа' : 'Сделать админом'}
                      </button>
                      <button
                        className={`${s.actionBtn} ${!user.is_banned ? s.actionBtnDanger : ''}`}
                        disabled={actionLoading === user.id}
                        onClick={() => handleBan(user.id, !user.is_banned)}
                      >
                        {user.is_banned ? 'Разбанить' : 'Забанить'}
                      </button>
                    </>
                  )}
                  <span className={s.expandIcon}>
                    {expandedUser === user.id ? <ChevronUp size={16} /> : <ChevronDown size={16} />}
                  </span>
                </div>
              </div>
              {expandedUser === user.id && (
                <div className={s.loginHistory}>
                  <div className={s.loginHistoryTitle}>История входов</div>
                  {historyLoading === user.id ? (
                    <div className={s.loginHistoryEmpty}>Загрузка...</div>
                  ) : !loginHistory[user.id] || loginHistory[user.id].length === 0 ? (
                    <div className={s.loginHistoryEmpty}>Нет данных</div>
                  ) : (
                    <table className={s.loginHistoryTable}>
                      <thead>
                        <tr>
                          <th>IP</th>
                          <th>User-Agent</th>
                          <th>Время</th>
                        </tr>
                      </thead>
                      <tbody>
                        {loginHistory[user.id].map((entry) => (
                          <tr key={entry.id}>
                            <td>{entry.ip || '—'}</td>
                            <td className={s.loginHistoryUa}>{entry.user_agent || '—'}</td>
                            <td>{new Date(entry.created_at).toLocaleString('ru-RU')}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                  )}
                </div>
              )}
            </div>
          ))}
        </div>
      )}

      {hasMore && (
        <div className={s.loadMore}>
          <button className={s.loadMoreBtn} onClick={loadMore} disabled={loadingMore}>
            {loadingMore ? 'Загрузка...' : 'Загрузить ещё'}
          </button>
        </div>
      )}
    </div>
  );
}
