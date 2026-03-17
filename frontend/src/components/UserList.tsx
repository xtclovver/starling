import { Link } from 'react-router-dom';
import Avatar from './Avatar';
import Spinner from './Spinner';
import s from '@/styles/profile.module.css';
import type { User } from '@/types';

interface UserListProps {
  users: User[];
  loading?: boolean;
  hasMore?: boolean;
  onLoadMore?: () => void;
}

export default function UserList({ users, loading, hasMore, onLoadMore }: UserListProps) {
  if (!loading && users.length === 0) {
    return <p className={s.empty}>Нет пользователей</p>;
  }

  return (
    <div>
      {users.map((u) => (
        <Link key={u.id} to={`/profile/${u.id}`} className={s.userItem}>
          <Avatar url={u.avatar_url} name={u.display_name || u.username} />
          <div className={s.userItemInfo}>
            <p className={s.userItemName}>{u.display_name || u.username}</p>
            <p className={s.userItemHandle}>@{u.username}</p>
            {u.bio && <p className={s.userItemBio}>{u.bio}</p>}
          </div>
        </Link>
      ))}
      {loading && <Spinner />}
      {hasMore && !loading && onLoadMore && (
        <button onClick={onLoadMore} className={s.loadMoreBtn}>Загрузить ещё</button>
      )}
    </div>
  );
}
