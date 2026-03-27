import { useEffect, useState, useCallback } from 'react';
import { useParams } from 'react-router-dom';
import { ArrowLeft, CalendarDays } from 'lucide-react';
import { getUser, follow, unfollow, getFollowers, getFollowing } from '@/api/users';
import { getUserPosts, getUserReposts } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import Avatar from '@/components/Avatar';
import PostCard from '@/components/PostCard';
import UserList from '@/components/UserList';
import SkeletonPost from '@/components/SkeletonPost';
import Spinner from '@/components/Spinner';
import l from '@/styles/layout.module.css';
import s from '@/styles/profile.module.css';
import type { User, Post } from '@/types';

type Tab = 'posts' | 'reposts' | 'followers' | 'following';

export default function Profile() {
  const { id } = useParams<{ id: string }>();
  const me = useAuthStore((st) => st.user);
  const [profile, setProfile] = useState<User | null>(null);
  const [loading, setLoading] = useState(true);
  const [isFollowing, setIsFollowing] = useState(false);
  const [followLoading, setFollowLoading] = useState(false);
  const [tab, setTab] = useState<Tab>('posts');

  const [posts, setPosts] = useState<Post[]>([]);
  const [postsCursor, setPostsCursor] = useState('');
  const [postsHasMore, setPostsHasMore] = useState(false);
  const [postsLoading, setPostsLoading] = useState(false);

  const [reposts, setReposts] = useState<Post[]>([]);
  const [repostsCursor, setRepostsCursor] = useState('');
  const [repostsHasMore, setRepostsHasMore] = useState(false);
  const [repostsLoading, setRepostsLoading] = useState(false);

  const [userList, setUserList] = useState<User[]>([]);
  const [userCursor, setUserCursor] = useState('');
  const [userHasMore, setUserHasMore] = useState(false);
  const [userLoading, setUserLoading] = useState(false);

  const isOwn = me?.id === id;

  useEffect(() => {
    if (!id) return;
    setLoading(true); setTab('posts'); setPosts([]); setReposts([]); setUserList([]);
    getUser(id).then((u) => {
      setProfile(u);
      setIsFollowing(u.is_following ?? false);
    }).catch(() => setProfile(null)).finally(() => setLoading(false));
  }, [id]);

  const loadPosts = useCallback(async (cursor = '') => {
    if (!id) return;
    setPostsLoading(true);
    try {
      const data = await getUserPosts(id, cursor);
      const fetched = data.posts || [];
      if (cursor) setPosts((p) => [...p, ...fetched]); else setPosts(fetched);
      setPostsCursor(data.pagination?.next_cursor || '');
      setPostsHasMore(data.pagination?.has_more || false);
    } catch {} finally { setPostsLoading(false); }
  }, [id]);

  useEffect(() => { if (tab === 'posts' && posts.length === 0 && id) loadPosts(); }, [tab, id, loadPosts, posts.length]);

  const loadReposts = useCallback(async (cursor = '') => {
    if (!id) return;
    setRepostsLoading(true);
    try {
      const data = await getUserReposts(id, cursor);
      const fetched = data.posts || [];
      if (cursor) setReposts((p) => [...p, ...fetched]); else setReposts(fetched);
      setRepostsCursor(data.pagination?.next_cursor || '');
      setRepostsHasMore(data.pagination?.has_more || false);
    } catch {} finally { setRepostsLoading(false); }
  }, [id]);

  useEffect(() => { if (tab === 'reposts' && reposts.length === 0 && id) loadReposts(); }, [tab, id, loadReposts, reposts.length]);

  const loadUsers = useCallback(async (type: 'followers' | 'following', cursor = '') => {
    if (!id) return;
    setUserLoading(true);
    try {
      const fn = type === 'followers' ? getFollowers : getFollowing;
      const data = await fn(id, cursor);
      const fetched = data.users || [];
      if (cursor) setUserList((p) => [...p, ...fetched]); else setUserList(fetched);
      setUserCursor(data.pagination?.next_cursor || '');
      setUserHasMore(data.pagination?.has_more || false);
    } catch {} finally { setUserLoading(false); }
  }, [id]);

  useEffect(() => {
    if ((tab === 'followers' || tab === 'following') && id) { setUserList([]); loadUsers(tab); }
  }, [tab, id, loadUsers]);

  const handleFollow = async () => {
    if (!id || followLoading) return;
    setFollowLoading(true);
    try {
      if (isFollowing) { await unfollow(id); setIsFollowing(false); }
      else { await follow(id); setIsFollowing(true); }
    } catch {} finally { setFollowLoading(false); }
  };

  const loadMorePosts = useCallback(() => {
    if (postsCursor && !postsLoading) loadPosts(postsCursor);
  }, [postsCursor, postsLoading, loadPosts]);

  const sentinelRef = useInfiniteScroll(loadMorePosts, postsHasMore, postsLoading);

  if (loading) return (<div><header className={l.pageHeader}><div style={{ height: 20, width: 128, background: 'var(--bg-tertiary)', borderRadius: 4, animation: 'pulse 1.5s ease-in-out infinite' }} /></header><Spinner /></div>);

  if (!profile) return (<div style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', paddingTop: 80 }}><p style={{ color: 'var(--text-secondary)', fontSize: 18 }}>Пользователь не найден</p></div>);

  const created = new Date(profile.created_at).toLocaleDateString('ru-RU', { month: 'long', year: 'numeric' });

  return (
    <div>
      <header className={l.pageHeader}>
        <button onClick={() => window.history.back()} className={l.backBtn}><ArrowLeft size={18} /></button>
        <div>
          <h1 className={l.pageHeaderTitle}>{profile.display_name || profile.username}</h1>
          <p className={l.pageHeaderSub}>{posts.length} постов</p>
        </div>
      </header>

      <div className={s.banner} style={profile.banner_url ? { background: `url(${profile.banner_url}) center/cover no-repeat` } : undefined} />

      <div className={s.profileHeader}>
        <div className={s.profileTopRow}>
          <Avatar url={profile.avatar_url} name={profile.display_name || profile.username} size="xl" className={s.profileAvatar} />
          {!isOwn && me && (
            <button onClick={handleFollow} disabled={followLoading} className={`${s.followBtn} ${isFollowing ? s.followBtnUnfollow : s.followBtnFollow}`}>
              {isFollowing ? 'Отписаться' : 'Подписаться'}
            </button>
          )}
          {isOwn && <a href="/settings" className={s.editBtn}>Настроить</a>}
        </div>

        <h2 className={s.displayName}>{profile.display_name || profile.username}</h2>
        <p className={s.handle}>@{profile.username}</p>
        {profile.bio && <p className={s.bio}>{profile.bio}</p>}
        <div className={s.joinDate}><CalendarDays size={14} /><span>Присоединился {created}</span></div>

        <div className={s.statsRow}>
          <button onClick={() => setTab('following')} className={s.statBtn}>
            <span className={s.statCount}>{profile.following_count ?? 0}</span> <span className={s.statText}>Подписки</span>
          </button>
          <button onClick={() => setTab('followers')} className={s.statBtn}>
            <span className={s.statCount}>{profile.followers_count ?? 0}</span> <span className={s.statText}>Подписчики</span>
          </button>
        </div>
      </div>

      <div className={s.tabs}>
        {(['posts', 'reposts', 'followers', 'following'] as Tab[]).map((t) => (
          <button key={t} onClick={() => setTab(t)} className={`${s.tab} ${tab === t ? s.tabActive : ''}`}>
            {t === 'posts' ? 'Посты' : t === 'reposts' ? 'Репосты' : t === 'followers' ? 'Подписчики' : 'Подписки'}
          </button>
        ))}
      </div>

      {tab === 'posts' && (
        <>
          {postsLoading && posts.length === 0 ? <>{[1,2,3].map((i) => <SkeletonPost key={i} />)}</> : (
            <>
              {posts.map((p) => <PostCard key={p.id} post={p} />)}
              <div ref={sentinelRef} />
              {postsLoading && posts.length > 0 && <Spinner />}
              {!postsLoading && posts.length === 0 && <p className={s.empty}>Нет постов</p>}
            </>
          )}
        </>
      )}

      {tab === 'reposts' && (
        <>
          {repostsLoading && reposts.length === 0 ? <>{[1,2,3].map((i) => <SkeletonPost key={i} />)}</> : (
            <>
              {reposts.map((p) => <PostCard key={p.id} post={p} />)}
              {repostsLoading && reposts.length > 0 && <Spinner />}
              {!repostsLoading && reposts.length === 0 && <p className={s.empty}>Нет репостов</p>}
              {repostsHasMore && !repostsLoading && (
                <button onClick={() => loadReposts(repostsCursor)} className={s.loadMoreBtn}>Загрузить ещё</button>
              )}
            </>
          )}
        </>
      )}

      {(tab === 'followers' || tab === 'following') && (
        <UserList users={userList} loading={userLoading} hasMore={userHasMore} onLoadMore={() => loadUsers(tab, userCursor)} />
      )}
    </div>
  );
}
