import { useState } from 'react';
import { Link, useNavigate, useLocation } from 'react-router-dom';
import { Feather } from 'lucide-react';
import { login } from '@/api/auth';
import { useAuthStore } from '@/store/auth';
import s from '@/styles/auth.module.css';

export default function Login() {
  const navigate = useNavigate();
  const location = useLocation();
  const setAuth = useAuthStore((st) => st.login);

  const [email, setEmail] = useState('');
  const [password, setPassword] = useState('');
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(false);

  const from = (location.state as { from?: string })?.from || '/';

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    setError('');
    setLoading(true);
    try {
      const data = await login(email, password);
      setAuth(data.user, data.access_token, data.refresh_token);
      navigate(from, { replace: true });
    } catch (err: unknown) {
      const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
      setError(msg || 'Неверный email или пароль');
    } finally { setLoading(false); }
  };

  return (
    <div className={s.page}>
      <div className={s.card}>
        <div className={s.logo}><Feather size={40} /></div>
        <h1 className={s.title}>Войти в аккаунт</h1>
        <form onSubmit={handleSubmit} className={s.form}>
          <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} placeholder="Email" required className={s.input} />
          <input type="password" value={password} onChange={(e) => setPassword(e.target.value)} placeholder="Пароль" required minLength={8} className={s.input} />
          {error && <p className={s.error}>{error}</p>}
          <button type="submit" disabled={loading} className={s.submitBtn}>{loading ? 'Входим...' : 'Войти'}</button>
        </form>
        <p className={s.footer}>
          Нет аккаунта? <Link to="/register" className={s.footerLink}>Зарегистрироваться</Link>
        </p>
      </div>
    </div>
  );
}
