import s from '@/styles/components.module.css';

export default function SkeletonPost() {
  return (
    <div className={s.skeleton}>
      <div className={s.skeletonRow}>
        <div className={s.skeletonCircle} />
        <div className={s.skeletonLines}>
          <div className={`${s.skeletonLine} ${s.skeletonLineShort}`} />
          <div className={s.skeletonLine} />
          <div className={`${s.skeletonLine} ${s.skeletonLineMedium}`} />
          <div className={s.skeletonActions}>
            <div className={s.skeletonActionBlock} />
            <div className={s.skeletonActionBlock} />
          </div>
        </div>
      </div>
    </div>
  );
}
