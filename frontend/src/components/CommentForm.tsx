import { useState } from 'react';
import { createComment } from '@/api/comments';
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
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');

  const handleSubmit = async () => {
    if (!content.trim() || loading) return;
    setLoading(true);
    setError('');
    try {
      const comment = await createComment(postId, content, parentId);
      setContent('');
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
      {error && <p className={s.errorText}>{error}</p>}
      <div className={s.commentFormActions}>
        {onCancel && <button onClick={onCancel} className={s.cancelBtn}>Отмена</button>}
        <button onClick={handleSubmit} disabled={!content.trim() || loading} className={s.commentSubmitBtn}>
          {loading ? '...' : 'Отправить'}
        </button>
      </div>
    </div>
  );
}
