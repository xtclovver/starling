import { useState, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, X } from 'lucide-react';
import { searchUsers } from '@/api/users';
import { useDebounce } from '@/hooks/useDebounce';
import Avatar from './Avatar';
import s from '@/styles/profile.module.css';
import type { User } from '@/types';

export default function SearchUsers() {
  const navigate = useNavigate();
  const [query, setQuery] = useState('');
  const [results, setResults] = useState<User[]>([]);
  const [open, setOpen] = useState(false);
  const debounced = useDebounce(query, 300);

  useEffect(() => {
    if (debounced.length < 2) return;
    let cancelled = false;
    searchUsers(debounced).then((data) => {
      if (!cancelled) setResults(data.users || []);
    }).catch(() => {});
    return () => { cancelled = true; };
  }, [debounced]);

  const handleSelect = (userId: string) => {
    navigate(`/profile/${userId}`);
    setQuery('');
    setOpen(false);
  };

  return (
    <div className={s.searchWrap}>
      <div style={{ position: 'relative' }}>
        <Search size={16} className={s.searchIcon} />
        <input
          value={query}
          onChange={(e) => { const v = e.target.value; setQuery(v); setOpen(true); if (v.length < 2) setResults([]); }}
          onFocus={() => setOpen(true)}
          placeholder="Поиск"
          className={s.searchInput}
        />
        {query && (
          <button onClick={() => { setQuery(''); setResults([]); }} className={s.searchClear}>
            <X size={14} />
          </button>
        )}
      </div>
      {open && results.length > 0 && (
        <>
          <div className={s.searchBackdrop} onClick={() => setOpen(false)} />
          <div className={s.searchDropdown}>
            {results.map((u) => (
              <button key={u.id} onClick={() => handleSelect(u.id)} className={s.searchItem}>
                <Avatar url={u.avatar_url} name={u.display_name || u.username} size="sm" />
                <div style={{ minWidth: 0 }}>
                  <p className={s.searchItemName}>{u.display_name || u.username}</p>
                  <p className={s.searchItemHandle}>@{u.username}</p>
                </div>
              </button>
            ))}
          </div>
        </>
      )}
    </div>
  );
}
