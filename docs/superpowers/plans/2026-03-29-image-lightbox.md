# Image Lightbox — Twitter Split Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Переработать `ImageLightbox` — заменить простой оверлей на Twitter-style split: изображение слева, боковая панель с постом и комментариями (infinite scroll) справа.

**Architecture:** `ImageLightbox` получает `post` и `allSrcs[]` вместо одного `src`. Сайдбар рендерит данные поста и встроенный `LightboxComments` — урезанная версия `CommentTree` с intersection observer для infinite scroll. Все вызывающие компоненты обновляют пропсы.

**Tech Stack:** React, TypeScript, CSS Modules, Lucide React, Intersection Observer API, существующий API `getCommentTree`.

---

## Карта файлов

| Файл | Действие |
|---|---|
| `frontend/src/components/ImageLightbox.tsx` | Полная переработка |
| `frontend/src/styles/image-lightbox.module.css` | Создать (стили для нового lightbox) |
| `frontend/src/components/PostCard.tsx` | Обновить пропсы ImageLightbox |
| `frontend/src/pages/PostPage.tsx` | Обновить пропсы ImageLightbox |
| `frontend/src/components/PostModal.tsx` | Обновить пропсы ImageLightbox |
| `frontend/src/components/CommentItem.tsx` | Обновить пропсы ImageLightbox |

---

### Task 1: CSS-модуль для нового lightbox

**Files:**
- Create: `frontend/src/styles/image-lightbox.module.css`

- [ ] **Step 1: Создать CSS-файл**

```css
/* frontend/src/styles/image-lightbox.module.css */

.overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  background: #000;
  display: flex;
  overflow: hidden;
}

/* ── Image area ── */
.imageArea {
  flex: 1 1 0;
  min-width: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  position: relative;
  background: #000;
  cursor: zoom-out;
}

.image {
  max-width: 100%;
  max-height: 100vh;
  object-fit: contain;
  cursor: default;
}

.closeBtn {
  position: absolute;
  top: 14px;
  left: 14px;
  width: 38px;
  height: 38px;
  background: rgba(0, 0, 0, 0.65);
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  cursor: pointer;
  z-index: 10;
}
.closeBtn:hover { background: rgba(255, 255, 255, 0.1); }

.arrowBtn {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  width: 42px;
  height: 42px;
  background: rgba(255, 255, 255, 0.12);
  border: none;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #fff;
  cursor: pointer;
  z-index: 10;
}
.arrowBtn:hover { background: rgba(255, 255, 255, 0.22); }
.arrowLeft { left: 14px; }
.arrowRight { right: 14px; }

.dots {
  position: absolute;
  bottom: 14px;
  left: 50%;
  transform: translateX(-50%);
  display: flex;
  gap: 6px;
}
.dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: rgba(255, 255, 255, 0.35);
}
.dotActive { background: #fff; }

/* ── Sidebar ── */
.sidebar {
  width: 36%;
  min-width: 300px;
  max-width: 420px;
  background: #000;
  border-left: 1px solid #2f3336;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  flex-shrink: 0;
}

.sidebarScroll {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}
.sidebarScroll::-webkit-scrollbar { width: 4px; }
.sidebarScroll::-webkit-scrollbar-track { background: transparent; }
.sidebarScroll::-webkit-scrollbar-thumb { background: #2f3336; border-radius: 4px; }

/* Post header */
.postHeader {
  display: flex;
  gap: 12px;
  margin-bottom: 14px;
}
.authorInfo { flex: 1; }
.authorName { font-weight: 800; font-size: 15px; color: #e7e9ea; }
.authorHandle { color: #71767b; font-size: 14px; }

.postText {
  font-size: 16px;
  line-height: 1.5;
  color: #e7e9ea;
  margin-bottom: 12px;
}
.postMeta {
  color: #71767b;
  font-size: 13px;
  margin-bottom: 12px;
}

.postStats {
  display: flex;
  gap: 16px;
  font-size: 13px;
  color: #71767b;
  padding: 10px 0;
  border-top: 1px solid #2f3336;
  border-bottom: 1px solid #2f3336;
  margin-bottom: 12px;
}
.postStats strong { color: #e7e9ea; }

.actionRow {
  display: flex;
  border-bottom: 1px solid #2f3336;
  margin-bottom: 14px;
}
.actionBtn {
  flex: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  padding: 10px 4px;
  background: none;
  border: none;
  color: #71767b;
  font-size: 13px;
  cursor: pointer;
  border-radius: 6px;
}
.actionBtn:hover { color: #1d9bf0; background: rgba(29, 155, 240, 0.08); }
.actionBtnLiked { color: #f91880; }
.actionBtnLiked:hover { color: #f91880; background: rgba(249, 24, 128, 0.08); }
.actionBtnReposted { color: #00ba7c; }

/* Comments section */
.commentsHeader {
  font-size: 13px;
  font-weight: 700;
  color: #71767b;
  margin-bottom: 12px;
}

.spinnerWrap {
  display: flex;
  justify-content: center;
  padding: 16px 0;
}

.sentinel { height: 1px; }

/* Reply box */
.replyBox {
  padding: 12px 16px;
  border-top: 1px solid #2f3336;
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.replyInput {
  flex: 1;
  background: transparent;
  border: none;
  color: #71767b;
  font-size: 15px;
  outline: none;
  cursor: text;
}
.replyBtn {
  background: #1d9bf0;
  color: #fff;
  border: none;
  border-radius: 20px;
  padding: 7px 18px;
  font-size: 14px;
  font-weight: 800;
  cursor: pointer;
}
.replyBtn:hover { background: #1a8cd8; }
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/styles/image-lightbox.module.css
git commit -m "feat: add image-lightbox CSS module"
```

---

### Task 2: Переработать ImageLightbox.tsx

**Files:**
- Modify: `frontend/src/components/ImageLightbox.tsx`

Новый интерфейс:
```tsx
interface Props {
  src: string;           // URL текущего изображения
  allSrcs?: string[];    // Все изображения поста (для навигации)
  post: Post;            // Данные поста (уже загружены в родителе)
  onClose: () => void;
}
```

Сайдбар содержит: шапку поста, текст, дату, статистику, кнопки действий, список комментариев с infinite scroll через IntersectionObserver, строку ответа.

- [ ] **Step 1: Написать новый ImageLightbox.tsx**

```tsx
import { useEffect, useState, useRef, useCallback } from 'react';
import { X, ChevronLeft, ChevronRight, Heart, MessageCircle, Repeat2, Share, Bookmark } from 'lucide-react';
import { getCommentTree, createComment } from '@/api/comments';
import { useAuthStore } from '@/store/auth';
import Avatar from './Avatar';
import CommentItem from './CommentItem';
import Spinner from './Spinner';
import s from '@/styles/image-lightbox.module.css';
import type { Post, Comment } from '@/types';

interface Props {
  src: string;
  allSrcs?: string[];
  post: Post;
  onClose: () => void;
}

export default function ImageLightbox({ src, allSrcs, post, onClose }: Props) {
  const user = useAuthStore((st) => st.user);
  const srcs = allSrcs && allSrcs.length > 0 ? allSrcs : [src];
  const [idx, setIdx] = useState(() => srcs.indexOf(src) >= 0 ? srcs.indexOf(src) : 0);

  // Comments state
  const [comments, setComments] = useState<Comment[]>([]);
  const [cursor, setCursor] = useState('');
  const [hasMore, setHasMore] = useState(false);
  const [commentsLoading, setCommentsLoading] = useState(true);
  const sentinelRef = useRef<HTMLDivElement>(null);
  const loadingRef = useRef(false);

  // Reply state
  const [replyText, setReplyText] = useState('');
  const [replyLoading, setReplyLoading] = useState(false);

  const loadComments = useCallback(async (c = '') => {
    if (!post || loadingRef.current) return;
    loadingRef.current = true;
    setCommentsLoading(true);
    try {
      const data = await getCommentTree(post.id, c);
      if (c) {
        setComments((prev) => [...prev, ...(data.comments || [])]);
      } else {
        setComments(data.comments || []);
      }
      setCursor(data.pagination?.next_cursor || '');
      setHasMore(data.pagination?.has_more || false);
    } catch { /* ignore */ } finally {
      setCommentsLoading(false);
      loadingRef.current = false;
    }
  }, [post?.id]);

  // Initial load
  useEffect(() => { loadComments(); }, [loadComments]);

  // Infinite scroll via IntersectionObserver
  useEffect(() => {
    if (!sentinelRef.current || !hasMore) return;
    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0].isIntersecting && hasMore && !loadingRef.current) {
          loadComments(cursor);
        }
      },
      { threshold: 0.1 }
    );
    observer.observe(sentinelRef.current);
    return () => observer.disconnect();
  }, [hasMore, cursor, loadComments]);

  // Keyboard
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
      if (e.key === 'ArrowLeft') setIdx((i) => Math.max(0, i - 1));
      if (e.key === 'ArrowRight') setIdx((i) => Math.min(srcs.length - 1, i + 1));
    };
    document.addEventListener('keydown', onKey);
    return () => document.removeEventListener('keydown', onKey);
  }, [onClose, srcs.length]);

  const handleReply = async () => {
    if (!replyText.trim() || replyLoading || !user) return;
    setReplyLoading(true);
    try {
      const comment = await createComment(post.id, replyText);
      setComments((prev) => [comment, ...prev]);
      setReplyText('');
    } catch { /* ignore */ } finally { setReplyLoading(false); }
  };

  const multiple = srcs.length > 1;

  return (
    <div className={s.overlay}>
      {/* Image area */}
      <div className={s.imageArea} onClick={onClose}>
        <button className={s.closeBtn} onClick={onClose}>
          <X size={20} />
        </button>

        {multiple && idx > 0 && (
          <button
            className={`${s.arrowBtn} ${s.arrowLeft}`}
            onClick={(e) => { e.stopPropagation(); setIdx((i) => i - 1); }}
          >
            <ChevronLeft size={24} />
          </button>
        )}

        <img
          src={srcs[idx]}
          alt=""
          className={s.image}
          onClick={(e) => e.stopPropagation()}
        />

        {multiple && idx < srcs.length - 1 && (
          <button
            className={`${s.arrowBtn} ${s.arrowRight}`}
            onClick={(e) => { e.stopPropagation(); setIdx((i) => i + 1); }}
          >
            <ChevronRight size={24} />
          </button>
        )}

        {multiple && (
          <div className={s.dots}>
            {srcs.map((_, i) => (
              <div key={i} className={`${s.dot} ${i === idx ? s.dotActive : ''}`} />
            ))}
          </div>
        )}
      </div>

      {/* Sidebar */}
      <div className={s.sidebar}>
        <div className={s.sidebarScroll}>
          {/* Post header */}
          <div className={s.postHeader}>
            <Avatar
              url={post.author?.avatar_url}
              name={post.author?.display_name || post.author?.username || '?'}
            />
            <div className={s.authorInfo}>
              <div className={s.authorName}>
                {post.author?.display_name || post.author?.username || 'Unknown'}
              </div>
              {post.author?.username && (
                <div className={s.authorHandle}>@{post.author.username}</div>
              )}
            </div>
          </div>

          <p className={s.postText}>{post.content}</p>
          <p className={s.postMeta}>
            {new Date(post.created_at).toLocaleString('ru-RU', {
              hour: '2-digit', minute: '2-digit',
              day: 'numeric', month: 'short', year: 'numeric',
            })}
          </p>

          <div className={s.postStats}>
            <span><strong>{post.reposts_count}</strong> Репостов</span>
            <span><strong>{post.likes_count}</strong> Лайков</span>
            {post.views_count > 0 && (
              <span><strong>{post.views_count}</strong> Просмотров</span>
            )}
          </div>

          <div className={s.actionRow}>
            <button className={s.actionBtn}><MessageCircle size={18} /> <span>{post.comments_count}</span></button>
            <button className={`${s.actionBtn} ${post.reposted ? s.actionBtnReposted : ''}`}><Repeat2 size={18} /></button>
            <button className={`${s.actionBtn} ${post.liked ? s.actionBtnLiked : ''}`}>
              <Heart size={18} fill={post.liked ? 'currentColor' : 'none'} />
            </button>
            <button className={s.actionBtn}><Bookmark size={18} /></button>
            <button className={s.actionBtn}><Share size={18} /></button>
          </div>

          {/* Comments */}
          <div className={s.commentsHeader}>Комментарии</div>
          {comments.map((c) => (
            <CommentItem
              key={c.id}
              comment={c}
              postId={post.id}
              onNewReply={(pid, reply) => {
                setComments((prev) =>
                  prev.map((cm) =>
                    cm.id === pid
                      ? { ...cm, children: [reply, ...(cm.children || [])] }
                      : cm
                  )
                );
              }}
              onDelete={(cid) => {
                setComments((prev) =>
                  prev
                    .map((cm) =>
                      cm.id === cid
                        ? cm.children?.length
                          ? { ...cm, content: '[удалено]', user_id: '' }
                          : null
                        : cm
                    )
                    .filter(Boolean) as Comment[]
                );
              }}
            />
          ))}

          {commentsLoading && (
            <div className={s.spinnerWrap}><Spinner /></div>
          )}

          {/* Sentinel for infinite scroll */}
          <div ref={sentinelRef} className={s.sentinel} />
        </div>

        {/* Reply box */}
        {user && (
          <div className={s.replyBox}>
            <Avatar url={user.avatar_url} name={user.display_name || user.username} size="sm" />
            <input
              className={s.replyInput}
              placeholder="Написать ответ..."
              value={replyText}
              onChange={(e) => setReplyText(e.target.value)}
              onKeyDown={(e) => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleReply(); } }}
            />
            <button
              className={s.replyBtn}
              onClick={handleReply}
              disabled={!replyText.trim() || replyLoading}
            >
              {replyLoading ? '...' : 'Ответить'}
            </button>
          </div>
        )}
      </div>
    </div>
  );
}
```

- [ ] **Step 2: Проверить что файл компилируется**

```bash
cd "frontend" && npx tsc --noEmit 2>&1 | head -30
```

Ожидаем ошибки о несовпадении пропсов в вызывающих компонентах — это нормально, исправим в следующих задачах.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/ImageLightbox.tsx
git commit -m "feat: rewrite ImageLightbox as Twitter-style split layout"
```

---

### Task 3: Обновить PostCard.tsx

**Files:**
- Modify: `frontend/src/components/PostCard.tsx` (строки 48, 289, 334)

PostCard уже имеет доступ к объекту `post`. Нужно:
1. Изменить тип состояния `lightboxSrc` — хранить `src` вместо `string | null`
2. Передать `post` и `allSrcs` в `ImageLightbox`

- [ ] **Step 1: Обновить состояние и вызов lightbox в PostCard**

Найти строку 48:
```tsx
const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
```
Заменить на:
```tsx
const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);
```
(без изменений — состояние остаётся тем же)

Найти строку 289:
```tsx
<MediaGrid media={displayPost.media} onImageClick={(url) => setLightboxSrc(url)} />
```
Оставить без изменений.

Найти строку 334:
```tsx
{lightboxSrc && <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
```
Заменить на:
```tsx
{lightboxSrc && (
  <ImageLightbox
    src={lightboxSrc}
    allSrcs={(displayPost.media || []).map((m) => m.url)}
    post={post}
    onClose={() => setLightboxSrc(null)}
  />
)}
```

- [ ] **Step 2: Проверить типы**

```bash
cd "frontend" && npx tsc --noEmit 2>&1 | grep PostCard
```

Ожидаем: нет ошибок по PostCard.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/PostCard.tsx
git commit -m "feat: pass post and allSrcs to ImageLightbox in PostCard"
```

---

### Task 4: Обновить PostPage.tsx

**Files:**
- Modify: `frontend/src/pages/PostPage.tsx` (строка 114)

PostPage имеет `post` в локальном state. Нужно передать его в lightbox.

- [ ] **Step 1: Обновить вызов ImageLightbox в PostPage**

Найти строку 114:
```tsx
{lightboxSrc && <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
```
Заменить на:
```tsx
{lightboxSrc && post && (
  <ImageLightbox
    src={lightboxSrc}
    allSrcs={(post.media || []).map((m) => m.url)}
    post={post}
    onClose={() => setLightboxSrc(null)}
  />
)}
```

- [ ] **Step 2: Проверить типы**

```bash
cd "frontend" && npx tsc --noEmit 2>&1 | grep PostPage
```

Ожидаем: нет ошибок по PostPage.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/PostPage.tsx
git commit -m "feat: pass post and allSrcs to ImageLightbox in PostPage"
```

---

### Task 5: Обновить PostModal.tsx

**Files:**
- Modify: `frontend/src/components/PostModal.tsx` (строка 111)

PostModal имеет `post` в локальном state.

- [ ] **Step 1: Обновить вызов ImageLightbox в PostModal**

Найти строку 111:
```tsx
{lightboxSrc && <ImageLightbox src={lightboxSrc} onClose={() => setLightboxSrc(null)} />}
```
Заменить на:
```tsx
{lightboxSrc && post && (
  <ImageLightbox
    src={lightboxSrc}
    allSrcs={(post.media || []).map((m) => m.url)}
    post={post}
    onClose={() => setLightboxSrc(null)}
  />
)}
```

- [ ] **Step 2: Проверить типы**

```bash
cd "frontend" && npx tsc --noEmit 2>&1 | grep PostModal
```

Ожидаем: нет ошибок по PostModal.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/PostModal.tsx
git commit -m "feat: pass post and allSrcs to ImageLightbox in PostModal"
```

---

### Task 6: Обновить CommentItem.tsx

**Files:**
- Modify: `frontend/src/components/CommentItem.tsx` (строка 294)

CommentItem открывает lightbox для одиночного изображения комментария. У него нет объекта `Post` — только `comment` и `postId`. Lightbox в этом случае должен показывать только изображение без сайдбара поста.

Решение: передать минимальный объект-заглушку для `post` — только то, что нужно для рендера (пустой пост с `id = postId`). Но это плохо, так как сайдбар покажет пустые данные.

Лучшее решение: добавить пропс `post?: Post` в `CommentItem` и пробрасывать его из `CommentTree`. Если `post` не передан — lightbox открывается без сайдбара (старое поведение: только `src`).

Однако это большое изменение цепочки пропсов. По спеке lightbox для комментариев не упоминается — оставить старое поведение (только изображение, без сайдбара). Для этого нужно добавить режим совместимости в `ImageLightbox` — когда `post` не передан, показывать только изображение.

- [ ] **Step 1: Добавить необязательный `post` в Props ImageLightbox**

В `frontend/src/components/ImageLightbox.tsx` изменить интерфейс Props:

```tsx
interface Props {
  src: string;
  allSrcs?: string[];
  post?: Post;           // необязательный — если не передан, только изображение
  onClose: () => void;
}
```

В теле компонента обернуть сайдбар в условие:

```tsx
// После закрывающего тега </div> image area, перед закрывающим </div> overlay:
{post && (
  <div className={s.sidebar}>
    {/* ...всё содержимое сайдбара... */}
  </div>
)}
```

Также изменить хук `loadComments` чтобы не вызывался без `post`:

```tsx
const loadComments = useCallback(async (c = '') => {
  if (!post || loadingRef.current) return;
  // ...остальное без изменений
}, [post?.id]);
```

И `useEffect` для начальной загрузки:
```tsx
useEffect(() => {
  if (post) loadComments();
}, [loadComments, post]);  // добавить post в deps
```

- [ ] **Step 2: Проверить типы**

```bash
cd "frontend" && npx tsc --noEmit 2>&1 | head -20
```

Ожидаем: 0 ошибок.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/ImageLightbox.tsx
git commit -m "feat: make post prop optional in ImageLightbox for comment images"
```

---

### Task 7: Финальная проверка

- [ ] **Step 1: Полная проверка типов**

```bash
cd "frontend" && npx tsc --noEmit
```

Ожидаем: пустой вывод (0 ошибок).

- [ ] **Step 2: Запустить dev-сервер и проверить вручную**

```bash
cd "frontend" && npm run dev
```

Проверить:
1. Открыть пост с одним изображением → lightbox открывается, справа виден пост и комментарии
2. Открыть пост с несколькими изображениями → стрелки навигации и точки видны
3. Прокрутить комментарии вниз → подгружаются следующие
4. Нажать Escape → lightbox закрывается
5. Кликнуть на тёмную область слева → lightbox закрывается
6. Открыть изображение из комментария → lightbox без сайдбара (только фото)

- [ ] **Step 3: Commit если были правки**

```bash
git add -p
git commit -m "fix: lightbox manual testing fixes"
```
