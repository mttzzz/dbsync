# DB Sync CLI - Global Installation Guide

## Автоматическая установка для глобального использования

### Windows (PowerShell)

1. **Добавить в PATH через PowerShell:**
```powershell
# Добавляем путь к dbsync в переменную PATH для текущего пользователя
$currentPath = [Environment]::GetEnvironmentVariable("PATH", [EnvironmentVariableTarget]::User)
$dbsyncPath = "C:\Users\kiril\projects\go\db-dump-http\bin"

if ($currentPath -notlike "*$dbsyncPath*") {
    [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$dbsyncPath", [EnvironmentVariableTarget]::User)
    Write-Host "✅ Путь к dbsync добавлен в PATH"
    Write-Host "🔄 Перезапустите терминал для применения изменений"
} else {
    Write-Host "✅ Путь к dbsync уже находится в PATH"
}
```

2. **Альтернативно - через GUI:**
   - Откройте "Системные переменные среды"
   - Найдите переменную PATH в пользовательских переменных
   - Добавьте путь: `C:\Users\kiril\projects\go\db-dump-http\bin`

### Настройка конфигурации

Приложение автоматически ищет конфигурацию в следующих местах:
1. `~/.dbsync.env` в домашней директории пользователя  
2. `.env` в папке `bin` рядом с исполняемым файлом dbsync.exe

✅ **Конфигурация скопирована в оба места:**
- `%USERPROFILE%\.dbsync.env` 
- `C:\Users\kiril\projects\go\db-dump-http\bin\.env`

### Использование

После добавления в PATH и перезапуска терминала:

```bash
# Проверка статуса подключений
dbsync status

# Просмотр доступных баз данных
dbsync list

# Интерактивная синхронизация
dbsync sync

# Синхронизация конкретной БД
dbsync sync my_database

# Справка
dbsync --help
```

### Проверка установки

```bash
# Проверить версию
dbsync --version

# Проверить что конфигурация загружается
dbsync config
```

## Готово! 🚀

Теперь вы можете использовать `dbsync` из любой директории.
