# 🚀 DB Sync CLI

<div align="center">
  
![Go](https://img.shields.io/badge/Go-1.24+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![MySQL](https://img.shields.io/badge/MySQL-4479A1?style=for-the-badge&logo=mysql&logoColor=white)
![Windows](https://img.shields.io/badge/Windows-0078D6?style=for-the-badge&logo=windows&logoColor=white)
![macOS](https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=apple&logoColor=white)

**Быстрая синхронизация MySQL баз данных с использованием MySQL Shell**

[![GitHub release](https://img.shields.io/github/v/release/mttzzz/dbsync)](https://github.com/mttzzz/dbsync/releases/latest)

</div>

---

## 🌟 Особенности

- ⚡ **Очень быстро**: ~30 сек для 665 MB базы (10x быстрее mydumper)
- 🐚 **MySQL Shell**: использует `util.dump-schemas` / `util.load-dump`
- 🎨 **Лаконичный вывод**: только важная информация
- 🔄 **Автообновление**: встроенная команда `upgrade`

## 📋 Требования

- **MySQL Shell 8.4+**: [Скачать](https://dev.mysql.com/downloads/shell/)
- **MySQL**: локальный сервер с `local_infile = 1`

## 📦 Установка

### Быстрая установка

**[Скачать последнюю версию](https://github.com/mttzzz/dbsync/releases/latest)**

| Платформа | Файл |
|-----------|------|
| Windows x64 | `dbsync-windows-amd64.zip` |
| macOS ARM | `dbsync-darwin-arm64.tar.gz` |

### MySQL Shell

```bash
# Windows (winget)
winget install Oracle.MySQLShell

# macOS (brew)
brew install mysql-shell
```

## ⚙️ Настройка

Создайте `.env` файл:

```env
# Удалённый сервер
DBSYNC_REMOTE_HOST=your-remote-host.com
DBSYNC_REMOTE_PORT=3306
DBSYNC_REMOTE_USER=username
DBSYNC_REMOTE_PASSWORD=password
DBSYNC_REMOTE_PROXY_URL=socks5://proxy.example.com:1080

# Локальный сервер
DBSYNC_LOCAL_HOST=localhost
DBSYNC_LOCAL_PORT=3306
DBSYNC_LOCAL_USER=root
DBSYNC_LOCAL_PASSWORD=password

# Настройки (опционально)
DBSYNC_DUMP_THREADS=8
```

Поддерживаются прокси `socks5://`, `socks5h://`, `http://` и `https://`. Для удалённого MySQL создаётся локальный TCP-туннель, поэтому прокси применяется и к проверкам подключения, и к `mysqlsh dump`.

## 📖 Использование

```bash
# Интерактивный выбор БД
dbsync

# Синхронизация конкретной БД
dbsync my_database

# Без подтверждения
dbsync my_database --force

# Список БД на удалённом сервере
dbsync list

# Проверка подключений
dbsync status

# Обновление программы
dbsync upgrade
```

## ⚡ Производительность

| База данных | Размер | Dump | Restore | Всего |
|-------------|--------|------|---------|-------|
| 63 таблицы, 2.3M строк | 665 MB | 19 сек | 13 сек | **32 сек** |

- Сжатие: zstd (665 MB → 182 MB)
- 8 параллельных потоков
- Индексы создаются после загрузки данных

## 🛡️ Безопасность

- Подтверждение перед заменой БД
- Флаг `--dry-run` для проверки
- Пароли передаются безопасно (не в командной строке)

---

<div align="center">

**[⭐ Star](https://github.com/mttzzz/dbsync)** • **[🐛 Issues](https://github.com/mttzzz/dbsync/issues)** • **[📦 Releases](https://github.com/mttzzz/dbsync/releases)**

</div>
