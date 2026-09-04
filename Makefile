.PHONY: dev-api dev-worker dev-frontend test vet docker-up setup-aws env

env:
	@test -f .env || cp .env.example .env

dev-api: env
	go run ./cmd/api

dev-worker: env
	go run ./cmd/worker

dev-frontend:
	cd frontend && npm run dev

test:
	go test ./...

vet:
	go vet ./...

docker-up:
	docker compose up -d

setup-aws:
	bash scripts/setup-local-aws.sh

build:
	go build -o bin/api ./cmd/api
	go build -o bin/worker ./cmd/worker
