.PHONY: proto-gen proto-lint build up down restart logs migrate-up migrate-down migrate-create test lint

proto-gen:
	MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd):/workspace" -w /workspace bufbuild/buf:latest generate

proto-lint:
	MSYS_NO_PATHCONV=1 docker run --rm -v "$$(pwd):/workspace" -w /workspace bufbuild/buf:latest lint

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
	docker compose run --rm migrate -path /migrations -database "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@postgres:5432/$${POSTGRES_DB}?sslmode=disable" up

migrate-down:
	docker compose run --rm migrate -path /migrations -database "postgres://$${POSTGRES_USER}:$${POSTGRES_PASSWORD}@postgres:5432/$${POSTGRES_DB}?sslmode=disable" down 1

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
