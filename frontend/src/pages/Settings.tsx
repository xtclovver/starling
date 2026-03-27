import { useState, useRef } from 'react';
import { useNavigate } from 'react-router-dom';
import { ArrowLeft, Camera, ImagePlus, X, ShieldAlert } from 'lucide-react';
import { updateUser } from '@/api/users';
import { uploadMedia, deleteMedia } from '@/api/media';
import { revokeAllSessions } from '@/api/auth';
import { useAuthStore } from '@/store/auth';
import Avatar from '@/components/Avatar';
import l from '@/styles/layout.module.css';
import s from '@/styles/profile.module.css';

export default function Settings() {
  const navigate = useNavigate();
  const { user, updateUser: updateStore, avatarMediaId, setAvatarMediaId, bannerMediaId, setBannerMediaId, logout } = useAuthStore();
  const avatarFileRef = useRef<HTMLInputElement>(null);
  const bannerFileRef = useRef<HTMLInputElement>(null);

  const [displayName, setDisplayName] = useState(user?.display_name || '');
  const [bio, setBio] = useState(user?.bio || '');

  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [avatarFile, setAvatarFile] = useState<File | null>(null);

  const [bannerPreview, setBannerPreview] = useState<string | null>(null);
  const [bannerFile, setBannerFile] = useState<File | null>(null);

  const [loading, setLoading] = useState(false);
  const [success, setSuccess] = useState(false);
  const [error, setError] = useState('');
  const [revoking, setRevoking] = useState(false);

  if (!user) { navigate('/login'); return null; }

  const handleAvatar = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleBanner = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setBannerFile(file);
    setBannerPreview(URL.createObjectURL(file));
  };

  const removeBanner = () => {
    setBannerFile(null);
    if (bannerPreview) URL.revokeObjectURL(bannerPreview);
    setBannerPreview(null);
    if (bannerFileRef.current) bannerFileRef.current.value = '';
  };

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setLoading(true); setError(''); setSuccess(false);
    try {
      let avatarUrl = user.avatar_url;
      let newAvatarMediaId: string | null = avatarMediaId;
      if (avatarFile) {
        const oldMediaId = avatarMediaId;
        const m = await uploadMedia(avatarFile);
        avatarUrl = m.url;
        newAvatarMediaId = m.id;
        if (oldMediaId) deleteMedia(oldMediaId).catch(() => {});
      }

      let bannerUrl = user.banner_url || '';
      let newBannerMediaId: string | null = bannerMediaId;
      if (bannerFile) {
        const oldMediaId = bannerMediaId;
        const m = await uploadMedia(bannerFile);
        bannerUrl = m.url;
        newBannerMediaId = m.id;
        if (oldMediaId) deleteMedia(oldMediaId).catch(() => {});
      }

      const updated = await updateUser(user.id, { display_name: displayName, bio, avatar_url: avatarUrl, banner_url: bannerUrl });
      updateStore(updated);
      setAvatarMediaId(newAvatarMediaId);
      setBannerMediaId(newBannerMediaId);
      setSuccess(true);
      setAvatarFile(null);
      setBannerFile(null);
      if (avatarPreview) { URL.revokeObjectURL(avatarPreview); setAvatarPreview(null); }
      if (bannerPreview) { URL.revokeObjectURL(bannerPreview); setBannerPreview(null); }
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Не удалось сохранить');
    } finally { setLoading(false); }
  };

  const handleRevokeAll = async () => {
    setRevoking(true);
    try {
      await revokeAllSessions();
      logout();
      navigate('/');
    } catch {
      setRevoking(false);
    }
  };

  const currentBanner = bannerPreview || user.banner_url;

  return (
    <div>
      <header className={l.pageHeader}>
        <button onClick={() => navigate(-1)} className={l.backBtn}><ArrowLeft size={18} /></button>
        <h1 className={l.pageHeaderTitle}>Настройки</h1>
      </header>

      <form onSubmit={handleSubmit} className={s.settingsForm}>

        {/* Banner */}
        <div>
          <label className={s.fieldLabel}>Фон профиля</label>
          <div className={s.bannerEdit} onClick={() => bannerFileRef.current?.click()}>
            {currentBanner ? (
              <img src={currentBanner} alt="" className={s.bannerEditImg} />
            ) : (
              <div className={s.bannerEditPlaceholder}>
                <ImagePlus size={24} color="var(--text-tertiary)" />
                <span>Нажмите чтобы загрузить</span>
              </div>
            )}
            <div className={s.bannerEditOverlay}><Camera size={22} color="white" /></div>
          </div>
          <input ref={bannerFileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleBanner} style={{ display: 'none' }} />
          {(bannerPreview || user.banner_url) && (
            <button type="button" onClick={removeBanner} className={s.bannerRemoveBtn}>
              <X size={14} /> Удалить фон
            </button>
          )}
        </div>

        {/* Avatar */}
        <div className={s.avatarEdit}>
          <div className={s.avatarEditBtn} onClick={() => avatarFileRef.current?.click()}>
            <Avatar url={avatarPreview || user.avatar_url} name={displayName || user.username} size="xl" />
            <div className={s.avatarOverlay}><Camera size={20} color="white" /></div>
          </div>
          <input ref={avatarFileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleAvatar} style={{ display: 'none' }} />
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

      <div className={s.securitySection}>
        <h2 className={s.securityTitle}><ShieldAlert size={18} /> Безопасность</h2>
        <p className={s.securityDesc}>Завершить все активные сессии на всех устройствах. Вам потребуется войти заново.</p>
        <button onClick={handleRevokeAll} disabled={revoking} className={s.revokeBtn}>
          {revoking ? 'Завершаем...' : 'Завершить все сессии'}
        </button>
      </div>
    </div>
  );
}
