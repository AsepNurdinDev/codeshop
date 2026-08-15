.PHONY: build run test test-race lint proto docker-build migrate-up migrate-down

APP_NAME=auth-service
DB_URL?=postgres://postgres:postgres@localhost:5432/codeshop_auth?sslmode=disable

build:
	go build -o bin/$(APP_NAME) ./cmd/server

run:
	go run ./cmd/server/main.go

test:
	go test -v ./...

test-race:
	go test -v -race ./...

lint:
	go vet ./...
	go fmt ./...

proto:
	mkdir -p proto/v1
	protoc --proto_path=proto --go_out=proto/v1 --go_opt=paths=source_relative --go-grpc_out=proto/v1 --go-grpc_opt=paths=source_relative proto/auth.proto

docker-build:
	docker build -t codeshop/$(APP_NAME):latest .

migrate-up:
	@echo "Applying migrations..."
	@which migrate > /dev/null || (echo "golang-migrate not installed, please install golang-migrate" && exit 1)
	migrate -path migrations -database "$(DB_URL)" up

migrate-down:
	@echo "Rolling back migrations..."
	@which migrate > /dev/null || (echo "golang-migrate not installed, please install golang-migrate" && exit 1)
	migrate -path migrations -database "$(DB_URL)" down 1
