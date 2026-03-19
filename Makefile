.PHONY: run test migrate-up migrate-down lint infra

infra:
	docker compose up -d

infra-down:
	docker compose down

run:
	go run cmd/server/main.go

test:
	go test ./... -v -count=1

migrate-up:
	migrate -path migrations -database "$(DB_DSN)" up

migrate-down:
	migrate -path migrations -database "$(DB_DSN)" down 1

lint:
	golangci-lint run ./...
