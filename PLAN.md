# Этап 5: Разработка микроблогинг-платформы

**Дедлайн:** 15 апреля 2026
**Стек:** React + Go + PostgreSQL + Redis + MinIO
**API:** REST (фронт → бэк) + gRPC (между сервисами) + WebSocket (live-лента)
**Инфраструктура:** Docker + docker-compose

> Этапы 1–4 (ТЗ, бизнес-логика, UML + DB schema, репозиторий + Docker + SQL) завершены.

---

## Архитектура

```
Frontend (React) ── REST ──→ API Gateway (Go) ←── WebSocket ── Frontend
                                  ↓ gRPC
                    ┌─────────────┼─────────────┐
                    ↓             ↓             ↓
                UserSvc       PostSvc       MediaSvc
                    ↓             ↓             ↓
                PostgreSQL    PostgreSQL     MinIO
                    └─────┬───────┘
                        Redis
                   (кэш + pub/sub)
```

Фронтенд ходит в API Gateway по REST и WebSocket. Внутри сервисы общаются по gRPC. Redis кэширует ленту, счётчики лайков и раздаёт live-обновления через pub/sub. Логи — structured JSON в stdout/stderr.

## Структура репозитория

```
/
├── frontend/              # React (Vite SPA)
│   ├── src/
│   ├── public/
│   ├── package.json
│   └── Dockerfile
├── api-gateway/           # REST + WebSocket → gRPC
│   ├── cmd/
│   ├── internal/
│   │   ├── handler/
│   │   ├── ws/
│   │   ├── middleware/
│   │   └── grpc_client/
│   ├── go.mod
│   └── Dockerfile
├── services/
│   ├── user-svc/
│   ├── post-svc/
│   ├── comment-svc/
│   └── media-svc/
├── proto/
│   ├── user.proto
│   ├── post.proto
│   ├── comment.proto
│   ├── media.proto
│   └── common.proto
├── migrations/
├── docker-compose.yml
├── buf.yaml
├── Makefile
├── .env.example
└── README.md
```

---

## Общая стратегия

Порядок разработки: снизу вверх. Сначала gRPC-сервисы (они независимы друг от друга), затем API Gateway (интеграционный слой), затем фронтенд. Каждый сервис проходит цикл: proto-генерация → repository-слой → бизнес-логика → gRPC-хендлеры → тесты.

### Зависимости между блоками

- 5.2 (UserSvc) блокирует 5.1 (Gateway) — JWT-верификация, GetUser
- 5.3 (PostSvc) блокирует 5.8 (Frontend: лента)
- 5.4 (CommentSvc) блокирует 5.10 (Frontend: комментарии)
- 5.5 (MediaSvc) блокирует загрузку аватаров и медиа в постах
- 5.1 (Gateway) блокирует весь фронтенд
- 5.6 (Frontend: инфраструктура) блокирует 5.7–5.10

### Рекомендуемый порядок

1. Все gRPC-сервисы параллельно (5.2, 5.3, 5.4, 5.5) — ~2 недели
2. API Gateway (5.1) — ~1 неделя, параллельно с хвостами сервисов
3. Frontend инфраструктура + auth (5.6, 5.7) — ~3-4 дня
4. Frontend фичи (5.8, 5.9, 5.10) — ~1 неделя
5. Интеграция и smoke tests (5.11) — ~2-3 дня

### Сводная таблица сроков

| Блок | Что | Срок | Длительность |
|------|-----|------|--------------|
| 5.2 | User Service | 17–21 марта | 5 дней |
| 5.3 | Post Service | 17–21 марта | 5 дней (параллельно с 5.2) |
| 5.4 | Comment Service | 22–26 марта | 4-5 дней |
| 5.5 | Media Service | 22–25 марта | 4 дня (параллельно с 5.4) |
| 5.1 | API Gateway | 27 марта – 2 апреля | 6 дней |
| 5.6 | Frontend: инфраструктура | 3–4 апреля | 2 дня |
| 5.7 | Frontend: auth | 4–5 апреля | 2 дня |
| 5.8 | Frontend: лента | 5–7 апреля | 2 дня |
| 5.9 | Frontend: профиль | 7–9 апреля | 2 дня |
| 5.10 | Frontend: комментарии | 9–10 апреля | 2 дня |
| 5.11 | Интеграция + README | 11–15 апреля | 4 дня |

---

# Фаза 1: gRPC-сервисы (17 марта – 31 марта)

---

## 5.2 User Service

### 5.2.1 Скаффолдинг (день 1)

- [x] Инициализация Go-модуля `services/user-svc`, структура `cmd/main.go`, `internal/`
- [x] Подключение зависимостей: `google.golang.org/grpc`, `github.com/jackc/pgx/v5`, `golang.org/x/crypto/bcrypt`, `github.com/golang-jwt/jwt/v5`
- [x] Конфигурация через env: `DB_URL`, `GRPC_PORT`, `JWT_SECRET`, `JWT_ACCESS_TTL`, `JWT_REFRESH_TTL`
- [x] Запуск gRPC-сервера на порту из env, graceful shutdown по SIGINT/SIGTERM
- [x] Health check endpoint (gRPC health v1)

### 5.2.2 Repository-слой (дни 2–3)

- [x] `internal/repository/user_repo.go` — интерфейс `UserRepository`
- [x] `Create(ctx, username, email, passwordHash) → User, error` — INSERT RETURNING, обработка duplicate key (username, email)
- [x] `GetByID(ctx, id) → User, error` — SELECT WHERE id=$1 AND deleted_at IS NULL
- [x] `GetByEmail(ctx, email) → User, error` — для авторизации
- [x] `Update(ctx, id, fields) → User, error` — динамический UPDATE только переданных полей
- [x] `SoftDelete(ctx, id) → error` — UPDATE SET deleted_at=NOW()
- [x] `Search(ctx, query, cursor, limit) → []User, nextCursor, error` — ILIKE или pg_trgm по username и display_name, cursor-based pagination по (created_at, id)
- [x] `GetByIDs(ctx, ids) → []User, error` — SELECT WHERE id = ANY($1), batch-запрос
- [x] `internal/repository/follow_repo.go` — интерфейс `FollowRepository`
- [x] `Follow(ctx, followerID, followingID) → error` — INSERT, обработка duplicate + no_self constraint
- [x] `Unfollow(ctx, followerID, followingID) → error` — DELETE
- [x] `GetFollowers(ctx, userID, cursor, limit) → []UUID, nextCursor, error`
- [x] `GetFollowing(ctx, userID, cursor, limit) → []UUID, nextCursor, error`
- [x] Все запросы parameterized, все SELECT фильтруют deleted_at IS NULL

### 5.2.3 JWT-модуль (день 3)

- [x] `internal/auth/jwt.go` — генерация access token (15 мин) и refresh token (7 дней)
- [x] Access: claims = `{sub: user_id, exp, iat}`
- [x] Refresh: случайный UUID, хэшируется SHA-256, хранится в Redis как `refresh:{hash} → user_id` с TTL 7d
- [x] `GenerateTokenPair(userID) → (accessToken, refreshToken, error)`
- [x] `ValidateAccessToken(token) → (userID, error)`
- [x] `RotateRefreshToken(oldToken) → (newAccess, newRefresh, error)` — удаляет старый, создаёт новый
- [x] Redis-клиент для refresh token storage

### 5.2.4 gRPC-хендлеры (дни 4–5)

- [x] `internal/grpc/server.go` — регистрация UserService
- [x] `Register` — валидация (email формат, username 3-50 символов, password 8+), bcrypt hash (cost=12), создание user, генерация JWT pair
- [x] `Login` — поиск по email, bcrypt compare, генерация JWT pair
- [x] `RefreshToken` — ротация через Redis
- [x] `GetUser` — по ID, возврат публичных полей
- [x] `UpdateUser` — проверка ownership (user_id из запроса), обновление только переданных полей
- [x] `SoftDeleteUser` — проверка ownership, soft delete
- [x] `SearchUsers` — проксирование в repository с cursor pagination
- [x] `GetUsersByIDs` — batch-запрос, возврат массива
- [x] `Follow` — проверка что target существует и не deleted, вызов FollowRepository
- [x] `Unfollow` — вызов FollowRepository
- [x] `GetFollowers` / `GetFollowing` — список с pagination, обогащение данными пользователей
- [x] Structured JSON логирование через `slog` на каждый RPC-вызов (method, duration, error)

### 5.2.5 Тесты (день 5)

- [x] Unit-тесты на JWT-модуль: генерация, валидация, ротация, expired token
- [ ] Unit-тесты на repository с testcontainers-go (PostgreSQL)
- [ ] Integration test: Register → Login → GetUser → UpdateUser → SoftDelete → GetUser (404)

---

## 5.3 Post Service

### 5.3.1 Скаффолдинг (день 1)

- [x] Go-модуль `services/post-svc`, структура аналогична UserSvc
- [x] Зависимости: grpc, pgx, redis (`github.com/redis/go-redis/v9`)
- [x] Env: `DB_URL`, `REDIS_URL`, `GRPC_PORT`, `LIKE_SYNC_INTERVAL` (default 30s)
- [x] gRPC-сервер + graceful shutdown + health check

### 5.3.2 Repository-слой (дни 2–3)

- [x] `internal/repository/post_repo.go`
- [x] `Create(ctx, userID, content, mediaURL) → Post, error` — INSERT RETURNING
- [x] `GetByID(ctx, id) → Post, error` — SELECT WHERE deleted_at IS NULL
- [x] `SoftDelete(ctx, id, userID) → error` — UPDATE SET deleted_at, проверка ownership
- [x] `GetFeed(ctx, userID, cursor, limit) → []Post, nextCursor, hasMore, error` — JOIN follows, cursor по (created_at, id) DESC, WHERE deleted_at IS NULL
- [x] `GetByUser(ctx, userID, cursor, limit) → []Post, nextCursor, hasMore, error`
- [x] `IncrementLikes(ctx, postID, delta) → error` — UPDATE SET likes_count = likes_count + $1
- [x] `internal/repository/like_repo.go`
- [x] `LikePost(ctx, postID, userID) → error` — INSERT, обработка duplicate
- [x] `UnlikePost(ctx, postID, userID) → error` — DELETE
- [x] `IsLiked(ctx, postID, userID) → bool, error` — для проверки

### 5.3.3 Redis-кэш ленты (день 3)

- [x] `internal/cache/feed_cache.go`
- [x] `GetFeed(ctx, userID, cursor, limit) → []string, error` — ZREVRANGEBYSCORE с cursor (Unix ms)
- [x] `SetFeed(ctx, userID, posts) → error` — ZADD + EXPIRE 60s
- [x] `InvalidateFeed(ctx, userIDs []string) → error` — DEL pipeline для массовой инвалидации
- [x] `internal/cache/like_counter.go`
- [x] `Increment(ctx, postID) → int64, error` — INCR `post:likes:{postID}`
- [x] `Decrement(ctx, postID) → int64, error` — DECR
- [x] `Get(ctx, postID) → int64, error` — GET с fallback на БД
- [x] Background goroutine: каждые 30s собирает dirty counters и flush в PostgreSQL через batch UPDATE

### 5.3.4 gRPC-хендлеры (дни 4–5)

- [x] `CreatePost` — валидация (content ≤ 280, не пустой), сохранение, получение follower_ids из UserSvc (gRPC call), инвалидация кэшей лент, PUBLISH в Redis `ws:new_post`
- [x] `GetPost` — fallback: Redis counter → merge с DB row
- [x] `DeletePost` — ownership check, soft delete, инвалидация кэшей, PUBLISH `ws:post_deleted`
- [x] `GetFeed` — cache hit: Redis sorted set → fetch posts by IDs; cache miss: DB query → populate cache
- [x] `LikePost` — INSERT like, INCR Redis counter, PUBLISH `ws:new_like`
- [x] `UnlikePost` — DELETE like, DECR Redis counter
- [x] `GetPostsByUser` — repository с cursor pagination
- [x] Structured JSON логирование

### 5.3.5 Тесты (день 5)

- [ ] Unit-тесты на cache-слой (mock Redis)
- [ ] Unit-тесты на repository (testcontainers)
- [ ] Integration: CreatePost → GetPost → GetFeed → LikePost → GetPost (likes_count changed) → DeletePost → GetPost (404)

> **Примечание:** Repository и integration тесты требуют testcontainers (Docker runtime)

---

## 5.4 Comment Service

### 5.4.1 Скаффолдинг (день 1)

- [x] Go-модуль `services/comment-svc`
- [x] Env: `DB_URL`, `GRPC_PORT`
- [x] gRPC-сервер + graceful shutdown + health check

### 5.4.2 Repository-слой (дни 2–3)

- [x] `internal/repository/comment_repo.go`
- [x] `Create(ctx, postID, userID, parentID, content) → Comment, error` — INSERT с вычислением depth: если parentID != nil, SELECT depth FROM comments WHERE id=$1, new depth = parent.depth + 1, CHECK depth ≤ 5
- [x] `GetTree(ctx, postID, cursor, limit) → []Comment, nextCursor, error` — двухэтапная выборка: (1) корневые комментарии с cursor pagination, (2) WITH RECURSIVE CTE для загрузки дочерних до depth=5
- [x] `SoftDelete(ctx, commentID, userID) → error` — ownership check, UPDATE SET deleted_at
- [x] `DecrementPostComments(ctx, postID) → error` — UPDATE posts SET comments_count = comments_count - 1
- [x] `IncrementPostComments(ctx, postID) → error` — UPDATE posts SET comments_count = comments_count + 1
- [x] `internal/repository/comment_like_repo.go`
- [x] `Like(ctx, commentID, userID) → error` — INSERT
- [x] `Unlike(ctx, commentID, userID) → error` — DELETE
- [x] `IncrementLikes(ctx, commentID, delta) → error`

### 5.4.3 Сборка дерева (день 3)

- [x] `internal/service/tree_builder.go`
- [x] Принимает flat list из CTE, строит вложенную структуру через map[parentID][]Comment
- [x] Возвращает `[]Comment` с заполненным `Children` полем
- [x] Deleted комментарии: если есть children, заменяем content на `[удалено]`, если нет children — исключаем из дерева

### 5.4.4 gRPC-хендлеры (день 4)

- [x] `CreateComment` — валидация (content ≤ 500, не пустой), проверка что post существует (direct DB check), если parentID задан — проверка depth < 5, сохранение, инкремент comments_count
- [x] `GetCommentTree` — repository + tree_builder, cursor pagination на root level
- [x] `DeleteComment` — ownership check, soft delete, декремент comments_count
- [x] `LikeComment` / `UnlikeComment` — аналогично постам, но без Redis-кэша (volume ниже)
- [x] Structured JSON логирование

### 5.4.5 Тесты (день 4–5)

- [x] Unit-тест tree_builder: flat list → nested tree, deleted nodes handling
- [ ] Integration: CreateComment (root) → CreateComment (reply) → CreateComment (depth 5) → CreateComment (depth 6, ожидаем ошибку) → GetTree → DeleteComment → GetTree (показывает [удалено])

---

## 5.5 Media Service

### 5.5.1 Скаффолдинг (день 1)

- [x] Go-модуль `services/media-svc`
- [x] Зависимости: grpc, pgx, `github.com/minio/minio-go/v7`
- [x] Env: `DB_URL`, `GRPC_PORT`, `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET`, `MINIO_USE_SSL`
- [x] gRPC-сервер + graceful shutdown + health check
- [x] При старте: создание bucket если не существует (`MakeBucket`)

### 5.5.2 Repository + MinIO client (дни 2–3)

- [x] `internal/repository/media_repo.go`
- [x] `Create(ctx, userID, postID, bucket, objectKey, contentType) → Media, error`
- [x] `GetByID(ctx, id) → Media, error`
- [x] `Delete(ctx, id, userID) → Media, error` — возвращает media для удаления объекта из MinIO
- [x] `internal/storage/minio.go`
- [x] `Upload(ctx, objectKey, data []byte, contentType string) → error` — PutObject
- [x] `GetPresignedURL(ctx, objectKey, expiry time.Duration) → string, error` — PresignedGetObject (expiry 1h)
- [x] `GetPresignedUploadURL(ctx, objectKey, contentType, expiry) → string, error` — PresignedPutObject
- [x] `Delete(ctx, objectKey) → error` — RemoveObject

### 5.5.3 Валидация файлов (день 2)

- [x] `internal/validation/media.go`
- [x] Допустимые MIME: `image/jpeg`, `image/png`, `image/gif`, `image/webp`
- [x] Максимальный размер: 10MB
- [x] Валидация по magic bytes (первые 4-8 байт), а не только по Content-Type header
- [x] Генерация objectKey: `{user_id}/{uuid}.{ext}` — изоляция по пользователям

### 5.5.4 gRPC-хендлеры (день 3)

- [x] `UploadMedia` — валидация файла, upload в MinIO, сохранение метаданных в PostgreSQL, возврат URL
- [x] `GetPresignedUploadURL` — генерация presigned PUT URL, создание записи media с pending статусом
- [x] `GetMediaURL` — генерация presigned GET URL (1h expiry)
- [x] `DeleteMedia` — ownership check, удаление из MinIO + БД
- [x] Structured JSON логирование

### 5.5.5 Тесты (день 4)

- [x] Unit-тесты валидации: допустимые типы, превышение размера, подмена content-type
- [ ] Integration с testcontainers (MinIO): Upload → GetURL → Delete → GetURL (404)

---

# Фаза 2: API Gateway (31 марта – 4 апреля)

---

## 5.1 API Gateway

### 5.1.1 Скаффолдинг (день 1)

- [x] Go-модуль `api-gateway`, структура: `cmd/main.go`, `internal/handler/`, `internal/middleware/`, `internal/ws/`, `internal/grpc_client/`
- [x] HTTP-сервер на stdlib (net/http)
- [x] Env: `PORT`, `USER_SVC_ADDR`, `POST_SVC_ADDR`, `COMMENT_SVC_ADDR`, `MEDIA_SVC_ADDR`, `REDIS_URL`, `JWT_SECRET`, `CORS_ORIGIN`
- [x] Graceful shutdown: HTTP server + gRPC connections + WebSocket hub

### 5.1.2 gRPC-клиенты (день 1)

- [x] `internal/grpc_client/clients.go` — фабрика для создания gRPC-соединений с retry policy
- [x] Каждый клиент: `grpc.NewClient` с `grpc.WithTransportCredentials(insecure.NewCredentials())` (внутренняя Docker-сеть)
- [x] Connection pooling через gRPC built-in (один `ClientConn` на сервис)
- [x] Keepalive: `grpc.WithKeepaliveParams` — ping каждые 30s, timeout 10s

### 5.1.3 Middleware-стек (дни 2–3)

- [x] `middleware/recovery.go` — panic recovery, возвращает 500 + логирует stack trace
- [x] `middleware/request_id.go` — генерация UUID, добавление в context и response header `X-Request-ID`
- [x] `middleware/logger.go` — structured JSON log: method, path, status, duration, request_id, user_id (если auth)
- [x] `middleware/cors.go` — CORS с whitelist из env: `Access-Control-Allow-Origin`, `Allow-Methods`, `Allow-Headers`, `Allow-Credentials`
- [x] `middleware/auth.go` — извлечение JWT из `Authorization: Bearer`, валидация, добавление user_id в context; пропуск для публичных роутов (login, register, GET posts, GET profiles)
- [x] `middleware/rate_limiter.go` — token bucket через Redis: INCR + EXPIRE; ключ = `rate_limit:{ip}:{route_group}`; лимиты: auth endpoints 5/min, upload 10/min, остальные auth 100/min, guest 30/min; возврат 429 с `Retry-After` header
- [x] `middleware/security_headers.go` — `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, `Content-Security-Policy: default-src 'self'`
- [x] `middleware/body_limit.go` — ограничение body: 1MB для JSON, 10MB для multipart/upload

### 5.1.4 REST-хендлеры (дни 3–5)

- [x] `handler/auth.go`
  - POST /api/auth/register — парсинг JSON, вызов UserSvc.Register, возврат 201 + tokens
  - POST /api/auth/login — парсинг JSON, вызов UserSvc.Login, возврат 200 + tokens
  - POST /api/auth/refresh — вызов UserSvc.RefreshToken, возврат новых tokens
- [x] `handler/user.go`
  - GET /api/users/:id — UserSvc.GetUser, возврат профиля
  - PUT /api/users/:id — auth required, ownership check (user_id из JWT == :id), UserSvc.UpdateUser
  - DELETE /api/users/:id — auth required, ownership, UserSvc.SoftDeleteUser
  - GET /api/users/:id/posts?cursor= — PostSvc.GetPostsByUser, обогащение author data
  - GET /api/users/search?q=&cursor= — UserSvc.SearchUsers
  - POST /api/users/:id/follow — auth required, UserSvc.Follow(jwt_user_id, :id)
  - DELETE /api/users/:id/follow — auth required, UserSvc.Unfollow
  - GET /api/users/:id/followers?cursor= — UserSvc.GetFollowers
  - GET /api/users/:id/following?cursor= — UserSvc.GetFollowing
- [x] `handler/post.go`
  - POST /api/posts — auth required, PostSvc.CreatePost, обогащение author
  - GET /api/posts/:id — PostSvc.GetPost, обогащение author
  - DELETE /api/posts/:id — auth required, PostSvc.DeletePost
  - GET /api/feed?cursor= — auth required, PostSvc.GetFeed, batch обогащение authors через UserSvc.GetUsersByIDs
  - POST /api/posts/:id/like — auth required, PostSvc.LikePost
  - DELETE /api/posts/:id/like — auth required, PostSvc.UnlikePost
- [x] `handler/comment.go`
  - POST /api/posts/:id/comments — auth required, CommentSvc.CreateComment
  - GET /api/posts/:id/comments?cursor= — CommentSvc.GetCommentTree, обогащение authors
  - DELETE /api/comments/:id — auth required, CommentSvc.DeleteComment
  - POST /api/comments/:id/like — auth required, CommentSvc.LikeComment
  - DELETE /api/comments/:id/like — auth required, CommentSvc.UnlikeComment
- [x] `handler/media.go`
  - POST /api/upload — auth required, multipart form, чтение файла, MediaSvc.UploadMedia
- [x] Все хендлеры: маппинг gRPC ошибок → HTTP статусы (NotFound→404, AlreadyExists→409, InvalidArgument→400, PermissionDenied→403, Unauthenticated→401)
- [x] Стандартный JSON response format: `{"data": ..., "error": null}` или `{"data": null, "error": {"code": ..., "message": ...}}`

### 5.1.5 WebSocket Hub (дни 5–6)

- [x] `internal/ws/hub.go` — центральный hub: map[userID]*Client, register/unregister channels
- [x] `internal/ws/client.go` — goroutine на read (ping/pong, close) и write (отправка events)
- [x] Подключение: WS /api/ws?token=JWT → валидация JWT → регистрация в hub
- [x] Redis subscriber: SUBSCRIBE `ws:channels:{user_id}` для каждого подключённого клиента
- [x] При получении сообщения из Redis: десериализация JSON → отправка клиенту
- [x] Keepalive: ping каждые 30s, pong timeout 10s, при timeout — закрытие соединения
- [x] При disconnect: UNSUBSCRIBE, удаление из hub
- [x] Типы событий: `new_post`, `new_like`, `new_comment`, `new_follower`, `post_deleted`
- [x] Формат сообщения клиенту: `{"type": "new_post", "data": {...}}`

### 5.1.6 Тесты (день 6)

- [x] Unit-тесты middleware: auth (valid/invalid/expired JWT), recovery, CORS, security headers, body limit, request ID
- [ ] Integration: HTTP request → gRPC call → response mapping
- [ ] WebSocket: connect → receive event → disconnect

---

# Фаза 3: Frontend (4 апреля – 11 апреля)

---

## 5.6 Frontend: инфраструктура (дни 1–2)

### 5.6.1 Проект (день 1)

- [x] `npm create vite@latest frontend -- --template react-ts`
- [x] Зависимости: `react-router-dom`, `zustand`, `axios`, `lucide-react`
- [x] Структура: `src/api/`, `src/components/`, `src/pages/`, `src/store/`, `src/hooks/`, `src/types/`, `src/lib/`
- [x] TypeScript типы из proto-контрактов: `User`, `Post`, `Comment`, `PaginationResponse`
- [x] Роуты: `/` (лента), `/login`, `/register`, `/profile/:id`, `/post/:id`, `/settings`

### 5.6.2 HTTP-клиент (день 1)

- [x] `src/api/client.ts` — axios instance с baseURL из env
- [x] Request interceptor: добавление `Authorization: Bearer {accessToken}` из store
- [x] Response interceptor: при 401 — попытка refresh через `/api/auth/refresh`, при неудаче — редирект на /login
- [x] Типизированные API-функции: `api.auth.register()`, `api.auth.login()`, `api.posts.create()`, и т.д.

### 5.6.3 WebSocket-клиент (день 2)

- [x] `src/lib/websocket.ts` — класс WSClient
- [x] Подключение: `ws://host/api/ws?token={accessToken}`
- [x] Auto-reconnect с exponential backoff: 1s, 2s, 4s, 8s, max 30s
- [x] Event emitter pattern: `ws.on('new_post', callback)`
- [x] Ping/pong handling
- [x] Disconnect при logout

### 5.6.4 Store (день 2)

- [x] `src/store/auth.ts` — Zustand: `user`, `accessToken`, `refreshToken`, `isAuthenticated`, `login()`, `logout()`, `updateUser()`, `setTokens()`
- [x] Tokens хранятся в localStorage
- [x] `src/store/feed.ts` — `posts[]`, `cursor`, `hasMore`, `loadMore()`, `prependNewPost()`
- [x] `src/store/ws.ts` — состояние WebSocket: `connected`, `reconnecting`

### 5.6.5 Layout (день 2)

- [x] `src/components/Layout.tsx` — sidebar (навигация), main content area, right panel (поиск, инфо)
- [x] Responsive: sidebar скрывается на мобильных
- [x] Protected route wrapper: проверка `isAuthenticated`, редирект на /login

---

## 5.7 Frontend: аутентификация (дни 3–4)

- [x] `src/pages/Register.tsx` — форма: username, email, password, confirm password
  - Валидация: username 3-50 символов, email формат, password 8+ символов, passwords match
  - При успехе: сохранение tokens в store, редирект на /
  - При ошибке: отображение серверных ошибок (duplicate email/username)
- [x] `src/pages/Login.tsx` — форма: email, password
  - При успехе: tokens в store, редирект на /
  - При ошибке: «неверный email или пароль»
- [x] Protected route HOC/wrapper: если !isAuthenticated → redirect /login с return URL
- [x] Auto-refresh: response interceptor пытается обновить access token при 401
- [x] Logout: очистка store, disconnect WebSocket, redirect /login

---

## 5.8 Frontend: лента (дни 4–5)

- [x] `src/pages/Home.tsx` — основная страница
- [x] `src/components/PostCard.tsx` — карточка поста: автор (аватар, username, display_name), контент, медиа (если есть), likes_count, comments_count, время, кнопки like/comment
- [x] `src/components/CreatePost.tsx` — textarea с счётчиком символов (max 280), кнопка прикрепления медиа (file input, preview), кнопка отправки
- [x] Infinite scroll: `IntersectionObserver` на последнем элементе, вызов loadMore() с cursor
- [x] WebSocket интеграция: при `new_post` event — показ баннера «N новых постов» сверху ленты, клик → prepend в список
- [x] Optimistic update для лайков: мгновенное обновление UI, откат при ошибке
- [x] Loading states: skeleton cards при загрузке, spinner при loadMore

---

## 5.9 Frontend: профиль и подписки (дни 6–7)

- [x] `src/pages/Profile.tsx` — header: аватар (большой), display_name, username, bio, счётчики followers/following, кнопка follow/unfollow (если не свой профиль), кнопка «Настройки» (если свой)
- [x] Список постов пользователя с cursor pagination (переиспользование PostCard)
- [x] `src/pages/Settings.tsx` — формы: изменение display_name, bio, загрузка аватара (preview перед upload)
- [x] `src/components/UserList.tsx` — переиспользуемый список пользователей (для followers, following, search results) с cursor pagination
- [x] `src/components/SearchUsers.tsx` — input с debounce (300ms), dropdown с результатами, клик → переход на профиль
- [x] Follow/unfollow: optimistic update на кнопке и счётчиках

---

## 5.10 Frontend: пост и комментарии (дни 7–8)

- [x] `src/pages/PostPage.tsx` — полный пост + дерево комментариев
- [x] `src/components/CommentTree.tsx` — рекурсивный рендер: отступ = depth * 24px, вертикальная линия для визуализации вложенности
- [x] `src/components/CommentItem.tsx` — автор, контент, время, like button, кнопка «Ответить»
- [x] `src/components/CommentForm.tsx` — inline textarea, появляется при клике «Ответить», отправка → prepend в дерево
- [x] Свёртывание/развёртывание веток: кнопка collapse/expand на комментариях с children
- [x] Deleted комментарии: серый текст `[удалено]`, children по-прежнему отображаются
- [x] Cursor pagination на root-level комментариях: «Загрузить ещё»

---

# Фаза 4: Интеграция и финальная сборка (11 апреля – 15 апреля)

---

## 5.11 Интеграция и финальная сборка

### 5.11.1 docker-compose проверка (день 1)

- [ ] `docker-compose up --build` — все контейнеры поднимаются: frontend, api-gateway, user-svc, post-svc, comment-svc, media-svc, postgres, redis, minio
- [ ] Health checks проходят на всех сервисах в течение 60 секунд
- [ ] gRPC-соединения между gateway и сервисами устанавливаются без ошибок
- [ ] Frontend доступен на localhost:3000
- [ ] MinIO console доступен, bucket создан

### 5.11.2 Smoke test сценарий (дни 1–2)

- [ ] Регистрация нового пользователя → получение JWT → корректный 201
- [ ] Логин → корректный 200 + tokens
- [ ] Создание поста → 201, пост появляется в ленте автора
- [ ] Регистрация второго пользователя → follow первого → лента второго содержит посты первого
- [ ] Лайк поста → likes_count увеличился
- [ ] Создание комментария → comments_count увеличился
- [ ] Создание вложенного комментария (depth 1,2,3,4,5) → все отображаются
- [ ] Попытка комментария depth 6 → ошибка
- [ ] WebSocket: второй пользователь подключён, первый создаёт пост → второй получает event
- [ ] Upload изображения → URL доступен, отображается в посте
- [ ] Soft delete поста → пост исчезает из ленты, GET возвращает 404
- [ ] Rate limit: 101-й запрос в минуту → 429
- [ ] Expired JWT → 401 → refresh → новый access token → повтор запроса успешен
- [ ] Гостевой доступ: GET /api/posts/:id без JWT → 200 (публичный доступ)
- [ ] Cursor pagination: создать 25 постов → первый запрос возвращает 20 + cursor → второй запрос с cursor возвращает 5

### 5.11.3 Edge cases и error handling (день 2)

- [ ] Duplicate registration (email/username) → 409
- [ ] Like уже лайкнутого поста → idempotent, без ошибки или 409
- [ ] Follow самого себя → 400
- [ ] Удаление чужого поста → 403
- [ ] Пост с пустым content → 400
- [ ] Пост с content > 280 символов → 400
- [ ] Upload файла > 10MB → 413
- [ ] Upload файла с неподдерживаемым MIME → 400
- [ ] Запрос несуществующего пользователя → 404
- [ ] Невалидный cursor → 400

### 5.11.4 README и документация (день 3)

- [ ] README.md в корне репозитория:
  - Описание проекта и архитектуры (краткое, со ссылкой на диаграмму компонентов)
  - Стек технологий
  - Инструкция по запуску: `git clone` → `cp .env.example .env` → `docker-compose up --build`
  - Переменные окружения: таблица с описанием каждой переменной и значениями по умолчанию
  - API endpoints: таблица или ссылка на отдельный файл
  - Структура репозитория
  - Известные ограничения
- [ ] .env.example с дефолтными значениями для локального запуска
- [ ] Комментарии в docker-compose.yml

### 5.11.5 Финальная ревизия кода (день 3)

- [ ] Все TODO и FIXME обработаны
- [ ] Нет hardcoded значений — всё через env
- [ ] Нет panic() в production коде — все ошибки обрабатываются
- [ ] Логирование единообразное: structured JSON, slog
- [ ] Все gRPC ошибки используют `status.Errorf` с правильными кодами
- [ ] Все SQL-запросы parameterized
- [ ] Нет утечек goroutine — все context propagated, все goroutine завершаются при shutdown

---

# Критерии готовности (Definition of Done)

- Все чекбоксы в этапах 5.1–5.11 отмечены
- `docker-compose up --build` поднимает систему с нуля за < 3 минут
- Smoke test сценарий (5.11.2) проходит полностью
- Все edge cases (5.11.3) обработаны корректно
- README содержит инструкцию по запуску, которой может следовать человек без контекста
- Нет критических ошибок в логах при нормальной работе
- REST API отвечает за < 200мс, gRPC < 50мс (на localhost)

---

# Справочник: Protobuf-контракты

## common.proto

```protobuf
syntax = "proto3";
package common;
option go_package = "github.com/yourorg/microtwitter/proto/common";

import "google/protobuf/timestamp.proto";

message PaginationRequest {
  string cursor = 1;
  int32  limit  = 2; // default 20, max 100
}

message PaginationResponse {
  string next_cursor = 1;
  bool   has_more    = 2;
}

message Error {
  int32  code    = 1;
  string message = 2;
  string details = 3;
}
```

## user.proto

```protobuf
syntax = "proto3";
package user;
option go_package = "github.com/yourorg/microtwitter/proto/user";

import "google/protobuf/timestamp.proto";
import "common.proto";

message User {
  string id           = 1;
  string username     = 2;
  string email        = 3;
  string display_name = 4;
  string bio          = 5;
  string avatar_url   = 6;
  google.protobuf.Timestamp created_at = 7;
}

message RegisterRequest {
  string username = 1;
  string email    = 2;
  string password = 3;
}
message RegisterResponse {
  User   user          = 1;
  string access_token  = 2;
  string refresh_token = 3;
}

message LoginRequest {
  string email    = 1;
  string password = 2;
}
message LoginResponse {
  User   user          = 1;
  string access_token  = 2;
  string refresh_token = 3;
}

message RefreshTokenRequest  { string refresh_token = 1; }
message RefreshTokenResponse {
  string access_token  = 1;
  string refresh_token = 2;
}

message GetUserRequest  { string user_id = 1; }
message GetUserResponse { User user = 1; }

message UpdateUserRequest {
  string user_id      = 1;
  optional string display_name = 2;
  optional string bio          = 3;
  optional string avatar_url   = 4;
}
message UpdateUserResponse { User user = 1; }

message SoftDeleteUserRequest  { string user_id = 1; }
message SoftDeleteUserResponse {}

message SearchUsersRequest {
  string query = 1;
  common.PaginationRequest pagination = 2;
}
message SearchUsersResponse {
  repeated User users = 1;
  common.PaginationResponse pagination = 2;
}

message GetUsersByIDsRequest  { repeated string user_ids = 1; }
message GetUsersByIDsResponse { repeated User users = 1; }

message FollowRequest   { string follower_id = 1; string following_id = 2; }
message UnfollowRequest { string follower_id = 1; string following_id = 2; }
message FollowResponse  {}
message UnfollowResponse {}

message GetFollowersRequest {
  string user_id = 1;
  common.PaginationRequest pagination = 2;
}
message GetFollowersResponse {
  repeated User users = 1;
  common.PaginationResponse pagination = 2;
}

service UserService {
  rpc Register(RegisterRequest)             returns (RegisterResponse);
  rpc Login(LoginRequest)                   returns (LoginResponse);
  rpc RefreshToken(RefreshTokenRequest)     returns (RefreshTokenResponse);
  rpc GetUser(GetUserRequest)               returns (GetUserResponse);
  rpc UpdateUser(UpdateUserRequest)         returns (UpdateUserResponse);
  rpc SoftDeleteUser(SoftDeleteUserRequest) returns (SoftDeleteUserResponse);
  rpc SearchUsers(SearchUsersRequest)       returns (SearchUsersResponse);
  rpc GetUsersByIDs(GetUsersByIDsRequest)   returns (GetUsersByIDsResponse);
  rpc Follow(FollowRequest)                 returns (FollowResponse);
  rpc Unfollow(UnfollowRequest)             returns (UnfollowResponse);
  rpc GetFollowers(GetFollowersRequest)     returns (GetFollowersResponse);
  rpc GetFollowing(GetFollowersRequest)     returns (GetFollowersResponse);
}
```

## post.proto

```protobuf
syntax = "proto3";
package post;
option go_package = "github.com/yourorg/microtwitter/proto/post";

import "google/protobuf/timestamp.proto";
import "common.proto";

message Post {
  string id             = 1;
  string user_id        = 2;
  string content        = 3;
  string media_url      = 4;
  int64  likes_count    = 5;
  int64  comments_count = 6;
  google.protobuf.Timestamp created_at = 7;
  google.protobuf.Timestamp updated_at = 8;
}

message CreatePostRequest {
  string user_id   = 1;
  string content   = 2;
  string media_url = 3;
}
message CreatePostResponse { Post post = 1; }

message GetPostRequest  { string post_id = 1; }
message GetPostResponse { Post post = 1; }

message DeletePostRequest {
  string post_id = 1;
  string user_id = 2;
}
message DeletePostResponse {}

message GetFeedRequest {
  string user_id = 1;
  common.PaginationRequest pagination = 2;
}
message GetFeedResponse {
  repeated Post posts = 1;
  common.PaginationResponse pagination = 2;
}

message LikePostRequest   { string post_id = 1; string user_id = 2; }
message LikePostResponse  { int64 likes_count = 1; }
message UnlikePostRequest  { string post_id = 1; string user_id = 2; }
message UnlikePostResponse { int64 likes_count = 1; }

message GetPostsByUserRequest {
  string user_id = 1;
  common.PaginationRequest pagination = 2;
}
message GetPostsByUserResponse {
  repeated Post posts = 1;
  common.PaginationResponse pagination = 2;
}

service PostService {
  rpc CreatePost(CreatePostRequest)         returns (CreatePostResponse);
  rpc GetPost(GetPostRequest)               returns (GetPostResponse);
  rpc DeletePost(DeletePostRequest)         returns (DeletePostResponse);
  rpc GetFeed(GetFeedRequest)               returns (GetFeedResponse);
  rpc LikePost(LikePostRequest)             returns (LikePostResponse);
  rpc UnlikePost(UnlikePostRequest)         returns (UnlikePostResponse);
  rpc GetPostsByUser(GetPostsByUserRequest) returns (GetPostsByUserResponse);
}
```

## comment.proto

```protobuf
syntax = "proto3";
package comment;
option go_package = "github.com/yourorg/microtwitter/proto/comment";

import "google/protobuf/timestamp.proto";
import "common.proto";

message Comment {
  string id          = 1;
  string post_id     = 2;
  string user_id     = 3;
  string parent_id   = 4; // empty string = root comment
  string content     = 5;
  int64  likes_count = 6;
  int32  depth       = 7;
  google.protobuf.Timestamp created_at = 8;
  google.protobuf.Timestamp updated_at = 9;
  repeated Comment children = 10;
}

message CreateCommentRequest {
  string post_id   = 1;
  string user_id   = 2;
  string parent_id = 3;
  string content   = 4;
}
message CreateCommentResponse { Comment comment = 1; }

message GetCommentTreeRequest {
  string post_id = 1;
  common.PaginationRequest pagination = 2;
}
message GetCommentTreeResponse {
  repeated Comment comments = 1;
  common.PaginationResponse pagination = 2;
}

message DeleteCommentRequest {
  string comment_id = 1;
  string user_id    = 2;
}
message DeleteCommentResponse {}

message LikeCommentRequest    { string comment_id = 1; string user_id = 2; }
message LikeCommentResponse   { int64 likes_count = 1; }
message UnlikeCommentRequest  { string comment_id = 1; string user_id = 2; }
message UnlikeCommentResponse { int64 likes_count = 1; }

service CommentService {
  rpc CreateComment(CreateCommentRequest)     returns (CreateCommentResponse);
  rpc GetCommentTree(GetCommentTreeRequest)   returns (GetCommentTreeResponse);
  rpc DeleteComment(DeleteCommentRequest)     returns (DeleteCommentResponse);
  rpc LikeComment(LikeCommentRequest)         returns (LikeCommentResponse);
  rpc UnlikeComment(UnlikeCommentRequest)     returns (UnlikeCommentResponse);
}
```

## media.proto

```protobuf
syntax = "proto3";
package media;
option go_package = "github.com/yourorg/microtwitter/proto/media";

import "google/protobuf/timestamp.proto";

message Media {
  string id           = 1;
  string user_id      = 2;
  string post_id      = 3;
  string bucket       = 4;
  string object_key   = 5;
  string content_type = 6;
  string url          = 7;
  google.protobuf.Timestamp created_at = 8;
}

message UploadMediaRequest {
  string user_id      = 1;
  string post_id      = 2;
  string filename     = 3;
  string content_type = 4;
  bytes  data         = 5;
}
message UploadMediaResponse {
  Media  media = 1;
  string url   = 2;
}

message GetPresignedUploadURLRequest {
  string user_id      = 1;
  string filename     = 2;
  string content_type = 3;
}
message GetPresignedUploadURLResponse {
  string upload_url  = 1;
  string object_key  = 2;
  string media_id    = 3;
}

message GetMediaURLRequest  { string media_id = 1; }
message GetMediaURLResponse { string url = 1; }

message DeleteMediaRequest {
  string media_id = 1;
  string user_id  = 2;
}
message DeleteMediaResponse {}

service MediaService {
  rpc UploadMedia(UploadMediaRequest)                     returns (UploadMediaResponse);
  rpc GetPresignedUploadURL(GetPresignedUploadURLRequest) returns (GetPresignedUploadURLResponse);
  rpc GetMediaURL(GetMediaURLRequest)                     returns (GetMediaURLResponse);
  rpc DeleteMedia(DeleteMediaRequest)                     returns (DeleteMediaResponse);
}
```

---

# Справочник: Схема БД (PostgreSQL DDL)

```sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE TABLE users (
    id            UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    username      VARCHAR(50)  NOT NULL,
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(100) NOT NULL DEFAULT '',
    bio           VARCHAR(500) NOT NULL DEFAULT '',
    avatar_url    TEXT         NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ,
    CONSTRAINT users_username_unique UNIQUE (username),
    CONSTRAINT users_email_unique    UNIQUE (email)
);

CREATE UNIQUE INDEX idx_users_username_active ON users(username) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX idx_users_email_active    ON users(email)    WHERE deleted_at IS NULL;
CREATE INDEX idx_users_username_trgm     ON users USING gin(username      gin_trgm_ops);
CREATE INDEX idx_users_display_name_trgm ON users USING gin(display_name  gin_trgm_ops);

CREATE TABLE posts (
    id             UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID         NOT NULL REFERENCES users(id),
    content        VARCHAR(280) NOT NULL,
    media_url      TEXT         NOT NULL DEFAULT '',
    likes_count    INT          NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    comments_count INT          NOT NULL DEFAULT 0 CHECK (comments_count >= 0),
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at     TIMESTAMPTZ
);

CREATE INDEX idx_posts_user_created ON posts(user_id, created_at DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_posts_created_at   ON posts(created_at DESC)           WHERE deleted_at IS NULL;

CREATE TABLE comments (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    post_id     UUID         NOT NULL REFERENCES posts(id),
    user_id     UUID         NOT NULL REFERENCES users(id),
    parent_id   UUID         REFERENCES comments(id),
    content     VARCHAR(500) NOT NULL,
    likes_count INT          NOT NULL DEFAULT 0 CHECK (likes_count >= 0),
    depth       INT          NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT comments_max_depth CHECK (depth <= 5)
);

CREATE INDEX idx_comments_post_root ON comments(post_id, created_at ASC) WHERE deleted_at IS NULL AND parent_id IS NULL;
CREATE INDEX idx_comments_parent    ON comments(parent_id, created_at ASC) WHERE deleted_at IS NULL AND parent_id IS NOT NULL;
CREATE INDEX idx_comments_user      ON comments(user_id) WHERE deleted_at IS NULL;

CREATE TABLE likes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id),
    post_id    UUID        REFERENCES posts(id),
    comment_id UUID        REFERENCES comments(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT likes_target_xor CHECK (
        (post_id IS NOT NULL AND comment_id IS NULL) OR
        (post_id IS NULL     AND comment_id IS NOT NULL)
    ),
    CONSTRAINT likes_post_uniq    UNIQUE (user_id, post_id),
    CONSTRAINT likes_comment_uniq UNIQUE (user_id, comment_id)
);

CREATE INDEX idx_likes_post_id    ON likes(post_id)    WHERE post_id    IS NOT NULL;
CREATE INDEX idx_likes_comment_id ON likes(comment_id) WHERE comment_id IS NOT NULL;

CREATE TABLE follows (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    follower_id  UUID        NOT NULL REFERENCES users(id),
    following_id UUID        NOT NULL REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT follows_no_self CHECK  (follower_id <> following_id),
    CONSTRAINT follows_unique  UNIQUE (follower_id, following_id)
);

CREATE INDEX idx_follows_follower  ON follows(follower_id);
CREATE INDEX idx_follows_following ON follows(following_id);

CREATE TABLE media (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID         NOT NULL REFERENCES users(id),
    post_id      UUID         REFERENCES posts(id),
    bucket       VARCHAR(63)  NOT NULL,
    object_key   TEXT         NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_media_post ON media(post_id) WHERE post_id IS NOT NULL;
CREATE INDEX idx_media_user ON media(user_id);

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$;

CREATE TRIGGER trg_users_updated_at    BEFORE UPDATE ON users    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_posts_updated_at    BEFORE UPDATE ON posts    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER trg_comments_updated_at BEFORE UPDATE ON comments FOR EACH ROW EXECUTE FUNCTION set_updated_at();
```

---

# Справочник: Redis-структуры

| Ключ | Тип | Содержимое | TTL | Операции |
|------|-----|------------|-----|----------|
| `feed:{user_id}` | Sorted Set | member = post_id, score = Unix timestamp (ms) | 60s | ZADD, ZREVRANGEBYSCORE, DEL при follow/post/delete |
| `post:likes:{post_id}` | String | INT64 счётчик лайков | 300s | INCR, DECR, GET, SET при синхронизации с БД |
| `rate_limit:{ip}:{endpoint}` | String | счётчик запросов в окне | per endpoint (12–60s) | INCR + EXPIRE при первом запросе в окне |
| `ws:channels:{user_id}` | Pub/Sub channel | JSON события: new_post, new_like, new_comment, new_follower | нет TTL | PUBLISH из сервисов, SUBSCRIBE в Gateway WS Hub |
| `refresh:{token_hash}` | String | user_id | 7d | SET EX при выдаче, DEL при использовании (rotation) |

Инвалидация кэша ленты:
- Создание нового поста: DEL feed всех подписчиков автора
- Soft delete поста: DEL feed всех подписчиков + ZREM из существующих Sorted Set
- Follow/Unfollow: DEL feed:{follower_id}

Синхронизация счётчика лайков с БД: PostSvc периодически (раз в 30s) или при eviction сбрасывает накопленные INCR/DECR из Redis в PostgreSQL через `UPDATE posts SET likes_count = $1 WHERE id = $2`.

---

# Справочник: API Endpoints

## REST (frontend → API Gateway)

**Auth:**
- POST /api/auth/register
- POST /api/auth/login
- POST /api/auth/refresh

**Users:**
- GET /api/users/:id
- PUT /api/users/:id
- DELETE /api/users/:id (soft)
- GET /api/users/:id/posts?cursor=
- GET /api/users/search?q=&cursor=

**Posts:**
- POST /api/posts
- GET /api/posts/:id
- DELETE /api/posts/:id (soft)
- GET /api/feed?cursor=

**Comments:**
- POST /api/posts/:id/comments
- GET /api/posts/:id/comments?cursor= (дерево)
- DELETE /api/comments/:id (soft)

**Likes:**
- POST /api/posts/:id/like
- DELETE /api/posts/:id/like
- POST /api/comments/:id/like
- DELETE /api/comments/:id/like

**Follows:**
- POST /api/users/:id/follow
- DELETE /api/users/:id/follow
- GET /api/users/:id/followers?cursor=
- GET /api/users/:id/following?cursor=

**Media:**
- POST /api/upload

## WebSocket

WS /api/ws — авторизация через JWT в query param. События от сервера: `new_post`, `new_like`, `new_comment`, `new_follower`, `post_deleted`.

## gRPC (между сервисами)

- **UserService:** Register, Login, RefreshToken, GetUser, UpdateUser, SoftDeleteUser, SearchUsers, GetUsersByIDs, Follow, Unfollow, GetFollowers, GetFollowing
- **PostService:** CreatePost, GetPost, DeletePost, GetFeed, LikePost, UnlikePost, GetPostsByUser
- **CommentService:** CreateComment, GetCommentTree, DeleteComment, LikeComment, UnlikeComment
- **MediaService:** UploadMedia, GetPresignedUploadURL, GetMediaURL, DeleteMedia

---

# Нефункциональные требования (справка)

- REST API < 200мс
- gRPC между сервисами < 50мс
- Всё в контейнерах
- Бэкенд масштабируется горизонтально
- Горячие данные кэшируются в Redis — лента TTL 60с, счётчики лайков TTL 300с
- Structured JSON логи в stdout/stderr
- Soft delete на постах, комментариях, пользователях
- Cursor-based pagination на всех списках

## Безопасность и rate limiting

- Token bucket через Redis — 100 req/min для авторизованных, 30 req/min для гостей
- Auth-эндпоинты ограничены до 5 req/min
- Upload ограничен до 10 req/min
- CORS с whitelist фронтенд-домена
- Валидация и санитизация всех входящих данных
- HTTP security headers — X-Content-Type-Options, X-Frame-Options, CSP
- Max body size — 1MB текст, 10MB upload
- Parameterized queries везде
- Санитизация пользовательского контента на выходе от XSS
