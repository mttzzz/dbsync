# BUILD.ps1 - Инструкция по использованию

PowerShell скрипт для сборки DB Sync CLI в Windows (замена Makefile).

## Быстрый старт

```powershell
# Показать все доступные команды
.\build.ps1 help

# Обычная сборка
.\build.ps1 build

# Release сборка с версией
.\build.ps1 build-release -Version "v1.0.0"

# Запустить тесты
.\build.ps1 test

# Полная проверка (форматирование + линтинг + тесты)
.\build.ps1 verify
```

## Основные команды

### Сборка
- `build` - Быстрая сборка для разработки
- `build-release` - Release сборка с метаданными
- `build-all` - Сборка для всех платформ (Linux, Windows, macOS)

### Тестирование
- `test` - Unit тесты
- `test-integration` - Интеграционные тесты (требует MySQL)
- `test-coverage` - Тесты с анализом покрытия

### Качество кода
- `lint` - Линтинг (go vet + golangci-lint)
- `format` - Форматирование кода (go fmt + goimports)
- `verify` - Полная проверка (format + lint + test)

### Зависимости
- `deps` - Установить Go зависимости
- `deps-dev` - Установить dev инструменты (golangci-lint, goimports)

### Docker
- `docker-build` - Собрать Docker образ
- `docker-test` - Запустить тестовое окружение
- `docker-clean` - Очистить Docker окружение

### Утилиты
- `run` - Собрать и запустить с --help
- `demo` - Демонстрация основных команд
- `clean` - Очистить артефакты сборки
- `version` - Показать информацию о версии

## Примеры использования

```powershell
# Первоначальная настройка
.\build.ps1 deps
.\build.ps1 deps-dev

# Разработка
.\build.ps1 format
.\build.ps1 build
.\build.ps1 test

# Подготовка к коммиту
.\build.ps1 verify

# Release
.\build.ps1 clean
.\build.ps1 build-release -Version "v2.1.0"

# Сборка для всех платформ
.\build.ps1 build-all -Version "v2.1.0"

# Docker разработка
.\build.ps1 docker-build
.\build.ps1 docker-test
# ... работа с приложением ...
.\build.ps1 docker-clean
```

## Параметры

- `-Version` - Указать версию для release сборки (по умолчанию "dev")

## Требования

- PowerShell 5.0+
- Go 1.21+
- Git (для получения commit hash)
- Docker (для Docker команд)
- MySQL (для интеграционных тестов)

## Дополнительные инструменты (опционально)

```powershell
# Установить дополнительные инструменты
.\build.ps1 deps-dev
```

Это установит:
- `golangci-lint` - Продвинутый линтер
- `goimports` - Автоматическое управление импортами

## Сравнение с Makefile

Все команды из Makefile портированы в build.ps1:

| Makefile | build.ps1 |
|----------|-----------|
| `make build` | `.\build.ps1 build` |
| `make test` | `.\build.ps1 test` |
| `make clean` | `.\build.ps1 clean` |
| `make build-release VERSION=v1.0.0` | `.\build.ps1 build-release -Version v1.0.0` |
| `make docker-build` | `.\build.ps1 docker-build` |

## Решение проблем

### Ошибка выполнения политики PowerShell
```powershell
Set-ExecutionPolicy -ExecutionPolicy RemoteSigned -Scope CurrentUser
```

### Go команды не найдены
Убедитесь что Go установлен и добавлен в PATH:
```powershell
go version
```

### Docker команды не работают
Убедитесь что Docker Desktop запущен:
```powershell
docker version
```
