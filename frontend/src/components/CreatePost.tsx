import { useState, useRef } from 'react';
import { ImagePlus, X } from 'lucide-react';
import { createPost } from '@/api/posts';
import { uploadMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import { useFeedStore } from '@/store/feed';
import Avatar from './Avatar';
import s from '@/styles/post.module.css';

const MAX_CHARS = 280;
const MAX_MEDIA = 10;

export default function CreatePost() {
  const user = useAuthStore((st) => st.user);
  const prependPost = useFeedStore((st) => st.prependPost);
  const [content, setContent] = useState('');
  const [mediaFiles, setMediaFiles] = useState<File[]>([]);
  const [mediaPreviews, setMediaPreviews] = useState<string[]>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  const fileRef = useRef<HTMLInputElement>(null);

  if (!user) return null;

  const remaining = MAX_CHARS - content.length;
  const canSubmit = content.trim().length > 0 && remaining >= 0 && !loading;

  const handleFiles = (e: React.ChangeEvent<HTMLInputElement>) => {
    const files = Array.from(e.target.files || []);
    if (!files.length) return;
    const allowed = MAX_MEDIA - mediaFiles.length;
    const toAdd = files.slice(0, allowed);
    setMediaFiles((prev) => [...prev, ...toAdd]);
    setMediaPreviews((prev) => [...prev, ...toAdd.map((f) => URL.createObjectURL(f))]);
    if (fileRef.current) fileRef.current.value = '';
  };

  const removeMedia = (idx: number) => {
    URL.revokeObjectURL(mediaPreviews[idx]);
    setMediaFiles((prev) => prev.filter((_, i) => i !== idx));
    setMediaPreviews((prev) => prev.filter((_, i) => i !== idx));
  };

  const clearAllMedia = () => {
    mediaPreviews.forEach((u) => URL.revokeObjectURL(u));
    setMediaFiles([]);
    setMediaPreviews([]);
    if (fileRef.current) fileRef.current.value = '';
  };

  const handleSubmit = async () => {
    if (!canSubmit) return;
    setLoading(true);
    setError('');
    try {
      const urls: string[] = [];
      for (const file of mediaFiles) {
        const m = await uploadMedia(file);
        urls.push(m.url);
      }
      const post = await createPost(content, urls);
      post.author = user;
      prependPost(post);
      setContent('');
      clearAllMedia();
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
          {mediaPreviews.length > 0 && (
            <div className={s.createPostMediaGrid}>
              {mediaPreviews.map((src, i) => {
                const file = mediaFiles[i];
                return (
                  <div key={i} className={s.createPostMediaPreview}>
                    {file.type.startsWith('video/') ? (
                      <video src={src} controls style={{ width: '100%', borderRadius: 8, maxHeight: 240 }} />
                    ) : file.type.startsWith('audio/') ? (
                      <audio src={src} controls style={{ width: '100%' }} />
                    ) : (
                      <img src={src} alt="" />
                    )}
                    <button onClick={() => removeMedia(i)} className={s.mediaRemoveBtn}><X size={14} /></button>
                  </div>
                );
              })}
            </div>
          )}
          {error && <p className={s.errorText}>{error}</p>}
          <div className={s.createPostFooter}>
            <div>
              <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp,video/mp4,video/webm,audio/mpeg,audio/ogg,audio/wav" multiple onChange={handleFiles} style={{ display: 'none' }} />
              <button onClick={() => fileRef.current?.click()} className={s.mediaUploadBtn} disabled={mediaFiles.length >= MAX_MEDIA}>
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
