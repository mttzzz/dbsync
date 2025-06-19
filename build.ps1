# DB Sync CLI - Build Script for Windows
# PowerShell equivalent of Makefile for Windows development

param(
    [Parameter(Position=0)]
    [string]$Command = "help",
    
    [Parameter()]
    [string]$Version = "dev"
)

# Переменные
$BINARY_NAME = "dbsync"
$BUILD_DIR = "bin"
$DOCKER_IMAGE = "dbsync"
$DOCKER_TAG = "latest"

# Цвета для PowerShell
$Colors = @{
    Green  = "Green"
    Yellow = "Yellow"
    Red    = "Red"
    Blue   = "Blue"
    Cyan   = "Cyan"
}

function Write-ColoredText {
    param([string]$Text, [string]$Color = "White")
    Write-Host $Text -ForegroundColor $Colors[$Color]
}

function Show-Help {
    Write-ColoredText "=== DB Sync CLI - Build Script для Windows ===" "Blue"
    Write-Host ""
    Write-ColoredText "Доступные команды:" "Yellow"
    Write-Host ""
    Write-ColoredText "  build              " "Green" -NoNewline; Write-Host "Собрать бинарный файл"
    Write-ColoredText "  build-release      " "Green" -NoNewline; Write-Host "Собрать release версию"
    Write-ColoredText "  build-all          " "Green" -NoNewline; Write-Host "Собрать для всех платформ"
    Write-ColoredText "  test               " "Green" -NoNewline; Write-Host "Запустить unit тесты"
    Write-ColoredText "  test-integration   " "Green" -NoNewline; Write-Host "Запустить интеграционные тесты"
    Write-ColoredText "  test-coverage      " "Green" -NoNewline; Write-Host "Тесты с покрытием кода"
    Write-ColoredText "  lint               " "Green" -NoNewline; Write-Host "Запустить линтеры"
    Write-ColoredText "  format             " "Green" -NoNewline; Write-Host "Форматировать код"
    Write-ColoredText "  clean              " "Green" -NoNewline; Write-Host "Очистить сборочные артефакты"
    Write-ColoredText "  deps               " "Green" -NoNewline; Write-Host "Установить зависимости"
    Write-ColoredText "  deps-dev           " "Green" -NoNewline; Write-Host "Установить dev зависимости"
    Write-ColoredText "  run                " "Green" -NoNewline; Write-Host "Собрать и запустить приложение"
    Write-ColoredText "  demo               " "Green" -NoNewline; Write-Host "Демонстрация команд"
    Write-ColoredText "  docker-build       " "Green" -NoNewline; Write-Host "Собрать Docker образ"
    Write-ColoredText "  docker-test        " "Green" -NoNewline; Write-Host "Запустить тестовое окружение"
    Write-ColoredText "  docker-clean       " "Green" -NoNewline; Write-Host "Очистить Docker окружение"
    Write-ColoredText "  version            " "Green" -NoNewline; Write-Host "Показать информацию о версии"
    Write-ColoredText "  verify             " "Green" -NoNewline; Write-Host "Полная проверка (lint + test + format)"
    Write-ColoredText "  help               " "Green" -NoNewline; Write-Host "Показать эту справку"
    Write-Host ""
    Write-ColoredText "Примеры использования:" "Yellow"
    Write-Host "  .\build.ps1 build"
    Write-Host "  .\build.ps1 build-release -Version v1.0.0"
    Write-Host "  .\build.ps1 test"
    Write-Host "  .\build.ps1 docker-build"
}

function Invoke-Build {
    Write-ColoredText "Сборка $BINARY_NAME..." "Yellow"
    
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    
    & go build -o "$BUILD_DIR\$BINARY_NAME.exe" .\cmd\dbsync
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Сборка завершена: $BUILD_DIR\$BINARY_NAME.exe" "Green"
    } else {
        Write-ColoredText "Ошибка сборки!" "Red"
        exit 1
    }
}

function Invoke-BuildRelease {
    Write-ColoredText "Сборка release версии $Version..." "Yellow"
    
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    
    $BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    $GitCommit = try { & git rev-parse --short HEAD 2>$null } catch { "unknown" }
    
    $ldflags = "-X 'db-sync-cli/internal/version.Version=$Version' -X 'db-sync-cli/internal/version.BuildDate=$BuildDate' -X 'db-sync-cli/internal/version.GitCommit=$GitCommit'"
    
    & go build -ldflags $ldflags -o "$BUILD_DIR\$BINARY_NAME.exe" .\cmd\dbsync
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Release сборка завершена: $BUILD_DIR\$BINARY_NAME.exe" "Green"
    } else {
        Write-ColoredText "Ошибка сборки!" "Red"
        exit 1
    }
}

function Invoke-BuildAll {
    Write-ColoredText "Сборка для всех платформ..." "Yellow"
    
    if (!(Test-Path $BUILD_DIR)) {
        New-Item -ItemType Directory -Path $BUILD_DIR | Out-Null
    }
    
    $BuildDate = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    $GitCommit = try { & git rev-parse --short HEAD 2>$null } catch { "unknown" }
    $ldflags = "-X 'db-sync-cli/internal/version.Version=$Version' -X 'db-sync-cli/internal/version.BuildDate=$BuildDate' -X 'db-sync-cli/internal/version.GitCommit=$GitCommit'"
    
    $platforms = @(
        @{OS="linux"; ARCH="amd64"; EXT=""},
        @{OS="linux"; ARCH="arm64"; EXT=""},
        @{OS="windows"; ARCH="amd64"; EXT=".exe"},
        @{OS="darwin"; ARCH="amd64"; EXT=""},
        @{OS="darwin"; ARCH="arm64"; EXT=""}
    )
    
    foreach ($platform in $platforms) {
        $outputName = "$BUILD_DIR\$BINARY_NAME-$($platform.OS)-$($platform.ARCH)$($platform.EXT)"
        Write-Host "Building for $($platform.OS) $($platform.ARCH)..."
        
        $env:GOOS = $platform.OS
        $env:GOARCH = $platform.ARCH
        
        & go build -ldflags $ldflags -o $outputName .\cmd\dbsync
        
        if ($LASTEXITCODE -ne 0) {
            Write-ColoredText "Ошибка сборки для $($platform.OS) $($platform.ARCH)!" "Red"
            exit 1
        }
    }
    
    # Сброс переменных окружения
    Remove-Item Env:\GOOS -ErrorAction SilentlyContinue
    Remove-Item Env:\GOARCH -ErrorAction SilentlyContinue
    
    Write-ColoredText "Сборка для всех платформ завершена!" "Green"
}

function Invoke-Test {
    Write-ColoredText "Запуск unit тестов..." "Yellow"
    
    $testDirs = @(
        ".\internal\config",
        ".\internal\models",
        ".\internal\version", 
        ".\pkg\utils",
        ".\internal\services",
        ".\internal\cli",
        ".\internal\ui"
    )
    
    foreach ($dir in $testDirs) {
        Write-Host "Testing $dir..."
        & go test -v $dir
        if ($LASTEXITCODE -ne 0) {
            Write-ColoredText "Тесты не прошли в $dir!" "Red"
            exit 1
        }
    }
    
    Write-ColoredText "Все unit тесты пройдены!" "Green"
}

function Invoke-TestIntegration {
    Write-ColoredText "Запуск интеграционных тестов..." "Yellow"
    Write-ColoredText "Предупреждение: Интеграционные тесты требуют запущенного MySQL сервера" "Red"
    
    & go test -v -tags=integration .\test\integration
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Интеграционные тесты пройдены!" "Green"
    } else {
        Write-ColoredText "Интеграционные тесты не прошли!" "Red"
        exit 1
    }
}

function Invoke-TestCoverage {
    Write-ColoredText "Анализ покрытия кода..." "Yellow"
    
    & go test -coverprofile=coverage.out .\internal\config .\internal\models .\internal\version .\pkg\utils .\internal\services .\internal\cli .\internal\ui
    
    if ($LASTEXITCODE -eq 0) {
        & go tool cover -html=coverage.out -o coverage.html
        Write-ColoredText "HTML отчёт создан: coverage.html" "Green"
        
        # Попытка открыть в браузере
        try {
            Start-Process "coverage.html"
        } catch {
            Write-Host "Откройте coverage.html в браузере"
        }
    } else {
        Write-ColoredText "Ошибка при анализе покрытия!" "Red"
        exit 1
    }
}

function Invoke-Lint {
    Write-ColoredText "Запуск линтеров..." "Yellow"
    
    & go vet .\...
    if ($LASTEXITCODE -ne 0) {
        Write-ColoredText "go vet обнаружил проблемы!" "Red"
        exit 1
    }
    
    # Проверка наличия golangci-lint
    $golangciLint = Get-Command golangci-lint -ErrorAction SilentlyContinue
    if ($golangciLint) {
        & golangci-lint run
        if ($LASTEXITCODE -ne 0) {
            Write-ColoredText "golangci-lint обнаружил проблемы!" "Red"
            exit 1
        }
    } else {
        Write-ColoredText "golangci-lint не установлен. Установите его для более детального анализа" "Yellow"
    }
    
    Write-ColoredText "Линтинг завершен успешно!" "Green"
}

function Invoke-Format {
    Write-ColoredText "Форматирование кода..." "Yellow"
    
    & go fmt .\...
    
    # Проверка наличия goimports
    $goimports = Get-Command goimports -ErrorAction SilentlyContinue
    if ($goimports) {
        $goFiles = Get-ChildItem -Recurse -Filter "*.go" | Where-Object { $_.FullName -notlike "*\vendor\*" }
        foreach ($file in $goFiles) {
            & goimports -w $file.FullName
        }
    } else {
        Write-ColoredText "goimports не установлен. Установите его для упорядочивания импортов" "Yellow"
    }
    
    Write-ColoredText "Форматирование завершено!" "Green"
}

function Invoke-Clean {
    Write-ColoredText "Очистка..." "Yellow"
    
    if (Test-Path $BUILD_DIR) {
        Remove-Item -Recurse -Force $BUILD_DIR
    }
    
    if (Test-Path "coverage.out") {
        Remove-Item "coverage.out"
    }
    
    if (Test-Path "coverage.html") {
        Remove-Item "coverage.html"
    }
    
    & go clean
    
    Write-ColoredText "Очистка завершена!" "Green"
}

function Invoke-Deps {
    Write-ColoredText "Установка зависимостей..." "Yellow"
    
    & go mod download
    & go mod tidy
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Зависимости установлены!" "Green"
    } else {
        Write-ColoredText "Ошибка установки зависимостей!" "Red"
        exit 1
    }
}

function Invoke-DevDeps {
    Write-ColoredText "Установка dev зависимостей..." "Yellow"
    
    & go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
    & go install golang.org/x/tools/cmd/goimports@latest
    
    Write-ColoredText "Dev зависимости установлены!" "Green"
}

function Invoke-Run {
    Invoke-Build
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Запуск $BINARY_NAME..." "Yellow"
        & ".\$BUILD_DIR\$BINARY_NAME.exe" --help
    }
}

function Invoke-Demo {
    Invoke-Build
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Демонстрация команд $BINARY_NAME..." "Yellow"
        
        Write-Host "1. Показать версию:"
        & ".\$BUILD_DIR\$BINARY_NAME.exe" version
        
        Write-Host "`n2. Показать конфигурацию:"
        & ".\$BUILD_DIR\$BINARY_NAME.exe" config --show
        
        Write-Host "`n3. Показать справку:"
        & ".\$BUILD_DIR\$BINARY_NAME.exe" --help
    }
}

function Invoke-DockerBuild {
    Write-ColoredText "Сборка Docker образа..." "Yellow"
    
    & docker build -t "$DOCKER_IMAGE`:$DOCKER_TAG" .
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Docker образ собран: $DOCKER_IMAGE`:$DOCKER_TAG" "Green"
    } else {
        Write-ColoredText "Ошибка сборки Docker образа!" "Red"
        exit 1
    }
}

function Invoke-DockerTest {
    Write-ColoredText "Запуск тестового окружения..." "Yellow"
    
    & docker-compose up -d mysql-local mysql-remote
    
    if ($LASTEXITCODE -eq 0) {
        Write-ColoredText "Ожидание готовности баз данных..." "Green"
        Start-Sleep -Seconds 30
        Write-ColoredText "Тестовое окружение готово!" "Green"
        Write-ColoredText "MySQL Local:  localhost:3306" "Blue"
        Write-ColoredText "MySQL Remote: localhost:3307" "Blue"
        Write-ColoredText "Adminer:      http://localhost:8080" "Blue"
    } else {
        Write-ColoredText "Ошибка запуска Docker окружения!" "Red"
        exit 1
    }
}

function Invoke-DockerClean {
    Write-ColoredText "Очистка Docker окружения..." "Yellow"
    
    & docker-compose down -v
    
    Write-ColoredText "Docker окружение очищено!" "Green"
}

function Show-Version {
    Write-ColoredText "DB Sync CLI" "Blue"
    Write-Host "Version: $Version"
    Write-Host "Build date: $((Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ"))"
    $GitCommit = try { & git rev-parse --short HEAD 2>$null } catch { "unknown" }
    Write-Host "Git commit: $GitCommit"
}

function Invoke-Verify {
    Write-ColoredText "Запуск полной проверки..." "Yellow"
    
    Invoke-Format
    Invoke-Lint
    Invoke-Test
    
    Write-ColoredText "Все проверки пройдены успешно!" "Green"
}

# Основная логика
switch ($Command.ToLower()) {
    "build" { Invoke-Build }
    "build-release" { Invoke-BuildRelease }
    "build-all" { Invoke-BuildAll }
    "test" { Invoke-Test }
    "test-integration" { Invoke-TestIntegration }
    "test-coverage" { Invoke-TestCoverage }
    "lint" { Invoke-Lint }
    "format" { Invoke-Format }
    "clean" { Invoke-Clean }
    "deps" { Invoke-Deps }
    "deps-dev" { Invoke-DevDeps }
    "run" { Invoke-Run }
    "demo" { Invoke-Demo }
    "docker-build" { Invoke-DockerBuild }
    "docker-test" { Invoke-DockerTest }
    "docker-clean" { Invoke-DockerClean }
    "version" { Show-Version }
    "verify" { Invoke-Verify }
    "help" { Show-Help }
    default { 
        Write-ColoredText "Неизвестная команда: $Command" "Red"
        Write-Host ""
        Show-Help
        exit 1
    }
}
