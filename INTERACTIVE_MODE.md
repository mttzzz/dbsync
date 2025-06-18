# 🎉 Интерактивный режим готов!

## Что изменилось

✅ **Интерактивный выбор базы данных реализован!**

Теперь вы можете запускать команду `sync` без указания имени базы данных, и появится красивый интерактивный селектор с возможностью поиска.

## Как использовать

### 1. Настройте конфигурацию

Скопируйте `.env.example` в `.env` и настройте подключения:

```bash
cp .env.example .env
```

Отредактируйте `.env`:
```bash
# Ваш удаленный сервер
DBSYNC_REMOTE_HOST=pushka.biz
DBSYNC_REMOTE_PORT=3306
DBSYNC_REMOTE_USER=your_username
DBSYNC_REMOTE_PASSWORD=your_password

# Локальный сервер
DBSYNC_LOCAL_HOST=localhost
DBSYNC_LOCAL_PORT=3306
DBSYNC_LOCAL_USER=root
DBSYNC_LOCAL_PASSWORD=your_local_password
```

### 2. Запустите интерактивный режим

```bash
.\bin\dbsync.exe sync
```

### 3. Управление в интерактивном режиме

- **↑/↓** - навигация по списку
- **/** - режим поиска (введите название БД)
- **Enter** - выбрать базу данных
- **Esc** - выход без выбора

### 4. Альтернативно - прямое указание БД

```bash
.\bin\dbsync.exe sync database_name
```

## Особенности интерактивного селектора

- 🔍 **Поиск в реальном времени** - начните вводить название БД
- 📊 **Информация о размере и таблицах** для каждой БД
- 🎨 **Красивый TUI** с подсветкой и стилями
- ⚡ **Быстрая навигация** стрелками

## Пример использования

```bash
# Запуск интерактивного режима
.\bin\dbsync.exe sync

# Появится список всех доступных БД:
# 📋 Select Database to Sync
# 
# > 1. admin_reestr (3.4 MB, 18 tables)
#   2. ai_pushka_biz (27.3 MB, 18 tables) 
#   3. belgiss_pushka_biz (7.1 GB, 3 tables)
#   ...
#
# Press / to search, ↑↓ to navigate, Enter to select, Esc to quit

# После выбора:
# 🔍 Running safety checks for database 'selected_db'...
# ✅ All safety checks passed!
# 📋 Operation Plan: ...
```

Теперь ваш интерактивный режим полностью функционален! 🚀
