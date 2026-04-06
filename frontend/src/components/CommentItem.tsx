import { useState, useRef } from 'react';
import { Link } from 'react-router-dom';
import s2 from '@/styles/post.module.css';
import { Heart, MessageCircle, Trash2, ChevronDown, ChevronUp, Pencil, X, Check, ImagePlus } from 'lucide-react';
import { likeComment, unlikeComment, deleteComment, updateComment } from '@/api/comments';
import { uploadMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import { timeAgo } from '@/lib/time';
import { getMediaKind } from '@/lib/media';
import Avatar from './Avatar';
import CommentForm from './CommentForm';
import ImageLightbox from './ImageLightbox';
import s from '@/styles/comment.module.css';
import type { Comment } from '@/types';

function renderContent(content: string) {
  const parts = content.split(/(#\w+|@\w+)/g);
  return parts.map((part, i) => {
    if (part.startsWith('#')) {
      const tag = part.slice(1).toLowerCase();
      return <Link key={i} to={`/hashtag/${tag}`} className={s2.hashtag}>{part}</Link>;
    }
    if (part.startsWith('@')) {
      return <Link key={i} to={`/u/${part.slice(1)}`} className={s2.mention}>{part}</Link>;
    }
    return part;
  });
}

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

  // Edit state
  const [editing, setEditing] = useState(false);
  const [editContent, setEditContent] = useState(comment.content);
  const [editMediaUrl, setEditMediaUrl] = useState(comment.media_url || '');
  const [editMediaFile, setEditMediaFile] = useState<File | null>(null);
  const [editMediaPreview, setEditMediaPreview] = useState<string | null>(null);
  const [editLoading, setEditLoading] = useState(false);
  const [localContent, setLocalContent] = useState(comment.content);
  const [localMediaUrl, setLocalMediaUrl] = useState(comment.media_url || '');
  const [localEditedAt, setLocalEditedAt] = useState(comment.edited_at);
  const editFileRef = useRef<HTMLInputElement>(null);

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
    try { await deleteComment(comment.id); onDelete?.(comment.id); } catch { /* ignore */ }
  };

  const startEdit = () => {
    setEditContent(localContent);
    setEditMediaUrl(localMediaUrl);
    setEditMediaFile(null);
    setEditMediaPreview(null);
    setEditing(true);
  };

  const cancelEdit = () => {
    setEditing(false);
    setEditMediaFile(null);
    if (editMediaPreview) URL.revokeObjectURL(editMediaPreview);
    setEditMediaPreview(null);
  };

  const handleEditFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setEditMediaFile(file);
    setEditMediaPreview(URL.createObjectURL(file));
    setEditMediaUrl('');
  };

  const removeEditMedia = () => {
    setEditMediaFile(null);
    if (editMediaPreview) URL.revokeObjectURL(editMediaPreview);
    setEditMediaPreview(null);
    setEditMediaUrl('');
    if (editFileRef.current) editFileRef.current.value = '';
  };

  const saveEdit = async () => {
    if (!editContent.trim() || editContent.length > 500 || editLoading) return;
    setEditLoading(true);
    try {
      let mediaUrl = editMediaUrl;
      if (editMediaFile) {
        const m = await uploadMedia(editMediaFile);
        mediaUrl = m.url;
      }
      const updated = await updateComment(comment.id, editContent, mediaUrl);
      setLocalContent(updated.content);
      setLocalMediaUrl(updated.media_url || '');
      setLocalEditedAt(updated.edited_at);
      setEditing(false);
      setEditMediaFile(null);
      if (editMediaPreview) URL.revokeObjectURL(editMediaPreview);
      setEditMediaPreview(null);
    } catch { /* ignore */ } finally { setEditLoading(false); }
  };

  const remaining = 500 - editContent.length;

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
              {comment.author?.display_name || comment.author?.username || 'Неизвестный'}
            </Link>
            {comment.author?.username && <span className={s.commentHandle}>@{comment.author.username}</span>}
            <span className={s.commentDot}>&middot;</span>
            <span className={s.commentTime}>{timeAgo(comment.created_at)}</span>
            {localEditedAt && <span className={s.commentEditedBadge}>изменено</span>}
            {isOwner && !editing && !isDeleted && (
              <button onClick={startEdit} className={`${s.commentActionBtn} ${s.commentEditBtn}`} title="Редактировать">
                <Pencil size={12} />
              </button>
            )}
          </div>

          {isDeleted ? (
            <p className={`${s.commentText} ${s.commentDeleted}`}>[удалено]</p>
          ) : editing ? (
            <div className={s.commentInlineEdit}>
              <textarea
                className={s.commentInlineTextarea}
                value={editContent}
                onChange={(e) => setEditContent(e.target.value)}
                autoFocus
                rows={3}
              />
              {(editMediaPreview || editMediaUrl) && (() => {
                const src = editMediaPreview || editMediaUrl;
                const kind = editMediaFile
                  ? (editMediaFile.type.startsWith('video/') ? 'video' : editMediaFile.type.startsWith('audio/') ? 'audio' : 'image')
                  : getMediaKind(src);
                return (
                  <div className={s.commentEditMediaPreview}>
                    {kind === 'video' ? <video src={src} controls style={{ width: '100%', borderRadius: 8, maxHeight: 200 }} /> :
                     kind === 'audio' ? <audio src={src} controls style={{ width: '100%' }} /> :
                     <img src={src} alt="" />}
                    <button onClick={removeEditMedia} className={s.commentMediaRemoveBtnEdit}><X size={13} /></button>
                  </div>
                );
              })()}
              <div className={s.commentInlineFooter}>
                <div className={s.commentInlineLeft}>
                  <input
                    ref={editFileRef}
                    type="file"
                    accept="image/jpeg,image/png,image/gif,image/webp,video/mp4,video/webm,audio/mpeg,audio/ogg,audio/wav"
                    onChange={handleEditFile}
                    style={{ display: 'none' }}
                  />
                  <button
                    onClick={() => editFileRef.current?.click()}
                    className={`${s.commentActionBtn} ${s.commentMediaBtn}`}
                    title="Прикрепить файл"
                  >
                    <ImagePlus size={14} />
                  </button>
                  <span
                    className={s.commentEditCharCount}
                    style={{ color: remaining < 0 ? 'var(--danger)' : remaining < 20 ? '#eab308' : 'var(--text-tertiary)' }}
                  >
                    {remaining}
                  </span>
                </div>
                <div className={s.commentInlineRight}>
                  <button onClick={cancelEdit} className={s.commentInlineCancelBtn}><X size={14} /></button>
                  <button
                    onClick={saveEdit}
                    disabled={editLoading || !editContent.trim() || editContent.length > 500}
                    className={s.commentInlineSaveBtn}
                  >
                    {editLoading ? '...' : <Check size={14} />}
                  </button>
                </div>
              </div>
            </div>
          ) : (
            <>
              <p className={s.commentText}>{renderContent(localContent)}</p>
              {localMediaUrl && (() => {
                const kind = getMediaKind(localMediaUrl);
                if (kind === 'video') return (
                  <div className={s.commentMedia}>
                    <video src={localMediaUrl} controls style={{ width: '100%', borderRadius: 8, maxHeight: 320 }} />
                  </div>
                );
                if (kind === 'audio') return (
                  <div style={{ marginTop: 6 }}>
                    <audio src={localMediaUrl} controls style={{ width: '100%' }} />
                  </div>
                );
                return (
                  <div className={s.commentMedia}>
                    <img
                      src={localMediaUrl}
                      alt=""
                      loading="lazy"
                      onClick={() => setLightboxSrc(localMediaUrl)}
                      style={{ cursor: 'zoom-in' }}
                    />
                  </div>
                );
              })()}
            </>
          )}

          {!isDeleted && !editing && (
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
