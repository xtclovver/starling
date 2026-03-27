import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import SearchUsers from './SearchUsers';
import { useWsStore } from '@/store/ws';
import { useAuthStore } from '@/store/auth';
import { useUIStore } from '@/store/ui';
import { getTrendingHashtags } from '@/api/posts';
import { getRecommendedUsers, follow } from '@/api/users';
import Avatar from './Avatar';
import s from '@/styles/layout.module.css';
import type { TrendingHashtag, User } from '@/types';

export default function RightPanel() {
  const connected = useWsStore((st) => st.connected);
  const user = useAuthStore((st) => st.user);
  const openAuthModal = useUIStore((st) => st.openAuthModal);
  const [trends, setTrends] = useState<TrendingHashtag[]>([]);
  const [recommended, setRecommended] = useState<User[]>([]);
  const [followedIds, setFollowedIds] = useState<Set<string>>(new Set());

  useEffect(() => {
    getTrendingHashtags().then(setTrends).catch(() => {});
    getRecommendedUsers().then(setRecommended).catch(() => {});
  }, []);

  const handleFollow = async (targetId: string) => {
    if (!user) { openAuthModal(); return; }
    try {
      await follow(targetId);
      setFollowedIds((prev) => new Set(prev).add(targetId));
    } catch {}
  };

  return (
    <aside className={s.rightPanel}>
      <SearchUsers />

      {trends.length > 0 && (
        <div className={s.infoBox}>
          <h3 className={s.infoBoxTitle}>Тренды</h3>
          {trends.slice(0, 5).map((t) => (
            <Link key={t.tag} to={`/hashtag/${t.tag}`} className={s.trendItem}>
              <span className={s.trendTag}>#{t.tag}</span>
              <span className={s.trendCount}>{t.post_count} постов</span>
            </Link>
          ))}
        </div>
      )}

      {recommended.length > 0 && (
        <div className={s.infoBox}>
          <h3 className={s.infoBoxTitle}>Кого читать</h3>
          {recommended.slice(0, 3).map((u) => (
            <div key={u.id} className={s.recommendItem}>
              <Link to={`/profile/${u.id}`} className={s.recommendUser}>
                <Avatar url={u.avatar_url} name={u.display_name || u.username} size="sm" />
                <div className={s.recommendInfo}>
                  <span className={s.recommendName}>{u.display_name || u.username}</span>
                  <span className={s.recommendHandle}>@{u.username}</span>
                </div>
              </Link>
              {!followedIds.has(u.id) && u.id !== user?.id && (
                <button onClick={() => handleFollow(u.id)} className={s.followBtn}>Читать</button>
              )}
              {followedIds.has(u.id) && (
                <span className={s.followedLabel}>Подписан</span>
              )}
            </div>
          ))}
        </div>
      )}

      <div className={s.infoBox}>
        <div className={s.statusRow}>
          <span className={`${s.statusDot} ${connected ? s.statusOnline : s.statusOffline}`} />
          {connected ? 'Live-обновления активны' : 'Офлайн'}
        </div>
      </div>
    </aside>
  );
}
