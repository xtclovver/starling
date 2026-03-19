import { useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { Heart, MessageCircle, UserPlus, Repeat2 } from 'lucide-react';
import { getNotifications, getUnreadCount, markRead, markAllRead } from '@/api/notifications';
import { useNotificationStore } from '@/store/notifications';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import Avatar from '@/components/Avatar';
import Spinner from '@/components/Spinner';
import { timeAgo } from '@/lib/time';
import s from '@/styles/layout.module.css';
import n from '@/styles/notification.module.css';

const TYPE_CONFIG: Record<string, { icon: typeof Heart; label: string; color: string }> = {
  like_post: { icon: Heart, label: 'понравился ваш пост', color: 'var(--like)' },
  like_comment: { icon: Heart, label: 'понравился ваш комментарий', color: 'var(--like)' },
  new_comment: { icon: MessageCircle, label: 'прокомментировал ваш пост', color: 'var(--accent)' },
  new_follower: { icon: UserPlus, label: 'подписался на вас', color: 'var(--success)' },
  repost: { icon: Repeat2, label: 'репостнул ваш пост', color: 'var(--success)' },
  quote: { icon: Repeat2, label: 'процитировал ваш пост', color: 'var(--accent)' },
};

export default function Notifications() {
  const { notifications, cursor, hasMore, loading, setNotifications, appendNotifications, setLoading, setUnreadCount, markAllAsRead } = useNotificationStore();

  const load = useCallback(async (c = '') => {
    setLoading(true);
    try {
      const data = await getNotifications(c);
      const items = data.notifications || [];
      if (c) appendNotifications(items, data.pagination?.next_cursor || '', data.pagination?.has_more || false);
      else setNotifications(items, data.pagination?.next_cursor || '', data.pagination?.has_more || false);
    } catch {}
    finally { setLoading(false); }
  }, [setNotifications, appendNotifications, setLoading]);

  useEffect(() => {
    load();
    getUnreadCount().then(setUnreadCount).catch(() => {});
  }, [load, setUnreadCount]);

  const loadMore = useCallback(() => {
    if (cursor && !loading) load(cursor);
  }, [cursor, loading, load]);

  const sentinelRef = useInfiniteScroll(loadMore, hasMore, loading);

  const handleMarkAllRead = async () => {
    try {
      await markAllRead();
      markAllAsRead();
    } catch {}
  };

  const handleClickNotification = async (id: string, read: boolean) => {
    if (!read) {
      try { await markRead(id); useNotificationStore.getState().markAsRead(id); } catch {}
    }
  };

  return (
    <div>
      <header className={s.pageHeader}>
        <h1 className={s.pageHeaderTitle}>Уведомления</h1>
        {notifications.some((n) => !n.read) && (
          <button onClick={handleMarkAllRead} className={n.markAllBtn}>Прочитать все</button>
        )}
      </header>
      {loading && notifications.length === 0 ? (
        <Spinner />
      ) : (
        <>
          {notifications.map((notif) => {
            const config = TYPE_CONFIG[notif.type] || TYPE_CONFIG.like_post;
            const Icon = config.icon;
            return (
              <div
                key={notif.id}
                className={`${n.item} ${!notif.read ? n.itemUnread : ''}`}
                onClick={() => handleClickNotification(notif.id, notif.read)}
              >
                <div className={n.iconWrap} style={{ color: config.color }}>
                  <Icon size={18} />
                </div>
                <div className={n.body}>
                  <Link to={`/profile/${notif.actor_id}`} className={n.actorRow}>
                    <Avatar url={notif.actor?.avatar_url} name={notif.actor?.display_name || notif.actor?.username || '?'} size="xs" />
                    <span className={n.actorName}>{notif.actor?.display_name || notif.actor?.username || 'Пользователь'}</span>
                  </Link>
                  <p className={n.text}>{config.label}</p>
                  <span className={n.time}>{timeAgo(notif.created_at)}</span>
                </div>
              </div>
            );
          })}
          <div ref={sentinelRef} />
          {loading && notifications.length > 0 && <Spinner />}
          {!loading && notifications.length === 0 && <p style={{ padding: 24, textAlign: 'center', color: 'var(--text-tertiary)' }}>Нет уведомлений</p>}
        </>
      )}
    </div>
  );
}
