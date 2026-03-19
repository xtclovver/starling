import { useState } from 'react';
import { X } from 'lucide-react';
import { updatePost } from '@/api/posts';
import { useFeedStore } from '@/store/feed';
import s from '@/styles/modal.module.css';
import type { Post } from '@/types';

export default function EditPostModal({ post, onClose }: { post: Post; onClose: () => void }) {
  const [content, setContent] = useState(post.content);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const updateFeedPost = useFeedStore((st) => st.updatePost);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!content.trim() || content.length > 280) return;
    setLoading(true);
    setError('');
    try {
      await updatePost(post.id, content);
      updateFeedPost(post.id, { content, edited_at: new Date().toISOString() });
      onClose();
    } catch {
      setError('Не удалось обновить пост');
    } finally { setLoading(false); }
  };

  const remaining = 280 - content.length;

  return (
    <div className={s.backdrop} onClick={onClose}>
      <div className={s.modal} onClick={(e) => e.stopPropagation()}>
        <button className={s.closeBtn} onClick={onClose}><X size={20} /></button>
        <h2 className={s.editTitle}>Редактировать пост</h2>
        <form onSubmit={handleSubmit}>
          <textarea
            className={s.editTextarea}
            value={content}
            onChange={(e) => setContent(e.target.value)}
            maxLength={280}
            autoFocus
          />
          {error && <p style={{ color: 'var(--danger)', fontSize: 14, marginTop: 8 }}>{error}</p>}
          <div className={s.editFooter}>
            <span className={s.editCharCount} style={{ color: remaining < 0 ? 'var(--danger)' : remaining < 20 ? '#eab308' : undefined }}>
              {remaining}
            </span>
            <button type="submit" disabled={loading || !content.trim() || content.length > 280} className={s.editSubmitBtn}>
              {loading ? 'Сохранение...' : 'Сохранить'}
            </button>
          </div>
        </form>
      </div>
    </div>
  );
}
