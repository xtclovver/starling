import { Link, useLocation, useNavigate } from 'react-router-dom';
import { Home, User, Settings, LogOut, Feather, Bookmark, Bell, Shield } from 'lucide-react';
import { logout as apiLogout } from '@/api/auth';
import { useAuthStore } from '@/store/auth';
import { useUIStore } from '@/store/ui';
import { useNotificationStore } from '@/store/notifications';
import Avatar from './Avatar';
import s from '@/styles/layout.module.css';

const NAV_AUTH = [
  { to: '/', icon: Home, label: 'Главная' },
  { to: '/notifications', icon: Bell, label: 'Уведомления', badge: true },
  { to: '/bookmarks', icon: Bookmark, label: 'Закладки' },
  { to: '/profile/:self', icon: User, label: 'Профиль' },
  { to: '/settings', icon: Settings, label: 'Настройки' },
];

export default function Sidebar() {
  const location = useLocation();
  const navigate = useNavigate();
  const { user, isAuthenticated, logout } = useAuthStore();
  const openAuthModal = useUIStore((st) => st.openAuthModal);
  const unreadCount = useNotificationStore((st) => st.unreadCount);

  const handleLogout = async () => {
    logout();
    navigate('/');
    apiLogout().catch(() => {});
  };

  return (
    <aside className={s.sidebar}>
      <div>
        <Link to="/" className={s.sidebarLogoRow}>
          <Feather size={28} className={s.sidebarLogoIcon} />
          <span className={s.sidebarLogoName}>Starling</span>
        </Link>
        <nav className={s.sidebarNav}>
          {isAuthenticated ? (
            <>
            {NAV_AUTH.map(({ to, icon: Icon, label, badge }) => {
              const href = to === '/profile/:self' ? `/profile/${user?.id}` : to;
              const active = location.pathname === href;
              return (
                <Link key={to} to={href} className={`${s.navItem} ${active ? s.navItemActive : ''}`}>
                  <span className={s.navIconWrap}>
                    <Icon size={24} strokeWidth={active ? 2.5 : 1.8} />
                    {badge && unreadCount > 0 && <span className={s.navBadge}>{unreadCount > 9 ? '9+' : unreadCount}</span>}
                  </span>
                  <span className={s.navLabel}>{label}</span>
                </Link>
              );
            })}
            {user?.is_admin && (
              <Link to="/admin" className={`${s.navItem} ${location.pathname === '/admin' ? s.navItemActive : ''}`}>
                <Shield size={24} strokeWidth={location.pathname === '/admin' ? 2.5 : 1.8} />
                <span className={s.navLabel}>Админ</span>
              </Link>
            )}
            </>
          ) : (
            <>
              <Link to="/" className={`${s.navItem} ${location.pathname === '/' ? s.navItemActive : ''}`}>
                <Home size={24} strokeWidth={location.pathname === '/' ? 2.5 : 1.8} />
                <span className={s.navLabel}>Главная</span>
              </Link>
              <button onClick={() => openAuthModal('login')} className={s.navItem}>
                <User size={24} strokeWidth={1.8} />
                <span className={s.navLabel}>Войти</span>
              </button>
            </>
          )}
        </nav>
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
