import { useState, useRef } from 'react';
import { ImagePlus, X } from 'lucide-react';
import { createComment } from '@/api/comments';
import { uploadMedia } from '@/api/media';
import s from '@/styles/comment.module.css';
import type { Comment } from '@/types';

interface CommentFormProps {
  postId: string;
  parentId?: string;
  onSubmit?: (comment: Comment) => void;
  onCancel?: () => void;
  compact?: boolean;
}

export default function CommentForm({ postId, parentId = '', onSubmit, onCancel, compact }: CommentFormProps) {
  const [content, setContent] = useState('');
  const [mediaFile, setMediaFile] = useState<File | null>(null);
  const [mediaPreview, setMediaPreview] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  const clearMedia = () => {
    setMediaFile(null);
    if (mediaPreview) URL.revokeObjectURL(mediaPreview);
    setMediaPreview(null);
    if (fileRef.current) fileRef.current.value = '';
  };

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setMediaFile(file);
    setMediaPreview(URL.createObjectURL(file));
  };

  const handleSubmit = async () => {
    if (!content.trim() || loading) return;
    setLoading(true);
    setError('');
    try {
      let mediaUrl = '';
      if (mediaFile) {
        const m = await uploadMedia(mediaFile);
        mediaUrl = m.url;
      }
      const comment = await createComment(postId, content, parentId, mediaUrl);
      setContent('');
      clearMedia();
      onSubmit?.(comment);
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Не удалось отправить');
    } finally { setLoading(false); }
  };

  return (
    <div className={compact ? s.commentFormCompact : s.commentForm}>
      <textarea
        value={content}
        onChange={(e) => setContent(e.target.value)}
        placeholder={parentId ? 'Написать ответ...' : 'Написать комментарий...'}
        rows={compact ? 2 : 3}
        className={s.commentTextarea}
      />
      {mediaPreview && mediaFile && (
        <div className={s.commentMediaPreview}>
          {mediaFile.type.startsWith('video/') ? (
            <video src={mediaPreview} controls style={{ width: '100%', borderRadius: 8, maxHeight: 200 }} />
          ) : mediaFile.type.startsWith('audio/') ? (
            <audio src={mediaPreview} controls style={{ width: '100%' }} />
          ) : (
            <img src={mediaPreview} alt="" />
          )}
          <button onClick={clearMedia} className={s.commentMediaRemoveBtn}><X size={14} /></button>
        </div>
      )}
      {error && <p className={s.errorText}>{error}</p>}
      <div className={s.commentFormActions}>
        <div className={s.commentFormLeft}>
          <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp,video/mp4,video/webm,audio/mpeg,audio/ogg,audio/wav" onChange={handleFile} style={{ display: 'none' }} />
          <button onClick={() => fileRef.current?.click()} className={s.commentMediaBtn} title="Прикрепить фото">
            <ImagePlus size={16} />
          </button>
        </div>
        <div className={s.commentFormRight}>
          {onCancel && <button onClick={onCancel} className={s.cancelBtn}>Отмена</button>}
          <button onClick={handleSubmit} disabled={!content.trim() || loading} className={s.commentSubmitBtn}>
            {loading ? <span className={s.commentSubmitSpinner} /> : 'Отправить'}
          </button>
        </div>
      </div>
    </div>
  );
}
