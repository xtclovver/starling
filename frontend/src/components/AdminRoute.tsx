import { Navigate } from 'react-router-dom';
import { useAuthStore } from '@/store/auth';

export default function AdminRoute({ children }: { children: React.ReactNode }) {
  const user = useAuthStore((s) => s.user);

  if (!user?.is_admin) {
    return <Navigate to="/" replace />;
  }
  return <>{children}</>;
}
