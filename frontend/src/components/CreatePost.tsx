import { useState, useRef } from 'react';
import { ImagePlus, X } from 'lucide-react';
import { createPost } from '@/api/posts';
import { uploadMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import Avatar from './Avatar';
import s from '@/styles/post.module.css';

const MAX_CHARS = 280;

export default function CreatePost() {
  const user = useAuthStore((st) => st.user);
  const prependPost = useFeedStore((st) => st.prependPost);
  const [content, setContent] = useState('');
  const [mediaFile, setMediaFile] = useState<File | null>(null);
  const [mediaPreview, setMediaPreview] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  if (!user) return null;

  const remaining = MAX_CHARS - content.length;
  const canSubmit = content.trim().length > 0 && remaining >= 0 && !loading;

  const handleFile = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setMediaFile(file);
    setMediaPreview(URL.createObjectURL(file));
  };

  const clearMedia = () => {
    setMediaFile(null);
    if (mediaPreview) URL.revokeObjectURL(mediaPreview);
    setMediaPreview(null);
    if (fileRef.current) fileRef.current.value = '';
  };

  const handleSubmit = async () => {
    if (!canSubmit) return;
    setLoading(true);
    setError('');
    try {
      let mediaUrl = '';
      if (mediaFile) { const m = await uploadMedia(mediaFile); mediaUrl = m.url; }
      const post = await createPost(content, mediaUrl);
      post.author = user;
      prependPost(post);
      setContent('');
      clearMedia();
    } catch (e: unknown) {
      const msg = (e as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Не удалось создать пост');
    } finally { setLoading(false); }
  };

  return (
    <div className={s.createPost}>
      <div className={s.createPostRow}>
        <Avatar url={user.avatar_url} name={user.display_name || user.username} />
        <div className={s.createPostBody}>
          <textarea
            value={content}
            onChange={(e) => setContent(e.target.value)}
            placeholder="Что нового?"
            rows={3}
            className={s.createPostTextarea}
          />
          {mediaPreview && (
            <div className={s.createPostMediaPreview}>
              <img src={mediaPreview} alt="" />
              <button onClick={clearMedia} className={s.mediaRemoveBtn}><X size={14} /></button>
            </div>
          )}
          {error && <p className={s.errorText}>{error}</p>}
          <div className={s.createPostFooter}>
            <div>
              <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleFile} style={{ display: 'none' }} />
              <button onClick={() => fileRef.current?.click()} className={s.mediaUploadBtn}>
                <ImagePlus size={18} />
              </button>
            </div>
            <div className={s.createPostRight}>
              {content.length > 0 && (
                <span className={`${s.charCount} ${remaining < 0 ? s.charCountOver : remaining < 20 ? s.charCountWarn : s.charCountOk}`}>
                  {remaining}
                </span>
              )}
              <button onClick={handleSubmit} disabled={!canSubmit} className={s.submitBtn}>
                {loading ? 'Отправка...' : 'Опубликовать'}
              </button>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
