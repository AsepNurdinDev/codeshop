.PHONY: run build test test-verbose lint vet fmt tidy proto migrate-up migrate-down docker-build docker-build-dev compose-up compose-down clean

APP_NAME := auth-service
BIN_DIR := bin
MIGRATE := migrate -path ./migrations -database "$${DATABASE_URL}"

## run: run the service locally (requires local Postgres/Redis/RabbitMQ, see README)
run:
	go run ./cmd/server

## build: compile the auth-service binary into ./bin
build:
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BIN_DIR)/$(APP_NAME) ./cmd/server

## test: run all tests
test:
	go test ./...

## test-verbose: run all tests with verbose output
test-verbose:
	go test ./... -v

## test-cover: run tests with coverage report
test-cover:
	go test ./... -coverprofile=coverage.out
	go tool cover -func=coverage.out

## vet: run go vet
vet:
	go vet ./...

## fmt: format all Go source files
fmt:
	gofmt -w .

## fmt-check: fail if any file is not gofmt-formatted
fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "the following files are not gofmt'd:"; gofmt -l .; exit 1)

## lint: fmt-check + vet, the minimum bar for CI
lint: fmt-check vet

## tidy: sync go.mod/go.sum with the code
tidy:
	go mod tidy

## proto: regenerate pkg/pb from proto/auth.proto (requires protoc, protoc-gen-go, protoc-gen-go-grpc on PATH)
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		--go_opt=Mproto/auth.proto=github.com/codeshop-platform/auth-service/pkg/pb \
		--go-grpc_opt=Mproto/auth.proto=github.com/codeshop-platform/auth-service/pkg/pb \
		proto/auth.proto
	mv proto/auth.pb.go proto/auth_grpc.pb.go pkg/pb/

## migrate-up: apply all pending migrations (requires golang-migrate CLI + DATABASE_URL env var)
migrate-up:
	$(MIGRATE) up

## migrate-down: roll back the last migration
migrate-down:
	$(MIGRATE) down 1

## docker-build: build the production image
docker-build:
	docker build -t codeshop/auth-service:latest .

## docker-build-dev: build the hot-reload dev image
docker-build-dev:
	docker build -f Dockerfile.dev -t codeshop/auth-service:dev .

## compose-up: start auth-service + its dependencies (Postgres, Redis, RabbitMQ) via docker compose
compose-up:
	docker compose -f docker-compose.yml up --build

## compose-down: stop and remove the docker compose stack
compose-down:
	docker compose -f docker-compose.yml down -v

## clean: remove build artifacts
clean:
	rm -rf $(BIN_DIR) tmp coverage.out
