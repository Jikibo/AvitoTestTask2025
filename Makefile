
.PHONY: help build run test test-unit test-e2e docker-up docker-down clean

help:
	@echo "Доступные команды:"
	@echo "  make build       - Собрать приложение"
	@echo "  make run         - Запустить приложение локально"
	@echo "  make test        - Запустить все тесты"
	@echo "  make test-unit   - Запустить unit тесты"
	@echo "  make test-e2e    - Запустить e2e тесты"
	@echo "  make docker-up   - Запустить через docker-compose"
	@echo "  make docker-down - Остановить docker-compose"
	@echo "  make clean       - Очистить артефакты сборки"

build:
	go build -o bin/pr-reviewer-service ./cmd/server

run:
	go run ./cmd/server

test:
	go test -v ./...

test-unit:
	go test -v ./internal/...

test-e2e:
	go test -v ./tests/e2e/...

docker-up:
	docker-compose up --build

docker-down:
	docker-compose down -v

clean:
	rm -rf bin/
	rm -f coverage.out
	go clean -testcache

lint:
	golangci-lint run ./...

load-test:
	k6 run load_test.js
