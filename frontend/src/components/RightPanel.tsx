import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import SearchUsers from './SearchUsers';
import { useAuthStore } from '@/store/auth';
import { useUIStore } from '@/store/ui';
import { getTrendingHashtags } from '@/api/posts';
import { getRecommendedUsers, follow } from '@/api/users';
import Avatar from './Avatar';
import s from '@/styles/layout.module.css';
import type { TrendingHashtag, User } from '@/types';

export default function RightPanel() {
  const user = useAuthStore((st) => st.user);
  const openAuthModal = useUIStore((st) => st.openAuthModal);
  const [trends, setTrends] = useState<TrendingHashtag[]>([]);
  const [recommended, setRecommended] = useState<User[]>([]);
  const [followedIds, setFollowedIds] = useState<Set<string>>(new Set());
  const [trendsLoading, setTrendsLoading] = useState(true);
  const [recommendedLoading, setRecommendedLoading] = useState(true);

  useEffect(() => {
    getTrendingHashtags()
      .then(setTrends)
      .catch(() => {})
      .finally(() => setTrendsLoading(false));
    getRecommendedUsers()
      .then(setRecommended)
      .catch(() => {})
      .finally(() => setRecommendedLoading(false));
  }, []);

  const handleFollow = async (targetId: string) => {
    if (!user) { openAuthModal(); return; }
    try {
      await follow(targetId);
      setFollowedIds((prev) => new Set(prev).add(targetId));
    } catch { /* ignore */ }
  };

  return (
    <aside className={s.rightPanel}>
      <SearchUsers />

      {trendsLoading ? (
        <div className={s.skeletonBox}>
          <div className={s.skeletonBoxTitle} />
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className={s.skeletonTrendRow}>
              <div className={s.skeletonTrendTag} />
              <div className={s.skeletonTrendCount} />
            </div>
          ))}
        </div>
      ) : trends.length > 0 ? (
        <div className={s.infoBox}>
          <h3 className={s.infoBoxTitle}>Тренды</h3>
          {trends.slice(0, 5).map((t) => (
            <Link key={t.tag} to={`/hashtag/${t.tag}`} className={s.trendItem}>
              <span className={s.trendTag}>#{t.tag}</span>
              <span className={s.trendCount}>{t.post_count} постов</span>
            </Link>
          ))}
        </div>
      ) : null}

      {recommendedLoading ? (
        <div className={s.skeletonBox}>
          <div className={s.skeletonBoxTitle} />
          {[1, 2, 3].map((i) => (
            <div key={i} className={s.skeletonRecommendRow}>
              <div className={s.skeletonRecommendCircle} />
              <div className={s.skeletonRecommendLines}>
                <div className={s.skeletonRecommendLine} style={{ width: '60%' }} />
                <div className={s.skeletonRecommendLine} style={{ width: '40%' }} />
              </div>
            </div>
          ))}
        </div>
      ) : recommended.length > 0 ? (
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
      ) : null}

    </aside>
  );
}
