# Loading Animations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add skeleton/spinner loading states everywhere users wait — app startup, page loads, action buttons, form submissions, and image loading.

**Architecture:** Reuse the existing `pulse` keyframe from `index.css` for all skeleton animations (it's already global). Each task is self-contained: one component or one page at a time. No new libraries, no new stores.

**Tech Stack:** React 19, TypeScript, CSS Modules, Zustand (existing), Axios (existing)

---

## Task 1: App Startup Skeleton

**Files:**
- Create: `frontend/src/components/AppSkeleton.tsx`
- Create: `frontend/src/styles/app-skeleton.module.css`
- Modify: `frontend/src/App.tsx`

The `useAuthStore` already has `initializing: true` and `setInitializing`. `App.tsx` currently doesn't use it — we just need to read it and show the skeleton.

- [ ] **Step 1: Create `AppSkeleton.tsx`**

```tsx
// frontend/src/components/AppSkeleton.tsx
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
```

- [ ] **Step 2: Create `app-skeleton.module.css`**

```css
/* frontend/src/styles/app-skeleton.module.css */
.shell {
  min-height: 100vh;
  display: flex;
  justify-content: center;
}

.sidebar {
  width: 68px;
  border-right: 1px solid var(--border);
  padding: 16px 12px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 20px;
  flex-shrink: 0;
}

.feed {
  flex: 1;
  max-width: 600px;
}

.logoMark {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  animation: pulse 1.5s ease-in-out infinite;
}

.navIcon {
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  animation: pulse 1.5s ease-in-out infinite;
}

.post {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  animation: pulse 1.5s ease-in-out infinite;
}

.avatar {
  width: 40px;
  height: 40px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  flex-shrink: 0;
}

.lines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding-top: 2px;
}

.line {
  height: 14px;
  background: var(--bg-tertiary);
  border-radius: 4px;
}

.lineShort { width: 45%; }
.lineMedium { width: 75%; }
```

- [ ] **Step 3: Modify `App.tsx` to show skeleton while initializing**

```tsx
// frontend/src/App.tsx
import { BrowserRouter, Routes, Route } from 'react-router-dom';
import { useAuthStore } from './store/auth';
import AppSkeleton from './components/AppSkeleton';
import Layout from './components/Layout';
import ProtectedRoute from './components/ProtectedRoute';
import Home from './pages/Home';
import Login from './pages/Login';
import Register from './pages/Register';
import Profile from './pages/Profile';
import PostPage from './pages/PostPage';
import Settings from './pages/Settings';
import Bookmarks from './pages/Bookmarks';
import HashtagPage from './pages/HashtagPage';
import Notifications from './pages/Notifications';
import UserByUsername from './pages/UserByUsername';
import NotFound from './pages/NotFound';

function App() {
  const initializing = useAuthStore((st) => st.initializing);

  if (initializing) return <AppSkeleton />;

  return (
    <BrowserRouter>
      <Routes>
        <Route path="/login" element={<Login />} />
        <Route path="/register" element={<Register />} />
        <Route element={<Layout />}>
          <Route path="/" element={<Home />} />
          <Route path="/profile/:id" element={<Profile />} />
          <Route path="/u/:username" element={<UserByUsername />} />
          <Route path="/post/:id" element={<PostPage />} />
          <Route path="/hashtag/:tag" element={<HashtagPage />} />
          <Route path="/bookmarks" element={<ProtectedRoute><Bookmarks /></ProtectedRoute>} />
          <Route path="/notifications" element={<ProtectedRoute><Notifications /></ProtectedRoute>} />
          <Route path="/settings" element={<ProtectedRoute><Settings /></ProtectedRoute>} />
        </Route>
        <Route path="*" element={<NotFound />} />
      </Routes>
    </BrowserRouter>
  );
}

export default App;
```

- [ ] **Step 4: Find where `setInitializing(false)` is called and verify it works**

Search for `setInitializing` in the codebase:
```bash
grep -r "setInitializing" frontend/src --include="*.ts" --include="*.tsx"
```

The store already has `initializing: true` as default. Make sure `setInitializing(false)` is called after the auth check completes (success or failure). If not found anywhere else, locate the auth initialization logic (likely in `Layout.tsx` or a hook) and ensure `setInitializing(false)` is called in the `finally` block.

- [ ] **Step 5: Run dev server and verify skeleton appears then disappears**

```bash
cd frontend && npm run dev
```

Open http://localhost:5173. On first load you should see the skeleton briefly, then the real layout.

- [ ] **Step 6: Commit**

```bash
git add frontend/src/components/AppSkeleton.tsx frontend/src/styles/app-skeleton.module.css frontend/src/App.tsx
git commit -m "feat: add app startup skeleton screen"
```

---

## Task 2: Skeleton for Notifications Page

**Files:**
- Modify: `frontend/src/pages/Notifications.tsx`

Currently shows `<Spinner />` on initial load. Replace with skeleton rows.

- [ ] **Step 1: Add skeleton rows to `Notifications.tsx`**

Replace the `loading && notifications.length === 0` branch. The full modified file:

```tsx
// frontend/src/pages/Notifications.tsx
import { useEffect, useCallback } from 'react';
import { Link } from 'react-router-dom';
import { Heart, MessageCircle, UserPlus, Repeat2, AtSign } from 'lucide-react';
import { getNotifications, getUnreadCount, markRead, markAllRead } from '@/api/notifications';
import { useNotificationStore } from '@/store/notifications';
import { useInfiniteScroll } from '@/hooks/useInfiniteScroll';
import Avatar from '@/components/Avatar';
import Spinner from '@/components/Spinner';
import { timeAgo } from '@/lib/time';
import s from '@/styles/layout.module.css';
import n from '@/styles/notification.module.css';

const TYPE_CONFIG: Record<string, { icon: typeof Heart; label: string; color: string }> = {
  like_post: { icon: Heart, label: 'понравился ваш пост', color: 'var(--like)' },
  like_comment: { icon: Heart, label: 'понравился ваш комментарий', color: 'var(--like)' },
  new_comment: { icon: MessageCircle, label: 'прокомментировал ваш пост', color: 'var(--accent)' },
  new_follower: { icon: UserPlus, label: 'подписался на вас', color: 'var(--success)' },
  repost: { icon: Repeat2, label: 'репостнул ваш пост', color: 'var(--success)' },
  quote: { icon: Repeat2, label: 'процитировал ваш пост', color: 'var(--accent)' },
  mention: { icon: AtSign, label: 'упомянул вас', color: 'var(--accent)' },
};

function SkeletonNotification() {
  return (
    <div className={n.skeletonItem}>
      <div className={n.skeletonIcon} />
      <div className={n.skeletonBody}>
        <div className={n.skeletonLine} style={{ width: '40%' }} />
        <div className={n.skeletonLine} style={{ width: '65%', marginTop: 6 }} />
        <div className={n.skeletonLine} style={{ width: '25%', marginTop: 6 }} />
      </div>
    </div>
  );
}

export default function Notifications() {
  const { notifications, cursor, hasMore, loading, setNotifications, appendNotifications, setLoading, setUnreadCount, markAllAsRead } = useNotificationStore();

  const load = useCallback(async (c = '') => {
    setLoading(true);
    try {
      const data = await getNotifications(c);
      const items = data.notifications || [];
      if (c) appendNotifications(items, data.pagination?.next_cursor || '', data.pagination?.has_more || false);
      else setNotifications(items, data.pagination?.next_cursor || '', data.pagination?.has_more || false);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [setNotifications, appendNotifications, setLoading]);

  useEffect(() => {
    load();
    getUnreadCount().then(setUnreadCount).catch(() => {});
  }, [load, setUnreadCount]);

  const loadMore = useCallback(() => {
    if (cursor && !loading) load(cursor);
  }, [cursor, loading, load]);

  const sentinelRef = useInfiniteScroll(loadMore, hasMore, loading);

  const handleMarkAllRead = async () => {
    try {
      await markAllRead();
      markAllAsRead();
    } catch { /* ignore */ }
  };

  const handleClickNotification = async (id: string, read: boolean) => {
    if (!read) {
      try { await markRead(id); useNotificationStore.getState().markAsRead(id); } catch { /* ignore */ }
    }
  };

  return (
    <div>
      <header className={s.pageHeader}>
        <h1 className={s.pageHeaderTitle}>Уведомления</h1>
        {notifications.some((n) => !n.read) && (
          <button onClick={handleMarkAllRead} className={n.markAllBtn}>Прочитать все</button>
        )}
      </header>
      {loading && notifications.length === 0 ? (
        <>{[1, 2, 3, 4, 5].map((i) => <SkeletonNotification key={i} />)}</>
      ) : (
        <>
          {notifications.map((notif) => {
            const config = TYPE_CONFIG[notif.type] || TYPE_CONFIG.like_post;
            const Icon = config.icon;
            return (
              <div
                key={notif.id}
                className={`${n.item} ${!notif.read ? n.itemUnread : ''}`}
                onClick={() => handleClickNotification(notif.id, notif.read)}
              >
                <div className={n.iconWrap} style={{ color: config.color }}>
                  <Icon size={18} />
                </div>
                <div className={n.body}>
                  <Link to={`/profile/${notif.actor_id}`} className={n.actorRow}>
                    <Avatar url={notif.actor?.avatar_url} name={notif.actor?.display_name || notif.actor?.username || '?'} size="xs" />
                    <span className={n.actorName}>{notif.actor?.display_name || notif.actor?.username || 'Пользователь'}</span>
                  </Link>
                  <p className={n.text}>{config.label}</p>
                  <span className={n.time}>{timeAgo(notif.created_at)}</span>
                </div>
              </div>
            );
          })}
          <div ref={sentinelRef} />
          {loading && notifications.length > 0 && <Spinner />}
          {!loading && notifications.length === 0 && <p style={{ padding: 24, textAlign: 'center', color: 'var(--text-tertiary)' }}>Нет уведомлений</p>}
        </>
      )}
    </div>
  );
}
```

- [ ] **Step 2: Add skeleton CSS to `notification.module.css`**

Append to the end of `frontend/src/styles/notification.module.css`:

```css
/* Skeleton */
.skeletonItem {
  display: flex;
  gap: 12px;
  padding: 12px 16px;
  border-bottom: 1px solid var(--border);
  animation: pulse 1.5s ease-in-out infinite;
}
.skeletonIcon {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  flex-shrink: 0;
}
.skeletonBody {
  flex: 1;
  display: flex;
  flex-direction: column;
}
.skeletonLine {
  height: 12px;
  background: var(--bg-tertiary);
  border-radius: 4px;
}
```

- [ ] **Step 3: Verify notification skeleton appears**

Navigate to `/notifications` while logged in. On first open you should see 5 pulsing skeleton rows, then real notifications.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/Notifications.tsx frontend/src/styles/notification.module.css
git commit -m "feat: add skeleton loading for notifications page"
```

---

## Task 3: Skeleton for PostPage

**Files:**
- Modify: `frontend/src/pages/PostPage.tsx`

Currently shows `<Spinner />` while loading. Replace with a `SkeletonPost` + 2 comment skeleton rows.

- [ ] **Step 1: Update `PostPage.tsx`**

Change the loading branch from `if (loading) return <div>{header}<Spinner /></div>;` to:

```tsx
// frontend/src/pages/PostPage.tsx
import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, Heart, MessageCircle, Trash2, Eye } from 'lucide-react';
import { getPost, likePost, unlikePost, deletePost, recordViews } from '@/api/posts';
import { useAuthStore } from '@/store/auth';
import Avatar from '@/components/Avatar';
import MediaGrid from '@/components/MediaGrid';
import CommentTree from '@/components/CommentTree';
import SkeletonPost from '@/components/SkeletonPost';
import ImageLightbox from '@/components/ImageLightbox';
import l from '@/styles/layout.module.css';
import s from '@/styles/post.module.css';
import c from '@/styles/components.module.css';
import type { Post } from '@/types';

function SkeletonComment() {
  return (
    <div style={{ display: 'flex', gap: 10, padding: '10px 16px', borderBottom: '1px solid var(--border)', animation: 'pulse 1.5s ease-in-out infinite' }}>
      <div className={c.skeletonCircle} style={{ width: 32, height: 32 }} />
      <div style={{ flex: 1, display: 'flex', flexDirection: 'column', gap: 7 }}>
        <div className={c.skeletonLine} style={{ width: '35%' }} />
        <div className={c.skeletonLine} style={{ width: '70%' }} />
      </div>
    </div>
  );
}

export default function PostPage() {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const user = useAuthStore((st) => st.user);
  const [post, setPost] = useState<Post | null>(null);
  const [loading, setLoading] = useState(true);
  const [likeLoading, setLikeLoading] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    getPost(id).then((p) => setPost(p)).catch(() => setPost(null)).finally(() => setLoading(false));
  }, [id]);

  useEffect(() => {
    if (!id) return;
    recordViews([id]).catch(() => {});
  }, [id]);

  const handleLike = async () => {
    if (!post || likeLoading || !user) return;
    setLikeLoading(true);
    const wasLiked = post.liked;
    setPost({ ...post, liked: !wasLiked, likes_count: post.likes_count + (wasLiked ? -1 : 1) });
    try {
      if (wasLiked) await unlikePost(post.id); else await likePost(post.id);
    } catch { setPost({ ...post }); }
    finally { setLikeLoading(false); }
  };

  const handleDelete = async () => {
    if (!post) return;
    try { await deletePost(post.id); navigate('/', { replace: true }); } catch { /* ignore */ }
  };

  const header = (
    <header className={l.pageHeader}>
      <button onClick={() => navigate(-1)} className={l.backBtn}><ArrowLeft size={18} /></button>
      <h1 className={l.pageHeaderTitle}>Пост</h1>
    </header>
  );

  if (loading) {
    return (
      <div>
        {header}
        <SkeletonPost />
        <SkeletonComment />
        <SkeletonComment />
      </div>
    );
  }

  if (!post) return <div>{header}<p style={{ color: 'var(--text-secondary)', textAlign: 'center', padding: '64px 0' }}>Пост не найден</p></div>;

  const isOwner = user?.id === post.user_id;

  return (
    <div>
      {header}
      <article className={s.postDetail}>
        <div className={s.postDetailHeader}>
          <Link to={`/profile/${post.user_id}`}>
            <Avatar url={post.author?.avatar_url} name={post.author?.display_name || post.author?.username} />
          </Link>
          <div className={s.postDetailAuthorInfo}>
            <Link to={`/profile/${post.user_id}`} className={s.postDetailName}>
              {post.author?.display_name || post.author?.username || 'Unknown'}
            </Link>
            {post.author?.username && <p className={s.postDetailHandle}>@{post.author.username}</p>}
          </div>
          {isOwner && (
            <button onClick={handleDelete} className={s.deleteBtn}><Trash2 size={16} /></button>
          )}
        </div>

        <p className={s.postDetailContent}>{post.content}</p>

        {post.media && post.media.length > 0 && (
          <div style={{ marginTop: 12 }}>
            <MediaGrid media={post.media} onImageClick={(url) => setLightboxSrc(url)} />
          </div>
        )}

        <p className={s.postDetailTimestamp}>
          {new Date(post.created_at).toLocaleString('ru-RU', { hour: '2-digit', minute: '2-digit', day: 'numeric', month: 'short', year: 'numeric' })}
        </p>

        <div className={s.postDetailStats}>
          <span><span className={s.statBold}>{post.likes_count}</span> <span className={s.statLabel}>нравится</span></span>
          <span><span className={s.statBold}>{post.comments_count}</span> <span className={s.statLabel}>комментариев</span></span>
          {post.views_count > 0 && (
            <span className={s.viewCount}><Eye size={15} /> <span className={s.statBold}>{post.views_count}</span> <span className={s.statLabel}>просмотров</span></span>
          )}
        </div>

        <div className={s.postDetailActions}>
          <button className={s.postDetailActionBtn}><MessageCircle size={20} /></button>
          <button
            onClick={handleLike}
            className={`${s.postDetailActionBtn} ${s.postDetailLikeBtn} ${post.liked ? s.postDetailLiked : ''}`}
          >
            <Heart size={20} fill={post.liked ? 'currentColor' : 'none'} />
          </button>
        </div>
      </article>

      <CommentTree postId={post.id} />
      {lightboxSrc && post && (
        <ImageLightbox
          src={lightboxSrc}
          allSrcs={(post.media || []).map((m) => m.url)}
          post={post}
          onClose={() => setLightboxSrc(null)}
        />
      )}
    </div>
  );
}
```

- [ ] **Step 2: Verify PostPage skeleton**

Navigate to any post URL. While loading you should see a skeleton post card + 2 skeleton comment rows.

- [ ] **Step 3: Commit**

```bash
git add frontend/src/pages/PostPage.tsx
git commit -m "feat: add skeleton loading for post page"
```

---

## Task 4: Button Action Pending States (PostCard)

**Files:**
- Modify: `frontend/src/components/PostCard.tsx`
- Modify: `frontend/src/styles/post.module.css`

Add `opacity: 0.5` + `pointer-events: none` on action buttons while their request is in flight. `likeLoading`, `bookmarkLoading`, `repostLoading` already exist — just wire them to CSS.

- [ ] **Step 1: Add `actionBtnPending` CSS class to `post.module.css`**

Append to `frontend/src/styles/post.module.css`:

```css
/* Pending state for action buttons */
.actionBtnPending {
  opacity: 0.5;
  pointer-events: none;
}
```

- [ ] **Step 2: Apply pending class to action buttons in `PostCard.tsx`**

Find the three action buttons (repost, like, bookmark) and add the pending class. Change these three buttons in the `postActions` div:

```tsx
<button
  onClick={handleRepost}
  className={`${s.actionBtn} ${post.reposted ? s.actionBtnReposted : ''} ${repostLoading ? s.actionBtnPending : ''}`}
>
  <Repeat2 size={17} />
  {post.reposts_count > 0 && <span>{post.reposts_count}</span>}
</button>
<button
  onClick={handleLike}
  className={`${s.actionBtn} ${s.actionBtnLike} ${post.liked ? s.actionBtnLiked : ''} ${likeLoading ? s.actionBtnPending : ''}`}
>
  <Heart size={17} fill={post.liked ? 'currentColor' : 'none'} />
  {post.likes_count > 0 && <span>{post.likes_count}</span>}
</button>
<button
  onClick={handleBookmark}
  className={`${s.actionBtn} ${post.bookmarked ? s.actionBtnBookmarked : ''} ${bookmarkLoading ? s.actionBtnPending : ''}`}
>
  <Bookmark size={17} fill={post.bookmarked ? 'currentColor' : 'none'} />
</button>
```

- [ ] **Step 3: Verify buttons go semi-transparent on click**

Click a like button rapidly — button should dim immediately and block double-clicks.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/PostCard.tsx frontend/src/styles/post.module.css
git commit -m "feat: add pending state to post action buttons"
```

---

## Task 5: Button Action Pending States (PostPage)

**Files:**
- Modify: `frontend/src/pages/PostPage.tsx`

The like button in PostPage doesn't have a pending class yet. Add it.

- [ ] **Step 1: Add pending class to the like button in `PostPage.tsx`**

Find the like button in the `postDetailActions` div and update it:

```tsx
<button
  onClick={handleLike}
  disabled={likeLoading}
  className={`${s.postDetailActionBtn} ${s.postDetailLikeBtn} ${post.liked ? s.postDetailLiked : ''}`}
  style={likeLoading ? { opacity: 0.5 } : undefined}
>
  <Heart size={20} fill={post.liked ? 'currentColor' : 'none'} />
</button>
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/PostPage.tsx
git commit -m "feat: add pending state to PostPage like button"
```

---

## Task 6: CreatePost Submit Button Spinner

**Files:**
- Modify: `frontend/src/components/CreatePost.tsx`
- Modify: `frontend/src/styles/post.module.css`

Currently the submit button shows `'Отправка...'` text when `loading`. Replace with a small inline spinner using the existing `spin` keyframe.

- [ ] **Step 1: Add `submitBtnSpinner` CSS to `post.module.css`**

Append to `frontend/src/styles/post.module.css`:

```css
/* Inline spinner for submit button */
.submitBtnSpinner {
  width: 14px;
  height: 14px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  display: inline-block;
}
```

- [ ] **Step 2: Update the submit button in `CreatePost.tsx`**

Change the submit button from:
```tsx
<button onClick={handleSubmit} disabled={!canSubmit} className={s.submitBtn}>
  {loading ? 'Отправка...' : 'Опубликовать'}
</button>
```

To:
```tsx
<button onClick={handleSubmit} disabled={!canSubmit} className={s.submitBtn}>
  {loading ? <span className={s.submitBtnSpinner} /> : 'Опубликовать'}
</button>
```

- [ ] **Step 3: Verify spinner appears on submit**

Type a post and click "Опубликовать". The button should show a spinning ring until the post is created.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/CreatePost.tsx frontend/src/styles/post.module.css
git commit -m "feat: add spinner to CreatePost submit button"
```

---

## Task 7: CommentForm Submit Button Spinner

**Files:**
- Modify: `frontend/src/components/CommentForm.tsx`
- Modify: `frontend/src/styles/comment.module.css`

Same pattern as Task 6, applied to `CommentForm`.

- [ ] **Step 1: Add `commentSubmitSpinner` CSS to `comment.module.css`**

Read `frontend/src/styles/comment.module.css` first to see where to append, then add:

```css
/* Inline spinner for comment submit */
.commentSubmitSpinner {
  width: 12px;
  height: 12px;
  border: 2px solid rgba(255, 255, 255, 0.3);
  border-top-color: white;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  display: inline-block;
}
```

- [ ] **Step 2: Update submit button in `CommentForm.tsx`**

Change:
```tsx
<button onClick={handleSubmit} disabled={!content.trim() || loading} className={s.commentSubmitBtn}>
  {loading ? '...' : 'Отправить'}
</button>
```

To:
```tsx
<button onClick={handleSubmit} disabled={!content.trim() || loading} className={s.commentSubmitBtn}>
  {loading ? <span className={s.commentSubmitSpinner} /> : 'Отправить'}
</button>
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/CommentForm.tsx frontend/src/styles/comment.module.css
git commit -m "feat: add spinner to CommentForm submit button"
```

---

## Task 8: Image Loading Shimmer in MediaGrid

**Files:**
- Modify: `frontend/src/components/MediaGrid.tsx`
- Modify: `frontend/src/styles/media-grid.module.css`

Add a shimmer background to image cells that hides once the image loads.

- [ ] **Step 1: Add shimmer and loaded CSS to `media-grid.module.css`**

Append to `frontend/src/styles/media-grid.module.css`:

```css
/* Image loading shimmer */
.imgWrapper {
  position: relative;
  width: 100%;
  height: 100%;
  background: var(--bg-tertiary);
  animation: pulse 1.5s ease-in-out infinite;
}
.imgWrapper.imgLoaded {
  animation: none;
  background: none;
}
.imgWrapper .img {
  opacity: 0;
  transition: opacity 0.2s;
}
.imgWrapper.imgLoaded .img {
  opacity: 1;
}
```

- [ ] **Step 2: Update `MediaGrid.tsx` to use wrapper with load state**

```tsx
// frontend/src/components/MediaGrid.tsx
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
```

- [ ] **Step 3: Verify image shimmer**

Open a post with images. The image area should show a pulsing grey block, then fade in the image once loaded.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/MediaGrid.tsx frontend/src/styles/media-grid.module.css
git commit -m "feat: add shimmer placeholder for images in MediaGrid"
```

---

## Task 9: Right Panel Skeleton

**Files:**
- Modify: `frontend/src/components/RightPanel.tsx`
- Modify: `frontend/src/styles/layout.module.css`

Add `trendsLoading` and `recommendedLoading` booleans. Show skeleton blocks while loading.

- [ ] **Step 1: Add skeleton CSS to `layout.module.css`**

Append to `frontend/src/styles/layout.module.css`:

```css
/* Right Panel Skeletons */
.skeletonBox {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 16px;
  padding: 16px;
  animation: pulse 1.5s ease-in-out infinite;
}
.skeletonBoxTitle {
  height: 16px;
  width: 50%;
  background: var(--bg-tertiary);
  border-radius: 4px;
  margin-bottom: 14px;
}
.skeletonTrendRow {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 8px 0;
  border-bottom: 1px solid var(--border-light);
}
.skeletonTrendRow:last-child { border-bottom: none; }
.skeletonTrendTag {
  height: 12px;
  width: 40%;
  background: var(--bg-tertiary);
  border-radius: 4px;
}
.skeletonTrendCount {
  height: 10px;
  width: 20%;
  background: var(--bg-tertiary);
  border-radius: 4px;
}
.skeletonRecommendRow {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
}
.skeletonRecommendCircle {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: var(--bg-tertiary);
  flex-shrink: 0;
}
.skeletonRecommendLines {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.skeletonRecommendLine {
  height: 11px;
  background: var(--bg-tertiary);
  border-radius: 4px;
}
```

- [ ] **Step 2: Update `RightPanel.tsx`**

```tsx
// frontend/src/components/RightPanel.tsx
import { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import SearchUsers from './SearchUsers';
import { useAuthStore } from '@/store/auth';
import { useUIStore } from '@/store/ui';
import { getTrendingHashtags } from '@/api/posts';
import { getRecommendedUsers, follow } from '@/api/users';
import Avatar from './Avatar';
import s from '@/styles/layout.module.css';
import type { TrendingHashtag, User } from '@/types';

export default function RightPanel() {
  const user = useAuthStore((st) => st.user);
  const openAuthModal = useUIStore((st) => st.openAuthModal);
  const [trends, setTrends] = useState<TrendingHashtag[]>([]);
  const [recommended, setRecommended] = useState<User[]>([]);
  const [followedIds, setFollowedIds] = useState<Set<string>>(new Set());
  const [trendsLoading, setTrendsLoading] = useState(true);
  const [recommendedLoading, setRecommendedLoading] = useState(true);

  useEffect(() => {
    getTrendingHashtags()
      .then(setTrends)
      .catch(() => {})
      .finally(() => setTrendsLoading(false));
    getRecommendedUsers()
      .then(setRecommended)
      .catch(() => {})
      .finally(() => setRecommendedLoading(false));
  }, []);

  const handleFollow = async (targetId: string) => {
    if (!user) { openAuthModal(); return; }
    try {
      await follow(targetId);
      setFollowedIds((prev) => new Set(prev).add(targetId));
    } catch { /* ignore */ }
  };

  return (
    <aside className={s.rightPanel}>
      <SearchUsers />

      {trendsLoading ? (
        <div className={s.skeletonBox}>
          <div className={s.skeletonBoxTitle} />
          {[1, 2, 3, 4, 5].map((i) => (
            <div key={i} className={s.skeletonTrendRow}>
              <div className={s.skeletonTrendTag} />
              <div className={s.skeletonTrendCount} />
            </div>
          ))}
        </div>
      ) : trends.length > 0 ? (
        <div className={s.infoBox}>
          <h3 className={s.infoBoxTitle}>Тренды</h3>
          {trends.slice(0, 5).map((t) => (
            <Link key={t.tag} to={`/hashtag/${t.tag}`} className={s.trendItem}>
              <span className={s.trendTag}>#{t.tag}</span>
              <span className={s.trendCount}>{t.post_count} постов</span>
            </Link>
          ))}
        </div>
      ) : null}

      {recommendedLoading ? (
        <div className={s.skeletonBox}>
          <div className={s.skeletonBoxTitle} />
          {[1, 2, 3].map((i) => (
            <div key={i} className={s.skeletonRecommendRow}>
              <div className={s.skeletonRecommendCircle} />
              <div className={s.skeletonRecommendLines}>
                <div className={s.skeletonRecommendLine} style={{ width: '60%' }} />
                <div className={s.skeletonRecommendLine} style={{ width: '40%' }} />
              </div>
            </div>
          ))}
        </div>
      ) : recommended.length > 0 ? (
        <div className={s.infoBox}>
          <h3 className={s.infoBoxTitle}>Кого читать</h3>
          {recommended.slice(0, 3).map((u) => (
            <div key={u.id} className={s.recommendItem}>
              <Link to={`/profile/${u.id}`} className={s.recommendUser}>
                <Avatar url={u.avatar_url} name={u.display_name || u.username} size="sm" />
                <div className={s.recommendInfo}>
                  <span className={s.recommendName}>{u.display_name || u.username}</span>
                  <span className={s.recommendHandle}>@{u.username}</span>
                </div>
              </Link>
              {!followedIds.has(u.id) && u.id !== user?.id && (
                <button onClick={() => handleFollow(u.id)} className={s.followBtn}>Читать</button>
              )}
              {followedIds.has(u.id) && (
                <span className={s.followedLabel}>Подписан</span>
              )}
            </div>
          ))}
        </div>
      ) : null}
    </aside>
  );
}
```

- [ ] **Step 3: Verify right panel skeleton**

Open the app on a wide screen (>1024px). The right panel should show pulsing skeleton blocks for trends and recommendations before data loads.

- [ ] **Step 4: Commit**

```bash
git add frontend/src/components/RightPanel.tsx frontend/src/styles/layout.module.css
git commit -m "feat: add skeleton loading for right panel"
```

---

## Task 10: Final Smoke Test

- [ ] **Step 1: Build the frontend to check for TypeScript errors**

```bash
cd frontend && npm run build
```

Expected: build completes with no errors.

- [ ] **Step 2: Check all loading states manually**

| Scenario | Expected |
|---|---|
| App cold start (Docker) | Full skeleton layout for 1-30s |
| Navigate to `/notifications` | 5 skeleton rows then real content |
| Navigate to `/post/:id` | SkeletonPost + 2 comment skeletons |
| Click like/repost/bookmark | Button goes 50% opacity, blocks double-click |
| Submit new post | Submit button shows spinning ring |
| Submit comment | Submit button shows spinning ring |
| Post with images | Grey shimmer block fades to image |
| Right panel (wide screen) | Skeleton trends + recommendations |

- [ ] **Step 3: Commit if any fixes were needed during testing**

```bash
git add -p
git commit -m "fix: loading animation edge cases from smoke test"
```
