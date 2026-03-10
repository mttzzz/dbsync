.PHONY: help build build-release build-all run install test test-unit test-integration test-coverage test-coverage-view lint format fmt deps deps-update deps-dev clean clean-all verify release-prepare version info docker-build docker-test docker-test-with-adminer docker-clean docker-run ci-test ci-build

BINARY_NAME := dbsync
VERSION ?= dev
BUILD_DIR := bin
DOCKER_IMAGE := dbsync
DOCKER_TAG := latest
GO_FILES := $(shell find . -name "*.go" -not -path "./vendor/*" -not -path "./coverage/*")
UNIT_TEST_PACKAGES := ./internal/config ./internal/models ./internal/version ./pkg/utils ./internal/services ./internal/cli ./internal/ui ./internal/tui
INSTALL_DIR ?= $(or $(GOBIN),$(shell go env GOPATH)/bin)

GREEN := \033[0;32m
YELLOW := \033[1;33m
RED := \033[0;31m
BLUE := \033[0;34m
NC := \033[0m

help: ## Показать справку
	@echo "$(BLUE)DB Sync CLI - Команды для разработки$(NC)"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  $(GREEN)%-24s$(NC) %s\n", $$1, $$2}'

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

run: build ## Собрать и запустить приложение
	@echo "$(YELLOW)Запуск приложения...$(NC)"
	@./$(BUILD_DIR)/$(BINARY_NAME)

install: build ## Установить бинарь в bin-каталог Go
	@echo "$(YELLOW)Установка $(BINARY_NAME) в $(INSTALL_DIR)...$(NC)"
	@mkdir -p $(INSTALL_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_DIR)/$(BINARY_NAME)
	@echo "$(GREEN)$(BINARY_NAME) установлен: $(INSTALL_DIR)/$(BINARY_NAME)$(NC)"

test: test-unit ## Запустить unit тесты
	@echo "$(GREEN)Все unit тесты пройдены!$(NC)"

test-unit: ## Запустить unit тесты
	@echo "$(YELLOW)Запуск unit тестов...$(NC)"
	@go test -v $(UNIT_TEST_PACKAGES)

test-integration: ## Запустить интеграционные тесты (требует MySQL и DBSYNC_TEST_*)
	@echo "$(YELLOW)Запуск интеграционных тестов...$(NC)"
	@echo "$(RED)Предупреждение: интеграционные тесты требуют настроенных DBSYNC_TEST_* переменных$(NC)"
	@go test -v -tags=integration ./test/integration

test-coverage: ## Запустить тесты с покрытием
	@echo "$(YELLOW)Анализ покрытия кода...$(NC)"
	@go test -coverprofile=coverage.out $(UNIT_TEST_PACKAGES)
	@go tool cover -html=coverage.out -o coverage.html
	@echo "$(GREEN)HTML отчёт создан: coverage.html$(NC)"

test-coverage-view: test-coverage ## Показать отчёт о покрытии в браузере
	@echo "$(YELLOW)Открываем отчёт о покрытии...$(NC)"
	@open coverage.html 2>/dev/null || xdg-open coverage.html 2>/dev/null || echo "Откройте coverage.html в браузере"

lint: ## Запустить линтеры
	@echo "$(YELLOW)Запуск линтеров...$(NC)"
	@if command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "$(RED)golangci-lint не установлен, выполняю go vet ./...$(NC)"; \
		go vet ./...; \
	fi

format: ## Форматировать код
	@echo "$(YELLOW)Форматирование кода...$(NC)"
	@go fmt ./...
	@if command -v goimports >/dev/null 2>&1; then \
		goimports -w $(GO_FILES); \
	fi

fmt: format ## Алиас для format

deps: ## Установить зависимости
	@echo "$(YELLOW)Установка зависимостей...$(NC)"
	@go mod download
	@go mod verify

deps-update: ## Обновить зависимости
	@echo "$(YELLOW)Обновление зависимостей...$(NC)"
	@go get -u ./...
	@go mod tidy

deps-dev: ## Установить dev зависимости
	@echo "$(YELLOW)Установка dev зависимостей...$(NC)"
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@go install golang.org/x/tools/cmd/goimports@latest

clean: ## Очистить артефакты сборки
	@echo "$(YELLOW)Очистка...$(NC)"
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@go clean
	@echo "$(GREEN)Очистка завершена$(NC)"

clean-all: clean docker-clean ## Полная очистка, включая Docker

verify: format lint test ## Полная проверка
	@echo "$(GREEN)Все проверки пройдены успешно!$(NC)"

release-prepare: verify build-release ## Подготовить release
	@echo "$(GREEN)Release готов: $(BUILD_DIR)/$(BINARY_NAME)$(NC)"

version: ## Показать версию сборки
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
	@echo "Install directory: $(INSTALL_DIR)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@$(MAKE) help

docker-build: ## Собрать Docker образ
	@echo "$(YELLOW)Сборка Docker образа...$(NC)"
	@docker build -t $(DOCKER_IMAGE):$(DOCKER_TAG) .
	@echo "$(GREEN)Docker образ собран: $(DOCKER_IMAGE):$(DOCKER_TAG)$(NC)"

docker-test: ## Запустить тестовое окружение в Docker
	@echo "$(YELLOW)Запуск тестового окружения...$(NC)"
	@docker compose up -d mysql-local mysql-remote
	@echo "$(GREEN)Ожидание готовности баз данных...$(NC)"
	@sleep 30
	@echo "$(GREEN)Тестовое окружение готово!$(NC)"

docker-test-with-adminer: ## Запустить тестовое окружение с Adminer
	@echo "$(YELLOW)Запуск тестового окружения с Adminer...$(NC)"
	@docker compose --profile dev up -d
	@echo "$(GREEN)Ожидание готовности сервисов...$(NC)"
	@sleep 30
	@echo "$(GREEN)Тестовое окружение готово!$(NC)"

docker-clean: ## Остановить и удалить Docker контейнеры
	@echo "$(YELLOW)Очистка Docker окружения...$(NC)"
	@docker compose down -v
	@echo "$(GREEN)Docker окружение очищено$(NC)"

docker-run: docker-build ## Запустить приложение в Docker
	@echo "$(YELLOW)Запуск приложения в Docker...$(NC)"
	@docker run --rm -it $(DOCKER_IMAGE):$(DOCKER_TAG)

ci-test: deps lint test-unit ## Команды для CI без интеграционных тестов
	@echo "$(GREEN)CI тесты пройдены!$(NC)"

ci-build: ci-test build-all ## Полная сборка для CI
	@echo "$(GREEN)CI сборка завершена!$(NC)"
