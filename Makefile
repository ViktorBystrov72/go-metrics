.PHONY: build build-with-version clean test vet

# Переменные для версии
VERSION ?= N/A
BUILD_DATE ?= $(shell date '+%Y-%m-%d %H:%M:%S')
BUILD_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo N/A)

# Флаги линковщика для установки версии
LDFLAGS = -X main.buildVersion=$(VERSION) -X 'main.buildDate=$(BUILD_DATE)' -X main.buildCommit=$(BUILD_COMMIT)

# Сборка всех приложений
build: build-server build-agent

# Сборка сервера
build-server:
	go build -ldflags "$(LDFLAGS)" -o bin/server cmd/server/main.go

# Сборка агента
build-agent:
	go build -ldflags "$(LDFLAGS)" -o bin/agent cmd/agent/main.go

# Сборка с версией
build-with-version: VERSION = v1.0.0
build-with-version: build

# Очистка
clean:
	rm -f bin/server bin/agent bin/server_with_version

# Запуск тестов
test:
	go test ./... -v -timeout=10s

# Проверка кода
vet:
	go vet -vettool=./statictest-darwin ./...

# Проверка покрытия
coverage:
	go test ./... -v -timeout=10s -coverprofile=coverage.out
	go tool cover -func=coverage.out | tail -1

# Форматирование кода
fmt:
	go fmt ./...
	goimports -w .

# Установка зависимостей
deps:
	go mod download
	go mod tidy
