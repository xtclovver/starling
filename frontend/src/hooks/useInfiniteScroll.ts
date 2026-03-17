import { useEffect, useRef, useCallback } from 'react';

export function useInfiniteScroll(onLoadMore: () => void, hasMore: boolean, loading: boolean) {
  const sentinelRef = useRef<HTMLDivElement | null>(null);

  const setSentinel = useCallback((node: HTMLDivElement | null) => {
    sentinelRef.current = node;
  }, []);

  useEffect(() => {
    const node = sentinelRef.current;
    if (!node || !hasMore || loading) return;

    const observer = new IntersectionObserver(
      (entries) => { if (entries[0].isIntersecting) onLoadMore(); },
      { rootMargin: '200px' }
    );
    observer.observe(node);
    return () => observer.disconnect();
  }, [onLoadMore, hasMore, loading]);

  return setSentinel;
}
