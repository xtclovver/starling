import s from '@/styles/components.module.css';

export default function Spinner() {
  return (
    <div className={s.spinner}>
      <div className={s.spinnerDot} />
    </div>
  );
}
