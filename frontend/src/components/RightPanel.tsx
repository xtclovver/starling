import SearchUsers from './SearchUsers';
import { useWsStore } from '@/store/ws';
import s from '@/styles/layout.module.css';

export default function RightPanel() {
  const connected = useWsStore((st) => st.connected);

  return (
    <aside className={s.rightPanel}>
      <SearchUsers />
      <div className={s.infoBox}>
        <h3 className={s.infoBoxTitle}>О платформе</h3>
        <p className={s.infoBoxText}>
          Микроблог — платформа для коротких постов, подписок и обсуждений.
        </p>
        <div className={s.statusRow}>
          <span className={`${s.statusDot} ${connected ? s.statusOnline : s.statusOffline}`} />
          {connected ? 'Live-обновления активны' : 'Офлайн'}
        </div>
      </div>
    </aside>
  );
}
