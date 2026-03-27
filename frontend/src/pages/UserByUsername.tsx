import { useEffect } from 'react';
import { useParams, useNavigate } from 'react-router-dom';
import { searchUsers } from '@/api/users';
import Spinner from '@/components/Spinner';

export default function UserByUsername() {
  const { username } = useParams<{ username: string }>();
  const navigate = useNavigate();

  useEffect(() => {
    if (!username) { navigate('/'); return; }
    searchUsers(username).then((data) => {
      const exact = data.users?.find((u) => u.username === username);
      if (exact) navigate(`/profile/${exact.id}`, { replace: true });
      else navigate('/', { replace: true });
    }).catch(() => navigate('/', { replace: true }));
  }, [username, navigate]);

  return <Spinner />;
}
