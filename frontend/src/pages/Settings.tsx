import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Camera } from 'lucide-react';
import { updateUser } from '@/api/users';
import { uploadMedia, deleteMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import Avatar from '@/components/Avatar';
import l from '@/styles/layout.module.css';
import s from '@/styles/profile.module.css';

export default function Settings() {
  const navigate = useNavigate();
  const { user, updateUser: updateStore, avatarMediaId, setAvatarMediaId } = useAuthStore();
  const fileRef = useRef<HTMLInputElement>(null);

  const [displayName, setDisplayName] = useState(user?.display_name || '');
  const [bio, setBio] = useState(user?.bio || '');
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState('');

  if (!user) { navigate('/login'); return null; }

  const handleAvatar = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError(''); setSuccess(false);
    try {
      let avatarUrl = user.avatar_url;
      let newMediaId: string | null = avatarMediaId;
      if (avatarFile) {
        const oldMediaId = avatarMediaId;
        const m = await uploadMedia(avatarFile);
        avatarUrl = m.url;
        newMediaId = m.id;
        if (oldMediaId) {
          deleteMedia(oldMediaId).catch(() => {});
        }
      }
      const updated = await updateUser(user.id, { display_name: displayName, bio, avatar_url: avatarUrl });
      updateStore(updated);
      setAvatarMediaId(newMediaId);
      setSuccess(true);
      setAvatarFile(null);
      if (avatarPreview) { URL.revokeObjectURL(avatarPreview); setAvatarPreview(null); }
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Не удалось сохранить');
    } finally { setLoading(false); }
  };

  return (
    <div>
      <header className={l.pageHeader}>
        <button onClick={() => navigate(-1)} className={l.backBtn}><ArrowLeft size={18} /></button>
        <h1 className={l.pageHeaderTitle}>Настройки</h1>
      </header>

      <form onSubmit={handleSubmit} className={s.settingsForm}>
        <div className={s.avatarEdit}>
          <div className={s.avatarEditBtn} onClick={() => fileRef.current?.click()}>
            <Avatar url={avatarPreview || user.avatar_url} name={displayName || user.username} size="xl" />
            <div className={s.avatarOverlay}><Camera size={20} color="white" /></div>
          </div>
          <input ref={fileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleAvatar} style={{ display: 'none' }} />
          <div>
            <p className={s.avatarHint}>Нажмите на аватар</p>
            <p className={s.avatarHintSub}>JPEG, PNG, GIF, WebP. До 10 МБ</p>
          </div>
        </div>

        <div>
          <label className={s.fieldLabel}>Отображаемое имя</label>
          <input type="text" value={displayName} onChange={(e) => setDisplayName(e.target.value)} maxLength={100} className={s.fieldInput} />
        </div>

        <div>
          <label className={s.fieldLabel}>О себе</label>
          <textarea value={bio} onChange={(e) => setBio(e.target.value)} maxLength={500} rows={4} className={s.fieldTextarea} />
          <p className={s.charHint}>{bio.length}/500</p>
        </div>

        {error && <p className={s.errorText}>{error}</p>}
        {success && <p className={s.successText}>Изменения сохранены</p>}

        <button type="submit" disabled={loading} className={s.saveBtn}>
          {loading ? 'Сохранение...' : 'Сохранить'}
        </button>
      </form>
    </div>
  );
}
