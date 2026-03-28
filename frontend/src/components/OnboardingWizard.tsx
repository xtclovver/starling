import { useState, useRef } from 'react';
import { Camera, ImagePlus } from 'lucide-react';
import { updateUser } from '@/api/users';
import { uploadMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import Avatar from './Avatar';
import s from '@/styles/modal.module.css';

interface Props {
  onClose: () => void;
}

const STEPS = [
  { key: 'name', title: 'Как вас зовут?', desc: 'Отображаемое имя видно всем пользователям' },
  { key: 'bio', title: 'Расскажите о себе', desc: 'Краткое описание для вашего профиля' },
  { key: 'avatar', title: 'Добавьте фото', desc: 'Выберите аватар для профиля' },
  { key: 'banner', title: 'Фон профиля', desc: 'Добавьте фоновое изображение' },
] as const;

export default function OnboardingWizard({ onClose }: Props) {
  const { user, updateUser: updateStore, setAvatarMediaId, setBannerMediaId } = useAuthStore();
  const [step, setStep] = useState(0);
  const [displayName, setDisplayName] = useState('');
  const [bio, setBio] = useState('');
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [bannerFile, setBannerFile] = useState<File | null>(null);
  const [bannerPreview, setBannerPreview] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const avatarRef = useRef<HTMLInputElement>(null);
  const bannerRef = useRef<HTMLInputElement>(null);

  if (!user) return null;

  const current = STEPS[step];
  const isLast = step === STEPS.length - 1;

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleBannerChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setBannerFile(file);
    setBannerPreview(URL.createObjectURL(file));
  };

  const handleFinish = async () => {
    setSaving(true);
    try {
      let avatarUrl = user.avatar_url;
      let bannerUrl = user.banner_url || '';
      let newAvatarMediaId: string | null = null;
      let newBannerMediaId: string | null = null;

      if (avatarFile) {
        const m = await uploadMedia(avatarFile);
        avatarUrl = m.url;
        newAvatarMediaId = m.id;
      }
      if (bannerFile) {
        const m = await uploadMedia(bannerFile);
        bannerUrl = m.url;
        newBannerMediaId = m.id;
      }

      const fields: Record<string, string> = {};
      if (displayName.trim()) fields.display_name = displayName;
      if (bio.trim()) fields.bio = bio;
      if (avatarUrl !== user.avatar_url) fields.avatar_url = avatarUrl;
      if (bannerUrl !== (user.banner_url || '')) fields.banner_url = bannerUrl;

      if (Object.keys(fields).length > 0) {
        const updated = await updateUser(user.id, fields);
        updateStore(updated);
      }

      if (newAvatarMediaId) setAvatarMediaId(newAvatarMediaId);
      if (newBannerMediaId) setBannerMediaId(newBannerMediaId);

      if (avatarPreview) URL.revokeObjectURL(avatarPreview);
      if (bannerPreview) URL.revokeObjectURL(bannerPreview);
    } catch {
      // Silently close — user can edit profile in settings later
    } finally {
      setSaving(false);
      onClose();
    }
  };

  const handleNext = () => {
    if (isLast) {
      handleFinish();
    } else {
      setStep(step + 1);
    }
  };

  const handleSkip = () => {
    if (isLast) {
      onClose();
    } else {
      setStep(step + 1);
    }
  };

  return (
    <div className={s.backdrop}>
      <div className={s.modal} onClick={(e) => e.stopPropagation()}>
        {/* Step dots */}
        <div className={s.wizardSteps}>
          {STEPS.map((_, i) => (
            <div key={i} className={`${s.wizardDot} ${i <= step ? s.wizardDotActive : ''}`} />
          ))}
        </div>

        <h2 className={s.wizardTitle}>{current.title}</h2>
        <p className={s.wizardDesc}>{current.desc}</p>

        {/* Step content */}
        {current.key === 'name' && (
          <div className={s.wizardField}>
            <input
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Отображаемое имя"
              maxLength={100}
              autoFocus
              className={s.wizardInput}
            />
          </div>
        )}

        {current.key === 'bio' && (
          <div className={s.wizardField}>
            <textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="О себе..."
              maxLength={500}
              rows={4}
              autoFocus
              className={s.wizardTextarea}
            />
            <p className={s.wizardCharHint}>{bio.length}/500</p>
          </div>
        )}

        {current.key === 'avatar' && (
          <div className={s.wizardAvatarArea}>
            <div className={s.wizardAvatarBtn} onClick={() => avatarRef.current?.click()}>
              <Avatar url={avatarPreview || user.avatar_url} name={displayName || user.username} size="xl" />
              <div className={s.wizardAvatarOverlay}><Camera size={20} color="white" /></div>
            </div>
            <input ref={avatarRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleAvatarChange} style={{ display: 'none' }} />
            <p className={s.wizardAvatarHint}>Нажмите чтобы выбрать</p>
          </div>
        )}

        {current.key === 'banner' && (
          <div className={s.wizardBannerArea}>
            <div className={s.wizardBannerBox} onClick={() => bannerRef.current?.click()}>
              {bannerPreview ? (
                <img src={bannerPreview} alt="" />
              ) : (
                <div className={s.wizardBannerPlaceholder}>
                  <ImagePlus size={24} color="var(--text-tertiary)" />
                  <span>Нажмите чтобы загрузить</span>
                </div>
              )}
              <div className={s.wizardBannerOverlay}><Camera size={22} color="white" /></div>
            </div>
            <input ref={bannerRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleBannerChange} style={{ display: 'none' }} />
          </div>
        )}

        {/* Footer */}
        <div className={s.wizardFooter}>
          <button onClick={handleSkip} className={s.wizardSkipBtn} type="button">
            Пропустить
          </button>
          <button onClick={handleNext} disabled={saving} className={s.wizardNextBtn} type="button">
            {saving ? 'Сохранение...' : isLast ? 'Завершить' : 'Далее'}
          </button>
        </div>
      </div>
    </div>
  );
}
