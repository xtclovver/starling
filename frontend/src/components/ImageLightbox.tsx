import { useEffect, useState, useRef, useCallback } from 'react';
import { X, ChevronLeft, ChevronRight, Heart, MessageCircle, Repeat2, Bookmark } from 'lucide-react';
import { getCommentTree, createComment } from '@/api/comments';
import { likePost, unlikePost, repostPost, unrepostPost, bookmarkPost, unbookmarkPost } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import Avatar from './Avatar';
import CommentItem from './CommentItem';
import Spinner from './Spinner';
import s from '@/styles/image-lightbox.module.css';
import type { Post, Comment } from '@/types';

interface Props {
  src: string;
  allSrcs?: string[];
  post?: Post;
  onClose: () => void;
}

export default function ImageLightbox({ src, allSrcs, post, onClose }: Props) {
  const user = useAuthStore((st) => st.user);
  const updateFeedPost = useFeedStore((st) => st.updatePost);
  const srcs = allSrcs && allSrcs.length > 0 ? allSrcs : [src];
  const [idx, setIdx] = useState(() => {
    const i = srcs.indexOf(src);
    return i >= 0 ? i : 0;
  });

  // Local post state for optimistic updates
  const [localPost, setLocalPost] = useState<Post | undefined>(post);

  // Comments state
  const [comments, setComments] = useState<Comment[]>([]);
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(false);
  const [commentsLoading, setCommentsLoading] = useState(false);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  // Reply state
  const [replyText, setReplyText] = useState('');
  const [replyLoading, setReplyLoading] = useState(false);

  const loadComments = useCallback(async (c = '') => {
    if (!post || loadingRef.current) return;
    loadingRef.current = true;
    setCommentsLoading(true);
    try {
      const data = await getCommentTree(post.id, c);
      if (c) {
        setComments((prev) => [...prev, ...(data.comments || [])]);
      } else {
        setComments(data.comments || []);
      }
      setCursor(data.pagination?.next_cursor || '');
      setHasMore(data.pagination?.has_more || false);
    } catch { /* ignore */ } finally {
      setCommentsLoading(false);
      loadingRef.current = false;
    }
  }, [post]);

  useEffect(() => {
    if (post) {
      setLocalPost(post);
      loadComments();
    }
  }, [loadComments, post]);

  // Infinite scroll via IntersectionObserver
  useEffect(() => {
    if (!sentinelRef.current || !hasMore) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingRef.current) {
          loadComments(cursor);
        }
      },
      { threshold: 0.1 }
    );
    observer.observe(sentinelRef.current);
    return () => observer.disconnect();
  }, [hasMore, cursor, loadComments]);

  // Keyboard
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
      if (e.key === 'ArrowLeft') setIdx((i) => Math.max(0, i - 1));
      if (e.key === 'ArrowRight') setIdx((i) => Math.min(srcs.length - 1, i + 1));
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, srcs.length]);

  const handleLike = async () => {
    if (!localPost || !user) return;
    const wasLiked = localPost.liked;
    const updated = { ...localPost, liked: !wasLiked, likes_count: localPost.likes_count + (wasLiked ? -1 : 1) };
    setLocalPost(updated);
    updateFeedPost(localPost.id, { liked: !wasLiked, likes_count: updated.likes_count });
    try {
      if (wasLiked) await unlikePost(localPost.id); else await likePost(localPost.id);
    } catch {
      setLocalPost(localPost);
      updateFeedPost(localPost.id, { liked: wasLiked, likes_count: localPost.likes_count });
    }
  };

  const handleRepost = async () => {
    if (!localPost || !user) return;
    const wasReposted = localPost.reposted;
    const updated = { ...localPost, reposted: !wasReposted, reposts_count: localPost.reposts_count + (wasReposted ? -1 : 1) };
    setLocalPost(updated);
    updateFeedPost(localPost.id, { reposted: !wasReposted, reposts_count: updated.reposts_count });
    try {
      if (wasReposted) await unrepostPost(localPost.id); else await repostPost(localPost.id);
    } catch {
      setLocalPost(localPost);
      updateFeedPost(localPost.id, { reposted: wasReposted, reposts_count: localPost.reposts_count });
    }
  };

  const handleBookmark = async () => {
    if (!localPost || !user) return;
    const wasBookmarked = localPost.bookmarked;
    setLocalPost({ ...localPost, bookmarked: !wasBookmarked });
    updateFeedPost(localPost.id, { bookmarked: !wasBookmarked });
    try {
      if (wasBookmarked) await unbookmarkPost(localPost.id); else await bookmarkPost(localPost.id);
    } catch {
      setLocalPost(localPost);
      updateFeedPost(localPost.id, { bookmarked: wasBookmarked });
    }
  };

  const handleReply = async () => {
    if (!replyText.trim() || replyLoading || !user || !post) return;
    setReplyLoading(true);
    try {
      const comment = await createComment(post.id, replyText);
      setComments((prev) => [comment, ...prev]);
      setReplyText('');
    } catch { /* ignore */ } finally {
      setReplyLoading(false);
    }
  };

  const multiple = srcs.length > 1;

  return (
    // stopPropagation prevents clicks inside lightbox from bubbling to PostCard/article
    <div className={s.overlay} onClick={(e) => e.stopPropagation()}>
      {/* Image area — clicking dark area closes lightbox */}
      <div className={s.imageArea} onClick={onClose}>
        <button className={s.closeBtn} onClick={onClose}>
          <X size={20} />
        </button>

        {multiple && idx > 0 && (
          <button
            className={`${s.arrowBtn} ${s.arrowLeft}`}
            onClick={(e) => { e.stopPropagation(); setIdx((i) => i - 1); }}
          >
            <ChevronLeft size={24} />
          </button>
        )}

        <img
          src={srcs[idx]}
          alt=""
          className={s.image}
          onClick={(e) => e.stopPropagation()}
        />

        {multiple && idx < srcs.length - 1 && (
          <button
            className={`${s.arrowBtn} ${s.arrowRight}`}
            onClick={(e) => { e.stopPropagation(); setIdx((i) => i + 1); }}
          >
            <ChevronRight size={24} />
          </button>
        )}

        {multiple && (
          <div className={s.dots}>
            {srcs.map((_, i) => (
              <div key={i} className={`${s.dot} ${i === idx ? s.dotActive : ''}`} />
            ))}
          </div>
        )}
      </div>

      {/* Sidebar — only when post is provided */}
      {localPost && (
        <div className={s.sidebar} onClick={(e) => e.stopPropagation()}>
          <div className={s.sidebarScroll}>
            <div className={s.postHeader}>
              <Avatar
                url={localPost.author?.avatar_url}
                name={localPost.author?.display_name || localPost.author?.username || '?'}
              />
              <div className={s.authorInfo}>
                <div className={s.authorName}>
                  {localPost.author?.display_name || localPost.author?.username || 'Unknown'}
                </div>
                {localPost.author?.username && (
                  <div className={s.authorHandle}>@{localPost.author.username}</div>
                )}
              </div>
            </div>

            <p className={s.postText}>{localPost.content}</p>
            <p className={s.postMeta}>
              {new Date(localPost.created_at).toLocaleString('ru-RU', {
                hour: '2-digit', minute: '2-digit',
                day: 'numeric', month: 'short', year: 'numeric',
              })}
            </p>

            <div className={s.postStats}>
              <span><strong>{localPost.reposts_count}</strong> Репостов</span>
              <span><strong>{localPost.likes_count}</strong> Лайков</span>
              {localPost.views_count > 0 && (
                <span><strong>{localPost.views_count}</strong> Просмотров</span>
              )}
            </div>

            <div className={s.actionRow}>
              <button className={s.actionBtn}>
                <MessageCircle size={18} />
                {localPost.comments_count > 0 && <span>{localPost.comments_count}</span>}
              </button>
              <button
                className={`${s.actionBtn} ${localPost.reposted ? s.actionBtnReposted : ''}`}
                onClick={handleRepost}
              >
                <Repeat2 size={18} />
              </button>
              <button
                className={`${s.actionBtn} ${localPost.liked ? s.actionBtnLiked : ''}`}
                onClick={handleLike}
              >
                <Heart size={18} fill={localPost.liked ? 'currentColor' : 'none'} />
              </button>
              <button
                className={`${s.actionBtn} ${localPost.bookmarked ? s.actionBtnBookmarked : ''}`}
                onClick={handleBookmark}
              >
                <Bookmark size={18} fill={localPost.bookmarked ? 'currentColor' : 'none'} />
              </button>
            </div>

            <div className={s.commentsHeader}>Комментарии</div>
            {comments.map((c) => (
              <CommentItem
                key={c.id}
                comment={c}
                postId={localPost.id}
                onNewReply={(pid, reply) => {
                  setComments((prev) =>
                    prev.map((cm) =>
                      cm.id === pid
                        ? { ...cm, children: [reply, ...(cm.children || [])] }
                        : cm
                    )
                  );
                }}
                onDelete={(cid) => {
                  setComments((prev) =>
                    prev
                      .map((cm) =>
                        cm.id === cid
                          ? cm.children?.length
                            ? { ...cm, content: '[удалено]', user_id: '' }
                            : null
                          : cm
                      )
                      .filter(Boolean) as Comment[]
                  );
                }}
              />
            ))}

            {commentsLoading && (
              <div className={s.spinnerWrap}><Spinner /></div>
            )}

            <div ref={sentinelRef} className={s.sentinel} />
          </div>

          {user && (
            <div className={s.replyBox}>
              <Avatar url={user.avatar_url} name={user.display_name || user.username} size="sm" />
              <input
                className={s.replyInput}
                placeholder="Написать ответ..."
                value={replyText}
                onChange={(e) => setReplyText(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && !e.shiftKey) {
                    e.preventDefault();
                    handleReply();
                  }
                }}
              />
              <button
                className={s.replyBtn}
                onClick={handleReply}
                disabled={!replyText.trim() || replyLoading}
              >
                {replyLoading ? '...' : 'Ответить'}
              </button>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
