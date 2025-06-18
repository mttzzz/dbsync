// Создадим Makefile для управления тестами и сборкой
.PHONY: test test-unit test-integration test-coverage build build-release clean help docker-build docker-test docker-clean

# Переменные
BINARY_NAME=dbsync
VERSION?=dev
BUILD_DIR=bin
GO_FILES=$(shell find . -name "*.go" -not -path "./vendor/*")
DOCKER_IMAGE=dbsync
DOCKER_TAG=latest

# Цвета для вывода
GREEN=\033[0;32m
YELLOW=\033[1;33m
RED=\033[0;31m
BLUE=\033[0;34m
NC=\033[0m # No Color

help: ## Показать справку
	@echo "$(BLUE)DB Sync CLI - Команды для разработки$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-20s$(NC) %s\n", $$1, $$2}'

# Сборка
build: ## Собрать бинарный файл
	@echo "$(YELLOW)Сборка $(BINARY_NAME)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbsync
	@echo "$(GREEN)Сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-release: ## Собрать release версию
	@echo "$(YELLOW)Сборка release версии $(VERSION)...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@go build -ldflags="-X 'db-sync-cli/internal/version.Version=$(VERSION)' -X 'db-sync-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'db-sync-cli/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/dbsync
	@echo "$(GREEN)Release сборка завершена: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

build-all: ## Собрать для всех платформ
	@echo "$(YELLOW)Сборка для всех платформ...$(NC)"
	@mkdir -p $(BUILD_DIR)
	@echo "Building for Linux AMD64..."
	@GOOS=linux GOARCH=amd64 go build -ldflags="-X 'db-sync-cli/internal/version.Version=$(VERSION)' -X 'db-sync-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'db-sync-cli/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./cmd/dbsync
	@echo "Building for Linux ARM64..."
	@GOOS=linux GOARCH=arm64 go build -ldflags="-X 'db-sync-cli/internal/version.Version=$(VERSION)' -X 'db-sync-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'db-sync-cli/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./cmd/dbsync
	@echo "Building for Windows AMD64..."
	@GOOS=windows GOARCH=amd64 go build -ldflags="-X 'db-sync-cli/internal/version.Version=$(VERSION)' -X 'db-sync-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'db-sync-cli/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./cmd/dbsync
	@echo "Building for macOS AMD64..."
	@GOOS=darwin GOARCH=amd64 go build -ldflags="-X 'db-sync-cli/internal/version.Version=$(VERSION)' -X 'db-sync-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'db-sync-cli/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 ./cmd/dbsync
	@echo "Building for macOS ARM64..."
	@GOOS=darwin GOARCH=arm64 go build -ldflags="-X 'db-sync-cli/internal/version.Version=$(VERSION)' -X 'db-sync-cli/internal/version.BuildDate=$(shell date -u +%Y-%m-%dT%H:%M:%SZ)' -X 'db-sync-cli/internal/version.GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'" -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./cmd/dbsync
	@echo "$(GREEN)Сборка для всех платформ завершена!$(NC)"

# Тестирование
test: test-unit ## Запустить все unit тесты
	@echo "$(GREEN)Все unit тесты пройдены!$(NC)"

test-unit: ## Запустить unit тесты
	@echo "$(YELLOW)Запуск unit тестов...$(NC)"
	@go test -v ./internal/config
	@go test -v ./internal/models  
	@go test -v ./internal/version
	@go test -v ./pkg/utils
	@go test -v ./internal/services
	@go test -v ./internal/cli
	@go test -v ./internal/ui

test-integration: ## Запустить интеграционные тесты (требует MySQL)
	@echo "$(YELLOW)Запуск интеграционных тестов...$(NC)"
	@echo "$(RED)Предупреждение: Интеграционные тесты требуют запущенного MySQL сервера$(NC)"
	@go test -v -tags=integration ./test/integration

test-coverage: ## Запустить тесты с покрытием
	@echo "$(YELLOW)Анализ покрытия кода...$(NC)"
	@go test -coverprofile=coverage.out ./internal/config ./internal/models ./internal/version ./pkg/utils ./internal/services ./internal/cli ./internal/ui
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)HTML отчёт создан: coverage.html$(NC)"

test-coverage-view: test-coverage ## Показать отчёт о покрытии в браузере
	@echo "$(YELLOW)Открываем отчёт о покрытии...$(NC)"
	@open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || start coverage.html 2>/dev/null || echo "Откройте coverage.html в браузере"

# Линтинг и проверки
lint: ## Запустить линтеры
	@echo "$(YELLOW)Запуск линтеров...$(NC)"
	@go vet ./...
	@gofmt -s -w .
	@which staticcheck > /dev/null && staticcheck ./... || echo "Установите staticcheck: go install honnef.co/go/tools/cmd/staticcheck@latest"

fmt: ## Форматировать код
	@echo "$(YELLOW)Форматирование кода...$(NC)"
	@gofmt -s -w .
	@which goimports > /dev/null && goimports -w . || echo "Установите goimports: go install golang.org/x/tools/cmd/goimports@latest"

# Docker команды
docker-build: ## Собрать Docker образ
	@echo "$(YELLOW)Сборка Docker образа...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker образ собран: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

docker-test: ## Запустить тестовое окружение в Docker
	@echo "$(YELLOW)Запуск тестового окружения...$(NC)"
	@docker-compose up -d mysql-local mysql-remote
	@echo "$(GREEN)Ожидание готовности баз данных...$(NC)"
	@sleep 30
	@echo "$(GREEN)Тестовое окружение готово!$(NC)"
	@echo "$(BLUE)MySQL Local:  localhost:3306$(NC)"
	@echo "$(BLUE)MySQL Remote: localhost:3307$(NC)"
	@echo "$(BLUE)Adminer:      http://localhost:8080$(NC)"

docker-test-with-adminer: ## Запустить тестовое окружение с Adminer
	@echo "$(YELLOW)Запуск тестового окружения с Adminer...$(NC)"
	@docker-compose --profile dev up -d
	@echo "$(GREEN)Ожидание готовности сервисов...$(NC)"
	@sleep 30
	@echo "$(GREEN)Тестовое окружение готово!$(NC)"
	@echo "$(BLUE)MySQL Local:  localhost:3306$(NC)"
	@echo "$(BLUE)MySQL Remote: localhost:3307$(NC)"
	@echo "$(BLUE)Adminer:      http://localhost:8080$(NC)"

docker-clean: ## Остановить и удалить Docker контейнеры
	@echo "$(YELLOW)Очистка Docker окружения...$(NC)"
	@docker-compose down -v
	@echo "$(GREEN)Docker окружение очищено$(NC)"

docker-run: docker-build ## Запустить приложение в Docker
	@echo "$(YELLOW)Запуск приложения в Docker...$(NC)"
	@docker run --rm -it $(DOCKER_IMAGE):$(DOCKER_TAG)

# Установка и зависимости
deps: ## Установить зависимости
	@echo "$(YELLOW)Установка зависимостей...$(NC)"
	@go mod download
	@go mod verify
	@echo "$(GREEN)Зависимости установлены$(NC)"

deps-update: ## Обновить зависимости
	@echo "$(YELLOW)Обновление зависимостей...$(NC)"
	@go get -u ./...
	@go mod tidy
	@echo "$(GREEN)Зависимости обновлены$(NC)"

# Очистка
clean: ## Очистить сборочные артефакты
	@echo "$(YELLOW)Очистка...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean
	@echo "$(GREEN)Очистка завершена$(NC)"

clean-all: clean docker-clean ## Полная очистка (включая Docker)

# Разработка
dev-setup: deps ## Настройка окружения для разработки
	@echo "$(YELLOW)Настройка окружения для разработки...$(NC)"
	@cp .env.example .env
	@echo "$(GREEN)Скопирован .env.example в .env$(NC)"
	@echo "$(BLUE)Отредактируйте .env файл под ваше окружение$(NC)"
	@echo "$(GREEN)Окружение для разработки готово!$(NC)"

run: build ## Собрать и запустить приложение
	@echo "$(YELLOW)Запуск приложения...$(NC)"
	@./$(BUILD_DIR)/$(BINARY_NAME) --help

# CI/CD
ci-test: deps lint test-unit ## Команды для CI (без интеграционных тестов)
	@echo "$(GREEN)CI тесты пройдены!$(NC)"

ci-build: ci-test build-all ## Полная сборка для CI
	@echo "$(GREEN)CI сборка завершена!$(NC)"

# Информация
version: ## Показать версию
	@echo "$(BLUE)DB Sync CLI$(NC)"
	@echo "Version: $(VERSION)"
	@echo "Build date: $(shell date -u +%Y-%m-%dT%H:%M:%SZ)"
	@echo "Git commit: $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)"

info: ## Показать информацию о проекте
	@echo "$(BLUE)=== DB Sync CLI Project Info ===$(NC)"
	@echo "Binary name: $(BINARY_NAME)"
	@echo "Version: $(VERSION)"
	@echo "Build directory: $(BUILD_DIR)"
	@echo "Docker image: $(DOCKER_IMAGE):$(DOCKER_TAG)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@$(MAKE) help
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)Отчет о покрытии сохранен: coverage.html$(NC)"

lint: ## Запустить линтер
	@echo "$(YELLOW)Запуск линтера...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint не установлен. Установите его: https://golangci-lint.run/usage/install/$(NC)"; \
		go vet ./...; \
	fi

format: ## Форматировать код
	@echo "$(YELLOW)Форматирование кода...$(NC)"
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w $(GO_FILES); \
	fi

clean: ## Очистить артефакты сборки
	@echo "$(YELLOW)Очистка...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "$(GREEN)Очистка завершена$(NC)"

deps: ## Установить зависимости
	@echo "$(YELLOW)Установка зависимостей...$(NC)"
	@go mod download
	@go mod tidy

deps-dev: ## Установить dev зависимости
	@echo "$(YELLOW)Установка dev зависимостей...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest

install: build ## Установить в $GOPATH/bin
	@echo "$(YELLOW)Установка $(BINARY_NAME)...$(NC)"
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(GOPATH)/bin/
	@echo "$(GREEN)$(BINARY_NAME) установлен в $(GOPATH)/bin/$(NC)"

run: build ## Собрать и запустить приложение
	@echo "$(YELLOW)Запуск $(BINARY_NAME)...$(NC)"
	@./$(BUILD_DIR)/$(BINARY_NAME) --help

demo: build ## Запустить демонстрацию
	@echo "$(YELLOW)Демонстрация команд $(BINARY_NAME)...$(NC)"
	@echo "1. Показать версию:"
	@./$(BUILD_DIR)/$(BINARY_NAME) version
	@echo "\n2. Показать конфигурацию:"
	@./$(BUILD_DIR)/$(BINARY_NAME) config --show
	@echo "\n3. Показать справку:"
	@./$(BUILD_DIR)/$(BINARY_NAME) --help

verify: format lint test ## Полная проверка (форматирование, линтинг, тесты)
	@echo "$(GREEN)Все проверки пройдены успешно!$(NC)"

release-prepare: verify build-release ## Подготовить release
	@echo "$(GREEN)Release готов: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

# Алиасы для удобства
t: test
b: build
r: run
c: clean
f: format
