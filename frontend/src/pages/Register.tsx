import { useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Feather } from 'lucide-react';
import { register } from '@/api/auth';
import { useAuthStore } from '@/store/auth';
import { translateBackendError } from '@/lib/errors';
import OnboardingWizard from '@/components/OnboardingWizard';
import s from '@/styles/auth.module.css';

export default function Register() {
  const navigate = useNavigate();
  const setAuth = useAuthStore((st) => st.login);

  const [username, setUsername] = useState('');
  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [confirmPassword, setConfirmPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);
  const [showOnboarding, setShowOnboarding] = useState(false);

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    if (username.length < 3 || username.length > 50) { setError('Имя пользователя: 3–50 символов'); return; }
    if (password.length < 8) { setError('Пароль: минимум 8 символов'); return; }
    if (password !== confirmPassword) { setError('Пароли не совпадают'); return; }

    setLoading(true);
    try {
      const data = await register(username, email, password);
      setAuth(data.user, data.access_token);
      setShowOnboarding(true);
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(translateBackendError(msg) || 'Не удалось зарегистрироваться');
    } finally { setLoading(false); }
  };

  return (
    <div className={s.page}>
      <div className={s.card}>
        <div className={s.logo}><Feather size={40} /></div>
        <h1 className={s.title}>Создать аккаунт</h1>
        <form onSubmit={handleSubmit} className={s.form}>
          <input type="text" value={username} onChange={(e) => setUsername(e.target.value)} placeholder="Имя пользователя" required minLength={3} maxLength={50} className={s.input} />
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" required className={s.input} />
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Пароль (мин. 8 символов)" required minLength={8} className={s.input} />
          <input type="password" value={confirmPassword} onChange={(e) => setConfirmPassword(e.target.value)} placeholder="Подтвердите пароль" required className={s.input} />
          {error && <p className={s.error}>{error}</p>}
          <button type="submit" disabled={loading} className={s.submitBtn}>{loading ? 'Регистрация...' : 'Зарегистрироваться'}</button>
        </form>
        <p className={s.footer}>
          Уже есть аккаунт? <Link to="/login" className={s.footerLink}>Войти</Link>
        </p>
      </div>
      {showOnboarding && (
        <OnboardingWizard onClose={() => navigate('/', { replace: true })} />
      )}
    </div>
  );
}
