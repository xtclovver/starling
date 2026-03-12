# Этап 4: Репозиторий, Docker, SQL-схемы — План для Claude Code

Дедлайн: 17 марта 2026
Стек: Go, React (Vite), PostgreSQL, Redis, MinIO, Docker, gRPC, buf

---

## Фаза 1: Инициализация репозитория и структура каталогов [DONE]

**Prompt 1.1 — Создание структуры проекта**

```
Создай структуру каталогов для микросервисного Go-проекта (микроблогинг-платформа). Инициализируй git-репозиторий.

Структура:

/
├── frontend/
│   ├── src/
│   ├── public/
│   └── package.json
├── api-gateway/
│   ├── cmd/gateway/main.go
│   ├── internal/
│   │   ├── handler/
│   │   ├── ws/
│   │   ├── middleware/
│   │   └── grpc_client/
│   └── go.mod
├── services/
│   ├── user-svc/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── repository/
│   │   │   ├── service/
│   │   │   └── server/
│   │   └── go.mod
│   ├── post-svc/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── repository/
│   │   │   ├── service/
│   │   │   └── server/
│   │   └── go.mod
│   ├── comment-svc/
│   │   ├── cmd/server/main.go
│   │   ├── internal/
│   │   │   ├── repository/
│   │   │   ├── service/
│   │   │   └── server/
│   │   └── go.mod
│   └── media-svc/
│       ├── cmd/server/main.go
│       ├── internal/
│       │   ├── repository/
│       │   ├── service/
│       │   └── server/
│       └── go.mod
├── proto/
│   ├── common.proto
│   ├── user.proto
│   ├── post.proto
│   ├── comment.proto
│   └── media.proto
├── migrations/
├── scripts/
├── .env.example
├── .gitignore
├── docker-compose.yml
├── buf.yaml
├── buf.gen.yaml
├── Makefile
└── README.md

Каждый Go-модуль (api-gateway и каждый сервис в services/) — отдельный go.mod с module path github.com/usedcvnt/microtwitter/<module-name>.

В main.go каждого сервиса — заглушка: log.Println("starting <service-name>...") и select{}.

Создай .gitignore для Go, Node.js, IDE, .env, /gen/.

Создай .env.example с переменными:
POSTGRES_USER=postgres
POSTGRES_PASSWORD=postgres
POSTGRES_DB=microtwitter
REDIS_URL=redis://redis:6379
MINIO_ROOT_USER=minioadmin
MINIO_ROOT_PASSWORD=minioadmin
MINIO_BUCKET=media
JWT_SECRET=change-me-in-production
API_GATEWAY_PORT=8080
USER_SVC_GRPC_PORT=50051
POST_SVC_GRPC_PORT=50052
COMMENT_SVC_GRPC_PORT=50053
MEDIA_SVC_GRPC_PORT=50054
```

---

## Фаза 2: Protobuf-контракты и buf [DONE] (proto в v1-директориях, buf lint + generate через Docker)

**Prompt 2.1 — Настройка buf и proto-файлы**

```
Настрой buf для генерации Go-кода из protobuf.

1. Создай buf.yaml в корне:
   - version: v2
   - modules: path: proto
   - lint rules: DEFAULT
   - breaking: FILE

2. Создай buf.gen.yaml:
   - plugins:
     - protocolbuffers/go с out: gen/go, opt: paths=source_relative
     - grpc/go с out: gen/go, opt: paths=source_relative

3. Заполни proto-файлы содержимым (все .proto уже определены в ТЗ):

proto/common.proto:
- package common
- go_package = "github.com/usedcvnt/microtwitter/gen/go/common"
- PaginationRequest (cursor string, limit int32)
- PaginationResponse (next_cursor string, has_more bool)
- Error (code int32, message string, details string)

proto/user.proto:
- package user
- go_package = "github.com/usedcvnt/microtwitter/gen/go/user"
- import common.proto и google/protobuf/timestamp.proto
- Messages: User, RegisterRequest/Response, LoginRequest/Response, RefreshTokenRequest/Response, GetUserRequest/Response, UpdateUserRequest/Response, SoftDeleteUserRequest/Response, SearchUsersRequest/Response, GetUsersByIDsRequest/Response, FollowRequest/Response, UnfollowRequest/Response, GetFollowersRequest/Response
- service UserService со всеми RPC (Register, Login, RefreshToken, GetUser, UpdateUser, SoftDeleteUser, SearchUsers, GetUsersByIDs, Follow, Unfollow, GetFollowers, GetFollowing)

proto/post.proto:
- package post
- go_package = "github.com/usedcvnt/microtwitter/gen/go/post"
- Messages: Post, CreatePostRequest/Response, GetPostRequest/Response, DeletePostRequest/Response, GetFeedRequest/Response, LikePostRequest/Response, UnlikePostRequest/Response, GetPostsByUserRequest/Response
- service PostService (CreatePost, GetPost, DeletePost, GetFeed, LikePost, UnlikePost, GetPostsByUser)

proto/comment.proto:
- package comment
- go_package = "github.com/usedcvnt/microtwitter/gen/go/comment"
- Messages: Comment (с repeated Comment children), CreateCommentRequest/Response, GetCommentTreeRequest/Response, DeleteCommentRequest/Response, LikeCommentRequest/Response, UnlikeCommentRequest/Response
- service CommentService (CreateComment, GetCommentTree, DeleteComment, LikeComment, UnlikeComment)

proto/media.proto:
- package media
- go_package = "github.com/usedcvnt/microtwitter/gen/go/media"
- Messages: Media, UploadMediaRequest/Response, GetPresignedUploadURLRequest/Response, GetMediaURLRequest/Response, DeleteMediaRequest/Response
- service MediaService (UploadMedia, GetPresignedUploadURL, GetMediaURL, DeleteMedia)

4. Добавь в Makefile таргет:

proto-gen:
	buf generate
proto-lint:
	buf lint

5. Запусти buf generate и убедись, что код генерируется в gen/go/ без ошибок.
```

---

## Фаза 3: SQL-миграции [DONE]

**Prompt 3.1 — Создание SQL-миграций (golang-migrate формат)**

```
Создай SQL-миграции в каталоге migrations/ в формате golang-migrate (пары файлов NNN_name.up.sql / NNN_name.down.sql).

001_users.up.sql:
- CREATE EXTENSION IF NOT EXISTS "pgcrypto"
- CREATE EXTENSION IF NOT EXISTS pg_trgm
- CREATE TABLE users (id UUID PK DEFAULT gen_random_uuid(), username VARCHAR(50) NOT NULL, email VARCHAR(255) NOT NULL, password_hash VARCHAR(255) NOT NULL, display_name VARCHAR(100) NOT NULL DEFAULT '', bio VARCHAR(500) NOT NULL DEFAULT '', avatar_url TEXT NOT NULL DEFAULT '', created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ, CONSTRAINT users_username_unique UNIQUE(username), CONSTRAINT users_email_unique UNIQUE(email))
- CREATE UNIQUE INDEX idx_users_username_active ON users(username) WHERE deleted_at IS NULL
- CREATE UNIQUE INDEX idx_users_email_active ON users(email) WHERE deleted_at IS NULL
- CREATE INDEX idx_users_username_trgm ON users USING gin(username gin_trgm_ops)
- CREATE INDEX idx_users_display_name_trgm ON users USING gin(display_name gin_trgm_ops)
- Триггер set_updated_at() + trg_users_updated_at

001_users.down.sql:
- DROP TRIGGER, DROP FUNCTION, DROP TABLE users

002_posts.up.sql:
- CREATE TABLE posts (id UUID PK DEFAULT gen_random_uuid(), user_id UUID NOT NULL REFERENCES users(id), content VARCHAR(280) NOT NULL, media_url TEXT NOT NULL DEFAULT '', likes_count INT NOT NULL DEFAULT 0 CHECK(likes_count >= 0), comments_count INT NOT NULL DEFAULT 0 CHECK(comments_count >= 0), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ)
- CREATE INDEX idx_posts_user_created ON posts(user_id, created_at DESC) WHERE deleted_at IS NULL
- CREATE INDEX idx_posts_created_at ON posts(created_at DESC) WHERE deleted_at IS NULL
- Триггер trg_posts_updated_at

002_posts.down.sql: DROP TABLE posts

003_comments.up.sql:
- CREATE TABLE comments (id UUID PK DEFAULT gen_random_uuid(), post_id UUID NOT NULL REFERENCES posts(id), user_id UUID NOT NULL REFERENCES users(id), parent_id UUID REFERENCES comments(id), content VARCHAR(500) NOT NULL, likes_count INT NOT NULL DEFAULT 0 CHECK(likes_count >= 0), depth INT NOT NULL DEFAULT 0, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), deleted_at TIMESTAMPTZ, CONSTRAINT comments_max_depth CHECK(depth <= 5))
- CREATE INDEX idx_comments_post_root ON comments(post_id, created_at ASC) WHERE deleted_at IS NULL AND parent_id IS NULL
- CREATE INDEX idx_comments_parent ON comments(parent_id, created_at ASC) WHERE deleted_at IS NULL AND parent_id IS NOT NULL
- CREATE INDEX idx_comments_user ON comments(user_id) WHERE deleted_at IS NULL
- Триггер trg_comments_updated_at

003_comments.down.sql: DROP TABLE comments

004_likes.up.sql:
- CREATE TABLE likes (id UUID PK DEFAULT gen_random_uuid(), user_id UUID NOT NULL REFERENCES users(id), post_id UUID REFERENCES posts(id), comment_id UUID REFERENCES comments(id), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), CONSTRAINT likes_target_xor CHECK((post_id IS NOT NULL AND comment_id IS NULL) OR (post_id IS NULL AND comment_id IS NOT NULL)), CONSTRAINT likes_post_uniq UNIQUE(user_id, post_id), CONSTRAINT likes_comment_uniq UNIQUE(user_id, comment_id))
- CREATE INDEX idx_likes_post_id ON likes(post_id) WHERE post_id IS NOT NULL
- CREATE INDEX idx_likes_comment_id ON likes(comment_id) WHERE comment_id IS NOT NULL

004_likes.down.sql: DROP TABLE likes

005_follows.up.sql:
- CREATE TABLE follows (id UUID PK DEFAULT gen_random_uuid(), follower_id UUID NOT NULL REFERENCES users(id), following_id UUID NOT NULL REFERENCES users(id), created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), CONSTRAINT follows_no_self CHECK(follower_id <> following_id), CONSTRAINT follows_unique UNIQUE(follower_id, following_id))
- CREATE INDEX idx_follows_follower ON follows(follower_id)
- CREATE INDEX idx_follows_following ON follows(following_id)

005_follows.down.sql: DROP TABLE follows

006_media.up.sql:
- CREATE TABLE media (id UUID PK DEFAULT gen_random_uuid(), user_id UUID NOT NULL REFERENCES users(id), post_id UUID REFERENCES posts(id), bucket VARCHAR(63) NOT NULL, object_key TEXT NOT NULL, content_type VARCHAR(100) NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT NOW())
- CREATE INDEX idx_media_post ON media(post_id) WHERE post_id IS NOT NULL
- CREATE INDEX idx_media_user ON media(user_id)

006_media.down.sql: DROP TABLE media

Убедись, что функция set_updated_at() создаётся в 001 и переиспользуется в 002 и 003 через CREATE TRIGGER ... EXECUTE FUNCTION set_updated_at(). DOWN-миграции должны DROP в обратном порядке зависимостей.
```

---

## Фаза 4: Dockerfiles [DONE]

**Prompt 4.1 — Dockerfile для Go-сервисов**

```
Создай multi-stage Dockerfiles для каждого Go-сервиса.

Шаблон (адаптируй пути для каждого):

api-gateway/Dockerfile:
- Stage 1 (builder): FROM golang:1.23-alpine, WORKDIR /build, COPY go.mod go.sum, RUN go mod download, COPY . ., COPY ../gen /build/gen (или используй go workspace), CGO_ENABLED=0 go build -o /app ./cmd/gateway
- Stage 2: FROM alpine:3.20, RUN apk add --no-cache ca-certificates, COPY --from=builder /app /app, EXPOSE 8080, ENTRYPOINT ["/app"]

Для каждого сервиса в services/ — аналогичный Dockerfile:
- services/user-svc/Dockerfile (EXPOSE 50051)
- services/post-svc/Dockerfile (EXPOSE 50052)
- services/comment-svc/Dockerfile (EXPOSE 50053)
- services/media-svc/Dockerfile (EXPOSE 50054)

Каждый билдится с CGO_ENABLED=0 GOOS=linux.

frontend/Dockerfile:
- Stage 1 (build): FROM node:20-alpine, WORKDIR /app, COPY package.json package-lock.json*, RUN npm ci, COPY . ., RUN npm run build
- Stage 2: FROM nginx:1.27-alpine, COPY --from=build /app/dist /usr/share/nginx/html, COPY nginx.conf /etc/nginx/conf.d/default.conf, EXPOSE 80

Создай frontend/nginx.conf:
- server на порту 80
- location / с try_files $uri $uri/ /index.html (SPA fallback)
- location /api/ с proxy_pass http://api-gateway:8080
- location /api/ws с proxy_pass, proxy_http_version 1.1, Upgrade и Connection headers для WebSocket

Важно: Go-сервисам нужен доступ к сгенерированному коду из gen/. Используй Go workspace (go.work) в корне или скопируй gen/ в контекст сборки. Добавь go.work в корне:
go 1.23
use (
    ./api-gateway
    ./services/user-svc
    ./services/post-svc
    ./services/comment-svc
    ./services/media-svc
)

Docker build context должен быть корнем репозитория (context: . в docker-compose), а dockerfile указывает на конкретный путь.
```

---

## Фаза 5: docker-compose.yml [DONE]

**Prompt 5.1 — Полный docker-compose**

```
Создай docker-compose.yml в корне проекта.

Сервисы:

postgres:
  image: postgres:16-alpine
  environment: POSTGRES_USER, POSTGRES_PASSWORD, POSTGRES_DB из .env
  volumes: pgdata:/var/lib/postgresql/data
  ports: "5432:5432"
  healthcheck: pg_isready -U $POSTGRES_USER -d $POSTGRES_DB, interval 5s, timeout 3s, retries 5

redis:
  image: redis:7-alpine
  command: redis-server --appendonly yes
  volumes: redisdata:/data
  ports: "6379:6379"
  healthcheck: redis-cli ping, interval 5s, timeout 3s, retries 5

minio:
  image: minio/minio:latest
  command: server /data --console-address ":9001"
  environment: MINIO_ROOT_USER, MINIO_ROOT_PASSWORD из .env
  volumes: miniodata:/data
  ports: "9000:9000", "9001:9001"
  healthcheck: curl -f http://localhost:9000/minio/health/live, interval 5s, timeout 3s, retries 5

minio-init:
  image: minio/mc:latest
  depends_on: minio (service_healthy)
  entrypoint: >
    /bin/sh -c "
    mc alias set local http://minio:9000 $$MINIO_ROOT_USER $$MINIO_ROOT_PASSWORD;
    mc mb --ignore-existing local/$$MINIO_BUCKET;
    mc anonymous set download local/$$MINIO_BUCKET;
    exit 0;
    "
  environment: MINIO_ROOT_USER, MINIO_ROOT_PASSWORD, MINIO_BUCKET из .env

migrate:
  image: migrate/migrate:v4.17.0
  depends_on: postgres (service_healthy)
  volumes: ./migrations:/migrations
  command: ["-path", "/migrations", "-database", "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable", "up"]

user-svc:
  build: context: ., dockerfile: services/user-svc/Dockerfile
  depends_on: postgres (service_healthy), redis (service_healthy), migrate (service_completed_successfully)
  environment: DB_URL=postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable, REDIS_URL, GRPC_PORT=50051, JWT_SECRET
  ports: "50051:50051"
  networks: [backend]
  healthcheck: grpc_health_probe или wget на readiness endpoint, interval 10s

post-svc:
  build: context: ., dockerfile: services/post-svc/Dockerfile
  depends_on: postgres, redis, migrate
  environment: DB_URL, REDIS_URL, GRPC_PORT=50052
  ports: "50052:50052"
  networks: [backend]

comment-svc:
  build: context: ., dockerfile: services/comment-svc/Dockerfile
  depends_on: postgres, redis, migrate
  environment: DB_URL, REDIS_URL, GRPC_PORT=50053
  ports: "50053:50053"
  networks: [backend]

media-svc:
  build: context: ., dockerfile: services/media-svc/Dockerfile
  depends_on: postgres, minio (service_healthy), minio-init (service_completed_successfully), migrate
  environment: DB_URL, MINIO_ENDPOINT=minio:9000, MINIO_ROOT_USER, MINIO_ROOT_PASSWORD, MINIO_BUCKET, GRPC_PORT=50054
  ports: "50054:50054"
  networks: [backend]

api-gateway:
  build: context: ., dockerfile: api-gateway/Dockerfile
  depends_on: user-svc, post-svc, comment-svc, media-svc, redis
  environment: PORT=8080, USER_SVC_ADDR=user-svc:50051, POST_SVC_ADDR=post-svc:50052, COMMENT_SVC_ADDR=comment-svc:50053, MEDIA_SVC_ADDR=media-svc:50054, REDIS_URL, JWT_SECRET
  ports: "8080:8080"
  networks: [backend]

frontend:
  build: context: ./frontend
  depends_on: [api-gateway]
  ports: "3000:80"
  networks: [backend]

networks:
  backend:
    driver: bridge

volumes:
  pgdata:
  redisdata:
  miniodata:

Все сервисы используют env_file: .env.
Health checks обязательны для postgres, redis, minio.
depends_on с condition: service_healthy / service_completed_successfully где нужно.
```

---

## Фаза 6: Makefile [DONE]

**Prompt 6.1 — Makefile с основными таргетами**

```
Создай Makefile в корне проекта со следующими таргетами:

.PHONY: proto-gen proto-lint build up down restart logs migrate-up migrate-down migrate-create test lint

proto-gen:
	buf generate

proto-lint:
	buf lint

build:
	docker compose build

up:
	docker compose up -d

down:
	docker compose down

restart:
	docker compose down && docker compose up -d --build

logs:
	docker compose logs -f

migrate-up:
	docker compose run --rm migrate -path /migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable" up

migrate-down:
	docker compose run --rm migrate -path /migrations -database "postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@postgres:5432/${POSTGRES_DB}?sslmode=disable" down 1

migrate-create:
	@read -p "Migration name: " name; \
	migrate create -ext sql -dir migrations -seq $$name

test:
	cd api-gateway && go test ./...
	cd services/user-svc && go test ./...
	cd services/post-svc && go test ./...
	cd services/comment-svc && go test ./...
	cd services/media-svc && go test ./...

lint:
	cd api-gateway && golangci-lint run
	cd services/user-svc && golangci-lint run
	cd services/post-svc && golangci-lint run
	cd services/comment-svc && golangci-lint run
	cd services/media-svc && golangci-lint run
```

---

## Фаза 7: Frontend scaffold [DONE] (без Tailwind)

**Prompt 7.1 — Инициализация React-проекта**

```
Инициализируй React-проект в frontend/:

1. npm create vite@latest . -- --template react-ts
2. npm install
3. npm install react-router-dom zustand axios
4. npm install -D tailwindcss @tailwindcss/vite

5. Настрой tailwind — добавь @tailwindcss/vite в vite.config.ts plugins, добавь @import "tailwindcss" в index.css.

6. Настрой vite.config.ts:
   - proxy: /api → http://localhost:8080, /api/ws → ws://localhost:8080 (для dev-режима)

7. Создай базовую структуру:
   src/
   ├── api/         # axios instance с interceptors
   │   └── client.ts
   ├── components/  # переиспользуемые компоненты
   ├── pages/       # страницы
   │   ├── Home.tsx
   │   ├── Login.tsx
   │   ├── Register.tsx
   │   ├── Profile.tsx
   │   ├── PostPage.tsx
   │   └── Settings.tsx
   ├── store/       # zustand stores
   ├── hooks/       # custom hooks
   ├── types/       # TypeScript типы
   ├── App.tsx      # роутинг
   └── main.tsx

8. В App.tsx настрой React Router:
   / → Home, /login → Login, /register → Register, /profile/:id → Profile, /post/:id → PostPage, /settings → Settings

9. В api/client.ts — axios instance с baseURL "/api", interceptor для добавления JWT из памяти, interceptor для auto-refresh при 401.

Все страницы — заглушки с заголовком и текстом "TODO".
```

---

## Фаза 8: README и финальная проверка

**Prompt 8.1 — README.md**

```
Создай README.md в корне проекта:

# MicroTwitter

Микроблогинг-платформа (аналог Twitter/X). Микросервисная архитектура на Go + React.

## Стек
- Frontend: React, Vite, TypeScript, Tailwind CSS
- API Gateway: Go, chi, WebSocket
- Сервисы: Go, gRPC, protobuf
- БД: PostgreSQL 16
- Кэш/Pub-Sub: Redis 7
- Файлы: MinIO (S3-compatible)
- Инфраструктура: Docker, docker-compose
- Protobuf: buf

## Быстрый старт

```bash
cp .env.example .env
make proto-gen
make up
```

Frontend: http://localhost:3000
API: http://localhost:8080
MinIO Console: http://localhost:9001

## Структура проекта
(краткое описание каталогов — 1 строка на каждый)

## Команды
(таблица с make-таргетами и описаниями)

## Архитектура
Frontend (React) → REST/WS → API Gateway (Go) → gRPC → {UserSvc, PostSvc, CommentSvc, MediaSvc} → PostgreSQL/Redis/MinIO

## Переменные окружения
(таблица из .env.example с описанием каждой переменной)
```

**Prompt 8.2 — Проверка**

```
Проверь что:
1. docker compose config — валидный (нет синтаксических ошибок)
2. Все Dockerfile существуют по путям, указанным в docker-compose
3. Все proto-файлы проходят buf lint
4. buf generate генерирует код без ошибок
5. go.work корректный и все модули резолвятся
6. Каждый go-модуль компилируется: go build ./...
7. frontend: npm run build проходит без ошибок
8. Все SQL-миграции синтаксически корректны (можно проверить через подключение к postgres и выполнение)
9. .gitignore включает: .env, gen/, node_modules/, dist/, bin/, *.exe, .idea/, .vscode/, __debug_bin
10. Сделай git init, git add -A, git commit -m "stage 4: repo structure, docker, migrations, proto"
```

---

## Порядок выполнения (summary)

| # | Фаза | Что делает | Критерий готовности |
|---|------|-----------|-------------------|
| 1 | Repo structure | Каталоги, go.mod, go.work, .gitignore, .env.example | `tree` показывает полную структуру |
| 2 | Protobuf | buf.yaml, buf.gen.yaml, все .proto, `buf generate` | gen/go/ содержит сгенерированные .pb.go и _grpc.pb.go |
| 3 | Migrations | 6 пар .up.sql/.down.sql | Применяются к чистой PostgreSQL без ошибок |
| 4 | Dockerfiles | Multi-stage для каждого сервиса + frontend | `docker compose build` проходит |
| 5 | docker-compose | Полный стек с health checks | `docker compose up -d` поднимает все 10+ контейнеров |
| 6 | Makefile | Все таргеты | `make proto-gen && make build && make up` работает |
| 7 | Frontend scaffold | Vite + React + router + axios + tailwind | `npm run build` без ошибок, dev-сервер открывается |
| 8 | README + verify | Документация, финальная проверка | Всё собирается, lint проходит, commit сделан |
