# Performance & Bug Fixes Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Fix empty feed on first load, broken follow notifications, slow comment loading, WebSocket reconnect with stale token, and Docker cold-start race conditions.

**Architecture:** Frontend changes optimize initialization order and parallelize fetches. Backend adds follow notification publishing. Docker compose adds healthchecks so api-gateway waits for gRPC services to be ready. WebSocket client gets a token factory to always use a fresh token on reconnect.

**Tech Stack:** React + Zustand (frontend), Go + gorilla/websocket (backend), Docker Compose

---

## File Map

| File | Change |
|------|--------|
| `frontend/src/lib/websocket.ts` | Accept `getToken: () => string` factory instead of static token string |
| `frontend/src/components/Layout.tsx` | Parallel `getUser` + `getFeed` + `getUnreadCount` after refresh; pass token factory to WSClient |
| `frontend/src/store/auth.ts` | `logout()` resets feedStore and notificationStore |
| `frontend/src/components/CommentTree.tsx` | Add optional `initialData` prop; skip initial fetch if provided |
| `frontend/src/pages/PostPage.tsx` | `Promise.all([getPost, getCommentTree])` on mount |
| `backend/api-gateway/internal/handler/user.go` | Add `notifier Notifier` to `UserHandler`; publish follow notification in `Follow()` |
| `backend/api-gateway/cmd/gateway/main.go` | Pass `publisher` to `NewUserHandler` |
| `docker-compose.yml` | TCP healthchecks on Go services; `condition: service_healthy` for api-gateway; `restart: unless-stopped` |

---

## Task 1: WSClient token factory

**Files:**
- Modify: `frontend/src/lib/websocket.ts`

- [ ] **Step 1: Replace `token` field with `getToken` factory**

Open `frontend/src/lib/websocket.ts`. Replace the entire file content:

```typescript
type EventCallback = (data: unknown) => void;

export class WSClient {
  private ws: WebSocket | null = null;
  private listeners = new Map<string, Set<EventCallback>>();
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private reconnectDelay = 1000;
  private maxDelay = 30000;
  private getToken: () => string;
  private disposed = false;
  private onStatusChange?: (connected: boolean, reconnecting: boolean) => void;

  constructor(getToken: () => string, onStatusChange?: (connected: boolean, reconnecting: boolean) => void) {
    this.getToken = getToken;
    this.onStatusChange = onStatusChange;
    this.connect();
  }

  private connect() {
    if (this.disposed) return;
    const token = this.getToken();
    if (!token) {
      this.scheduleReconnect();
      return;
    }
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const host = window.location.host;
    this.ws = new WebSocket(`${protocol}//${host}/api/ws?token=${token}`);

    this.ws.onopen = () => {
      this.reconnectDelay = 1000;
      this.onStatusChange?.(true, false);
    };

    this.ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        const cbs = this.listeners.get(msg.type);
        if (cbs) cbs.forEach((cb) => cb(msg.data));
      } catch { /* ignore */ }
    };

    this.ws.onclose = () => {
      this.onStatusChange?.(false, true);
      this.scheduleReconnect();
    };

    this.ws.onerror = () => { this.ws?.close(); };
  }

  private scheduleReconnect() {
    if (this.disposed) return;
    this.reconnectTimer = setTimeout(() => {
      this.reconnectDelay = Math.min(this.reconnectDelay * 2, this.maxDelay);
      this.connect();
    }, this.reconnectDelay);
  }

  on(event: string, callback: EventCallback) {
    if (!this.listeners.has(event)) this.listeners.set(event, new Set());
    this.listeners.get(event)!.add(callback);
    return () => { this.listeners.get(event)?.delete(callback); };
  }

  disconnect() {
    this.disposed = true;
    if (this.reconnectTimer) clearTimeout(this.reconnectTimer);
    this.ws?.close();
    this.listeners.clear();
    this.onStatusChange?.(false, false);
  }
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/lib/websocket.ts
git commit -m "fix: WSClient accepts token factory to avoid stale token on reconnect"
```

---

## Task 2: Parallel initialization + WSClient wiring in Layout.tsx

**Files:**
- Modify: `frontend/src/components/Layout.tsx`

- [ ] **Step 1: Rewrite Layout.tsx**

Replace the entire file:

```tsx
import { useEffect, useRef } from 'react';
import { Outlet } from 'react-router-dom';
import axios from 'axios';
import { useAuthStore } from '@/store/auth';
import { useWsStore } from '@/store/ws';
import { useFeedStore } from '@/store/feed';
import { useNotificationStore } from '@/store/notifications';
import { getUser } from '@/api/users';
import { getUnreadCount } from '@/api/notifications';
import { getFeed } from '@/api/posts';
import { WSClient } from '@/lib/websocket';
import Sidebar from './Sidebar';
import RightPanel from './RightPanel';
import AuthModal from './AuthModal';
import AppSkeleton from './AppSkeleton';
import s from '@/styles/layout.module.css';
import type { Post, Notification } from '@/types';

export default function Layout() {
  const { isAuthenticated, accessToken, user, login: setUser, logout, setAccessToken, initializing, setInitializing } = useAuthStore();
  const { setConnected, setReconnecting } = useWsStore();
  const { setPosts, addPendingPost } = useFeedStore();
  const { prependNotification, incrementUnread, setUnreadCount } = useNotificationStore();
  const wsRef = useRef<WSClient | null>(null);
  const triedRefresh = useRef(false);

  // Silent refresh on app load — attempt to restore session via httpOnly cookie
  useEffect(() => {
    if (accessToken || triedRefresh.current) {
      setInitializing(false);
      return;
    }
    triedRefresh.current = true;
    axios.post('/api/auth/refresh', {}, { withCredentials: true })
      .then(({ data }) => {
        const newAccess = data.data.access_token;
        setAccessToken(newAccess);
        const payload = JSON.parse(atob(newAccess.split('.')[1]));
        // Run user fetch, feed fetch, and unread count in parallel
        return Promise.all([
          getUser(payload.sub),
          getFeed('').catch(() => null),
          getUnreadCount().catch(() => 0),
        ]).then(([u, feedData, unreadCount]) => {
          setUser(u, newAccess);
          if (feedData) {
            setPosts(
              feedData.posts || [],
              feedData.pagination?.next_cursor || '',
              feedData.pagination?.has_more || false,
            );
          }
          setUnreadCount(unreadCount as number);
        });
      })
      .catch(() => { /* no valid session */ })
      .finally(() => setInitializing(false));
  }, [accessToken, setAccessToken, setUser, setInitializing, setPosts, setUnreadCount]);

  // Fetch user profile when we have a token but no user object
  useEffect(() => {
    if (!isAuthenticated || user || !accessToken) return;
    try {
      const payload = JSON.parse(atob(accessToken.split('.')[1]));
      const userId = payload.sub;
      if (userId) {
        getUser(userId).then((u) => setUser(u, accessToken)).catch(() => logout());
      }
    } catch { logout(); }
  }, [isAuthenticated, user, accessToken, setUser, logout]);

  // WebSocket connection
  useEffect(() => {
    if (!isAuthenticated || !accessToken) return;
    wsRef.current = new WSClient(
      () => useAuthStore.getState().accessToken ?? '',
      (connected, reconnecting) => {
        setConnected(connected);
        setReconnecting(reconnecting);
      }
    );
    const unsubPost = wsRef.current.on('new_post', (data) => addPendingPost(data as Post));
    const unsubNotif = wsRef.current.on('notification', (data) => {
      prependNotification(data as Notification);
      incrementUnread();
    });
    return () => { unsubPost(); unsubNotif(); wsRef.current?.disconnect(); };
  }, [isAuthenticated, accessToken, setConnected, setReconnecting, addPendingPost, prependNotification, incrementUnread]);

  if (initializing) {
    return <AppSkeleton />;
  }

  return (
    <div className={s.shell}>
      <div className={s.shellInner}>
        <div className={s.sidebarCol}><Sidebar /></div>
        <main className={s.mainCol}><Outlet /></main>
        <div className={s.rightCol}><RightPanel /></div>
      </div>
      <AuthModal />
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/Layout.tsx
git commit -m "perf: parallel feed+user+unread fetch on app init, fresh WS token on reconnect"
```

---

## Task 3: Reset stores on logout

**Files:**
- Modify: `frontend/src/store/auth.ts`

- [ ] **Step 1: Add store resets to logout action**

In `frontend/src/store/auth.ts`, add imports at the top and update `logout`:

```typescript
import { create } from 'zustand';
import type { User } from '@/types';

interface AuthState {
  user: User | null;
  accessToken: string | null;
  isAuthenticated: boolean;
  avatarMediaId: string | null;
  bannerMediaId: string | null;
  initializing: boolean;
  login: (user: User, accessToken: string) => void;
  logout: () => void;
  updateUser: (user: Partial<User>) => void;
  setAvatarMediaId: (id: string | null) => void;
  setBannerMediaId: (id: string | null) => void;
  setAccessToken: (accessToken: string) => void;
  setInitializing: (v: boolean) => void;
}

export const useAuthStore = create<AuthState>((set) => ({
  user: null,
  accessToken: null,
  isAuthenticated: false,
  avatarMediaId: null,
  bannerMediaId: null,
  initializing: true,

  login: (user, accessToken) => {
    set({ user, accessToken, isAuthenticated: true });
  },

  logout: () => {
    // Lazy imports to avoid circular dependency
    const { useFeedStore } = require('@/store/feed');
    const { useNotificationStore } = require('@/store/notifications');
    useFeedStore.getState().reset();
    useNotificationStore.getState().reset();
    set({ user: null, accessToken: null, isAuthenticated: false, avatarMediaId: null, bannerMediaId: null });
  },

  updateUser: (partial) =>
    set((state) => ({
      user: state.user ? { ...state.user, ...partial } : null,
    })),

  setAvatarMediaId: (id) => set({ avatarMediaId: id }),
  setBannerMediaId: (id) => set({ bannerMediaId: id }),

  setAccessToken: (accessToken) => {
    set({ accessToken, isAuthenticated: true });
  },

  setInitializing: (v) => set({ initializing: v }),
}));
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/store/auth.ts
git commit -m "fix: reset feed and notification stores on logout to prevent stale data"
```

---

## Task 4: CommentTree initialData prop

**Files:**
- Modify: `frontend/src/components/CommentTree.tsx`

- [ ] **Step 1: Add initialData prop and skip initial fetch when provided**

Replace the entire `frontend/src/components/CommentTree.tsx`:

```tsx
import { useState, useEffect, useCallback } from 'react';
import { getCommentTree } from '@/api/comments';
import { useAuthStore } from '@/store/auth';
import CommentItem from './CommentItem';
import CommentForm from './CommentForm';
import Spinner from './Spinner';
import s from '@/styles/profile.module.css';
import type { Comment } from '@/types';

interface InitialData {
  comments: Comment[];
  cursor: string;
  hasMore: boolean;
}

function addReplyToTree(comments: Comment[], parentId: string, reply: Comment): Comment[] {
  return comments.map((c) => {
    if (c.id === parentId) return { ...c, children: [reply, ...(c.children || [])] };
    if (c.children?.length) return { ...c, children: addReplyToTree(c.children, parentId, reply) };
    return c;
  });
}

function removeFromTree(comments: Comment[], commentId: string): Comment[] {
  return comments.map((c) => {
    if (c.id === commentId) {
      if (c.children?.length) return { ...c, content: '[удалено]', user_id: '' };
      return null;
    }
    if (c.children?.length) return { ...c, children: removeFromTree(c.children, commentId) };
    return c;
  }).filter(Boolean) as Comment[];
}

export default function CommentTree({ postId, initialData }: { postId: string; initialData?: InitialData }) {
  const user = useAuthStore((st) => st.user);
  const [comments, setComments] = useState<Comment[]>(initialData?.comments ?? []);
  const [cursor, setCursor] = useState(initialData?.cursor ?? '');
  const [hasMore, setHasMore] = useState(initialData?.hasMore ?? false);
  const [loading, setLoading] = useState(!initialData);

  const load = useCallback(async (c = '') => {
    setLoading(true);
    try {
      const data = await getCommentTree(postId, c);
      if (c) setComments((prev) => [...prev, ...(data.comments || [])]);
      else setComments(data.comments || []);
      setCursor(data.pagination?.next_cursor || '');
      setHasMore(data.pagination?.has_more || false);
    } catch { /* ignore */ }
    finally { setLoading(false); }
  }, [postId]);

  useEffect(() => {
    if (!initialData) load();
  }, [load, initialData]);

  return (
    <div>
      {user && <CommentForm postId={postId} onSubmit={(c) => setComments((prev) => [c, ...prev])} />}
      <div style={{ padding: '0 16px' }}>
        {comments.map((c) => (
          <CommentItem
            key={c.id}
            comment={c}
            postId={postId}
            onNewReply={(pid, reply) => setComments((prev) => addReplyToTree(prev, pid, reply))}
            onDelete={(cid) => setComments((prev) => removeFromTree(prev, cid))}
          />
        ))}
      </div>
      {loading && <Spinner />}
      {hasMore && !loading && (
        <button onClick={() => load(cursor)} className={s.loadMoreBtn}>Загрузить ещё</button>
      )}
      {!loading && comments.length === 0 && <p className={s.empty}>Нет комментариев</p>}
    </div>
  );
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/components/CommentTree.tsx
git commit -m "feat: CommentTree accepts initialData prop to skip redundant fetch"
```

---

## Task 5: Parallel post + comments fetch in PostPage

**Files:**
- Modify: `frontend/src/pages/PostPage.tsx`

- [ ] **Step 1: Replace sequential fetches with Promise.all**

In `frontend/src/pages/PostPage.tsx`, replace the import block and state/effect section. Full file:

```tsx
import { useEffect, useState } from 'react';
import { useParams, useNavigate, Link } from 'react-router-dom';
import { ArrowLeft, Heart, MessageCircle, Trash2, Eye } from 'lucide-react';
import { getPost, likePost, unlikePost, deletePost, recordViews } from '@/api/posts';
import { getCommentTree } from '@/api/comments';
import { useAuthStore } from '@/store/auth';
import Avatar from '@/components/Avatar';
import MediaGrid from '@/components/MediaGrid';
import CommentTree from '@/components/CommentTree';
import SkeletonPost from '@/components/SkeletonPost';
import ImageLightbox from '@/components/ImageLightbox';
import l from '@/styles/layout.module.css';
import s from '@/styles/post.module.css';
import c from '@/styles/components.module.css';
import type { Post, Comment } from '@/types';

interface CommentInitialData {
  comments: Comment[];
  cursor: string;
  hasMore: boolean;
}

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
  const [commentInitialData, setCommentInitialData] = useState<CommentInitialData | undefined>(undefined);
  const [loading, setLoading] = useState(true);
  const [likeLoading, setLikeLoading] = useState(false);
  const [lightboxSrc, setLightboxSrc] = useState<string | null>(null);

  useEffect(() => {
    if (!id) return;
    setLoading(true);
    Promise.all([
      getPost(id),
      getCommentTree(id, '').catch(() => null),
    ]).then(([p, commentData]) => {
      setPost(p);
      if (commentData) {
        setCommentInitialData({
          comments: commentData.comments || [],
          cursor: commentData.pagination?.next_cursor || '',
          hasMore: commentData.pagination?.has_more || false,
        });
      }
    }).catch(() => setPost(null)).finally(() => setLoading(false));
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
            disabled={likeLoading}
            className={`${s.postDetailActionBtn} ${s.postDetailLikeBtn} ${post.liked ? s.postDetailLiked : ''}`}
            style={likeLoading ? { opacity: 0.5 } : undefined}
          >
            <Heart size={20} fill={post.liked ? 'currentColor' : 'none'} />
          </button>
        </div>
      </article>

      <CommentTree postId={post.id} initialData={commentInitialData} />
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

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/PostPage.tsx
git commit -m "perf: fetch post and comments in parallel on PostPage"
```

---

## Task 6: Follow notification in UserHandler (Go)

**Files:**
- Modify: `backend/api-gateway/internal/handler/user.go`
- Modify: `backend/api-gateway/cmd/gateway/main.go`

- [ ] **Step 1: Add notifier to UserHandler and publish follow notification**

Replace the top of `backend/api-gateway/internal/handler/user.go` (struct + constructor + Follow method). The rest of the file stays unchanged.

Change the import block to add `"context"`:

```go
import (
	"context"
	"encoding/json"
	"net/http"

	commonpb "github.com/usedcvnt/microtwitter/gen/go/common/v1"
	postpb "github.com/usedcvnt/microtwitter/gen/go/post/v1"
	userpb "github.com/usedcvnt/microtwitter/gen/go/user/v1"
)
```

Replace the struct and constructor:

```go
type UserHandler struct {
	user     userpb.UserServiceClient
	post     postpb.PostServiceClient
	notifier Notifier
}

func NewUserHandler(user userpb.UserServiceClient, post postpb.PostServiceClient, notifier Notifier) *UserHandler {
	return &UserHandler{user: user, post: post, notifier: notifier}
}
```

Replace the `Follow` method:

```go
func (h *UserHandler) Follow(w http.ResponseWriter, r *http.Request) {
	targetID := r.PathValue("id")
	userID := getUserID(r)

	_, err := h.user.Follow(r.Context(), &userpb.FollowRequest{
		FollowerId:  userID,
		FollowingId: targetID,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	if h.notifier != nil {
		go func() {
			nr, err := h.user.CreateNotification(context.Background(), &userpb.CreateNotificationRequest{
				UserId:  targetID,
				ActorId: userID,
				Type:    "follow",
			})
			if err == nil {
				h.notifier.PublishNotification(context.Background(), targetID, notificationToMap(nr.GetNotification()))
			}
		}()
	}

	writeJSON(w, http.StatusOK, nil)
}
```

- [ ] **Step 2: Wire publisher in main.go**

In `backend/api-gateway/cmd/gateway/main.go`, find the line:

```go
userH := handler.NewUserHandler(clients.User, clients.Post)
```

Replace it with:

```go
userH := handler.NewUserHandler(clients.User, clients.Post, publisher)
```

- [ ] **Step 3: Build to verify it compiles**

```bash
cd backend/api-gateway && go build ./...
```

Expected: no errors.

- [ ] **Step 4: Commit**

```bash
git add backend/api-gateway/internal/handler/user.go backend/api-gateway/cmd/gateway/main.go
git commit -m "feat: publish follow notification via WebSocket"
```

---

## Task 7: Docker healthchecks and startup order

**Files:**
- Modify: `docker-compose.yml`

- [ ] **Step 1: Add healthchecks and restart policies**

Replace the entire `docker-compose.yml`:

```yaml
services:
  postgres:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: ${POSTGRES_USER}
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
      POSTGRES_DB: ${POSTGRES_DB}
    volumes:
      - pgdata:/var/lib/postgresql/data
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${POSTGRES_USER} -d ${POSTGRES_DB}"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - backend
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes
    volumes:
      - redisdata:/data
    ports:
      - "6379:6379"
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - backend
    restart: unless-stopped

  minio:
    image: minio/minio:latest
    command: server /data --console-address ":9001"
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
    volumes:
      - miniodata:/data
    ports:
      - "9000:9000"
      - "9001:9001"
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:9000/minio/health/live"]
      interval: 5s
      timeout: 3s
      retries: 5
    networks:
      - backend
    restart: unless-stopped

  minio-init:
    image: minio/mc:latest
    depends_on:
      minio:
        condition: service_healthy
    entrypoint: >
      /bin/sh -c "
      mc alias set local http://minio:9000 $$MINIO_ROOT_USER $$MINIO_ROOT_PASSWORD;
      mc mb --ignore-existing local/$$MINIO_BUCKET;
      mc anonymous set download local/$$MINIO_BUCKET;
      exit 0;
      "
    environment:
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
      MINIO_BUCKET: ${MINIO_BUCKET}
    networks:
      - backend

  migrate:
    image: migrate/migrate:v4.17.0
    depends_on:
      postgres:
        condition: service_healthy
    volumes:
      - ./backend/migrations:/migrations
    command:
      [
        "-path", "/migrations",
        "-database", "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable",
        "up"
      ]
    networks:
      - backend

  user-svc:
    build:
      context: ./backend
      dockerfile: services/user-svc/Dockerfile
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    environment:
      DB_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      REDIS_URL: ${REDIS_URL}
      GRPC_PORT: "50051"
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "50051:50051"
    healthcheck:
      test: ["CMD-SHELL", "nc -z localhost 50051 || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    networks:
      - backend
    restart: unless-stopped

  post-svc:
    build:
      context: ./backend
      dockerfile: services/post-svc/Dockerfile
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    environment:
      DB_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      REDIS_URL: ${REDIS_URL}
      GRPC_PORT: "50052"
    ports:
      - "50052:50052"
    healthcheck:
      test: ["CMD-SHELL", "nc -z localhost 50052 || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    networks:
      - backend
    restart: unless-stopped

  comment-svc:
    build:
      context: ./backend
      dockerfile: services/comment-svc/Dockerfile
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
      migrate:
        condition: service_completed_successfully
    environment:
      DB_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      REDIS_URL: ${REDIS_URL}
      GRPC_PORT: "50053"
    ports:
      - "50053:50053"
    healthcheck:
      test: ["CMD-SHELL", "nc -z localhost 50053 || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    networks:
      - backend
    restart: unless-stopped

  media-svc:
    build:
      context: ./backend
      dockerfile: services/media-svc/Dockerfile
    depends_on:
      postgres:
        condition: service_healthy
      minio:
        condition: service_healthy
      minio-init:
        condition: service_completed_successfully
      migrate:
        condition: service_completed_successfully
    environment:
      DB_URL: postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable
      MINIO_ENDPOINT: minio:9000
      MINIO_PUBLIC_ENDPOINT: ${MINIO_PUBLIC_ENDPOINT:-http://localhost:9000}
      MINIO_ROOT_USER: ${MINIO_ROOT_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_ROOT_PASSWORD}
      MINIO_BUCKET: ${MINIO_BUCKET}
      GRPC_PORT: "50054"
    ports:
      - "50054:50054"
    healthcheck:
      test: ["CMD-SHELL", "nc -z localhost 50054 || exit 1"]
      interval: 5s
      timeout: 3s
      retries: 10
      start_period: 10s
    networks:
      - backend
    restart: unless-stopped

  api-gateway:
    build:
      context: ./backend
      dockerfile: api-gateway/Dockerfile
    depends_on:
      user-svc:
        condition: service_healthy
      post-svc:
        condition: service_healthy
      comment-svc:
        condition: service_healthy
      media-svc:
        condition: service_healthy
      redis:
        condition: service_healthy
    environment:
      PORT: "8080"
      USER_SVC_ADDR: user-svc:50051
      POST_SVC_ADDR: post-svc:50052
      COMMENT_SVC_ADDR: comment-svc:50053
      MEDIA_SVC_ADDR: media-svc:50054
      REDIS_URL: ${REDIS_URL}
      JWT_SECRET: ${JWT_SECRET}
    ports:
      - "8080:8080"
    networks:
      - backend
    restart: unless-stopped

  frontend:
    build:
      context: ./frontend
    depends_on:
      - api-gateway
    ports:
      - "3000:80"
    networks:
      - backend
    restart: unless-stopped

networks:
  backend:
    driver: bridge

volumes:
  pgdata:
  redisdata:
  miniodata:
```

Note: `nc` (netcat) is available in `alpine:3.20` by default via `busybox`.

- [ ] **Step 2: Commit**

```bash
git add docker-compose.yml
git commit -m "fix: add gRPC healthchecks and service_healthy deps to prevent cold-start race"
```

---

## Self-Review

**Spec coverage:**
- ✅ Change 1 (parallel init) → Task 2
- ✅ Change 2 (store reset on logout) → Task 3
- ✅ Change 3 (follow notification) → Task 6
- ✅ Change 4 (parallel post+comments) → Task 5
- ✅ Change 5 (CommentTree initialData) → Task 4
- ✅ Change 6 (WS token factory) → Task 1
- ✅ Change 7 (Docker healthchecks) → Task 7

**Type consistency:**
- `InitialData` interface defined in Task 4 (CommentTree.tsx) and referenced by the same name in Task 5 (PostPage.tsx uses local `CommentInitialData` — compatible shape, no cross-import needed)
- `Notifier` interface already defined in `handler/post.go` — reused in Task 6 without redefinition
- `WSClient` constructor signature changed in Task 1, consumed in Task 2 — consistent

**No placeholders:** All steps contain complete code.
