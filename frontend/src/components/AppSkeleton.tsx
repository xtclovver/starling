import s from '@/styles/app-skeleton.module.css';

export default function AppSkeleton() {
  return (
    <div className={s.shell}>
      <div className={s.sidebarCol}>
        <div className={s.sidebar}>
          <div className={s.logoRow}>
            <div className={s.logoIcon} />
            <div className={s.logoText} />
          </div>
          <div className={s.nav}>
            {[1, 2, 3, 4, 5].map((i) => (
              <div key={i} className={s.navItem}>
                <div className={s.navIcon} />
                <div className={s.navLabel} />
              </div>
            ))}
          </div>
        </div>
      </div>
      <div className={s.mainCol}>
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
      <div className={s.rightCol}>
        <div className={s.searchBar} />
        <div className={s.rightBlock}>
          <div className={s.blockTitle} />
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className={s.trendRow}>
              <div className={s.trendTag} />
              <div className={s.trendCount} />
            </div>
          ))}
        </div>
        <div className={s.rightBlock}>
          <div className={s.blockTitle} />
          {[1, 2, 3].map((i) => (
            <div key={i} className={s.userRow}>
              <div className={s.userAvatar} />
              <div className={s.userLines}>
                <div className={s.userLine} />
                <div className={`${s.userLine} ${s.userLineShort}`} />
              </div>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
