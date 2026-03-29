import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, Heart, MessageCircle, Trash2, Eye } from 'lucide-react';
import { getPost, likePost, unlikePost, deletePost, recordViews } from '@/api/posts';
import { getCommentTree } from '@/api/comments';
import { useAuthStore } from '@/store/auth';
import Avatar from '@/components/Avatar';
import MediaGrid from '@/components/MediaGrid';
import CommentTree from '@/components/CommentTree';
import SkeletonPost from '@/components/SkeletonPost';
import ImageLightbox from '@/components/ImageLightbox';
import l from '@/styles/layout.module.css';
import s from '@/styles/post.module.css';
import c from '@/styles/components.module.css';
import type { Post, Comment } from '@/types';

interface CommentInitialData {
  comments: Comment[];
  cursor: string;
  hasMore: boolean;
}

function SkeletonComment() {
  return (
    <div style={{ display: 'flex', gap: 10, padding: '10px 16px', borderBottom: '1px solid var(--border)', animation: 'pulse 1.5s ease-in-out infinite' }}>
      <div className={c.skeletonCircle} style={{ width: 32, height: 32 }} />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 7 }}>
        <div className={c.skeletonLine} style={{ width: '35%' }} />
        <div className={c.skeletonLine} style={{ width: '70%' }} />
      </div>
    </div>
  );
}

export default function PostPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const [post, setPost] = useState<Post | null>(null);
  const [commentInitialData, setCommentInitialData] = useState<CommentInitialData | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [likeLoading, setLikeLoading] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([
      getPost(id),
      getCommentTree(id, '').catch(() => null),
    ]).then(([p, commentData]) => {
      setPost(p);
      if (commentData) {
        setCommentInitialData({
          comments: commentData.comments || [],
          cursor: commentData.pagination?.next_cursor || '',
          hasMore: commentData.pagination?.has_more || false,
        });
      }
    }).catch(() => setPost(null)).finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!id) return;
    recordViews([id]).catch(() => {});
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
    try { await deletePost(post.id); navigate('/', { replace: true }); } catch { /* ignore */ }
  };

  const header = (
    <header className={l.pageHeader}>
      <button onClick={() => navigate(-1)} className={l.backBtn}><ArrowLeft size={18} /></button>
      <h1 className={l.pageHeaderTitle}>Пост</h1>
    </header>
  );

  if (loading) {
    return (
      <div>
        {header}
        <SkeletonPost />
        <SkeletonComment />
        <SkeletonComment />
      </div>
    );
  }
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

        {post.media && post.media.length > 0 && (
          <div style={{ marginTop: 12 }}>
            <MediaGrid media={post.media} onImageClick={(url) => setLightboxSrc(url)} />
          </div>
        )}

        <p className={s.postDetailTimestamp}>
          {new Date(post.created_at).toLocaleString('ru-RU', { hour: '2-digit', minute: '2-digit', day: 'numeric', month: 'short', year: 'numeric' })}
        </p>

        <div className={s.postDetailStats}>
          <span><span className={s.statBold}>{post.likes_count}</span> <span className={s.statLabel}>нравится</span></span>
          <span><span className={s.statBold}>{post.comments_count}</span> <span className={s.statLabel}>комментариев</span></span>
          {post.views_count > 0 && (
            <span className={s.viewCount}><Eye size={15} /> <span className={s.statBold}>{post.views_count}</span> <span className={s.statLabel}>просмотров</span></span>
          )}
        </div>

        <div className={s.postDetailActions}>
          <button className={s.postDetailActionBtn}><MessageCircle size={20} /></button>
          <button
            onClick={handleLike}
            disabled={likeLoading}
            className={`${s.postDetailActionBtn} ${s.postDetailLikeBtn} ${post.liked ? s.postDetailLiked : ''}`}
            style={likeLoading ? { opacity: 0.5 } : undefined}
          >
            <Heart size={20} fill={post.liked ? 'currentColor' : 'none'} />
          </button>
        </div>
      </article>

      <CommentTree postId={post.id} initialData={commentInitialData} />
      {lightboxSrc && post && (
        <ImageLightbox
          src={lightboxSrc}
          allSrcs={(post.media || []).map((m) => m.url)}
          post={post}
          onClose={() => setLightboxSrc(null)}
        />
      )}
    </div>
  );
}
