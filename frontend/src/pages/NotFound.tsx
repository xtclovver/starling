import { Link } from 'react-router-dom';
import { Feather } from 'lucide-react';
import s from '@/styles/not-found.module.css';

export default function NotFound() {
  return (
    <div className={s.page}>
      <Link to="/" className={s.logoRow}>
        <Feather size={28} />
        <span className={s.logoName}>Starling</span>
      </Link>

      <span className={s.emoji}>🔭</span>
      <div className={s.code}>404</div>
      <h1 className={s.title}>Страница улетела в космос</h1>
      <p className={s.subtitle}>
        Мы не можем найти то, что ты ищешь. Возможно, ссылка устарела или страница была удалена.
      </p>

      <button className={s.btn} onClick={() => history.back()}>
        ← Вернуться назад
      </button>
      <Link to="/" className={s.homeLink}>или перейти на главную</Link>
    </div>
  );
}
