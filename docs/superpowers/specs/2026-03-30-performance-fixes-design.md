# Performance & Bug Fixes Design

**Date:** 2026-03-30

## Problem Summary

Three user-visible issues after Docker cold start:

1. Feed shows empty on first load — requires manual refresh to see posts
2. Notifications do not work for follow events
3. Comments load slowly on post pages

## Root Causes

### 1. Sequential initialization chain (Layout.tsx)
On app load: `/auth/refresh` → `getUser` → render Home → `getFeed`. Three sequential HTTP requests before posts appear. Feed only loads after `Home` mounts, which only happens after auth is restored.

### 2. Feed store not reset on logout
`posts.length > 0` after logout means `Home` skips `loadFeed()` on next login. If the same user logs back in, they see stale data. If a different user logs in, they may see wrong data briefly.

### 3. Follow action missing notification (user.go)
`UserHandler.Follow()` calls `h.user.Follow()` but never calls `CreateNotification` or `PublishNotification`. The `UserHandler` struct has no `notifier` field at all. Contrast with `PostHandler.LikePost` which correctly creates and publishes a notification via goroutine.

### 4. Sequential post + comments fetch (PostPage.tsx)
`PostPage` renders `CommentTree` only after `getPost` resolves (`loading=false`). `CommentTree` then starts its own fetch. Two sequential requests that could be parallel.

## Solution: Variant B — Optimized Initialization + Targeted Fixes

### Change 1: Parallel initialization in Layout.tsx

After `/auth/refresh` returns a token, run `getUser`, `getFeed`, and `getUnreadCount` in parallel via `Promise.all`. Store feed results directly into `feedStore` before `Home` mounts. `Home` checks `posts.length > 0` and skips its own fetch if data is already present.

```
Before: refresh → getUser → render Home → getFeed   (3 sequential)
After:  refresh → getUser + getFeed + getUnreadCount (parallel)
```

### Change 2: Reset stores on logout (store/auth.ts)

`logout()` action calls `useFeedStore.getState().reset()` and `useNotificationStore.getState().reset()` to clear stale data between sessions.

### Change 3: Follow notification (backend/api-gateway)

- Add `notifier Notifier` field to `UserHandler` struct
- Update `NewUserHandler` constructor to accept `notifier Notifier`
- In `Follow()`: after successful gRPC call, launch goroutine that calls `CreateNotification(type="follow")` then `PublishNotification`
- Wire up in `cmd/gateway/main.go`: pass `publisher` to `NewUserHandler`

### Change 4: Parallel post + comments fetch (PostPage.tsx)

Run `getPost(id)` and `getCommentTree(id)` simultaneously via `Promise.all`. Pass the initial comments data to `CommentTree` as prop `initialData`.

### Change 5: CommentTree accepts initialData prop

Add optional `initialData?: { comments: Comment[]; cursor: string; hasMore: boolean }` prop. If provided, initialize state from it and skip the initial `load()` call in `useEffect`.

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/Layout.tsx` | Parallel init after refresh |
| `frontend/src/store/auth.ts` | Reset feed + notification stores on logout |
| `frontend/src/pages/PostPage.tsx` | Promise.all for post + comments |
| `frontend/src/components/CommentTree.tsx` | Accept initialData prop |
| `backend/api-gateway/internal/handler/user.go` | Add notifier, publish follow notification |
| `backend/api-gateway/cmd/gateway/main.go` | Pass publisher to NewUserHandler |

### Change 6: WebSocket token refresh (websocket.ts)

When `WSClient` reconnects after a disconnect, it reuses the token passed at construction time. If the access token expired, the server returns 401 and the client loops forever reconnecting with a bad token.

Fix: accept a `getToken: () => string` factory function instead of a plain `token: string`. On each `connect()` call, invoke `getToken()` to get the current token from the auth store. In `Layout.tsx`, pass `() => useAuthStore.getState().accessToken ?? ''` as the factory.

### Change 7: Docker startup order (docker-compose.yml)

`api-gateway` depends on `user-svc`, `post-svc`, `comment-svc`, `media-svc` with no `condition` — Docker starts api-gateway as soon as containers start, not when gRPC is ready. This causes connection errors during cold start.

Fix:
- Add TCP healthchecks to all four Go services (no binary changes needed — `nc` or `/dev/tcp` check on their gRPC port)
- Change `api-gateway` depends_on to use `condition: service_healthy` for all four services
- Add `restart: unless-stopped` to all application services (user-svc, post-svc, comment-svc, media-svc, api-gateway, frontend)

## Files Changed

| File | Change |
|------|--------|
| `frontend/src/components/Layout.tsx` | Parallel init after refresh; pass token factory to WSClient |
| `frontend/src/store/auth.ts` | Reset feed + notification stores on logout |
| `frontend/src/pages/PostPage.tsx` | Promise.all for post + comments |
| `frontend/src/components/CommentTree.tsx` | Accept initialData prop |
| `frontend/src/lib/websocket.ts` | Accept `getToken` factory instead of static token |
| `backend/api-gateway/internal/handler/user.go` | Add notifier, publish follow notification |
| `backend/api-gateway/cmd/gateway/main.go` | Pass publisher to NewUserHandler |
| `docker-compose.yml` | Healthchecks for Go services, service_healthy conditions, restart policies |
