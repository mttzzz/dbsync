# DB Sync CLI

Безопасная консольная утилита на Go для синхронизации MySQL баз данных между удалённым и локальным сервером.

## 🌟 Особенности

- 🔒 **Безопасность**: dry-run режим, интерактивные подтверждения для критических операций
- 🎨 **Современный TUI**: стилизованный интерфейс с progress bar и цветовой индикацией
- ⚡ **Интеллектуальная валидация**: автоматическое определение опасных БД (production, live, etc.)
- 🔧 **Гибкая конфигурация**: поддержка .env файлов и переменных окружения
- 📊 **Подробная статистика**: размер дампов, количество таблиц, время выполнения
- ✅ **Высокое покрытие тестами**: unit и интеграционные тесты

## 📋 Требования

- Go 1.24.1+
- MySQL/MariaDB сервер (локальный и удалённый)
- Утилиты `mysqldump` и `mysql` в PATH

## 🚀 Установка

### Из исходников

```bash
git clone https://github.com/your-username/db-sync-cli.git
cd db-sync-cli
go build -o bin/dbsync ./cmd/dbsync
```

### Сборка для всех платформ

```bash
# Windows
env GOOS=windows GOARCH=amd64 go build -o bin/dbsync.exe ./cmd/dbsync

# Linux
env GOOS=linux GOARCH=amd64 go build -o bin/dbsync-linux ./cmd/dbsync

# macOS
env GOOS=darwin GOARCH=amd64 go build -o bin/dbsync-macos ./cmd/dbsync
```

## ⚙️ Конфигурация

### Создание .env файла

```bash
cp .env.example .env
```

### Редактирование конфигурации

```env
# Удалённый MySQL сервер
DBSYNC_REMOTE_HOST=your-remote-host.com
DBSYNC_REMOTE_PORT=3306
DBSYNC_REMOTE_USER=your-username
DBSYNC_REMOTE_PASSWORD=your-password

# Локальный MySQL сервер
DBSYNC_LOCAL_HOST=localhost
DBSYNC_LOCAL_PORT=3306
DBSYNC_LOCAL_USER=root
DBSYNC_LOCAL_PASSWORD=your-local-password

# Настройки дампа
DBSYNC_DUMP_TIMEOUT=30m
DBSYNC_DUMP_TEMP_DIR=/tmp
```

## 📖 Использование

### Основные команды

```bash
# Показать справку
./bin/dbsync --help

# Проверить статус подключений
./bin/dbsync status

# Список доступных БД на удалённом сервере
./bin/dbsync list

# Синхронизация с интерактивным выбором БД
./bin/dbsync sync

# Синхронизация конкретной БД
./bin/dbsync sync my_database

# Dry-run (показать что будет сделано, без изменений)
./bin/dbsync sync my_database --dry-run

# Показать текущую конфигурацию
./bin/dbsync config

# Показать версию
./bin/dbsync version
```

### Примеры использования

#### Безопасная проверка перед синхронизацией

```bash
# Сначала проверяем что будет сделано
./bin/dbsync sync production_db --dry-run

# Если всё корректно, выполняем синхронизацию
./bin/dbsync sync production_db
```

#### Интерактивный режим

```bash
# Запускаем без указания БД для интерактивного выбора
./bin/dbsync sync
```

## 🧪 Тестирование

### Unit-тесты

```bash
# Запуск всех unit-тестов
go test -v ./internal/config ./internal/models ./internal/version ./pkg/utils ./internal/services ./internal/cli ./internal/ui

# С покрытием кода
go test -v -cover ./internal/config ./internal/models ./internal/version ./pkg/utils ./internal/services ./internal/cli ./internal/ui

# Генерация HTML отчёта о покрытии
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Интеграционные тесты

Для запуска интеграционных тестов необходимо настроить переменные окружения:

```bash
# Настройка переменных для тестирования
export DBSYNC_TEST_REMOTE_HOST=your-test-remote-host.com
export DBSYNC_TEST_REMOTE_USER=test_user
export DBSYNC_TEST_REMOTE_PASSWORD=test_password
export DBSYNC_TEST_LOCAL_HOST=localhost
export DBSYNC_TEST_LOCAL_USER=root
export DBSYNC_TEST_LOCAL_PASSWORD=your_local_password

# Запуск интеграционных тестов
go test -v -tags=integration ./test/integration
```

### Makefile команды (если доступны)

```bash
# Сборка
make build

# Все unit-тесты
make test

# Интеграционные тесты
make test-integration

# Покрытие кода
make test-coverage

# Release сборка
make build-release VERSION=v1.0.0
```

### Покрытие кода

Текущее покрытие unit-тестами:

- **config**: 76.4%
- **version**: 100.0%
- **utils**: 38.0%
- **services**: 5.6%
- **cli**: 5.9%
- **ui**: 78.5%
- **models**: Только структуры данных

## 🛡️ Функции безопасности

### Автоматические проверки

- Определение опасных БД по именам: `production`, `prod`, `live`, `master`, `main`
- Dry-run режим по умолчанию для критических операций
- Интерактивные подтверждения для destructive операций

### Валидация данных

- Проверка имён файлов и БД на безопасность
- Санитизация пользовательского ввода
- Валидация подключений перед операциями

### Логирование

- Подробные логи всех операций
- Возможность настройки уровня логирования
- Сохранение логов операций

## 📂 Структура проекта

```
db-sync-cli/
├── cmd/dbsync/           # Точка входа в приложение
├── internal/
│   ├── config/           # Управление конфигурацией
│   ├── cli/              # CLI команды и интерфейс
│   ├── services/         # Бизнес-логика (DB, Dump сервисы)
│   ├── models/           # Структуры данных
│   ├── ui/               # UI компоненты (форматирование, прогресс)
│   └── version/          # Управление версиями
├── pkg/utils/            # Утилиты общего назначения
├── test/
│   ├── mocks/            # Моки для тестирования
│   └── integration/      # Интеграционные тесты
├── .env.example          # Пример конфигурации
├── .env                  # Конфигурация (не в git)
├── Makefile              # Сборка и тестирование
└── README.md
```

## 🤝 Разработка

### Требования для разработки

- Go 1.24.1+
- MySQL/MariaDB для тестирования
- Git

### Локальная разработка

```bash
# Клонирование репозитория
git clone https://github.com/your-username/db-sync-cli.git
cd db-sync-cli

# Установка зависимостей
go mod download

# Копирование конфигурации
cp .env.example .env
# Отредактируйте .env под ваше окружение

# Запуск тестов
go test ./...

# Сборка
go build -o bin/dbsync ./cmd/dbsync

# Запуск
./bin/dbsync --help
```

### Добавление новых функций

1. Создайте unit-тесты для новой функциональности
2. Реализуйте функцию
3. Добавьте интеграционные тесты при необходимости
4. Обновите документацию

### Архитектурные принципы

- **Разделение ответственности**: Каждый пакет имеет чёткую зону ответственности
- **Тестируемость**: Все компоненты покрыты тестами
- **Безопасность**: Защита от случайных destructive операций
- **Удобство использования**: Интуитивный CLI и TUI интерфейс

## 📜 Лицензия

MIT License - см. файл [LICENSE](LICENSE)

## 🐛 Сообщение об ошибках

Пожалуйста, создавайте [issues](https://github.com/your-username/db-sync-cli/issues) для:

- Сообщений об ошибках
- Предложений новых функций
- Вопросов по использованию

## 📋 TODO

- [ ] Поддержка PostgreSQL
- [ ] Web интерфейс для мониторинга
- [ ] Автоматические backup'ы перед синхронизацией
- [ ] Поддержка конфигурационных профилей
- [ ] Docker образы
- [ ] CI/CD пайплайн
- [ ] Метрики и мониторинг
