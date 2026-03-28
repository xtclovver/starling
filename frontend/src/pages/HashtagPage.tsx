import { useEffect, useCallback, useState } from 'react';
import { useParams } from 'react-router-dom';
import { getPostsByHashtag } from '@/api/posts';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import PostCard from '@/components/PostCard';
import SkeletonPost from '@/components/SkeletonPost';
import Spinner from '@/components/Spinner';
import s from '@/styles/layout.module.css';
import p from '@/styles/profile.module.css';
import type { Post } from '@/types';

export default function HashtagPage() {
  const { tag } = useParams<{ tag: string }>();
  const [posts, setPosts] = useState<Post[]>([]);
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(true);
  const [loading, setLoading] = useState(false);

  const load = useCallback(async (c = '') => {
    if (!tag) return;
    setLoading(true);
    try {
      const data = await getPostsByHashtag(tag, c);
      const items = data.posts || [];
      if (c) setPosts((prev) => [...prev, ...items]);
      else setPosts(items);
      setCursor(data.pagination?.next_cursor || '');
      setHasMore(data.pagination?.has_more || false);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [tag]);

  useEffect(() => { setPosts([]); setCursor(''); setHasMore(true); load(); }, [load]);

  const loadMore = useCallback(() => {
    if (cursor && !loading) load(cursor);
  }, [cursor, loading, load]);

  const sentinelRef = useInfiniteScroll(loadMore, hasMore, loading);

  return (
    <div>
      <header className={s.pageHeader}>
        <h1 className={s.pageHeaderTitle}>#{tag}</h1>
      </header>
      {loading && posts.length === 0 ? (
        <>{[1,2,3].map((i) => <SkeletonPost key={i} />)}</>
      ) : (
        <>
          {posts.map((post) => <PostCard key={post.id} post={post} />)}
          <div ref={sentinelRef} />
          {loading && posts.length > 0 && <Spinner />}
          {!hasMore && posts.length > 0 && <p className={p.empty}>Все посты загружены</p>}
          {!loading && posts.length === 0 && <p className={p.empty}>Нет постов с #{tag}</p>}
        </>
      )}
    </div>
  );
}
