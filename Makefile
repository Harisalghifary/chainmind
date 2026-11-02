.PHONY: up down run-backend run-frontend seed test lint build

up:
	docker compose up -d

down:
	docker compose down

run-backend:
	cd backend && go run cmd/server/main.go

run-frontend:
	cd frontend && npm run dev

seed:
	cd backend && go run scripts/seed_exploits.go

test:
	cd backend && go test ./... -v -cover

lint:
	cd backend && golangci-lint run ./...

build:
	cd frontend && npm run build