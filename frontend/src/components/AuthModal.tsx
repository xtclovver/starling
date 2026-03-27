import { useState, useEffect } from 'react';
import { Feather, X } from 'lucide-react';
import { login, register } from '@/api/auth';
import { useAuthStore } from '@/store/auth';
import { useUIStore } from '@/store/ui';
import { useFeedStore } from '@/store/feed';
import s from '@/styles/modal.module.css';
import a from '@/styles/auth.module.css';

export default function AuthModal() {
  const { authModalOpen, authModalTab, closeAuthModal } = useUIStore();
  const setAuth = useAuthStore((st) => st.login);
  const resetFeed = useFeedStore((st) => st.reset);

  const [tab, setTab] = useState<'login' | 'register'>(authModalTab);
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [username, setUsername] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  useEffect(() => { setTab(authModalTab); }, [authModalTab]);
  useEffect(() => { if (authModalOpen) { setError(''); setEmail(''); setPassword(''); setUsername(''); setConfirmPassword(''); } }, [authModalOpen]);

  if (!authModalOpen) return null;

  const handleLogin = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await login(email, password);
      setAuth(data.user, data.access_token);
      resetFeed();
      closeAuthModal();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Неверный email или пароль');
    } finally { setLoading(false); }
  };

  const handleRegister = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (username.length < 3 || username.length > 50) { setError('Имя пользователя: 3–50 символов'); return; }
    if (password.length < 8) { setError('Пароль: минимум 8 символов'); return; }
    if (password !== confirmPassword) { setError('Пароли не совпадают'); return; }
    setLoading(true);
    try {
      const data = await register(username, email, password);
      setAuth(data.user, data.access_token);
      resetFeed();
      closeAuthModal();
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Не удалось зарегистрироваться');
    } finally { setLoading(false); }
  };

  return (
    <div className={s.backdrop} onClick={closeAuthModal}>
      <div className={s.modal} onClick={(e) => e.stopPropagation()}>
        <button className={s.closeBtn} onClick={closeAuthModal}><X size={20} /></button>
        <div className={a.logo}><Feather size={36} /></div>
        {tab === 'login' ? (
          <>
            <h2 className={a.title}>Войти</h2>
            <form onSubmit={handleLogin} className={a.form}>
              <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" required className={a.input} />
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Пароль" required minLength={8} className={a.input} />
              {error && <p className={a.error}>{error}</p>}
              <button type="submit" disabled={loading} className={a.submitBtn}>{loading ? 'Входим...' : 'Войти'}</button>
            </form>
            <p className={a.footer}>
              Нет аккаунта? <button onClick={() => { setTab('register'); setError(''); }} className={s.switchBtn}>Зарегистрироваться</button>
            </p>
          </>
        ) : (
          <>
            <h2 className={a.title}>Создать аккаунт</h2>
            <form onSubmit={handleRegister} className={a.form}>
              <input type="text" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="Имя пользователя" required minLength={3} maxLength={50} className={a.input} />
              <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" required className={a.input} />
              <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Пароль (мин. 8 символов)" required minLength={8} className={a.input} />
              <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} placeholder="Подтвердите пароль" required className={a.input} />
              {error && <p className={a.error}>{error}</p>}
              <button type="submit" disabled={loading} className={a.submitBtn}>{loading ? 'Регистрация...' : 'Зарегистрироваться'}</button>
            </form>
            <p className={a.footer}>
              Уже есть аккаунт? <button onClick={() => { setTab('login'); setError(''); }} className={s.switchBtn}>Войти</button>
            </p>
          </>
        )}
      </div>
    </div>
  );
}
