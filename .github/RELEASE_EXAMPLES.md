# 📋 Примеры создания релизов

## 🎯 Создание стабильного релиза

### 1. Подготовка релиза

```bash
# Убедитесь, что вы на main ветке
git checkout main
git pull origin main

# Проверьте, что все тесты проходят
.\build.ps1 verify

# Обновите CHANGELOG.md
# Измените раздел [Unreleased] на [v1.2.0] с текущей датой
```

### 2. Создание и отправка тега

```bash
# Создайте тег с версией
git tag -a v1.2.0 -m "Release version 1.2.0

🚀 Основные изменения:
- Добавлена поддержка PostgreSQL
- Улучшен интерактивный режим  
- Исправлены критические ошибки

📦 Полные release notes: https://github.com/your-username/dbsync/releases/tag/v1.2.0"

# Отправьте тег в репозиторий
git push origin v1.2.0
```

### 3. Автоматическая сборка

После отправки тега GitHub Actions автоматически:

1. ✅ Запустит все тесты
2. 🔨 Соберёт бинарные файлы для всех платформ
3. 📦 Создаст архивы с checksums
4. 🚀 Опубликует GitHub Release
5. 🐳 Соберёт и опубликует Docker образ

## 🌙 Nightly релизы

Nightly релизы создаются автоматически при каждом коммите в `main`:

```bash
# Просто делайте коммит в main
git checkout main
git add .
git commit -m "feat: добавлена новая функция"
git push origin main

# Автоматически будет создан nightly релиз
```

Результат:
- Релиз с тегом `nightly`
- Файлы с версией типа `v0.0.0-main-abc1234`
- Пометка как pre-release

## 📅 Weekly релизы

Weekly релизы создаются автоматически каждое воскресенье в 02:00 UTC из `develop` ветки.

Для ручного запуска:

```bash
# Перейдите в GitHub Actions
# Найдите workflow "CI/CD"
# Нажмите "Run workflow" на develop ветке
```

## 🔧 Локальная сборка релиза

### Windows (PowerShell)

```powershell
# Сборка для всех платформ
.\build.ps1 build-all

# Release сборка с версией
.\build.ps1 build-release -Version "v1.2.0"

# Результаты в папке dist/
dir dist/
```

### Результат локальной сборки:
```
dist/
├── dbsync-v1.2.0-linux-amd64.tar.gz
├── dbsync-v1.2.0-linux-arm64.tar.gz
├── dbsync-v1.2.0-windows-amd64.zip
├── dbsync-v1.2.0-darwin-amd64.tar.gz
├── dbsync-v1.2.0-darwin-arm64.tar.gz
└── checksums.txt
```

## � Бинарные релизы

### Автоматическая публикация

Бинарные файлы автоматически публикуются в GitHub Releases:

```bash
# Для стабильных релизов доступны все платформы:
# Windows: dbsync-v1.2.0-windows-amd64.zip
# Linux: dbsync-v1.2.0-linux-amd64.tar.gz
# macOS: dbsync-v1.2.0-darwin-amd64.tar.gz
# ... и другие архитектуры
```

## 📝 Release Notes

### Автоматические Release Notes

CI/CD автоматически генерирует release notes на основе:
- Коммитов с момента последнего релиза
- Pull Request'ов
- Списка изменённых файлов

### Кастомные Release Notes

Создайте или обновите `CHANGELOG.md`:

```markdown
## [v1.2.0] - 2025-06-19

### ✨ Новые функции
- Поддержка PostgreSQL (#123)
- Улучшенный интерактивный режим (#145)
- Экспорт в JSON формат (#156)

### 🐛 Исправления
- Исправлена утечка памяти при больших дампах (#134)
- Улучшена обработка timeout'ов (#142)

### 🔧 Улучшения
- Увеличена скорость синхронизации на 30%
- Лучшая цветовая схема в TUI

### ⚠️ Критические изменения
- Изменён формат конфигурации (см. migration guide)
- Удалена поддержка MySQL < 5.7
```

## 🚨 Откат релиза

### Если релиз содержит критические ошибки:

```bash
# 1. Удалите проблемный релиз и тег
gh release delete v1.2.0 --yes
git tag -d v1.2.0
git push origin --delete v1.2.0

# 2. Создайте hotfix ветку от предыдущей версии
git checkout v1.1.0
git checkout -b hotfix/v1.2.1

# 3. Исправьте ошибки
# ... внесите изменения ...

# 4. Создайте новый релиз
git add .
git commit -m "fix: критическое исправление безопасности"
git tag -a v1.2.1 -m "Hotfix release v1.2.1"
git push origin v1.2.1
```

## 📊 Мониторинг релизов

### GitHub Actions

Проверьте статус сборки:
- Перейдите в раздел "Actions"
- Найдите workflow для вашего тега
- Проверьте логи каждого шага

### Проверка релиза

```bash
# Проверьте, что релиз создан
gh release view v1.2.0

# Тестирование сборки
curl -L https://github.com/your-username/dbsync/releases/download/v1.2.0/dbsync-v1.2.0-linux-amd64.tar.gz | tar -xz
./dbsync-v1.2.0-linux-amd64 version
```

## 🎯 Полезные алиасы Git

Добавьте в `~/.gitconfig`:

```ini
[alias]
    # Создание релиза
    release = "!f() { git tag -a $1 -m \"Release $1\" && git push origin $1; }; f"
    
    # Просмотр тегов
    tags = tag -l --sort=-version:refname
    
    # Последний тег
    lasttag = describe --tags --abbrev=0
    
    # Изменения с последнего тега
    changelog = "!git log $(git describe --tags --abbrev=0)..HEAD --oneline"
```

Использование:
```bash
# Создание релиза
git release v1.2.0

# Просмотр изменений
git changelog

# Последняя версия
git lasttag
```

## 📋 Checklist релиза

### Перед созданием тега:
- [ ] Все тесты проходят локально (`.\build.ps1 verify`)
- [ ] Обновлён CHANGELOG.md
- [ ] Обновлена документация (если нужно)  
- [ ] Проверена совместимость с предыдущими версиями
- [ ] Выполнено интеграционное тестирование

### После создания тега:
- [ ] GitHub Actions workflow выполнился успешно
- [ ] GitHub Release создан
- [ ] Все платформы представлены в релизе
- [ ] Docker образ опубликован
- [ ] Checksums файл создан
- [ ] Release notes корректны

### После публикации:
- [ ] Протестированы сборки для основных платформ
- [ ] Release notes корректны
- [ ] Checksums файл создан
- [ ] Документация актуальна
- [ ] Сообщество уведомлено (если крупный релиз)

---

💡 **Совет**: Используйте [Semantic Versioning](https://semver.org/) для нумерации версий:
- `MAJOR.MINOR.PATCH` (например, `1.2.3`)
- Префикс `v` для тегов (например, `v1.2.3`)
