import { useEffect, useCallback, useState } from 'react';
import { getBookmarks } from '@/api/posts';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import PostCard from '@/components/PostCard';
import SkeletonPost from '@/components/SkeletonPost';
import Spinner from '@/components/Spinner';
import s from '@/styles/layout.module.css';
import p from '@/styles/profile.module.css';
import type { Post } from '@/types';

export default function Bookmarks() {
  const [posts, setPosts] = useState<Post[]>([]);
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (c = '') => {
    setLoading(true);
    try {
      const data = await getBookmarks(c);
      const items = data.posts || [];
      if (c) setPosts((prev) => [...prev, ...items]);
      else setPosts(items);
      setCursor(data.pagination?.next_cursor || '');
      setHasMore(data.pagination?.has_more || false);
    } catch {}
    finally { setLoading(false); }
  }, []);

  useEffect(() => { load(); }, [load]);

  const loadMore = useCallback(() => {
    if (cursor && !loading) load(cursor);
  }, [cursor, loading, load]);

  const sentinelRef = useInfiniteScroll(loadMore, hasMore, loading);

  return (
    <div>
      <header className={s.pageHeader}>
        <h1 className={s.pageHeaderTitle}>Закладки</h1>
      </header>
      {loading && posts.length === 0 ? (
        <>{[1,2,3].map((i) => <SkeletonPost key={i} />)}</>
      ) : (
        <>
          {posts.map((post) => (
            <PostCard
              key={post.id}
              post={post}
              onUnbookmark={() => setPosts((prev) => prev.filter((p) => p.id !== post.id))}
            />
          ))}
          <div ref={sentinelRef} />
          {loading && posts.length > 0 && <Spinner />}
          {!hasMore && posts.length > 0 && <p className={p.empty}>Все закладки загружены</p>}
          {!loading && posts.length === 0 && <p className={p.empty}>Нет закладок</p>}
        </>
      )}
    </div>
  );
}
