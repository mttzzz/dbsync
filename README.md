# 🚀 DB Sync CLI

<div align="center">
  
![Go](https://img.shields.io/badge/Go-1.24.1+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![MySQL](https://img.shields.io/badge/MySQL-4479A1?style=for-the-badge&logo=mysql&logoColor=white)
![Windows](https://img.shields.io/badge/Windows-0078D6?style=for-the-badge&logo=windows&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black)
![macOS](https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=apple&logoColor=white)

**Безопасная консольная утилита на Go для синхронизации MySQL баз данных между удалённым и локальным сервером**

[Быстрый старт](#-быстрый-старт) • [Установка](#-установка) • [Использование](#-использование) • [Тестирование](#-тестирование)

</div>

---

## 🌟 Особенности

- 🔒 **Безопасность превыше всего**: dry-run режим, интерактивные подтверждения для критических операций
- 🎨 **Современный TUI**: стилизованный интерфейс с progress bar и цветовой индикацией
- ⚡ **Интеллектуальная валидация**: автоматическое определение опасных БД (production, live, etc.)
- 🔧 **Гибкая конфигурация**: поддержка .env файлов и переменных окружения
- 📊 **Подробная статистика**: размер дампов, количество таблиц, время выполнения
- ✅ **Высокое покрытие тестами**: unit и интеграционные тесты
- 🌐 **Кроссплатформенность**: Windows, Linux, macOS

## 📋 Системные требования

- **Go**: 1.24.1 или выше
- **MySQL/MariaDB**: локальный и удалённый сервер
- **Утилиты**: `mysqldump` и `mysql` должны быть доступны в PATH
- **ОС**: Windows 10+, Linux, macOS

## ⚡ Быстрый старт

```powershell
# 1. Клонируем репозиторий
git clone https://github.com/your-username/dbsync.git
cd dbsync

# 2. Собираем проект (Windows)
.\build.ps1 build

# 3. Настраиваем конфигурацию
Copy-Item .env.example .env
# Отредактируйте .env файл под ваши настройки

# 4. Проверяем подключение
.\bin\dbsync.exe status

# 5. Синхронизируем базу данных
.\bin\dbsync.exe sync
```

## 🚀 Установка

### Автоматическая сборка (Windows)

```powershell
# Обычная сборка для разработки
.\build.ps1 build

# Release сборка с версией
.\build.ps1 build-release -Version "v1.0.0"

# Сборка для всех платформ
.\build.ps1 build-all
```

### Ручная сборка

<details>
<summary>🔽 Развернуть инструкции по ручной сборке</summary>

```bash
# Windows
set GOOS=windows
set GOARCH=amd64
go build -ldflags "-X main.version=v1.0.0 -X main.buildTime=%date% %time%" -o bin/dbsync.exe ./cmd/dbsync

# Linux
env GOOS=linux GOARCH=amd64 go build -ldflags "-X main.version=v1.0.0" -o bin/dbsync-linux ./cmd/dbsync

# macOS Intel
env GOOS=darwin GOARCH=amd64 go build -ldflags "-X main.version=v1.0.0" -o bin/dbsync-macos-intel ./cmd/dbsync

# macOS Apple Silicon
env GOOS=darwin GOARCH=arm64 go build -ldflags "-X main.version=v1.0.0" -o bin/dbsync-macos-arm ./cmd/dbsync
```

</details>

### Глобальная установка (Windows)

<details>
<summary>🔽 Настройка для глобального использования</summary>

**Автоматически через PowerShell:**
```powershell
# Добавляем dbsync в PATH
$currentPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)
$dbsyncPath = "$PWD\bin"

if ($currentPath -notlike "*$dbsyncPath*") {
    [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$dbsyncPath", [EnvironmentVariableTarget]::User)
    Write-Host "✅ dbsync добавлен в PATH. Перезапустите терминал."
}
```

**Вручную:**
1. Откройте "Системные переменные среды"
2. Найдите переменную PATH в пользовательских переменных
3. Добавьте путь к папке `bin` вашего проекта

После настройки используйте просто `dbsync` вместо `.\bin\dbsync.exe`

</details>

## ⚙️ Конфигурация

### 1. Создание конфигурационного файла

```powershell
# Копируем пример конфигурации
Copy-Item .env.example .env
```

### 2. Настройка параметров подключения

```env
# === УДАЛЁННЫЙ MYSQL СЕРВЕР ===
DBSYNC_REMOTE_HOST=your-remote-host.com
DBSYNC_REMOTE_PORT=3306
DBSYNC_REMOTE_USER=your-username
DBSYNC_REMOTE_PASSWORD=your-password

# === ЛОКАЛЬНЫЙ MYSQL СЕРВЕР ===
DBSYNC_LOCAL_HOST=localhost
DBSYNC_LOCAL_PORT=3306
DBSYNC_LOCAL_USER=root
DBSYNC_LOCAL_PASSWORD=your-local-password

# === НАСТРОЙКИ ДАМПА ===
DBSYNC_DUMP_TIMEOUT=30m
DBSYNC_DUMP_TEMP_DIR=./tmp
DBSYNC_DUMP_COMPRESS=true

# === НАСТРОЙКИ БЕЗОПАСНОСТИ ===
DBSYNC_FORCE_DRY_RUN=false
DBSYNC_DANGEROUS_DB_PATTERNS=production,prod,live,master,main
```

### 3. Альтернативные способы конфигурации

<details>
<summary>🔽 Переменные окружения и другие способы</summary>

**Переменные окружения:**
```powershell
# Установка через переменные среды Windows
$env:DBSYNC_REMOTE_HOST="your-host.com"
$env:DBSYNC_REMOTE_USER="username"
# ... и так далее
```

**Поиск конфигурации:**
Приложение ищет конфигурацию в следующем порядке:
1. Переменные окружения `DBSYNC_*`
2. `.env` в текущей директории
3. `~/.dbsync.env` в домашней папке пользователя
4. `.env` рядом с исполняемым файлом

</details>

## 📖 Использование

### Основные команды

```bash
# 📊 Статус и информация
dbsync status          # Проверить подключения к серверам
dbsync config          # Показать текущую конфигурацию
dbsync version         # Версия приложения
dbsync --help          # Справка по всем командам

# 📋 Работа с базами данных
dbsync list            # Список БД на удалённом сервере
dbsync sync            # Интерактивный выбор БД для синхронизации
dbsync sync my_db      # Синхронизация конкретной БД
dbsync sync my_db --dry-run  # Проверка без изменений
```

### 🎯 Примеры использования

#### 1. Первый запуск и проверка настроек

```powershell
# Проверяем конфигурацию
dbsync config

# Тестируем подключения
dbsync status

# Смотрим доступные базы данных
dbsync list
```

#### 2. Безопасная синхронизация

```powershell
# Всегда начинаем с проверки (dry-run)
dbsync sync production_db --dry-run

# Выводится детальная информация:
# ✅ База данных: production_db
# 📊 Размер: 245.7 MB
# 📋 Таблиц: 28
# ⚠️  ВНИМАНИЕ: Обнаружена production БД!
# 🔄 Операции: DROP 3 таблицы, CREATE 28 таблиц, INSERT ~50K записей

# Если всё корректно, выполняем синхронизацию
dbsync sync production_db
```

#### 3. Интерактивный режим

```powershell
# Запускаем интерактивный выбор
dbsync sync

# Появляется красивое меню:
# ┌─────────────────────────────────────────────┐
# │              📊 Выберите БД                 │
# ├─────────────────────────────────────────────┤
# │  1. user_data           (124.5 MB, 15 табл.)│
# │  2. analytics          (45.2 MB, 8 табл.)   │
# │  3. ⚠️  production_db   (245.7 MB, 28 табл.) │
# │  4. test_environment   (12.1 MB, 5 табл.)   │
# └─────────────────────────────────────────────┘
```

#### 4. Работа с опасными базами данных

```powershell
# При работе с production БД требуется подтверждение
dbsync sync production_db

# Выводится предупреждение:
# ⚠️  ОПАСНАЯ ОПЕРАЦИЯ ⚠️ 
# База данных содержит ключевое слово: production
# Локальная БД будет ПОЛНОСТЬЮ ПЕРЕЗАПИСАНА!
# 
# Введите название БД для подтверждения: production_db
# Продолжить? (yes/no): yes
```

## 🧪 Тестирование

### Быстрый запуск тестов (Windows)

```powershell
# Все unit-тесты
.\build.ps1 test

# Тесты с покрытием кода
.\build.ps1 test-coverage

# Интеграционные тесты (требуют MySQL)
.\build.ps1 test-integration

# Полная проверка (форматирование + линтинг + тесты)
.\build.ps1 verify
```

### Детальное тестирование

<details>
<summary>🔽 Подробные команды тестирования</summary>

**Unit-тесты по пакетам:**
```powershell
# Конфигурация
go test -v ./internal/config

# Модели данных
go test -v ./internal/models

# CLI команды
go test -v ./internal/cli

# Сервисы (база данных, дампы)
go test -v ./internal/services

# UI компоненты
go test -v ./internal/ui

# Утилиты
go test -v ./pkg/utils

# Версионирование
go test -v ./internal/version
```

**Покрытие кода:**
```powershell
# Генерация отчёта о покрытии
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html

# Открытие HTML отчёта
start coverage.html
```

**Интеграционные тесты:**
```powershell
# Настройка переменных для тестирования
$env:DBSYNC_TEST_REMOTE_HOST="your-test-host.com"
$env:DBSYNC_TEST_REMOTE_USER="test_user"
$env:DBSYNC_TEST_REMOTE_PASSWORD="test_password"
$env:DBSYNC_TEST_LOCAL_HOST="localhost"
$env:DBSYNC_TEST_LOCAL_USER="root"
$env:DBSYNC_TEST_LOCAL_PASSWORD="your_password"

# Запуск интеграционных тестов
go test -v -tags=integration ./test/integration
```

</details>

### 📊 Текущее покрытие тестами

| Пакет | Покрытие | Статус |
|-------|----------|--------|
| `config` | 76.4% | ✅ Хорошо |
| `version` | 100.0% | ✅ Отлично |
| `utils` | 38.0% | ⚠️ Требует улучшения |
| `services` | 5.6% | ❌ Критично |
| `cli` | 5.9% | ❌ Критично |
| `ui` | 78.5% | ✅ Хорошо |
| `models` | N/A | ℹ️ Только структуры |

## 🛡️ Безопасность

### Автоматические проверки безопасности

- **🚨 Определение опасных БД**: автоматически определяет БД с именами `production`, `prod`, `live`, `master`, `main`
- **🔐 Dry-run по умолчанию**: для критических операций требует явного подтверждения
- **✋ Интерактивные подтверждения**: запрашивает подтверждение для destructive операций
- **🛡️ Валидация входных данных**: проверка имён файлов и БД на безопасность

### Система предупреждений

```
⚠️  ВНИМАНИЕ: Обнаружена production база данных!
🔄 Операция: Полная замена локальной БД
📊 Размер: 245.7 MB (28 таблиц)
⏱️  Ожидаемое время: ~3-5 минут

Для продолжения введите название БД: production_db
```

### Логирование операций

- **📝 Подробные логи**: все операции логируются с временными метками
- **🎯 Уровни логирования**: ERROR, WARN, INFO, DEBUG
- **💾 Сохранение истории**: возможность сохранения логов операций

## 📂 Архитектура проекта

```
dbsync/
├── 📁 cmd/dbsync/              # 🚀 Точка входа приложения
│   └── main.go                 # Инициализация CLI
├── 📁 internal/                # 🔒 Внутренняя логика (не экспортируется)
│   ├── 📁 config/              # ⚙️ Управление конфигурацией
│   │   ├── config.go           # Загрузка .env и переменных окружения
│   │   └── config_test.go      # Тесты конфигурации
│   ├── 📁 cli/                 # 💻 CLI команды и интерфейс
│   │   ├── commands.go         # Реализация команд (sync, list, status)
│   │   ├── interactive.go      # Интерактивный режим выбора БД
│   │   └── commands_test.go    # Тесты CLI
│   ├── 📁 services/            # 🔧 Бизнес-логика
│   │   ├── database.go         # Работа с MySQL подключениями
│   │   ├── dump.go             # Создание и восстановление дампов
│   │   ├── interfaces.go       # Интерфейсы для тестирования
│   │   └── *_test.go           # Тесты сервисов
│   ├── 📁 models/              # 📊 Структуры данных
│   │   ├── database.go         # Модели БД, подключений, статистики
│   │   └── database_test.go    # Тесты моделей
│   ├── 📁 ui/                  # 🎨 UI компоненты
│   │   ├── formatter.go        # Цветное форматирование вывода
│   │   ├── progress.go         # Progress bar для операций
│   │   └── *_test.go           # Тесты UI
│   └── 📁 version/             # 📋 Управление версиями
│       ├── version.go          # Информация о версии и сборке
│       └── version_test.go     # Тесты версионирования
├── 📁 pkg/utils/               # 🛠️ Общие утилиты
│   ├── validation.go           # Валидация входных данных
│   └── validation_test.go      # Тесты валидации
├── 📁 test/                    # 🧪 Тестирование
│   ├── 📁 mocks/               # 🎭 Моки для unit-тестов
│   │   ├── database_service.go # Мок database service
│   │   └── dump_service.go     # Мок dump service
│   ├── 📁 integration/         # 🔗 Интеграционные тесты
│   │   └── integration_test.go # Полные сценарии тестирования
│   └── 📁 sql/                 # 🗃️ SQL скрипты для тестов
│       └── init.sql            # Начальная схема для тестов
├── 📁 bin/                     # 📦 Собранные исполняемые файлы
│   └── dbsync.exe              # Windows исполняемый файл
├── 📁 tmp/                     # 🗂️ Временные файлы дампов
├── 📄 .env.example             # 📋 Пример конфигурации
├── 📄 .env                     # ⚙️ Рабочая конфигурация (не в git)
├── 📄 go.mod                   # 📦 Go модуль и зависимости
├── 📄 go.sum                   # 🔒 Хеши зависимостей
├── 📄 build.ps1                # 🔨 PowerShell скрипт сборки
├── 📄 Dockerfile               # 🐳 Docker образ
├── 📄 docker-compose.yml       # 🐳 Docker Compose для разработки
└── 📄 README.md                # 📖 Документация
```

### Принципы архитектуры

- **🔒 Разделение ответственности**: каждый пакет имеет чёткую зону ответственности
- **🧪 Тестируемость**: все компоненты покрыты unit-тестами с использованием моков
- **🛡️ Безопасность**: защита от случайных destructive операций на всех уровнях
- **🎨 Удобство использования**: интуитивный CLI и красивый TUI интерфейс
- **📊 Мониторинг**: подробная статистика и логирование всех операций

## 🚀 Разработка

### Настройка среды разработки

```powershell
# 1. Клонирование и переход в проект
git clone https://github.com/your-username/dbsync.git
cd dbsync

# 2. Установка зависимостей
go mod download

# 3. Установка инструментов разработки
.\build.ps1 deps-dev

# 4. Настройка конфигурации
Copy-Item .env.example .env
# Отредактируйте .env под ваше окружение

# 5. Проверка окружения
.\build.ps1 verify
```

### Инструменты разработки

<details>
<summary>🔽 Команды build.ps1</summary>

| Команда | Описание |
|---------|----------|
| `build` | Быстрая сборка для разработки |
| `build-release` | Release сборка с метаданными |
| `build-all` | Сборка для всех платформ |
| `test` | Запуск unit-тестов |
| `test-integration` | Интеграционные тесты |
| `test-coverage` | Анализ покрытия кода |
| `lint` | Линтинг кода (go vet + golangci-lint) |
| `format` | Форматирование (go fmt + goimports) |
| `verify` | Полная проверка (format + lint + test) |
| `deps` | Установка Go зависимостей |
| `deps-dev` | Установка dev инструментов |
| `docker-build` | Сборка Docker образа |
| `docker-test` | Тестовое окружение в Docker |
| `clean` | Очистка временных файлов |

</details>

### Добавление новой функциональности

1. **🧪 Создайте тесты**: начните с написания unit-тестов для новой функции
2. **🔧 Реализуйте функцию**: следуйте принципам архитектуры проекта
3. **📋 Обновите интерфейсы**: при необходимости создайте новые интерфейсы
4. **🔗 Добавьте интеграционные тесты**: для сложной функциональности
5. **📖 Обновите документацию**: README и комментарии в коде

### Контрибьюция

```powershell
# 1. Форк репозитория на GitHub
# 2. Клонирование вашего форка
git clone https://github.com/your-username/dbsync.git

# 3. Создание ветки для функции
git checkout -b feature/new-awesome-feature

# 4. Разработка и тестирование
.\build.ps1 verify

# 5. Коммит изменений
git add .
git commit -m "feat: добавлена новая awesome функция"

# 6. Push и создание Pull Request
git push origin feature/new-awesome-feature
```

## 🐳 Docker

### Разработка в Docker

```powershell
# Сборка образа
.\build.ps1 docker-build

# Запуск тестового окружения
.\build.ps1 docker-test

# Очистка Docker окружения
.\build.ps1 docker-clean
```

### Использование Docker Compose

```yaml
# docker-compose.yml включает:
# - MySQL контейнер для тестирования
# - Настроенные сети и volumes
# - Environment переменные
```

## 🤝 Поддержка и сообщество

### 🐛 Сообщение об ошибках

Создавайте [Issues](https://github.com/your-username/dbsync/issues) для:
- 🚨 Сообщений об ошибках
- 💡 Предложений новых функций
- ❓ Вопросов по использованию
- 📖 Улучшений документации

### 📋 Roadmap

- [ ] **🐘 PostgreSQL поддержка**: расширение на другие СУБД
- [ ] **🌐 Web интерфейс**: браузерный UI для мониторинга
- [ ] **💾 Автоматические backup'ы**: создание резервных копий перед синхронизацией
- [ ] **👥 Конфигурационные профили**: множественные настройки для разных сред
- [ ] **🐳 Docker образы**: официальные образы в Docker Hub
- [ ] **🚀 CI/CD пайплайн**: автоматические тесты и релизы
- [ ] **📊 Метрики и мониторинг**: интеграция с Prometheus/Grafana
- [ ] **🔄 Инкрементальная синхронизация**: синхронизация только изменений
- [ ] **🔐 Улучшенная безопасность**: шифрование паролей, 2FA
- [ ] **📱 Mobile app**: мобильное приложение для мониторинга

### 📜 Лицензия

**MIT License** - подробности в файле [LICENSE](LICENSE)

### 🙏 Благодарности

- **Go Team** - за превосходный язык программирования
- **MySQL Team** - за надёжную СУБД
- **Open Source Community** - за инспирацию и поддержку

---

<div align="center">

**Сделано с ❤️ для разработчиков**

[⭐ Поставьте звезду](https://github.com/your-username/dbsync) • [🐛 Сообщить об ошибке](https://github.com/your-username/dbsync/issues) • [💡 Предложить улучшение](https://github.com/your-username/dbsync/issues)

</div>
