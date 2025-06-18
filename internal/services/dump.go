package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

// DumpService предоставляет функции для создания и восстановления дампов
type DumpService struct {
	config    *config.Config
	dbService DatabaseServiceInterface
}

// NewDumpService создает новый экземпляр DumpService
func NewDumpService(cfg *config.Config, dbService DatabaseServiceInterface) *DumpService {
	return &DumpService{
		config:    cfg,
		dbService: dbService,
	}
}

// ValidateDumpOperation проверяет возможность выполнения операции дампа
func (ds *DumpService) ValidateDumpOperation(databaseName string) error {
	// Валидация имени базы данных
	if err := ds.dbService.ValidateDatabaseName(databaseName); err != nil {
		return fmt.Errorf("invalid database name: %w", err)
	}

	// Проверяем что удаленная БД существует
	exists, err := ds.dbService.DatabaseExists(databaseName, true)
	if err != nil {
		return fmt.Errorf("failed to check remote database: %w", err)
	}
	if !exists {
		return fmt.Errorf("database '%s' not found on remote server", databaseName)
	}

	// Проверяем подключение к удаленному серверу
	remoteConn, err := ds.dbService.TestConnection(true)
	if err != nil || !remoteConn.Connected {
		return fmt.Errorf("cannot connect to remote server: %s", remoteConn.Error)
	}

	// Проверяем подключение к локальному серверу
	localConn, err := ds.dbService.TestConnection(false)
	if err != nil || !localConn.Connected {
		return fmt.Errorf("cannot connect to local server: %s", localConn.Error)
	}

	// Проверяем что mysqldump доступен
	if err := ds.validateMysqldumpAvailable(); err != nil {
		return fmt.Errorf("mysqldump validation failed: %w", err)
	}

	// Проверяем что mysql клиент доступен
	if err := ds.validateMysqlAvailable(); err != nil {
		return fmt.Errorf("mysql client validation failed: %w", err)
	}

	// Проверяем что временная директория доступна для записи
	if err := ds.validateTempDirectory(); err != nil {
		return fmt.Errorf("temp directory validation failed: %w", err)
	}

	return nil
}

// PlanDumpOperation возвращает план операции без выполнения
func (ds *DumpService) PlanDumpOperation(databaseName string) (*models.SyncResult, error) {
	// Валидация операции
	if err := ds.ValidateDumpOperation(databaseName); err != nil {
		return nil, err
	}

	// Получаем информацию о БД
	dbInfo, err := ds.dbService.GetDatabaseInfo(databaseName, true)
	if err != nil {
		return nil, fmt.Errorf("failed to get database info: %w", err)
	}

	// Проверяем существует ли локальная БД
	localExists, err := ds.dbService.DatabaseExists(databaseName, false)
	if err != nil {
		return nil, fmt.Errorf("failed to check local database: %w", err)
	}

	result := &models.SyncResult{
		Success:      true, // В dry-run режиме всегда успешно если валидация прошла
		DatabaseName: databaseName,
		DumpSize:     dbInfo.Size,
		TablesCount:  dbInfo.Tables,
		StartTime:    time.Now(),
		EndTime:      time.Now(),
		Duration:     0, // Для dry-run
	}

	// Добавляем информацию о том что произойдет
	action := "create"
	if localExists {
		action = "replace"
	}

	result.Error = fmt.Sprintf("DRY RUN: Would %s local database '%s' with %d tables (%.1f MB)",
		action, databaseName, dbInfo.Tables, float64(dbInfo.Size)/(1024*1024))

	return result, nil
}

// CreateDump создает дамп удаленной базы данных
func (ds *DumpService) CreateDump(databaseName string, dryRun bool) (*models.SyncResult, string, error) {
	startTime := time.Now()

	// Получаем информацию о базе данных для правильного подсчета таблиц
	dbInfo, err := ds.dbService.GetDatabaseInfo(databaseName, true) // true для remote
	if err != nil {
		return nil, "", fmt.Errorf("failed to get database info: %w", err)
	}

	if dryRun {
		result, err := ds.PlanDumpOperation(databaseName)
		return result, "", err
	}

	// Создаём временный файл для дампа
	tempDir := ds.config.Dump.TempDir
	if tempDir == "" {
		tempDir = "./tmp"
	}

	// Убеждаемся что директория существует
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	dumpPath := filepath.Join(tempDir, fmt.Sprintf("%s_%d.sql", databaseName, time.Now().Unix()))

	// Строим команду mysqldump с оптимизациями для скорости
	cmd := exec.Command(
		ds.config.Dump.MysqldumpPath,
		"--single-transaction",
		"--routines",
		"--triggers",
		"--no-tablespaces",
		"--set-gtid-purged=OFF", // Исправляет проблему с GTID
		"--opt",                 // Включает несколько оптимизаций
		"--quick",               // Получает строки по одной (экономит память)
		"--compress",            // Сжимает соединение
		"--disable-keys",        // Отключает ключи при вставке (ускоряет)
		"--extended-insert",     // Использует многострочные INSERT (быстрее)
		"--host="+ds.config.Remote.Host,
		"--port="+fmt.Sprintf("%d", ds.config.Remote.Port),
		"--user="+ds.config.Remote.User,
		"--password="+ds.config.Remote.Password,
		databaseName,
	)

	// Создаём файл дампа
	file, err := os.Create(dumpPath)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create dump file: %w", err)
	}
	defer file.Close()

	cmd.Stdout = file

	// Запускаем команду в фоне
	if err := cmd.Start(); err != nil {
		os.Remove(dumpPath)
		return nil, "", fmt.Errorf("failed to start mysqldump: %w", err)
	}

	// Мониторим прогресс
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	// Показываем прогресс каждые 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	fmt.Printf("📦 Dumping database '%s' (estimated %s)...\n", databaseName, FormatSize(dbInfo.Size))

	for {
		select {
		case err := <-done:
			// Команда завершена
			if err != nil {
				os.Remove(dumpPath)
				return nil, "", fmt.Errorf("mysqldump failed: %w", err)
			}

			// Финальный прогресс
			fmt.Printf("\r✅ Dump completed                                        \n")

			// Получаем информацию о файле
			fileInfo, err := file.Stat()
			if err != nil {
				return nil, "", fmt.Errorf("failed to get dump file info: %w", err)
			}

			endTime := time.Now()

			result := &models.SyncResult{
				Success:      true,
				DatabaseName: databaseName,
				Duration:     endTime.Sub(startTime),
				DumpSize:     fileInfo.Size(),
				TablesCount:  dbInfo.Tables, // Используем реальное количество таблиц
				StartTime:    startTime,
				EndTime:      endTime,
			}

			return result, dumpPath, nil

		case <-ticker.C:
			// Обновляем прогресс
			if stat, err := os.Stat(dumpPath); err == nil {
				progress := float64(stat.Size()) / float64(dbInfo.Size)
				if progress > 1.0 {
					progress = 1.0
				}

				// Показываем прогресс с размером файла
				fmt.Printf("\r📦 Dumping... %s / %s (%.1f%%)     ",
					FormatSize(stat.Size()),
					FormatSize(dbInfo.Size),
					progress*100)
			}
		}
	}
}

// RestoreDump восстанавливает дамп в локальную БД
func (ds *DumpService) RestoreDump(dumpPath string, databaseName string, dryRun bool) error {
	if dryRun {
		// В dry-run режиме просто проверяем что файл существует
		if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
			return fmt.Errorf("dump file does not exist: %s", dumpPath)
		}
		return nil
	}

	// Проверяем что файл дампа существует
	if _, err := os.Stat(dumpPath); os.IsNotExist(err) {
		return fmt.Errorf("dump file does not exist: %s", dumpPath)
	}

	// Сначала удаляем существующую БД если она есть
	dropCmd := exec.Command(
		ds.config.Dump.MysqlPath,
		"--host="+ds.config.Local.Host,
		"--port="+fmt.Sprintf("%d", ds.config.Local.Port),
		"--user="+ds.config.Local.User,
		"--password="+ds.config.Local.Password,
		"-e", fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName),
	)

	if err := dropCmd.Run(); err != nil {
		return fmt.Errorf("failed to drop existing database: %w", err)
	}

	// Создаём новую БД
	createCmd := exec.Command(
		ds.config.Dump.MysqlPath,
		"--host="+ds.config.Local.Host,
		"--port="+fmt.Sprintf("%d", ds.config.Local.Port),
		"--user="+ds.config.Local.User,
		"--password="+ds.config.Local.Password,
		"-e", fmt.Sprintf("CREATE DATABASE %s", databaseName),
	)

	if err := createCmd.Run(); err != nil {
		return fmt.Errorf("failed to create database: %w", err)
	}

	// Получаем размер файла дампа для прогресса
	dumpInfo, err := os.Stat(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to get dump file info: %w", err)
	}

	// Импортируем дамп с оптимизациями
	restoreCmd := exec.Command(
		ds.config.Dump.MysqlPath,
		"--host="+ds.config.Local.Host,
		"--port="+fmt.Sprintf("%d", ds.config.Local.Port),
		"--user="+ds.config.Local.User,
		"--password="+ds.config.Local.Password,
		"--compress",               // Сжимает соединение
		"--quick",                  // Экономит память
		"--max_allowed_packet=1GB", // Увеличивает размер пакета
		databaseName,
	)

	// Открываем файл дампа
	file, err := os.Open(dumpPath)
	if err != nil {
		return fmt.Errorf("failed to open dump file: %w", err)
	}
	defer file.Close()

	// Захватываем stderr для диагностики ошибок
	var stderr strings.Builder
	restoreCmd.Stdin = file
	restoreCmd.Stderr = &stderr

	// Запускаем команду в фоне для мониторинга прогресса
	if err := restoreCmd.Start(); err != nil {
		return fmt.Errorf("failed to start mysql restore: %w", err)
	}

	// Мониторим прогресс восстановления
	done := make(chan error)
	go func() {
		done <- restoreCmd.Wait()
	}()

	// Показываем прогресс каждые 500ms
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	fmt.Printf("🔄 Restoring dump to local database '%s' (%s)...\n", databaseName, FormatSize(dumpInfo.Size()))

	startTime := time.Now()

	for {
		select {
		case err := <-done:
			// Команда завершена
			if err != nil {
				errMsg := "mysql restore failed"
				if stderrOutput := stderr.String(); stderrOutput != "" {
					errMsg += ": " + strings.TrimSpace(stderrOutput)
				}
				return fmt.Errorf("%s: %w", errMsg, err)
			}

			// Финальный прогресс
			fmt.Printf("\r✅ Restore completed                                        \n")
			return nil

		case <-ticker.C:
			// Оцениваем прогресс по времени (приблизительно)
			elapsed := time.Since(startTime)
			if elapsed.Seconds() > 0 {
				// Примерная скорость восстановления: 10-50 MB/min для типичных БД
				estimatedSpeed := int64(20 * 1024 * 1024 / 60) // 20MB/min приблизительно
				estimatedRead := int64(elapsed.Seconds()) * estimatedSpeed

				if estimatedRead > dumpInfo.Size() {
					estimatedRead = dumpInfo.Size()
				}

				progress := float64(estimatedRead) / float64(dumpInfo.Size()) * 100

				fmt.Printf("\r🔄 Restoring... %s / %s (%.1f%%)     ",
					FormatSize(estimatedRead),
					FormatSize(dumpInfo.Size()),
					progress)
			}
		}
	}
}

// ExecuteSync выполняет полную синхронизацию базы данных
func (ds *DumpService) ExecuteSync(databaseName string) (*models.SyncResult, error) {
	startTime := time.Now()

	// Валидация операции
	if err := ds.ValidateDumpOperation(databaseName); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Создаём дамп
	fmt.Printf("📦 Creating dump of remote database '%s'...\n", databaseName)
	dumpStartTime := time.Now()
	dumpResult, dumpPath, err := ds.CreateDump(databaseName, false)
	if err != nil {
		return nil, fmt.Errorf("dump creation failed: %w", err)
	}
	dumpDuration := time.Since(dumpStartTime)

	// Восстанавливаем дамп
	fmt.Printf("🔄 Restoring dump to local database '%s'...\n", databaseName)
	restoreStartTime := time.Now()
	if err := ds.RestoreDump(dumpPath, databaseName, false); err != nil {
		// Удаляем файл дампа при ошибке
		os.Remove(dumpPath)
		return nil, fmt.Errorf("dump restoration failed: %w", err)
	}
	restoreDuration := time.Since(restoreStartTime)

	// Очищаем временный файл
	if err := os.Remove(dumpPath); err != nil {
		fmt.Printf("⚠️  Warning: failed to cleanup dump file: %v\n", err)
	}

	endTime := time.Now()

	return &models.SyncResult{
		Success:         true,
		DatabaseName:    databaseName,
		Duration:        endTime.Sub(startTime),
		DumpDuration:    dumpDuration,
		RestoreDuration: restoreDuration,
		DumpSize:        dumpResult.DumpSize,
		TablesCount:     dumpResult.TablesCount,
		StartTime:       startTime,
		EndTime:         endTime,
	}, nil
}

// validateMysqldumpAvailable проверяет доступность mysqldump
func (ds *DumpService) validateMysqldumpAvailable() error {
	cmd := exec.Command(ds.config.Dump.MysqldumpPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysqldump not found at '%s': %w", ds.config.Dump.MysqldumpPath, err)
	}
	return nil
}

// validateMysqlAvailable проверяет доступность mysql клиента
func (ds *DumpService) validateMysqlAvailable() error {
	cmd := exec.Command(ds.config.Dump.MysqlPath, "--version")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("mysql client not found at '%s': %w", ds.config.Dump.MysqlPath, err)
	}
	return nil
}

// validateTempDirectory проверяет доступность временной директории
func (ds *DumpService) validateTempDirectory() error {
	// Создаем директорию если не существует
	if err := os.MkdirAll(ds.config.Dump.TempDir, 0755); err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Проверяем права на запись
	testFile := filepath.Join(ds.config.Dump.TempDir, "test_write.tmp")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("temp directory is not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// GetDumpCommand возвращает команду mysqldump (для отображения в dry-run)
func (ds *DumpService) GetDumpCommand(databaseName string) []string {
	args := ds.config.Remote.GetMysqldumpArgs(databaseName)
	return append([]string{ds.config.Dump.MysqldumpPath}, args...)
}

// GetRestoreCommand возвращает команду mysql для восстановления (для отображения в dry-run)
func (ds *DumpService) GetRestoreCommand(databaseName string) []string {
	args := ds.config.Local.GetMysqlArgs(databaseName)
	return append([]string{ds.config.Dump.MysqlPath}, args...)
}

// GetSafetyChecks возвращает список проверок безопасности
func (ds *DumpService) GetSafetyChecks(databaseName string) ([]string, error) {
	var checks []string

	// Проверяем что это не системная БД
	systemDatabases := []string{"information_schema", "performance_schema", "mysql", "sys"}
	for _, sysDB := range systemDatabases {
		if databaseName == sysDB {
			checks = append(checks, fmt.Sprintf("❌ CRITICAL: '%s' is a system database", databaseName))
			return checks, fmt.Errorf("cannot sync system database")
		}
	}
	checks = append(checks, "✅ Database name is safe")

	// Проверяем подключения
	remoteConn, err := ds.dbService.TestConnection(true)
	if err != nil || !remoteConn.Connected {
		checks = append(checks, "❌ Remote server connection failed")
		return checks, fmt.Errorf("remote connection failed")
	}
	checks = append(checks, "✅ Remote server connection OK")

	localConn, err := ds.dbService.TestConnection(false)
	if err != nil || !localConn.Connected {
		checks = append(checks, "❌ Local server connection failed")
		return checks, fmt.Errorf("local connection failed")
	}
	checks = append(checks, "✅ Local server connection OK")

	// Проверяем утилиты
	if err := ds.validateMysqldumpAvailable(); err != nil {
		checks = append(checks, "❌ mysqldump not available")
		return checks, err
	}
	checks = append(checks, "✅ mysqldump available")

	if err := ds.validateMysqlAvailable(); err != nil {
		checks = append(checks, "❌ mysql client not available")
		return checks, err
	}
	checks = append(checks, "✅ mysql client available")

	// Проверяем существование БД
	exists, err := ds.dbService.DatabaseExists(databaseName, true)
	if err != nil {
		checks = append(checks, "❌ Failed to check remote database")
		return checks, err
	}
	if !exists {
		checks = append(checks, fmt.Sprintf("❌ Database '%s' not found on remote", databaseName))
		return checks, fmt.Errorf("database not found")
	}
	checks = append(checks, fmt.Sprintf("✅ Database '%s' exists on remote", databaseName))

	// Проверяем существование локальной БД
	localExists, err := ds.dbService.DatabaseExists(databaseName, false)
	if err != nil {
		checks = append(checks, "❌ Failed to check local database")
		return checks, err
	}
	if localExists {
		checks = append(checks, fmt.Sprintf("⚠️  Local database '%s' will be REPLACED", databaseName))
	} else {
		checks = append(checks, fmt.Sprintf("✅ Local database '%s' will be created", databaseName))
	}

	return checks, nil
}
