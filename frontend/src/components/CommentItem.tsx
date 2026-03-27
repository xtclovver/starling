import { useState } from 'react';
import { Link } from 'react-router-dom';
import { Heart, MessageCircle, Trash2, ChevronDown, ChevronUp } from 'lucide-react';
import { likeComment, unlikeComment, deleteComment } from '@/api/comments';
import { useAuthStore } from '@/store/auth';
import { timeAgo } from '@/lib/time';
import Avatar from './Avatar';
import CommentForm from './CommentForm';
import ImageLightbox from './ImageLightbox';
import s from '@/styles/comment.module.css';
import type { Comment } from '@/types';

interface Props {
  comment: Comment;
  postId: string;
  onNewReply?: (parentId: string, comment: Comment) => void;
  onDelete?: (commentId: string) => void;
}

export default function CommentItem({ comment, postId, onNewReply, onDelete }: Props) {
  const user = useAuthStore((st) => st.user);
  const [liked, setLiked] = useState(comment.liked ?? false);
  const [likesCount, setLikesCount] = useState(comment.likes_count);
  const [replying, setReplying] = useState(false);
  const [collapsed, setCollapsed] = useState(false);
  const [likeLoading, setLikeLoading] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  const isOwner = user?.id === comment.user_id;
  const isDeleted = !comment.content || comment.content === '[удалено]';
  const hasChildren = comment.children && comment.children.length > 0;

  const handleLike = async () => {
    if (likeLoading || !user) return;
    setLikeLoading(true);
    const wasLiked = liked;
    setLiked(!wasLiked);
    setLikesCount((c) => c + (wasLiked ? -1 : 1));
    try {
      if (wasLiked) await unlikeComment(comment.id); else await likeComment(comment.id);
    } catch {
      setLiked(wasLiked);
      setLikesCount(comment.likes_count);
    } finally { setLikeLoading(false); }
  };

  const handleDelete = async () => {
    try { await deleteComment(comment.id); onDelete?.(comment.id); } catch {}
  };

  return (
    <>
    <div style={{ paddingLeft: comment.depth > 0 ? 20 : 0 }}>
      <div className={s.commentItem}>
        {comment.depth > 0 && <div className={s.depthLine} />}
        <Link to={`/profile/${comment.user_id}`} style={{ flexShrink: 0 }}>
          <Avatar url={comment.author?.avatar_url} name={comment.author?.display_name || comment.author?.username} size="sm" />
        </Link>
        <div className={s.commentContent}>
          <div className={s.commentMeta}>
            <Link to={`/profile/${comment.user_id}`} className={s.commentAuthor}>
              {comment.author?.display_name || comment.author?.username || 'Unknown'}
            </Link>
            {comment.author?.username && <span className={s.commentHandle}>@{comment.author.username}</span>}
            <span className={s.commentDot}>&middot;</span>
            <span className={s.commentTime}>{timeAgo(comment.created_at)}</span>
          </div>

          {isDeleted ? (
            <p className={`${s.commentText} ${s.commentDeleted}`}>[удалено]</p>
          ) : (
            <>
              <p className={s.commentText}>{comment.content}</p>
              {comment.media_url && (
                <div className={s.commentMedia}>
                  <img
                    src={comment.media_url}
                    alt=""
                    loading="lazy"
                    onClick={() => setLightboxSrc(comment.media_url!)}
                    style={{ cursor: 'zoom-in' }}
                  />
                </div>
              )}
            </>
          )}

          {!isDeleted && (
            <div className={s.commentActions}>
              <button onClick={handleLike} className={`${s.commentActionBtn} ${s.commentLikeBtn} ${liked ? s.commentLiked : ''}`}>
                <Heart size={13} fill={liked ? 'currentColor' : 'none'} />
                {likesCount > 0 && <span>{likesCount}</span>}
              </button>
              {user && comment.depth < 5 && (
                <button onClick={() => setReplying(!replying)} className={`${s.commentActionBtn} ${s.commentReplyBtn}`}>
                  <MessageCircle size={13} /><span>Ответить</span>
                </button>
              )}
              {isOwner && (
                <button onClick={handleDelete} className={`${s.commentActionBtn} ${s.commentDeleteBtn}`}>
                  <Trash2 size={13} />
                </button>
              )}
              {hasChildren && (
                <button onClick={() => setCollapsed(!collapsed)} className={s.commentCollapseBtn}>
                  {collapsed ? <ChevronDown size={13} /> : <ChevronUp size={13} />}
                  <span>{collapsed ? 'Развернуть' : 'Свернуть'}</span>
                </button>
              )}
            </div>
          )}

          {replying && (
            <div className={s.replyForm}>
              <CommentForm
                postId={postId}
                parentId={comment.id}
                onSubmit={(c) => { onNewReply?.(comment.id, c); setReplying(false); }}
                onCancel={() => setReplying(false)}
                compact
              />
            </div>
          )}
        </div>
      </div>

      {!collapsed && hasChildren && (
        <div className={s.commentChildren}>
          {comment.children.map((child) => (
            <CommentItem key={child.id} comment={child} postId={postId} onNewReply={onNewReply} onDelete={onDelete} />
          ))}
        </div>
      )}
    </div>
    {lightboxSrc && <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
  </>
  );
}
