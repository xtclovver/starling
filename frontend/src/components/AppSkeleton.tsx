import s from '@/styles/app-skeleton.module.css';

export default function AppSkeleton() {
  return (
    <div className={s.shell}>
      <div className={s.sidebar}>
        <div className={s.logoMark} />
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className={s.navIcon} />
        ))}
      </div>
      <div className={s.feed}>
        {[1, 2, 3, 4].map((i) => (
          <div key={i} className={s.post}>
            <div className={s.avatar} />
            <div className={s.lines}>
              <div className={`${s.line} ${s.lineShort}`} />
              <div className={s.line} />
              <div className={`${s.line} ${s.lineMedium}`} />
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}
