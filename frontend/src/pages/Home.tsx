import { useEffect, useCallback, useState } from 'react';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import { getFeed, getGlobalFeed } from '@/api/posts';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import CreatePost from '@/components/CreatePost';
import PostCard from '@/components/PostCard';
import SkeletonPost from '@/components/SkeletonPost';
import Spinner from '@/components/Spinner';
import s from '@/styles/layout.module.css';
import p from '@/styles/profile.module.css';
import type { Post } from '@/types';

export default function Home() {
  const isAuthenticated = useAuthStore((st) => st.isAuthenticated);
  const { posts, cursor, hasMore, loading, pendingPosts, setPosts, appendPosts, flushPendingPosts, setLoading } = useFeedStore();
  const [guestPosts, setGuestPosts] = useState<Post[]>([]);
  const [guestCursor, setGuestCursor] = useState('');
  const [guestHasMore, setGuestHasMore] = useState(true);
  const [guestLoading, setGuestLoading] = useState(false);

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

  const loadGuestFeed = useCallback(async (c = '') => {
    if (isAuthenticated) return;
    setGuestLoading(true);
    try {
      const data = await getGlobalFeed(c);
      const feedPosts = data.posts || [];
      if (c) {
        setGuestPosts((prev) => [...prev, ...feedPosts]);
      } else {
        setGuestPosts(feedPosts);
      }
      setGuestCursor(data.pagination?.next_cursor || '');
      setGuestHasMore(data.pagination?.has_more || false);
    } catch {}
    finally { setGuestLoading(false); }
  }, [isAuthenticated]);

  useEffect(() => {
    if (isAuthenticated && posts.length === 0) loadFeed();
    if (!isAuthenticated && guestPosts.length === 0) loadGuestFeed();
  }, [isAuthenticated, loadFeed, loadGuestFeed, posts.length, guestPosts.length]);

  const loadMore = useCallback(() => {
    if (isAuthenticated) {
      if (cursor && !loading) loadFeed(cursor);
    } else {
      if (guestCursor && !guestLoading) loadGuestFeed(guestCursor);
    }
  }, [isAuthenticated, cursor, loading, loadFeed, guestCursor, guestLoading, loadGuestFeed]);

  const currentPosts = isAuthenticated ? posts : guestPosts;
  const currentHasMore = isAuthenticated ? hasMore : guestHasMore;
  const currentLoading = isAuthenticated ? loading : guestLoading;
  const sentinelRef = useInfiniteScroll(loadMore, currentHasMore, currentLoading);

  return (
    <div>
      <header className={s.pageHeader}>
        <h1 className={s.pageHeaderTitle}>{isAuthenticated ? 'Главная' : 'Лента'}</h1>
      </header>
      {isAuthenticated && <CreatePost />}
      {isAuthenticated && pendingPosts.length > 0 && (
        <button className={s.newPostsBanner} onClick={flushPendingPosts}>
          {pendingPosts.length === 1 ? '1 новый пост' : `${pendingPosts.length} новых постов`}
        </button>
      )}
      {currentLoading && currentPosts.length === 0 ? (
        <>{[1,2,3,4].map((i) => <SkeletonPost key={i} />)}</>
      ) : (
        <>
          {currentPosts.map((post) => <PostCard key={post.id} post={post} />)}
          <div ref={sentinelRef} />
          {currentLoading && currentPosts.length > 0 && <Spinner />}
          {!currentHasMore && currentPosts.length > 0 && <p className={p.empty}>Вы прочитали все посты</p>}
          {!currentLoading && currentPosts.length === 0 && (
            <p className={p.empty}>
              {isAuthenticated ? 'Лента пуста. Подпишитесь на кого-нибудь!' : 'Пока нет постов'}
            </p>
          )}
        </>
      )}
    </div>
  );
}
