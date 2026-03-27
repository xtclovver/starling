import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, Heart, MessageCircle, Trash2 } from 'lucide-react';
import { getPost, likePost, unlikePost, deletePost } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import { getMediaKind } from '@/lib/media';
import Avatar from '@/components/Avatar';
import CommentTree from '@/components/CommentTree';
import Spinner from '@/components/Spinner';
import l from '@/styles/layout.module.css';
import s from '@/styles/post.module.css';
import type { Post } from '@/types';

export default function PostPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const [post, setPost] = useState<Post | null>(null);
  const [loading, setLoading] = useState(true);
  const [likeLoading, setLikeLoading] = useState(false);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    getPost(id).then((p) => setPost(p)).catch(() => setPost(null)).finally(() => setLoading(false));
  }, [id]);

  const handleLike = async () => {
    if (!post || likeLoading || !user) return;
    setLikeLoading(true);
    const wasLiked = post.liked;
    setPost({ ...post, liked: !wasLiked, likes_count: post.likes_count + (wasLiked ? -1 : 1) });
    try {
      if (wasLiked) await unlikePost(post.id); else await likePost(post.id);
    } catch { setPost({ ...post }); }
    finally { setLikeLoading(false); }
  };

  const handleDelete = async () => {
    if (!post) return;
    try { await deletePost(post.id); navigate('/', { replace: true }); } catch {}
  };

  const header = (
    <header className={l.pageHeader}>
      <button onClick={() => navigate(-1)} className={l.backBtn}><ArrowLeft size={18} /></button>
      <h1 className={l.pageHeaderTitle}>Пост</h1>
    </header>
  );

  if (loading) return <div>{header}<Spinner /></div>;
  if (!post) return <div>{header}<p style={{ color: 'var(--text-secondary)', textAlign: 'center', padding: '64px 0' }}>Пост не найден</p></div>;

  const isOwner = user?.id === post.user_id;

  return (
    <div>
      {header}
      <article className={s.postDetail}>
        <div className={s.postDetailHeader}>
          <Link to={`/profile/${post.user_id}`}>
            <Avatar url={post.author?.avatar_url} name={post.author?.display_name || post.author?.username} />
          </Link>
          <div className={s.postDetailAuthorInfo}>
            <Link to={`/profile/${post.user_id}`} className={s.postDetailName}>
              {post.author?.display_name || post.author?.username || 'Unknown'}
            </Link>
            {post.author?.username && <p className={s.postDetailHandle}>@{post.author.username}</p>}
          </div>
          {isOwner && (
            <button onClick={handleDelete} className={s.deleteBtn}><Trash2 size={16} /></button>
          )}
        </div>

        <p className={s.postDetailContent}>{post.content}</p>

        {post.media_url && (() => {
          const kind = getMediaKind(post.media_url);
          if (kind === 'video') return (
            <div className={s.postMedia} style={{ marginTop: 12 }}>
              <video src={post.media_url} controls style={{ width: '100%', borderRadius: 12, maxHeight: 480 }} />
            </div>
          );
          if (kind === 'audio') return (
            <div style={{ marginTop: 12 }}>
              <audio src={post.media_url} controls style={{ width: '100%' }} />
            </div>
          );
          return (
            <div className={s.postMedia} style={{ marginTop: 12 }}>
              <img src={post.media_url} alt="" style={{ maxHeight: 600 }} loading="lazy" />
            </div>
          );
        })()}

        <p className={s.postDetailTimestamp}>
          {new Date(post.created_at).toLocaleString('ru-RU', { hour: '2-digit', minute: '2-digit', day: 'numeric', month: 'short', year: 'numeric' })}
        </p>

        <div className={s.postDetailStats}>
          <span><span className={s.statBold}>{post.likes_count}</span> <span className={s.statLabel}>нравится</span></span>
          <span><span className={s.statBold}>{post.comments_count}</span> <span className={s.statLabel}>комментариев</span></span>
        </div>

        <div className={s.postDetailActions}>
          <button className={s.postDetailActionBtn}><MessageCircle size={20} /></button>
          <button
            onClick={handleLike}
            className={`${s.postDetailActionBtn} ${s.postDetailLikeBtn} ${post.liked ? s.postDetailLiked : ''}`}
          >
            <Heart size={20} fill={post.liked ? 'currentColor' : 'none'} />
          </button>
        </div>
      </article>

      <CommentTree postId={post.id} />
    </div>
  );
}
