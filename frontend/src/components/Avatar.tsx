import s from '@/styles/components.module.css';

interface AvatarProps {
  url?: string;
  name?: string;
  size?: 'xs' | 'sm' | 'md' | 'lg' | 'xl';
  className?: string;
}

const sizeClass = { xs: s.avatarXs, sm: s.avatarSm, md: s.avatarMd, lg: s.avatarLg, xl: s.avatarXl };

export default function Avatar({ url, name = '?', size = 'md', className = '' }: AvatarProps) {
  const initial = name.charAt(0).toUpperCase();
  const sc = sizeClass[size];

  if (url && /^(https?|blob):/i.test(url)) {
    return <img src={url} alt={name} className={`${s.avatar} ${sc} ${className}`} />;
  }

  const hash = name.split('').reduce((acc, c) => acc + c.charCodeAt(0), 0);
  const hue = hash % 360;

  return (
    <div
      className={`${s.avatarFallback} ${sc} ${className}`}
      style={{ background: `hsl(${hue}, 50%, 40%)` }}
    >
      {initial}
    </div>
  );
}
