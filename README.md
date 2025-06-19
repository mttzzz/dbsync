# 🚀 DB Sync CLI

<div align="center">
  
![Go](https://img.shields.io/badge/Go-1.24.1+-00ADD8?style=for-the-badge&logo=go&logoColor=white)
![MySQL](https://img.shields.io/badge/MySQL-4479A1?style=for-the-badge&logo=mysql&logoColor=white)
![Windows](https://img.shields.io/badge/Windows-0078D6?style=for-the-badge&logo=windows&logoColor=white)
![Linux](https://img.shields.io/badge/Linux-FCC624?style=for-the-badge&logo=linux&logoColor=black)
![macOS](https://img.shields.io/badge/macOS-000000?style=for-the-badge&logo=apple&logoColor=white)

**Безопасная консольная утилита для синхронизации MySQL баз данных между удалённым и локальным сервером**

[![GitHub release](https://img.shields.io/github/v/release/your-username/dbsync)](https://github.com/your-username/dbsync/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/your-username/dbsync/total)](https://github.com/your-username/dbsync/releases)

[📦 Скачать релиз](#-установка) • [📖 Использование](#-использование) • [🐛 Сообщить об ошибке](https://github.com/your-username/dbsync/issues)

</div>

---

## 🌟 Особенности

- 🔒 **Безопасность превыше всего**: dry-run режим, интерактивные подтверждения для критических операций
- 🎨 **Современный интерфейс**: стилизованный TUI с progress bar и цветовой индикацией
- ⚡ **Интеллектуальная валидация**: автоматическое определение опасных БД (production, live, etc.)
- 🔧 **Простая настройка**: конфигурация через .env файл или переменные окружения
- 📊 **Подробная статистика**: размер дампов, количество таблиц, время выполнения
- 🌐 **Кроссплатформенность**: Windows, Linux, macOS для всех основных архитектур

## 📋 Системные требования

- **MySQL/MariaDB**: локальный и удалённый сервер
- **Утилиты**: `mysqldump` и `mysql` должны быть доступны в системе
- **ОС**: Windows 10+, Linux (любой дистрибутив), macOS 10.14+

## 📦 Установка

### 🎯 Быстрая установка (рекомендуется)

Скачайте готовую сборку для вашей системы:

**🔗 [Скачать последнюю версию](https://github.com/your-username/dbsync/releases/latest)**

| Платформа | Архитектура | Файл для скачивания |
|-----------|-------------|---------------------|
| **Windows** | 64-bit (Intel/AMD) | `dbsync-vX.X.X-windows-amd64.zip` |
| **Windows** | 32-bit | `dbsync-vX.X.X-windows-386.zip` |
| **Linux** | 64-bit (Intel/AMD) | `dbsync-vX.X.X-linux-amd64.tar.gz` |
| **Linux** | ARM64 | `dbsync-vX.X.X-linux-arm64.tar.gz` |
| **Linux** | 32-bit | `dbsync-vX.X.X-linux-386.tar.gz` |
| **macOS** | Intel | `dbsync-vX.X.X-darwin-amd64.tar.gz` |
| **macOS** | Apple Silicon (M1/M2) | `dbsync-vX.X.X-darwin-arm64.tar.gz` |

### 🔧 Установка по платформам

<details>
<summary>🪟 <strong>Windows</strong></summary>

1. **Скачайте архив** для вашей архитектуры (обычно amd64)
2. **Распакуйте** архив в любую папку (например, `C:\dbsync\`)
3. **Добавьте в PATH** (опционально):
   ```powershell
   # Добавление в PATH для текущего пользователя
   $env:PATH += ";C:\dbsync"
   ```
4. **Проверьте установку**:
   ```cmd
   dbsync.exe version
   ```

</details>

<details>
<summary>🐧 <strong>Linux</strong></summary>

```bash
# Скачивание и установка (замените X.X.X на актуальную версию)
curl -L "https://github.com/your-username/dbsync/releases/latest/download/dbsync-vX.X.X-linux-amd64.tar.gz" | tar -xz

# Перемещение в системную папку
sudo mv dbsync-vX.X.X-linux-amd64 /usr/local/bin/dbsync

# Установка прав
sudo chmod +x /usr/local/bin/dbsync

# Проверка
dbsync version
```

</details>

<details>
<summary>🍎 <strong>macOS</strong></summary>

```bash
# Для Intel Mac
curl -L "https://github.com/your-username/dbsync/releases/latest/download/dbsync-vX.X.X-darwin-amd64.tar.gz" | tar -xz

# Для Apple Silicon (M1/M2)
curl -L "https://github.com/your-username/dbsync/releases/latest/download/dbsync-vX.X.X-darwin-arm64.tar.gz" | tar -xz

# Перемещение в системную папку
sudo mv dbsync-vX.X.X-darwin-* /usr/local/bin/dbsync

# Установка прав
sudo chmod +x /usr/local/bin/dbsync

# Проверка
dbsync version
```

**Примечание**: При первом запуске macOS может показать предупреждение безопасности. Разрешите выполнение в "Системные настройки" → "Безопасность и конфиденциальность".

</details>

### 🔄 Типы релизов

| Тип релиза | Статус | Описание |
|------------|--------|----------|
| 🏷️ **Стабильные** | [![GitHub release](https://img.shields.io/github/v/release/your-username/dbsync)](https://github.com/your-username/dbsync/releases/latest) | Релизы с тегами версий для продуктивного использования |
| 🌙 **Nightly** | [![Nightly](https://img.shields.io/badge/nightly-latest-orange)](https://github.com/your-username/dbsync/releases/tag/nightly) | Ежедневные сборки с последними изменениями |
| 📅 **Weekly** | [![Weekly](https://img.shields.io/badge/weekly-dev-blue)](https://github.com/your-username/dbsync/releases/tag/weekly) | Еженедельные сборки для тестирования новых функций |

## ⚙️ Настройка

### 1. Создание конфигурационного файла

Создайте файл `.env` в папке с программой или в домашней директории:

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

# === ДОПОЛНИТЕЛЬНЫЕ НАСТРОЙКИ ===
DBSYNC_DUMP_TIMEOUT=30m
DBSYNC_DUMP_TEMP_DIR=./tmp
```

### 2. Проверка настроек

```bash
# Проверить конфигурацию
dbsync config

# Проверить подключения к серверам
dbsync status
```

## 📖 Использование

### Основные команды

```bash
# 📊 Информация и статус
dbsync version         # Версия программы
dbsync config          # Показать текущую конфигурацию
dbsync status          # Проверить подключения к серверам
dbsync --help          # Справка по всем командам

# 📋 Работа с базами данных
dbsync list            # Список БД на удалённом сервере
dbsync sync            # Интерактивный выбор БД для синхронизации
dbsync sync my_db      # Синхронизация конкретной БД
dbsync sync my_db --dry-run  # Проверка без изменений (безопасно)
```

### 🎯 Примеры использования

#### 1. Первый запуск и проверка настроек

```bash
# Проверяем конфигурацию
dbsync config

# Тестируем подключения
dbsync status

# Смотрим доступные базы данных
dbsync list
```

#### 2. Безопасная синхронизация

```bash
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

## 🤝 Поддержка и сообщество

### 🐛 Сообщение об ошибках

Используйте [GitHub Issues](https://github.com/your-username/dbsync/issues/new/choose) с готовыми шаблонами:

- 🐛 **Bug Report** - для сообщения об ошибках
- ✨ **Feature Request** - для предложения новых функций  
- ❓ **Question** - для вопросов по использованию

### 📈 Обновления

| Тип | Расписание | Назначение |
|-----|------------|------------|
| 🏷️ **Stable** | По мере готовности | Стабильные релизы для продуктива |
| 🌙 **Nightly** | Каждый коммит в main | Тестирование последних изменений |
| 📅 **Weekly** | Воскресенье, 02:00 UTC | Dev-сборки из develop ветки |

### 📋 Roadmap

- [ ] **🐘 PostgreSQL поддержка**: расширение на другие СУБД
- [ ] **🌐 Web интерфейс**: браузерный UI для мониторинга
- [ ] **💾 Автоматические backup'ы**: создание резервных копий перед синхронизацией
- [ ] **👥 Конфигурационные профили**: множественные настройки для разных сред
- [ ] **📊 Метрики и мониторинг**: интеграция с Prometheus/Grafana
- [ ] **🔄 Инкрементальная синхронизация**: синхронизация только изменений
- [ ] **🔐 Улучшенная безопасность**: шифрование паролей, 2FA

### 📜 Лицензия

**MIT License** - подробности в файле [LICENSE](LICENSE)

### 🙏 Благодарности

- **Go Team** - за превосходный язык программирования
- **MySQL Team** - за надёжную СУБД
- **GitHub Actions** - за отличную CI/CD платформу
- **Open Source Community** - за инспирацию и поддержку

---

<div align="center">

**Сделано с ❤️ для разработчиков**

[![GitHub stars](https://img.shields.io/github/stars/your-username/dbsync?style=social)](https://github.com/your-username/dbsync/stargazers)
[![GitHub forks](https://img.shields.io/github/forks/your-username/dbsync?style=social)](https://github.com/your-username/dbsync/network/members)
[![GitHub issues](https://img.shields.io/github/issues/your-username/dbsync)](https://github.com/your-username/dbsync/issues)

[⭐ Поставьте звезду](https://github.com/your-username/dbsync) • [🐛 Сообщить об ошибке](https://github.com/your-username/dbsync/issues/new?template=bug_report.md) • [💡 Предложить улучшение](https://github.com/your-username/dbsync/issues/new?template=feature_request.md) • [📦 Релизы](https://github.com/your-username/dbsync/releases)

</div>
