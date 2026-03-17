import { useEffect, useCallback } from 'react';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import { getFeed } from '@/api/posts';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import CreatePost from '@/components/CreatePost';
import PostCard from '@/components/PostCard';
import SkeletonPost from '@/components/SkeletonPost';
import Spinner from '@/components/Spinner';
import s from '@/styles/layout.module.css';
import a from '@/styles/auth.module.css';
import p from '@/styles/profile.module.css';

export default function Home() {
  const isAuthenticated = useAuthStore((st) => st.isAuthenticated);
  const { posts, cursor, hasMore, loading, setPosts, appendPosts, setLoading } = useFeedStore();

  const loadFeed = useCallback(async (c = '') => {
    if (!isAuthenticated) return;
    setLoading(true);
    try {
      const data = await getFeed(c);
      const feedPosts = data.posts || [];
      if (c) appendPosts(feedPosts, data.pagination?.next_cursor || '', data.pagination?.has_more || false);
      else setPosts(feedPosts, data.pagination?.next_cursor || '', data.pagination?.has_more || false);
    } catch {}
    finally { setLoading(false); }
  }, [isAuthenticated, setPosts, appendPosts, setLoading]);

  useEffect(() => {
    if (posts.length === 0 && isAuthenticated) loadFeed();
  }, [loadFeed, posts.length, isAuthenticated]);

  const loadMore = useCallback(() => {
    if (cursor && !loading) loadFeed(cursor);
  }, [cursor, loading, loadFeed]);

  const sentinelRef = useInfiniteScroll(loadMore, hasMore, loading);

  if (!isAuthenticated) {
    return (
      <div className={a.welcome}>
        <h1 className={a.welcomeTitle}>Добро пожаловать</h1>
        <p className={a.welcomeText}>Делитесь мыслями, подписывайтесь на интересных людей, участвуйте в обсуждениях.</p>
        <div className={a.welcomeActions}>
          <a href="/register" className={a.btnPrimary}>Создать аккаунт</a>
          <a href="/login" className={a.btnOutline}>Войти</a>
        </div>
      </div>
    );
  }

  return (
    <div>
      <header className={s.pageHeader}>
        <h1 className={s.pageHeaderTitle}>Главная</h1>
      </header>
      <CreatePost />
      {loading && posts.length === 0 ? (
        <>{[1,2,3,4].map((i) => <SkeletonPost key={i} />)}</>
      ) : (
        <>
          {posts.map((post) => <PostCard key={post.id} post={post} />)}
          <div ref={sentinelRef} />
          {loading && posts.length > 0 && <Spinner />}
          {!hasMore && posts.length > 0 && <p className={p.empty}>Вы прочитали все посты</p>}
          {!loading && posts.length === 0 && <p className={p.empty}>Лента пуста. Подпишитесь на кого-нибудь!</p>}
        </>
      )}
    </div>
  );
}
