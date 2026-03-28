import { useEffect, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { X, Heart, Eye } from 'lucide-react';
import { getPost, likePost, unlikePost, recordViews } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import MediaGrid from './MediaGrid';
import CommentTree from './CommentTree';
import Avatar from './Avatar';
import ImageLightbox from './ImageLightbox';
import Spinner from './Spinner';
import s from '@/styles/post-modal.module.css';
import ps from '@/styles/post.module.css';
import type { Post } from '@/types';

interface Props {
  postId: string;
  onClose: () => void;
}

export default function PostModal({ postId, onClose }: Props) {
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const updateFeedPost = useFeedStore((st) => st.updatePost);
  const [post, setPost] = useState<Post | null>(null);
  const [loading, setLoading] = useState(true);
  const [likeLoading, setLikeLoading] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  useEffect(() => {
    setLoading(true);
    getPost(postId)
      .then((p) => setPost(p))
      .catch(() => setPost(null))
      .finally(() => setLoading(false));
    recordViews([postId]).catch(() => {});
  }, [postId]);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    document.addEventListener('keydown', onKey);
    document.body.style.overflow = 'hidden';
    return () => {
      document.removeEventListener('keydown', onKey);
      document.body.style.overflow = '';
    };
  }, [onClose]);

  useEffect(() => {
    window.history.pushState({ postModal: true }, '', `/post/${postId}`);
    const onPop = () => onClose();
    window.addEventListener('popstate', onPop);
    return () => window.removeEventListener('popstate', onPop);
  }, [postId, onClose]);

  const handleLike = async () => {
    if (!post || likeLoading || !user) return;
    setLikeLoading(true);
    const wasLiked = post.liked;
    const newPost = { ...post, liked: !wasLiked, likes_count: post.likes_count + (wasLiked ? -1 : 1) };
    setPost(newPost);
    updateFeedPost(post.id, { liked: !wasLiked, likes_count: newPost.likes_count });
    try {
      if (wasLiked) await unlikePost(post.id); else await likePost(post.id);
    } catch {
      setPost(post);
      updateFeedPost(post.id, { liked: wasLiked, likes_count: post.likes_count });
    } finally { setLikeLoading(false); }
  };

  const hasMedia = post?.media && post.media.length > 0;

  return (
    <>
      <div className={s.backdrop} onClick={onClose}>
        <div className={`${s.modal} ${hasMedia ? s.modalWithMedia : ''}`} onClick={(e) => e.stopPropagation()}>
          <button className={s.closeBtn} onClick={onClose}><X size={22} /></button>

          {loading ? (
            <div className={s.loading}><Spinner /></div>
          ) : !post ? (
            <div className={s.loading}><p style={{ color: 'var(--text-secondary)' }}>Пост не найден</p></div>
          ) : hasMedia ? (
            <div className={s.twoCol}>
              <div className={s.mediaCol}>
                <MediaGrid media={post.media} onImageClick={(url) => setLightboxSrc(url)} />
              </div>
              <div className={s.contentCol}>
                <PostContent
                  post={post}
                  onLike={handleLike}
                  navigate={navigate}
                />
                <CommentTree postId={post.id} />
              </div>
            </div>
          ) : (
            <div className={s.singleCol}>
              <PostContent
                post={post}
                onLike={handleLike}
                navigate={navigate}
              />
              <CommentTree postId={post.id} />
            </div>
          )}
        </div>
      </div>
      {lightboxSrc && <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
    </>
  );
}

function PostContent({ post, onLike, navigate }: { post: Post; onLike: () => void; navigate: ReturnType<typeof useNavigate> }) {
  return (
    <div className={s.postContent}>
      <div className={s.header}>
        <div onClick={() => navigate(`/profile/${post.user_id}`)} style={{ cursor: 'pointer' }}>
          <Avatar url={post.author?.avatar_url} name={post.author?.display_name || post.author?.username || '?'} />
        </div>
        <div>
          <div className={s.authorName} onClick={() => navigate(`/profile/${post.user_id}`)}>
            {post.author?.display_name || post.author?.username || 'Unknown'}
          </div>
          {post.author?.username && <div className={s.handle}>@{post.author.username}</div>}
        </div>
      </div>
      <p className={s.text}>{post.content}</p>
      <p className={s.timestamp}>
        {new Date(post.created_at).toLocaleString('ru-RU', { hour: '2-digit', minute: '2-digit', day: 'numeric', month: 'short', year: 'numeric' })}
        {post.edited_at && <span className={ps.editedBadge}> &middot; изменено</span>}
      </p>
      <div className={s.stats}>
        <span><strong>{post.likes_count}</strong> нравится</span>
        <span><strong>{post.comments_count}</strong> комментариев</span>
        {post.views_count > 0 && <span><Eye size={14} /> <strong>{post.views_count}</strong> просмотров</span>}
      </div>
      <div className={s.actions}>
        <button onClick={onLike} className={`${s.actionBtn} ${post.liked ? s.liked : ''}`}>
          <Heart size={20} fill={post.liked ? 'currentColor' : 'none'} />
        </button>
      </div>
    </div>
  );
}
