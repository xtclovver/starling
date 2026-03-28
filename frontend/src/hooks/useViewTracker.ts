import { useEffect, useRef, useCallback } from 'react';
import { recordViews } from '@/api/posts';

const FLUSH_INTERVAL = 5000; // 5 seconds
const VISIBILITY_THRESHOLD = 1000; // 1 second

export function useViewTracker() {
  const pendingRef = useRef<Set<string>>(new Set());
  const sentRef = useRef<Set<string>>(new Set());
  const timersRef = useRef<Map<string, ReturnType<typeof setTimeout>>>(new Map());

  const flush = useCallback(() => {
    const ids = Array.from(pendingRef.current);
    if (!ids.length) return;
    pendingRef.current.clear();
    recordViews(ids).catch(() => {});
  }, []);

  useEffect(() => {
    const interval = setInterval(flush, FLUSH_INTERVAL);
    return () => {
      clearInterval(interval);
      flush();
    };
  }, [flush]);

  const trackRef = useCallback((postId: string) => {
    return (el: HTMLElement | null) => {
      if (!el) {
        const timer = timersRef.current.get(postId);
        if (timer) {
          clearTimeout(timer);
          timersRef.current.delete(postId);
        }
        return;
      }

      const observer = new IntersectionObserver(
        ([entry]) => {
          if (entry.isIntersecting) {
            if (sentRef.current.has(postId)) return;
            const timer = setTimeout(() => {
              pendingRef.current.add(postId);
              sentRef.current.add(postId);
              timersRef.current.delete(postId);
            }, VISIBILITY_THRESHOLD);
            timersRef.current.set(postId, timer);
          } else {
            const timer = timersRef.current.get(postId);
            if (timer) {
              clearTimeout(timer);
              timersRef.current.delete(postId);
            }
          }
        },
        { threshold: 0.5 }
      );

      observer.observe(el);

      // Store cleanup on the element
      (el as any).__viewObserver = observer;
    };
  }, []);

  return { trackRef };
}
