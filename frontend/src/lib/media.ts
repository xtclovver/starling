const VIDEO_EXTS = new Set(['mp4', 'webm']);
const AUDIO_EXTS = new Set(['mp3', 'ogg', 'wav', 'mpeg']);

export type MediaKind = 'image' | 'video' | 'audio';

export function getMediaKind(url: string): MediaKind {
  const ext = url.split('?')[0].split('.').pop()?.toLowerCase() ?? '';
  if (VIDEO_EXTS.has(ext)) return 'video';
  if (AUDIO_EXTS.has(ext)) return 'audio';
  return 'image';
}
