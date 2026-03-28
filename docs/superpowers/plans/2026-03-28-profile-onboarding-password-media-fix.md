# Profile Onboarding, Password Change, Settings Label & Media Edit Fix

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add post-registration onboarding wizard, avatar label in settings, password change with session invalidation, and fix media not updating after post edit.

**Architecture:** 4 independent changes: (1) New `OnboardingWizard` React component shown after registration, (2) "Аватар" label added to Settings.tsx, (3) Full-stack password change: proto → gRPC → API gateway → frontend in Security tab, (4) Bug fix in PostCard.tsx to propagate media_url from edit response.

**Tech Stack:** React/TypeScript (frontend), Go/gRPC/protobuf (backend), PostgreSQL, Redis, bcrypt

---

## File Structure

### New Files
- `frontend/src/components/OnboardingWizard.tsx` — Multi-step modal wizard (display name → bio → avatar → banner)

### Modified Files
- `frontend/src/pages/Register.tsx` — Show OnboardingWizard after successful registration
- `frontend/src/pages/Settings.tsx` — Add "Аватар" label, add password change form in Security section
- `frontend/src/styles/profile.module.css` — Styles for password change form, avatar label
- `frontend/src/styles/modal.module.css` — Styles for onboarding wizard
- `frontend/src/api/auth.ts` — Add `changePassword()` API call
- `frontend/src/components/PostCard.tsx` — Fix media_url propagation after edit
- `backend/proto/user/v1/user.proto` — Add ChangePassword RPC + messages
- `backend/services/user-svc/internal/grpc/server.go` — Implement ChangePassword method
- `backend/api-gateway/internal/handler/auth.go` — Add ChangePassword HTTP handler
- `backend/api-gateway/cmd/gateway/main.go` — Register new route

---

## Task 1: Fix media not updating after post edit (bug fix)

**Files:**
- Modify: `frontend/src/components/PostCard.tsx:149-166`

The bug: when saving an edited post with new media, `saveEdit` calls `updatePost()` which returns the updated post. The `media_url` from the server response is correctly used in `updateFeedPost`, but if the user added media to a post that previously had none, the `editMediaUrl` state is cleared (set to `''` on line 137) when a new file is selected. The `updatePost(post.id, editContent, mediaUrl)` call on line 159 passes the uploaded URL correctly. The feed store update on line 160 uses `updated.media_url` — this should work.

The actual bug is that `updatePost` in `api/posts.ts` sends `media_url` as empty string when no file is selected AND the original media was kept. Let me re-examine... Actually, the real issue is on line 137: `setEditMediaUrl('')` clears the existing URL when a new file is selected, but the `mediaUrl` local variable on line 154 starts as `editMediaUrl` which is already `''`. After upload, it gets the new URL. This seems correct.

Looking more carefully: the issue is that `updateFeedPost` on line 160 correctly passes `media_url: updated.media_url`, but the response from the API might not include the media_url properly. Actually, looking at the backend `UpdatePost` in post-svc — when `media_url` is provided, it updates it; when not, it keeps the old one. The frontend `updatePost` in `api/posts.ts` line 47 always sends `media_url` param. If `mediaUrl` is empty string, the backend will keep the old one. This is correct behavior.

The actual bug is simpler: when editing and adding media, the `editMediaPreview` (blob URL) is shown during edit. After save, the preview is revoked (line 163), but the post in the feed store now has the correct `media_url` from the server. The issue is that `updated.media_url` from the response might be missing because the API gateway's `postToMap` might not be returning it properly after update...

Actually, re-reading the user's bug report: "Если средактировать пост и добавить туда медиа, то оно появится только после обновления страницы." This means the media DOES get saved (visible after refresh), but doesn't appear immediately. The `updateFeedPost` call uses `updated.media_url` from the server response. If this value is correct but the UI doesn't update, it could be that the post card re-renders with the old `post` prop before the store update propagates.

Looking at line 160: `updateFeedPost(post.id, { content: updated.content, media_url: updated.media_url, edited_at: updated.edited_at || new Date().toISOString() });`

This uses `updated.media_url` from the API response. If the API returns the correct URL, the feed store should update the post, and the component should re-render. The post card uses `post.media_url` from props — but props come from the parent which reads from the feed store. This should work.

Wait — the `PostCard` component receives `post` as a prop. If the parent component (e.g., `Home`, `Profile`, `PostPage`) uses its own local state for posts rather than the feed store, then updating the feed store won't update the local state. Let me check if PostPage uses its own state.

Actually, the most likely cause: the `PostCard` receives the `post` object from the feed store, and `updateFeedPost` updates the store. But the `editing` state hides the media display (line 211-253 shows the edit UI when `editing` is true). When `setEditing(false)` is called on line 161, the component should re-render and show the non-editing view with `post.media_url`. Since `updateFeedPost` was called before `setEditing(false)`, the store should already have the updated `media_url`.

But there's a subtle timing issue: `updateFeedPost` and `setEditing(false)` are in the same synchronous block after await, so React batches these. The `post` prop from the store and the local `editing` state should both update in the same render.

The most probable cause is that `PostPage` (single post view) fetches the post independently and doesn't use the feed store. Let me check the parent rendering flow more carefully. The key insight: if the post is rendered from a parent's local state (not the feed store), then `updateFeedPost` updates the wrong store.

The fix should be: return the full updated post from `saveEdit` and also update the local post state if the parent provides a callback, OR simply ensure `updateFeedPost` is sufficient by making all parents read from the feed store.

Actually, the simplest and most correct fix: the `saveEdit` function should update the local rendering as well. Looking at line 160, `updateFeedPost(post.id, {...})` updates the feed store. If the `PostCard`'s parent re-renders from feed store, this works. But if the parent uses its own state (like PostPage), the card needs to be re-rendered with new post data.

The real fix: the `post` prop passed to PostCard may come from a parent that doesn't use the feed store. We need to ensure the update propagates regardless. The simplest approach: after successful edit, call `updateFeedPost` as before (for feed pages), and the PostCard should work since it references `post.media_url` which comes from the parent. Since `updateFeedPost` only updates the feed store, pages that don't use feed store (PostPage, Profile) won't reflect changes.

**The fix:** After the `updateFeedPost` call, also update the parent's state. Since the `PostCard` receives `post` as a prop and doesn't control it, the simplest fix is to not rely solely on the feed store. We should update the feed store AND ensure PostCard re-renders properly. The issue is likely specific to non-feed contexts.

After all this analysis, the cleanest fix is: `saveEdit` should pass the updated post fields back in a way that the PostCard shows immediately. Since `updateFeedPost` updates the store, and most list views use the feed store, the remaining case is PostPage. We can simply ensure that `updateFeedPost` is called (which it is), and for PostPage we can see if it subscribes to the feed store.

**Simplest fix identified:** The `PostCard` component uses `post.media_url` (from props) for the non-editing view. After save, the feed store is updated, but the prop comes from the parent. For Feed/Home pages this works because they read from the feed store. For PostPage/Profile that use local state, the update doesn't propagate.

**The actual simplest fix:** After the edit completes successfully, update the `post` reference that PostCard uses. Since PostCard gets `post` from props, we can't change that inside PostCard. But we can check: does PostCard re-read from the feed store? Looking at line 34: `const updateFeedPost = useFeedStore((st) => st.updatePost);` — it only grabs the updater, not the post itself.

**Final answer: the fix is to also update the local edit states so the non-editing view renders correctly.** When editing is false, PostCard renders `post.media_url` from props. If props don't change (parent doesn't use feed store), the old media_url is shown. The simplest and correct fix: we need to make PostCard aware of the updated media_url even if props don't change. We can do this by tracking `overrideMediaUrl` in local state, or by ensuring all parents re-render.

**Chosen approach:** Use a local state override for the post data after edit. Add a `localPost` state that merges with props and is used for rendering, updated on successful edit.

- [ ] **Step 1: Add local post override state to PostCard**

In `frontend/src/components/PostCard.tsx`, add a `localOverrides` state and merge with props for rendering:

```tsx
// After line 48 (const editFileRef = ...)
const [localOverrides, setLocalOverrides] = useState<Partial<Post>>({});
const displayPost = { ...post, ...localOverrides };
```

Then replace all rendering references from `post.media_url`, `post.content`, `post.edited_at` to use `displayPost` instead. Specifically in the non-editing view (lines 256-280).

- [ ] **Step 2: Update saveEdit to set local overrides**

In `saveEdit` function, after the `updateFeedPost` call (line 160), add:

```tsx
setLocalOverrides({ content: updated.content, media_url: updated.media_url, edited_at: updated.edited_at || new Date().toISOString() });
```

- [ ] **Step 3: Reset overrides when props change**

Add a useEffect to reset overrides when the post prop changes (e.g., when navigating):

```tsx
useEffect(() => {
  setLocalOverrides({});
}, [post.id, post.updated_at]);
```

- [ ] **Step 4: Update rendering to use displayPost**

Replace `post.content`, `post.media_url`, `post.edited_at` references in the non-editing rendering section with `displayPost.content`, `displayPost.media_url`, `displayPost.edited_at`.

Lines to change in the non-editing view:
- Line 185: `{post.edited_at &&` → `{displayPost.edited_at &&`
- Line 256: `<p className={s.postContent}>{renderContent(post.content)}</p>` → `<p className={s.postContent}>{renderContent(displayPost.content)}</p>`
- Line 257: `{post.media_url && (() => {` → `{displayPost.media_url && (() => {`
- Line 258: `const kind = getMediaKind(post.media_url);` → `const kind = getMediaKind(displayPost.media_url);`
- Lines 261, 266, 271, 275: all `post.media_url` → `displayPost.media_url`

- [ ] **Step 5: Commit**

```bash
git add frontend/src/components/PostCard.tsx
git commit -m "fix: media not updating after post edit without page refresh"
```

---

## Task 2: Add "Аватар" label to Settings page

**Files:**
- Modify: `frontend/src/pages/Settings.tsx:138-149`

- [ ] **Step 1: Add label above avatar section**

In `frontend/src/pages/Settings.tsx`, change the avatar section (lines 138-149):

```tsx
{/* Avatar */}
<div>
  <label className={s.fieldLabel}>Аватар</label>
  <div className={s.avatarEdit}>
    <div className={s.avatarEditBtn} onClick={() => avatarFileRef.current?.click()}>
      <Avatar url={avatarPreview || user.avatar_url} name={displayName || user.username} size="xl" />
      <div className={s.avatarOverlay}><Camera size={20} color="white" /></div>
    </div>
    <input ref={avatarFileRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleAvatar} style={{ display: 'none' }} />
    <div>
      <p className={s.avatarHint}>Нажмите на аватар</p>
      <p className={s.avatarHintSub}>JPEG, PNG, GIF, WebP. До 10 МБ</p>
    </div>
  </div>
</div>
```

The change: wrap the avatar block in a `<div>` with a `<label className={s.fieldLabel}>Аватар</label>` before `avatarEdit` div, matching the pattern of "Фон профиля", "Отображаемое имя", "О себе".

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/Settings.tsx
git commit -m "feat: add 'Аватар' label to settings page"
```

---

## Task 3: Backend — ChangePassword proto + gRPC implementation

**Files:**
- Modify: `backend/proto/user/v1/user.proto`
- Modify: `backend/services/user-svc/internal/grpc/server.go`

- [ ] **Step 1: Add ChangePassword messages and RPC to proto**

In `backend/proto/user/v1/user.proto`, add before the closing of the service block:

After `MarkAllReadResponse` (line 218), add:

```protobuf
message ChangePasswordRequest {
  string user_id = 1;
  string current_password = 2;
  string new_password = 3;
  string access_token = 4;
}

message ChangePasswordResponse {}
```

In the `service UserService` block, add after `MarkAllRead` RPC:

```protobuf
  rpc ChangePassword(ChangePasswordRequest) returns (ChangePasswordResponse);
```

- [ ] **Step 2: Regenerate protobuf Go code**

Run from the project root:

```bash
cd backend && make proto
```

Or if no Makefile, run the protoc command directly. Check how proto generation works in this project first.

- [ ] **Step 3: Implement ChangePassword in user-svc gRPC server**

In `backend/services/user-svc/internal/grpc/server.go`, add the `ChangePassword` method:

```go
func (s *Server) ChangePassword(ctx context.Context, req *pb.ChangePasswordRequest) (*pb.ChangePasswordResponse, error) {
	start := time.Now()
	defer func() { s.log.Info("ChangePassword", "duration", time.Since(start)) }()

	if len(req.GetNewPassword()) < 8 {
		return nil, status.Error(codes.InvalidArgument, "new password must be at least 8 characters")
	}

	user, err := s.userRepo.GetByID(ctx, req.GetUserId())
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, status.Error(codes.NotFound, "user not found")
		}
		s.log.Error("get user failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.GetCurrentPassword())); err != nil {
		return nil, status.Error(codes.Unauthenticated, "current password is incorrect")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.GetNewPassword()), 12)
	if err != nil {
		s.log.Error("bcrypt hash failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	_, err = s.userRepo.Update(ctx, req.GetUserId(), map[string]string{"password_hash": hash})
	if err != nil {
		s.log.Error("update password failed", "error", err)
		return nil, status.Error(codes.Internal, "internal error")
	}

	// Revoke all sessions except current
	if err := s.jwt.RevokeAllTokens(ctx, req.GetUserId(), req.GetAccessToken()); err != nil {
		s.log.Error("revoke tokens after password change failed", "error", err)
		// Don't fail the request — password was already changed
	}

	return &pb.ChangePasswordResponse{}, nil
}
```

**Important:** The `userRepo.Update` method takes `map[string]string` for fields. The password_hash is a string (bcrypt output as `string(hash)`), so this needs to be `map[string]string{"password_hash": string(hash)}`.

- [ ] **Step 4: Commit backend changes**

```bash
git add backend/proto/user/v1/user.proto backend/services/user-svc/internal/grpc/server.go
git commit -m "feat: add ChangePassword gRPC method with session invalidation"
```

---

## Task 4: Backend — ChangePassword API gateway handler + route

**Files:**
- Modify: `backend/api-gateway/internal/handler/auth.go`
- Modify: `backend/api-gateway/cmd/gateway/main.go`

- [ ] **Step 1: Add ChangePassword handler to auth.go**

In `backend/api-gateway/internal/handler/auth.go`, add after `RevokeAll` method:

```go
type changePasswordRequest struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

func (h *AuthHandler) ChangePassword(w http.ResponseWriter, r *http.Request) {
	userID := getUserID(r)
	if userID == "" {
		writeError(w, http.StatusUnauthorized, "unauthorized")
		return
	}

	var req changePasswordRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	accessToken := extractBearerToken(r)

	_, err := h.user.ChangePassword(r.Context(), &userpb.ChangePasswordRequest{
		UserId:          userID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
		AccessToken:     accessToken,
	})
	if err != nil {
		handleGRPCError(w, err)
		return
	}

	h.clearRefreshCookie(w)
	writeJSON(w, http.StatusOK, nil)
}
```

- [ ] **Step 2: Register the route in main.go**

In `backend/api-gateway/cmd/gateway/main.go`, add after the revoke-all route (line 76):

```go
mux.Handle("POST /api/auth/change-password", auth.Required(http.HandlerFunc(authH.ChangePassword)))
```

- [ ] **Step 3: Commit API gateway changes**

```bash
git add backend/api-gateway/internal/handler/auth.go backend/api-gateway/cmd/gateway/main.go
git commit -m "feat: add change-password HTTP endpoint in API gateway"
```

---

## Task 5: Frontend — Password change API function

**Files:**
- Modify: `frontend/src/api/auth.ts`

- [ ] **Step 1: Add changePassword function**

In `frontend/src/api/auth.ts`, add after `revokeAllSessions`:

```ts
export async function changePassword(currentPassword: string, newPassword: string) {
  await client.post('/auth/change-password', { current_password: currentPassword, new_password: newPassword });
}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/api/auth.ts
git commit -m "feat: add changePassword API function"
```

---

## Task 6: Frontend — Password change UI in Settings Security section

**Files:**
- Modify: `frontend/src/pages/Settings.tsx`
- Modify: `frontend/src/styles/profile.module.css`

- [ ] **Step 1: Add password change state and handler to Settings**

In `frontend/src/pages/Settings.tsx`:

Add import for `changePassword`:
```tsx
import { revokeAllSessions, changePassword } from '@/api/auth';
```

Add import for `Lock` icon:
```tsx
import { ArrowLeft, Camera, ImagePlus, X, ShieldAlert, Lock } from 'lucide-react';
```

Add state variables after existing state declarations (after line 30):
```tsx
const [currentPassword, setCurrentPassword] = useState('');
const [newPassword, setNewPassword] = useState('');
const [confirmNewPassword, setConfirmNewPassword] = useState('');
const [pwLoading, setPwLoading] = useState(false);
const [pwSuccess, setPwSuccess] = useState(false);
const [pwError, setPwError] = useState('');
```

Add handler after `handleRevokeAll`:
```tsx
const handleChangePassword = async (e: React.FormEvent) => {
  e.preventDefault();
  setPwError(''); setPwSuccess(false);
  if (newPassword.length < 8) { setPwError('Новый пароль: минимум 8 символов'); return; }
  if (newPassword !== confirmNewPassword) { setPwError('Пароли не совпадают'); return; }
  setPwLoading(true);
  try {
    await changePassword(currentPassword, newPassword);
    setPwSuccess(true);
    setCurrentPassword(''); setNewPassword(''); setConfirmNewPassword('');
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
    setPwError(msg || 'Не удалось сменить пароль');
  } finally { setPwLoading(false); }
};
```

- [ ] **Step 2: Add password change form to Security section**

In `frontend/src/pages/Settings.tsx`, replace the security section (lines 170-176) with:

```tsx
<div className={s.securitySection}>
  <h2 className={s.securityTitle}><ShieldAlert size={18} /> Безопасность</h2>

  <form onSubmit={handleChangePassword} className={s.passwordForm}>
    <h3 className={s.passwordTitle}><Lock size={16} /> Смена пароля</h3>
    <input
      type="password"
      value={currentPassword}
      onChange={(e) => setCurrentPassword(e.target.value)}
      placeholder="Текущий пароль"
      required
      className={s.fieldInput}
    />
    <input
      type="password"
      value={newPassword}
      onChange={(e) => setNewPassword(e.target.value)}
      placeholder="Новый пароль (мин. 8 символов)"
      required
      minLength={8}
      className={s.fieldInput}
    />
    <input
      type="password"
      value={confirmNewPassword}
      onChange={(e) => setConfirmNewPassword(e.target.value)}
      placeholder="Подтвердите новый пароль"
      required
      className={s.fieldInput}
    />
    {pwError && <p className={s.errorText}>{pwError}</p>}
    {pwSuccess && <p className={s.successText}>Пароль изменён. Все остальные сессии завершены.</p>}
    <button type="submit" disabled={pwLoading} className={s.saveBtn}>
      {pwLoading ? 'Сохранение...' : 'Сменить пароль'}
    </button>
  </form>

  <div className={s.sessionSection}>
    <p className={s.securityDesc}>Завершить все активные сессии на всех устройствах. Вам потребуется войти заново.</p>
    <button onClick={handleRevokeAll} disabled={revoking} className={s.revokeBtn}>
      {revoking ? 'Завершаем...' : 'Завершить все сессии'}
    </button>
  </div>
</div>
```

- [ ] **Step 3: Add CSS styles**

In `frontend/src/styles/profile.module.css`, add after `.revokeBtn:disabled`:

```css
.passwordForm {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin-bottom: 24px;
  max-width: 400px;
}
.passwordTitle {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  font-weight: 600;
  color: var(--text-secondary);
}
.sessionSection {
  padding-top: 16px;
  border-top: 1px solid var(--border);
}
```

- [ ] **Step 4: Commit**

```bash
git add frontend/src/pages/Settings.tsx frontend/src/styles/profile.module.css
git commit -m "feat: add password change form to settings security section"
```

---

## Task 7: Frontend — Onboarding wizard component

**Files:**
- Create: `frontend/src/components/OnboardingWizard.tsx`
- Modify: `frontend/src/styles/modal.module.css`

- [ ] **Step 1: Add onboarding wizard styles**

In `frontend/src/styles/modal.module.css`, add at the end:

```css
/* ── Onboarding Wizard ── */
.wizardTitle {
  font-family: var(--font-display);
  font-size: 22px;
  font-weight: 700;
  margin-bottom: 8px;
}
.wizardDesc {
  font-size: 14px;
  color: var(--text-secondary);
  margin-bottom: 24px;
}
.wizardField {
  width: 100%;
  margin-bottom: 16px;
}
.wizardInput {
  width: 100%;
  padding: 10px 16px;
  background: var(--bg-tertiary);
  border: 1px solid var(--border);
  border-radius: 12px;
  font-size: 15px;
  color: var(--text-primary);
  outline: none;
  transition: border-color 0.15s;
}
.wizardInput:focus { border-color: var(--accent); }
.wizardTextarea {
  composes: wizardInput;
  resize: none;
}
.wizardCharHint {
  font-size: 12px;
  color: var(--text-tertiary);
  text-align: right;
  margin-top: 4px;
}
.wizardAvatarArea {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}
.wizardAvatarBtn {
  position: relative;
  cursor: pointer;
}
.wizardAvatarOverlay {
  position: absolute;
  inset: 0;
  border-radius: 50%;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.15s;
}
.wizardAvatarBtn:hover .wizardAvatarOverlay { opacity: 1; }
.wizardAvatarHint {
  font-size: 14px;
  color: var(--text-secondary);
}
.wizardBannerArea {
  margin-bottom: 16px;
}
.wizardBannerBox {
  width: 100%;
  height: 100px;
  border-radius: 12px;
  border: 1px dashed var(--border);
  overflow: hidden;
  cursor: pointer;
  position: relative;
}
.wizardBannerBox img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.wizardBannerPlaceholder {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  height: 100%;
  gap: 4px;
  color: var(--text-tertiary);
  font-size: 13px;
}
.wizardBannerOverlay {
  position: absolute;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  opacity: 0;
  transition: opacity 0.15s;
}
.wizardBannerBox:hover .wizardBannerOverlay { opacity: 1; }
.wizardFooter {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-top: 8px;
}
.wizardSkipBtn {
  font-size: 14px;
  color: var(--text-secondary);
  transition: color 0.15s;
}
.wizardSkipBtn:hover { color: var(--text-primary); }
.wizardNextBtn {
  padding: 8px 24px;
  border-radius: 50px;
  background: var(--accent);
  color: white;
  font-weight: 600;
  font-size: 14px;
  transition: background 0.15s;
}
.wizardNextBtn:hover { background: var(--accent-hover); }
.wizardNextBtn:disabled { opacity: 0.5; cursor: not-allowed; }
.wizardSteps {
  display: flex;
  gap: 6px;
  justify-content: center;
  margin-bottom: 20px;
}
.wizardDot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--border);
  transition: background 0.2s;
}
.wizardDotActive {
  background: var(--accent);
}
```

- [ ] **Step 2: Create OnboardingWizard component**

Create `frontend/src/components/OnboardingWizard.tsx`:

```tsx
import { useState, useRef } from 'react';
import { Camera, ImagePlus } from 'lucide-react';
import { updateUser } from '@/api/users';
import { uploadMedia, deleteMedia } from '@/api/media';
import { useAuthStore } from '@/store/auth';
import Avatar from './Avatar';
import s from '@/styles/modal.module.css';

interface Props {
  onClose: () => void;
}

const STEPS = [
  { key: 'name', title: 'Как вас зовут?', desc: 'Отображаемое имя видно всем пользователям' },
  { key: 'bio', title: 'Расскажите о себе', desc: 'Краткое описание для вашего профиля' },
  { key: 'avatar', title: 'Добавьте фото', desc: 'Выберите аватар для профиля' },
  { key: 'banner', title: 'Фон профиля', desc: 'Добавьте фоновое изображение' },
] as const;

export default function OnboardingWizard({ onClose }: Props) {
  const { user, updateUser: updateStore, setAvatarMediaId, setBannerMediaId } = useAuthStore();
  const [step, setStep] = useState(0);
  const [displayName, setDisplayName] = useState('');
  const [bio, setBio] = useState('');
  const [avatarFile, setAvatarFile] = useState<File | null>(null);
  const [avatarPreview, setAvatarPreview] = useState<string | null>(null);
  const [bannerFile, setBannerFile] = useState<File | null>(null);
  const [bannerPreview, setBannerPreview] = useState<string | null>(null);
  const [saving, setSaving] = useState(false);
  const avatarRef = useRef<HTMLInputElement>(null);
  const bannerRef = useRef<HTMLInputElement>(null);

  if (!user) return null;

  const current = STEPS[step];
  const isLast = step === STEPS.length - 1;

  const handleAvatarChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setAvatarFile(file);
    setAvatarPreview(URL.createObjectURL(file));
  };

  const handleBannerChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (!file) return;
    setBannerFile(file);
    setBannerPreview(URL.createObjectURL(file));
  };

  const handleFinish = async () => {
    setSaving(true);
    try {
      let avatarUrl = user.avatar_url;
      let bannerUrl = user.banner_url || '';
      let newAvatarMediaId: string | null = null;
      let newBannerMediaId: string | null = null;

      if (avatarFile) {
        const m = await uploadMedia(avatarFile);
        avatarUrl = m.url;
        newAvatarMediaId = m.id;
      }
      if (bannerFile) {
        const m = await uploadMedia(bannerFile);
        bannerUrl = m.url;
        newBannerMediaId = m.id;
      }

      const fields: Record<string, string> = {};
      if (displayName.trim()) fields.display_name = displayName;
      if (bio.trim()) fields.bio = bio;
      if (avatarUrl !== user.avatar_url) fields.avatar_url = avatarUrl;
      if (bannerUrl !== (user.banner_url || '')) fields.banner_url = bannerUrl;

      if (Object.keys(fields).length > 0) {
        const updated = await updateUser(user.id, fields);
        updateStore(updated);
      }

      if (newAvatarMediaId) setAvatarMediaId(newAvatarMediaId);
      if (newBannerMediaId) setBannerMediaId(newBannerMediaId);

      if (avatarPreview) URL.revokeObjectURL(avatarPreview);
      if (bannerPreview) URL.revokeObjectURL(bannerPreview);
    } catch {
      // Silently close — user can edit profile in settings later
    } finally {
      setSaving(false);
      onClose();
    }
  };

  const handleNext = () => {
    if (isLast) {
      handleFinish();
    } else {
      setStep(step + 1);
    }
  };

  const handleSkip = () => {
    if (isLast) {
      onClose();
    } else {
      setStep(step + 1);
    }
  };

  return (
    <div className={s.backdrop} onClick={onClose}>
      <div className={s.modal} onClick={(e) => e.stopPropagation()}>
        {/* Step dots */}
        <div className={s.wizardSteps}>
          {STEPS.map((_, i) => (
            <div key={i} className={`${s.wizardDot} ${i <= step ? s.wizardDotActive : ''}`} />
          ))}
        </div>

        <h2 className={s.wizardTitle}>{current.title}</h2>
        <p className={s.wizardDesc}>{current.desc}</p>

        {/* Step content */}
        {current.key === 'name' && (
          <div className={s.wizardField}>
            <input
              type="text"
              value={displayName}
              onChange={(e) => setDisplayName(e.target.value)}
              placeholder="Отображаемое имя"
              maxLength={100}
              autoFocus
              className={s.wizardInput}
            />
          </div>
        )}

        {current.key === 'bio' && (
          <div className={s.wizardField}>
            <textarea
              value={bio}
              onChange={(e) => setBio(e.target.value)}
              placeholder="О себе..."
              maxLength={500}
              rows={4}
              autoFocus
              className={s.wizardTextarea}
            />
            <p className={s.wizardCharHint}>{bio.length}/500</p>
          </div>
        )}

        {current.key === 'avatar' && (
          <div className={s.wizardAvatarArea}>
            <div className={s.wizardAvatarBtn} onClick={() => avatarRef.current?.click()}>
              <Avatar url={avatarPreview || user.avatar_url} name={displayName || user.username} size="xl" />
              <div className={s.wizardAvatarOverlay}><Camera size={20} color="white" /></div>
            </div>
            <input ref={avatarRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleAvatarChange} style={{ display: 'none' }} />
            <p className={s.wizardAvatarHint}>Нажмите чтобы выбрать</p>
          </div>
        )}

        {current.key === 'banner' && (
          <div className={s.wizardBannerArea}>
            <div className={s.wizardBannerBox} onClick={() => bannerRef.current?.click()}>
              {bannerPreview ? (
                <img src={bannerPreview} alt="" />
              ) : (
                <div className={s.wizardBannerPlaceholder}>
                  <ImagePlus size={24} color="var(--text-tertiary)" />
                  <span>Нажмите чтобы загрузить</span>
                </div>
              )}
              <div className={s.wizardBannerOverlay}><Camera size={22} color="white" /></div>
            </div>
            <input ref={bannerRef} type="file" accept="image/jpeg,image/png,image/gif,image/webp" onChange={handleBannerChange} style={{ display: 'none' }} />
          </div>
        )}

        {/* Footer */}
        <div className={s.wizardFooter}>
          <button onClick={handleSkip} className={s.wizardSkipBtn}>
            {isLast ? 'Пропустить' : 'Пропустить'}
          </button>
          <button onClick={handleNext} disabled={saving} className={s.wizardNextBtn}>
            {saving ? 'Сохранение...' : isLast ? 'Завершить' : 'Далее'}
          </button>
        </div>
      </div>
    </div>
  );
}
```

- [ ] **Step 3: Commit**

```bash
git add frontend/src/components/OnboardingWizard.tsx frontend/src/styles/modal.module.css
git commit -m "feat: add onboarding wizard component"
```

---

## Task 8: Frontend — Show onboarding wizard after registration

**Files:**
- Modify: `frontend/src/pages/Register.tsx`

- [ ] **Step 1: Add wizard state and rendering to Register**

In `frontend/src/pages/Register.tsx`:

Add import:
```tsx
import OnboardingWizard from '@/components/OnboardingWizard';
```

Add state after existing state declarations:
```tsx
const [showOnboarding, setShowOnboarding] = useState(false);
```

Modify `handleSubmit` — change the `navigate('/', { replace: true })` line to show the wizard instead:

```tsx
const handleSubmit = async (e: React.FormEvent) => {
  e.preventDefault();
  setError('');
  if (username.length < 3 || username.length > 50) { setError('Имя пользователя: 3–50 символов'); return; }
  if (password.length < 8) { setError('Пароль: минимум 8 символов'); return; }
  if (password !== confirmPassword) { setError('Пароли не совпадают'); return; }

  setLoading(true);
  try {
    const data = await register(username, email, password);
    setAuth(data.user, data.access_token);
    setShowOnboarding(true);
  } catch (err: unknown) {
    const msg = (err as { response?: { data?: { error?: { message?: string } } } })?.response?.data?.error?.message;
    setError(msg || 'Не удалось зарегистрироваться');
  } finally { setLoading(false); }
};
```

Add the wizard rendering — inside the return, before the closing `</div>` of the page:

```tsx
{showOnboarding && (
  <OnboardingWizard onClose={() => navigate('/', { replace: true })} />
)}
```

- [ ] **Step 2: Commit**

```bash
git add frontend/src/pages/Register.tsx
git commit -m "feat: show onboarding wizard after registration"
```

---

## Task 9: Regenerate protobuf and verify build

**Files:**
- Generated: `backend/gen/go/user/v1/*.go`

- [ ] **Step 1: Check proto generation setup**

```bash
ls backend/Makefile 2>/dev/null || ls backend/scripts/ 2>/dev/null || echo "check generate approach"
```

Look at how proto files are generated in this project.

- [ ] **Step 2: Regenerate proto**

Run the appropriate proto generation command for this project.

- [ ] **Step 3: Verify backend compiles**

```bash
cd backend/services/user-svc && go build ./...
cd backend/api-gateway && go build ./...
```

- [ ] **Step 4: Verify frontend compiles**

```bash
cd frontend && npx tsc --noEmit
```

- [ ] **Step 5: Commit generated code**

```bash
git add backend/gen/
git commit -m "chore: regenerate protobuf code for ChangePassword"
```
