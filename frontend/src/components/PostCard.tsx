import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Heart, MessageCircle, Trash2 } from 'lucide-react';
import { likePost, unlikePost, deletePost } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import { timeAgo } from '@/lib/time';
import Avatar from './Avatar';
import s from '@/styles/post.module.css';
import type { Post } from '@/types';

export default function PostCard({ post, onDelete }: { post: Post; onDelete?: () => void }) {
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const updatePost = useFeedStore((st) => st.updatePost);
  const [likeLoading, setLikeLoading] = useState(false);
  const isOwner = user?.id === post.user_id;

  const handleLike = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (likeLoading || !user) return;
    setLikeLoading(true);
    const wasLiked = post.liked;
    updatePost(post.id, { liked: !wasLiked, likes_count: post.likes_count + (wasLiked ? -1 : 1) });
    try {
      if (wasLiked) await unlikePost(post.id); else await likePost(post.id);
    } catch {
      updatePost(post.id, { liked: wasLiked, likes_count: post.likes_count });
    } finally { setLikeLoading(false); }
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try { await deletePost(post.id); useFeedStore.getState().removePost(post.id); onDelete?.(); } catch {}
  };

  return (
    <article className={s.postCard} onClick={() => navigate(`/post/${post.id}`)}>
      <div className={s.postRow}>
        <Link to={`/profile/${post.user_id}`} onClick={(e) => e.stopPropagation()}>
          <Avatar url={post.author?.avatar_url} name={post.author?.display_name || post.author?.username || '?'} />
        </Link>
        <div className={s.postBody}>
          <div className={s.postMeta}>
            <Link to={`/profile/${post.user_id}`} onClick={(e) => e.stopPropagation()} className={s.postAuthor}>
              {post.author?.display_name || post.author?.username || 'Unknown'}
            </Link>
            {post.author?.username && <span className={s.postHandle}>@{post.author.username}</span>}
            <span className={s.postDot}>&middot;</span>
            <span className={s.postTime}>{timeAgo(post.created_at)}</span>
            {isOwner && (
              <button onClick={handleDelete} className={s.deleteBtn}><Trash2 size={14} /></button>
            )}
          </div>
          <p className={s.postContent}>{post.content}</p>
          {post.media_url && (
            <div className={s.postMedia}><img src={post.media_url} alt="" loading="lazy" /></div>
          )}
          <div className={s.postActions}>
            <button
              onClick={(e) => { e.stopPropagation(); navigate(`/post/${post.id}`); }}
              className={s.actionBtn}
            >
              <MessageCircle size={17} />
              {post.comments_count > 0 && <span>{post.comments_count}</span>}
            </button>
            <button
              onClick={handleLike}
              className={`${s.actionBtn} ${s.actionBtnLike} ${post.liked ? s.actionBtnLiked : ''}`}
            >
              <Heart size={17} fill={post.liked ? 'currentColor' : 'none'} />
              {post.likes_count > 0 && <span>{post.likes_count}</span>}
            </button>
          </div>
        </div>
      </div>
    </article>
  );
}
