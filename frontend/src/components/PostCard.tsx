import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Heart, MessageCircle, Trash2, Bookmark, Repeat2, Pencil } from 'lucide-react';
import { likePost, unlikePost, deletePost, bookmarkPost, unbookmarkPost, repostPost, unrepostPost } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import { useUIStore } from '@/store/ui';
import { timeAgo } from '@/lib/time';
import Avatar from './Avatar';
import EditPostModal from './EditPostModal';
import s from '@/styles/post.module.css';
import type { Post } from '@/types';

function renderContent(content: string) {
  const parts = content.split(/(#\w+)/g);
  return parts.map((part, i) => {
    if (part.startsWith('#')) {
      const tag = part.slice(1).toLowerCase();
      return <Link key={i} to={`/hashtag/${tag}`} onClick={(e) => e.stopPropagation()} className={s.hashtag}>{part}</Link>;
    }
    return part;
  });
}

export default function PostCard({ post, onDelete }: { post: Post; onDelete?: () => void }) {
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const updatePost = useFeedStore((st) => st.updatePost);
  const openAuthModal = useUIStore((st) => st.openAuthModal);
  const [likeLoading, setLikeLoading] = useState(false);
  const [bookmarkLoading, setBookmarkLoading] = useState(false);
  const [repostLoading, setRepostLoading] = useState(false);
  const [editing, setEditing] = useState(false);
  const isOwner = user?.id === post.user_id;

  const requireAuth = (action: () => void) => {
    if (!user) { openAuthModal(); return; }
    action();
  };

  const handleLike = async (e: React.MouseEvent) => {
    e.stopPropagation();
    requireAuth(async () => {
      if (likeLoading) return;
      setLikeLoading(true);
      const wasLiked = post.liked;
      updatePost(post.id, { liked: !wasLiked, likes_count: post.likes_count + (wasLiked ? -1 : 1) });
      try {
        if (wasLiked) await unlikePost(post.id); else await likePost(post.id);
      } catch {
        updatePost(post.id, { liked: wasLiked, likes_count: post.likes_count });
      } finally { setLikeLoading(false); }
    });
  };

  const handleBookmark = async (e: React.MouseEvent) => {
    e.stopPropagation();
    requireAuth(async () => {
      if (bookmarkLoading) return;
      setBookmarkLoading(true);
      const was = post.bookmarked;
      updatePost(post.id, { bookmarked: !was });
      try {
        if (was) await unbookmarkPost(post.id); else await bookmarkPost(post.id);
      } catch {
        updatePost(post.id, { bookmarked: was });
      } finally { setBookmarkLoading(false); }
    });
  };

  const handleRepost = async (e: React.MouseEvent) => {
    e.stopPropagation();
    requireAuth(async () => {
      if (repostLoading) return;
      setRepostLoading(true);
      const was = post.reposted;
      updatePost(post.id, { reposted: !was, reposts_count: post.reposts_count + (was ? -1 : 1) });
      try {
        if (was) await unrepostPost(post.id); else await repostPost(post.id);
      } catch {
        updatePost(post.id, { reposted: was, reposts_count: post.reposts_count });
      } finally { setRepostLoading(false); }
    });
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    try { await deletePost(post.id); useFeedStore.getState().removePost(post.id); onDelete?.(); } catch {}
  };

  return (
    <>
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
              {post.edited_at && <span className={s.editedBadge}>изменено</span>}
              {isOwner && (
                <>
                  <button onClick={(e) => { e.stopPropagation(); setEditing(true); }} className={s.deleteBtn}><Pencil size={14} /></button>
                  <button onClick={handleDelete} className={s.deleteBtn}><Trash2 size={14} /></button>
                </>
              )}
            </div>
            <p className={s.postContent}>{renderContent(post.content)}</p>
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
                onClick={handleRepost}
                className={`${s.actionBtn} ${post.reposted ? s.actionBtnReposted : ''}`}
              >
                <Repeat2 size={17} />
                {post.reposts_count > 0 && <span>{post.reposts_count}</span>}
              </button>
              <button
                onClick={handleLike}
                className={`${s.actionBtn} ${s.actionBtnLike} ${post.liked ? s.actionBtnLiked : ''}`}
              >
                <Heart size={17} fill={post.liked ? 'currentColor' : 'none'} />
                {post.likes_count > 0 && <span>{post.likes_count}</span>}
              </button>
              <button
                onClick={handleBookmark}
                className={`${s.actionBtn} ${post.bookmarked ? s.actionBtnBookmarked : ''}`}
              >
                <Bookmark size={17} fill={post.bookmarked ? 'currentColor' : 'none'} />
              </button>
            </div>
          </div>
        </div>
      </article>
      {editing && <EditPostModal post={post} onClose={() => setEditing(false)} />}
    </>
  );
}
