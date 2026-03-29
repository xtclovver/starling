import { useState } from 'react';
import { getMediaKind } from '@/lib/media';
import s from '@/styles/media-grid.module.css';
import type { MediaItem } from '@/types';

interface Props {
  media: MediaItem[];
  onImageClick?: (url: string) => void;
}

function ImageCell({ url, onImageClick }: { url: string; onImageClick?: (url: string) => void }) {
  const [loaded, setLoaded] = useState(false);
  return (
    <div
      className={`${s.imgWrapper} ${loaded ? s.imgLoaded : ''}`}
      onClick={() => onImageClick?.(url)}
      style={{ cursor: onImageClick ? 'zoom-in' : undefined }}
    >
      <img
        src={url}
        alt=""
        loading="lazy"
        className={s.img}
        onLoad={() => setLoaded(true)}
        onError={() => setLoaded(true)}
      />
    </div>
  );
}

export default function MediaGrid({ media, onImageClick }: Props) {
  if (!media.length) return null;

  const count = media.length;

  return (
    <div
      className={`${s.grid} ${count === 1 ? s.grid1 : count === 2 ? s.grid2 : count === 3 ? s.grid3 : s.grid4}`}
      onClick={(e) => e.stopPropagation()}
    >
      {media.map((m, i) => {
        const kind = getMediaKind(m.url);
        if (kind === 'video') {
          return (
            <div key={i} className={s.cell}>
              <video src={m.url} controls className={s.video} />
            </div>
          );
        }
        if (kind === 'audio') {
          return (
            <div key={i} className={s.cell}>
              <audio src={m.url} controls style={{ width: '100%' }} />
            </div>
          );
        }
        return (
          <div key={i} className={s.cell}>
            <ImageCell url={m.url} onImageClick={onImageClick} />
          </div>
        );
      })}
    </div>
  );
}
