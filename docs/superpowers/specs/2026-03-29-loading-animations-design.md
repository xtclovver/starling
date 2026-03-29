# Loading Animations Design

**Date:** 2026-03-29
**Status:** Approved

## Problem

After Docker restart the backend takes 10–30 seconds to come up. During that time the frontend shows a blank black screen with no feedback. Additionally, several pages and interactive elements lack loading states, making the UI feel unresponsive.

## Scope

Five areas of improvement, in priority order:

1. App startup skeleton
2. Page-level skeletons (missing pages)
3. Button action states
4. Post/comment submission
5. Image loading placeholders

---

## 1. App Startup Skeleton

**Where:** `App.tsx` — during the initial auth check (`useEffect` that calls the backend to verify session).

**What:** Full-layout skeleton matching the real UI structure:
- Sidebar skeleton (logo placeholder + 4 icon circles)
- Main content area with 4 `SkeletonPost` components, shimmer animation

**How:** Add an `appLoading` state to `useAuthStore` (or local state in `App.tsx`). Set it `true` on mount, `false` when the auth check resolves (success or failure). While `appLoading === true`, render `<AppSkeleton />` instead of `<Routes>`.

**Component:** New `AppSkeleton.tsx` + `AppSkeleton.module.css`. Reuses the existing `shimmer` keyframe already defined in `SkeletonPost.module.css` — extract it to a shared `_animations.css` or duplicate locally.

---

## 2. Page-Level Skeletons (Missing Pages)

Existing skeleton coverage: Home ✓, Profile ✓, Bookmarks ✓
Missing: PostPage, HashtagPage, Notifications, Settings

**PostPage:** Show a single large `SkeletonPost` for the main post + 2 smaller skeleton comment rows while `loading === true`.

**HashtagPage:** Same pattern as Home — 4 `SkeletonPost` items.

**Notifications:** New `SkeletonNotification` component — a row with circle avatar placeholder + two lines of text shimmer.

**Settings:** Simple skeleton — 3 section headers + input-shaped blocks.

All follow the existing pattern: `if (loading && items.length === 0) return <>{skeletons}</>`.

---

## 3. Button Action States

Affected interactions: like, repost, bookmark, follow/unfollow.

**Pattern:** These already use optimistic updates (state flips immediately). Add a visual disabled state during the in-flight request to prevent double-clicks and signal activity.

**Implementation:** Each action button gets a local `pending` boolean. While `pending === true`:
- Button is `disabled`
- Icon replaced with a small inline spinner (16px, same color as icon) OR icon gets `opacity: 0.5` + CSS pulse animation

Prefer the opacity+pulse approach — simpler, no DOM changes, consistent with the dark theme.

**Revert on error:** Already implemented in PostPage for likes. Apply the same pattern to all action handlers.

---

## 4. Post/Comment Submission

**CreatePost component:** The submit button shows a spinner and becomes `disabled` from the moment the user clicks until the API call resolves. On success, the new post is prepended to the feed. On error, button re-enables (silent failure stays, consistent with current error handling approach).

**Comment submission:** Same — submit button disabled + spinner while in flight.

No changes to optimistic feed insertion logic (posts already appear immediately in some places — keep that behavior).

---

## 5. Image Loading Placeholders

**Where:** Post images rendered via `<img>` tags inside post cards.

**What:** Shimmer placeholder shown until `onLoad` fires. On `onError`, show a neutral dark rectangle (no broken-image icon).

**How:** Wrap each `<img>` in a container that has the shimmer background by default. Add `onLoad` handler that adds a CSS class (`loaded`) to hide the shimmer. Use `object-fit: cover` so image fills the container without layout shift.

No new component needed — inline state (`imgLoaded`) per image instance is sufficient.

---

## Architecture Notes

- **Shared shimmer:** The `@keyframes shimmer` animation is currently duplicated in `SkeletonPost.module.css`. Extract to `src/styles/animations.css` (global, not a module) and import once in `main.tsx`. All skeleton components reference the class name directly.
- **No new state management:** All loading states are local (`useState`) or already in existing Zustand stores. No new stores needed.
- **No new dependencies:** Pure CSS animations + React state. No animation libraries.

---

## 6. Right Panel Skeleton

**Where:** `RightPanel.tsx` — loads trending hashtags and recommended users via two API calls on mount. Currently renders empty while data is in flight.

**What:** Skeleton placeholders for both sections while `trends.length === 0` and `recommended.length === 0`:
- Search bar: already visible (no skeleton needed)
- Trends block: heading placeholder + 5 tag-row skeletons (tag name + count)
- Recommended block: heading placeholder + 3 user-row skeletons (avatar circle + two lines)

**How:** Add `trendsLoading` and `recommendedLoading` local booleans. Show skeleton blocks while loading, replace with real content on resolve. If data comes back empty (no trends/no recommendations), render nothing — same as current behavior.

**Component:** New `SkeletonRightPanel.tsx` (or inline JSX in `RightPanel.tsx` — prefer inline since it's self-contained).

---

## Out of Scope

- Toast/snackbar error notifications (separate feature)
- Page transition animations (route-level)
