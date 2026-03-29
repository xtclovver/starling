import { useState, useRef, useEffect } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Heart, MessageCircle, Trash2, Bookmark, Repeat2, Pencil, X, ImagePlus, Check, Eye } from 'lucide-react';
import { likePost, unlikePost, deletePost, bookmarkPost, unbookmarkPost, repostPost, unrepostPost, updatePost } from '@/api/posts';
import { uploadMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import { useUIStore } from '@/store/ui';
import { timeAgo } from '@/lib/time';
import { getMediaKind } from '@/lib/media';
import Avatar from './Avatar';
import MediaGrid from './MediaGrid';
import ImageLightbox from './ImageLightbox';
import s from '@/styles/post.module.css';
import type { Post, MediaItem } from '@/types';

function renderContent(content: string) {
  const parts = content.split(/(#\w+|@\w+)/g);
  return parts.map((part, i) => {
    if (part.startsWith('#')) {
      const tag = part.slice(1).toLowerCase();
      return <Link key={i} to={`/hashtag/${tag}`} onClick={(e) => e.stopPropagation()} className={s.hashtag}>{part}</Link>;
    }
    if (part.startsWith('@')) {
      const username = part.slice(1);
      return <Link key={i} to={`/u/${username}`} onClick={(e) => e.stopPropagation()} className={s.mention}>{part}</Link>;
    }
    return part;
  });
}

export default function PostCard({ post, onDelete, onUnbookmark, onOpen }: { post: Post; onDelete?: () => void; onUnbookmark?: () => void; onOpen?: (id: string) => void }) {
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const updateFeedPost = useFeedStore((st) => st.updatePost);
  const openAuthModal = useUIStore((st) => st.openAuthModal);
  const [likeLoading, setLikeLoading] = useState(false);
  const [bookmarkLoading, setBookmarkLoading] = useState(false);
  const [repostLoading, setRepostLoading] = useState(false);
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(post.content);
  const [editMediaItems, setEditMediaItems] = useState<MediaItem[]>(post.media || []);
  const [editNewFiles, setEditNewFiles] = useState<File[]>([]);
  const [editNewPreviews, setEditNewPreviews] = useState<string[]>([]);
  const [editLoading, setEditLoading] = useState(false);
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false);
  const [deleteLoading, setDeleteLoading] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
  const editFileRef = useRef<HTMLInputElement>(null);
  const totalEditMedia = editMediaItems.length + editNewFiles.length;
  const [localOverrides, setLocalOverrides] = useState<Partial<Post>>({});
  const displayPost = { ...post, ...localOverrides };

  useEffect(() => {
    setLocalOverrides({});
  }, [post.id, post.updated_at]);

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
      updateFeedPost(post.id, { liked: !wasLiked, likes_count: post.likes_count + (wasLiked ? -1 : 1) });
      try {
        if (wasLiked) await unlikePost(post.id); else await likePost(post.id);
      } catch {
        updateFeedPost(post.id, { liked: wasLiked, likes_count: post.likes_count });
      } finally { setLikeLoading(false); }
    });
  };

  const handleBookmark = async (e: React.MouseEvent) => {
    e.stopPropagation();
    requireAuth(async () => {
      if (bookmarkLoading) return;
      setBookmarkLoading(true);
      const was = post.bookmarked;
      updateFeedPost(post.id, { bookmarked: !was });
      try {
        if (was) { await unbookmarkPost(post.id); onUnbookmark?.(); } else await bookmarkPost(post.id);
      } catch {
        updateFeedPost(post.id, { bookmarked: was });
      } finally { setBookmarkLoading(false); }
    });
  };

  const handleRepost = async (e: React.MouseEvent) => {
    e.stopPropagation();
    requireAuth(async () => {
      if (repostLoading) return;
      setRepostLoading(true);
      const was = post.reposted;
      updateFeedPost(post.id, { reposted: !was, reposts_count: post.reposts_count + (was ? -1 : 1) });
      try {
        if (was) await unrepostPost(post.id); else await repostPost(post.id);
      } catch {
        updateFeedPost(post.id, { reposted: was, reposts_count: post.reposts_count });
      } finally { setRepostLoading(false); }
    });
  };

  const handleDelete = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (deleteLoading) return;
    setDeleteLoading(true);
    try {
      await deletePost(post.id);
      useFeedStore.getState().removePost(post.id);
      onDelete?.();
    } catch { /* ignore */ } finally {
      setDeleteLoading(false);
      setShowDeleteConfirm(false);
    }
  };

  const startEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    setEditContent(post.content);
    setEditMediaItems(post.media || []);
    setEditNewFiles([]);
    setEditNewPreviews([]);
    setEditing(true);
  };

  const cancelEdit = (e: React.MouseEvent) => {
    e.stopPropagation();
    setEditing(false);
    setEditNewFiles([]);
    editNewPreviews.forEach((u) => URL.revokeObjectURL(u));
    setEditNewPreviews([]);
  };

  const handleEditFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (!files.length) return;
    const allowed = 10 - totalEditMedia;
    const toAdd = files.slice(0, allowed);
    setEditNewFiles((prev) => [...prev, ...toAdd]);
    setEditNewPreviews((prev) => [...prev, ...toAdd.map((f) => URL.createObjectURL(f))]);
    if (editFileRef.current) editFileRef.current.value = '';
  };

  const removeExistingMedia = (idx: number) => {
    setEditMediaItems((prev) => prev.filter((_, i) => i !== idx));
  };

  const removeNewMedia = (idx: number) => {
    URL.revokeObjectURL(editNewPreviews[idx]);
    setEditNewFiles((prev) => prev.filter((_, i) => i !== idx));
    setEditNewPreviews((prev) => prev.filter((_, i) => i !== idx));
  };

  const saveEdit = async (e: React.MouseEvent) => {
    e.stopPropagation();
    if (!editContent.trim() || editContent.length > 280 || editLoading) return;
    setEditLoading(true);
    try {
      const existingUrls = editMediaItems.map((m) => m.url);
      const newUrls: string[] = [];
      for (const file of editNewFiles) {
        const m = await uploadMedia(file);
        newUrls.push(m.url);
      }
      const allUrls = [...existingUrls, ...newUrls];
      const updated = await updatePost(post.id, editContent, allUrls);
      const editedFields: Partial<Post> = { content: updated.content, media: updated.media || [], edited_at: updated.edited_at || new Date().toISOString() };
      updateFeedPost(post.id, editedFields);
      setLocalOverrides(editedFields);
      setEditing(false);
      setEditNewFiles([]);
      editNewPreviews.forEach((u) => URL.revokeObjectURL(u));
      setEditNewPreviews([]);
    } catch { /* ignore */ } finally { setEditLoading(false); }
  };

  const remaining = 280 - editContent.length;

  return (
    <>
    <article className={s.postCard} onClick={() => !editing && !lightboxSrc && (onOpen ? onOpen(post.id) : navigate(`/post/${post.id}`))}>
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
            {displayPost.edited_at && <span className={s.editedBadge}>изменено</span>}
            {isOwner && !editing && (
              <div className={s.ownerActions}>
                <button onClick={startEdit} className={s.ownerBtn} title="Редактировать"><Pencil size={14} /></button>
                <div className={s.deleteWrap}>
                  <button
                    onClick={(e) => { e.stopPropagation(); setShowDeleteConfirm((v) => !v); }}
                    className={s.ownerBtn}
                    title="Удалить"
                  ><Trash2 size={14} /></button>
                  {showDeleteConfirm && (
                    <div className={s.deletePopup} onClick={(e) => e.stopPropagation()}>
                      <p className={s.deletePopupText}>Удалить пост?</p>
                      <div className={s.deletePopupActions}>
                        <button onClick={() => setShowDeleteConfirm(false)} className={s.deletePopupCancel}>Отмена</button>
                        <button onClick={handleDelete} disabled={deleteLoading} className={s.deletePopupConfirm}>
                          {deleteLoading ? '...' : 'Удалить'}
                        </button>
                      </div>
                    </div>
                  )}
                </div>
              </div>
            )}
          </div>

          {editing ? (
            <div className={s.inlineEdit} onClick={(e) => e.stopPropagation()}>
              <textarea
                className={s.inlineEditTextarea}
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                autoFocus
                rows={3}
              />
              {(editMediaItems.length > 0 || editNewPreviews.length > 0) && (
                <div className={s.editMediaGrid}>
                  {editMediaItems.map((m, i) => {
                    const kind = getMediaKind(m.url);
                    return (
                      <div key={`existing-${i}`} className={s.editMediaPreview}>
                        {kind === 'video' ? <video src={m.url} style={{ width: '100%', borderRadius: 8, maxHeight: 200 }} /> :
                         kind === 'audio' ? <audio src={m.url} controls style={{ width: '100%' }} /> :
                         <img src={m.url} alt="" />}
                        <button onClick={() => removeExistingMedia(i)} className={s.mediaRemoveBtn}><X size={14} /></button>
                      </div>
                    );
                  })}
                  {editNewPreviews.map((src, i) => {
                    const file = editNewFiles[i];
                    const kind = file.type.startsWith('video/') ? 'video' : file.type.startsWith('audio/') ? 'audio' : 'image';
                    return (
                      <div key={`new-${i}`} className={s.editMediaPreview}>
                        {kind === 'video' ? <video src={src} style={{ width: '100%', borderRadius: 8, maxHeight: 200 }} /> :
                         kind === 'audio' ? <audio src={src} controls style={{ width: '100%' }} /> :
                         <img src={src} alt="" />}
                        <button onClick={() => removeNewMedia(i)} className={s.mediaRemoveBtn}><X size={14} /></button>
                      </div>
                    );
                  })}
                </div>
              )}
              <div className={s.inlineEditFooter}>
                <div className={s.inlineEditLeft}>
                  <input ref={editFileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp,video/mp4,video/webm,audio/mpeg,audio/ogg,audio/wav" multiple onChange={handleEditFiles} style={{ display: 'none' }} />
                  <button onClick={(e) => { e.stopPropagation(); editFileRef.current?.click(); }} className={s.mediaUploadBtn} title="Прикрепить медиа" disabled={totalEditMedia >= 10}>
                    <ImagePlus size={16} />
                  </button>
                  <span className={s.editCharCount} style={{ color: remaining < 0 ? 'var(--danger)' : remaining < 20 ? '#eab308' : 'var(--text-tertiary)' }}>
                    {remaining}
                  </span>
                </div>
                <div className={s.inlineEditRight}>
                  <button onClick={cancelEdit} className={s.inlineEditCancelBtn}><X size={16} /></button>
                  <button
                    onClick={saveEdit}
                    disabled={editLoading || !editContent.trim() || editContent.length > 280}
                    className={s.inlineEditSaveBtn}
                  >
                    {editLoading ? '...' : <Check size={16} />}
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <>
              <p className={s.postContent}>{renderContent(displayPost.content)}</p>
              {displayPost.media && displayPost.media.length > 0 && (
                <MediaGrid media={displayPost.media} onImageClick={(url) => setLightboxSrc(url)} />
              )}
            </>
          )}

          {!editing && (
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
              {post.views_count > 0 && (
                <span className={s.viewCount}>
                  <Eye size={15} />
                  <span>{post.views_count}</span>
                </span>
              )}
            </div>
          )}
        </div>
      </div>
    </article>
    {lightboxSrc && (
      <ImageLightbox
        src={lightboxSrc}
        allSrcs={(displayPost.media || []).map((m) => m.url)}
        post={post}
        onClose={() => setLightboxSrc(null)}
      />
    )}
  </>
  );
}
