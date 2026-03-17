import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Home, User, Settings, LogOut, LogIn, UserPlus, Feather } from 'lucide-react';
import { useAuthStore } from '@/store/auth';
import Avatar from './Avatar';
import s from '@/styles/layout.module.css';

const NAV_AUTH = [
  { to: '/', icon: Home, label: 'Главная' },
  { to: '/profile/:self', icon: User, label: 'Профиль' },
  { to: '/settings', icon: Settings, label: 'Настройки' },
];

const NAV_GUEST = [
  { to: '/login', icon: LogIn, label: 'Войти' },
  { to: '/register', icon: UserPlus, label: 'Регистрация' },
];

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated, logout } = useAuthStore();
  const items = isAuthenticated ? NAV_AUTH : NAV_GUEST;

  const handleLogout = () => { logout(); navigate('/login'); };

  return (
    <aside className={s.sidebar}>
      <div>
        <Link to="/" className={s.sidebarLogo}><Feather size={28} /></Link>
        <nav className={s.sidebarNav}>
          {items.map(({ to, icon: Icon, label }) => {
            const href = to === '/profile/:self' ? `/profile/${user?.id}` : to;
            const active = location.pathname === href;
            return (
              <Link key={to} to={href} className={`${s.navItem} ${active ? s.navItemActive : ''}`}>
                <Icon size={24} strokeWidth={active ? 2.5 : 1.8} />
                <span className={s.navLabel}>{label}</span>
              </Link>
            );
          })}
        </nav>
        {isAuthenticated && (
          <button onClick={() => navigate('/')} className={s.sidebarPostBtn}>Опубликовать</button>
        )}
      </div>

      {isAuthenticated && user && (
        <div className={s.sidebarProfile}>
          <Avatar url={user.avatar_url} name={user.display_name || user.username} size="sm" />
          <div className={s.sidebarProfileInfo}>
            <p className={s.sidebarProfileName}>{user.display_name || user.username}</p>
            <p className={s.sidebarProfileHandle}>@{user.username}</p>
          </div>
          <button onClick={(e) => { e.stopPropagation(); handleLogout(); }} className={s.logoutBtn} title="Выйти">
            <LogOut size={16} />
          </button>
        </div>
      )}
    </aside>
  );
}
