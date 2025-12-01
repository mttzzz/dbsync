package services

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

// MyDumperService предоставляет функции для создания и восстановления дампов через mydumper/myloader
type MyDumperService struct {
	config    *config.Config
	dbService DatabaseServiceInterface
}

// NewMyDumperService создает новый экземпляр MyDumperService
func NewMyDumperService(cfg *config.Config, dbService DatabaseServiceInterface) *MyDumperService {
	return &MyDumperService{
		config:    cfg,
		dbService: dbService,
	}
}

// ValidateDumpOperation проверяет возможность выполнения операции дампа через mydumper
func (mds *MyDumperService) ValidateDumpOperation(databaseName string) error {
	// Валидация имени базы данных
	if err := mds.dbService.ValidateDatabaseName(databaseName); err != nil {
		return fmt.Errorf("invalid database name: %w", err)
	}

	// Проверяем что удаленная БД существует
	exists, err := mds.dbService.DatabaseExists(databaseName, true)
	if err != nil {
		return fmt.Errorf("failed to check remote database: %w", err)
	}
	if !exists {
		return fmt.Errorf("database '%s' not found on remote server", databaseName)
	}

	// Проверяем подключение к удаленному серверу
	remoteConn, err := mds.dbService.TestConnection(true)
	if err != nil || !remoteConn.Connected {
		return fmt.Errorf("cannot connect to remote server: %s", remoteConn.Error)
	}

	// Проверяем подключение к локальному серверу
	localConn, err := mds.dbService.TestConnection(false)
	if err != nil || !localConn.Connected {
		return fmt.Errorf("cannot connect to local server: %s", localConn.Error)
	}

	// Проверяем что Docker доступен
	if err := mds.validateDockerAvailable(); err != nil {
		return fmt.Errorf("docker validation failed: %w", err)
	}

	// Проверяем что временная директория доступна для записи
	if err := mds.validateTempDirectory(); err != nil {
		return fmt.Errorf("temp directory validation failed: %w", err)
	}

	return nil
}

// validateDockerAvailable проверяет доступность Docker и чистит старые контейнеры
func (mds *MyDumperService) validateDockerAvailable() error {
	cmd := exec.Command("docker", "version", "--format", "{{.Server.Version}}")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("docker not available: %w", err)
	}
	fmt.Printf("🐳 Docker version: %s\n", strings.TrimSpace(string(output)))

	// Очищаем старые/зависшие контейнеры mydumper (если остались от прерванных операций)
	listCmd := exec.Command("docker", "ps", "-aq", "--filter", "ancestor=mydumper/mydumper")
	if ids, err := listCmd.Output(); err == nil && len(strings.TrimSpace(string(ids))) > 0 {
		for _, id := range strings.Fields(string(ids)) {
			exec.Command("docker", "rm", "-f", id).Run()
		}
	}

	return nil
}

// validateTempDirectory проверяет доступность системной временной директории
func (mds *MyDumperService) validateTempDirectory() error {
	tempDir := os.TempDir()

	// Проверяем права на запись
	testFile := filepath.Join(tempDir, "dbsync_test_write.tmp")
	f, err := os.Create(testFile)
	if err != nil {
		return fmt.Errorf("temp directory is not writable: %w", err)
	}
	f.Close()
	os.Remove(testFile)

	return nil
}

// getAbsoluteTempDir возвращает путь к системной временной директории
func (mds *MyDumperService) getAbsoluteTempDir() (string, error) {
	return os.TempDir(), nil
}

// convertPathForDocker конвертирует Windows путь для Docker (если необходимо)
func (mds *MyDumperService) convertPathForDocker(path string) string {
	if runtime.GOOS == "windows" {
		// Конвертируем C:\path\to\dir в /c/path/to/dir для Docker
		path = strings.ReplaceAll(path, "\\", "/")
		if len(path) >= 2 && path[1] == ':' {
			path = "/" + strings.ToLower(string(path[0])) + path[2:]
		}
	}
	return path
}

// getDockerHost возвращает адрес хоста для Docker контейнера
func (mds *MyDumperService) getDockerHost(originalHost string) string {
	// Если хост localhost или 127.0.0.1, используем host.docker.internal
	if originalHost == "localhost" || originalHost == "127.0.0.1" {
		return "host.docker.internal"
	}
	return originalHost
}

// GetDatabaseInfoViaDocker получает информацию о БД через Docker mysql (без локального клиента)
func (mds *MyDumperService) GetDatabaseInfoViaDocker(databaseName string, isRemote bool) (*models.Database, error) {
	var host string
	var port int
	var user, password string

	if isRemote {
		host = mds.getDockerHost(mds.config.Remote.Host)
		port = mds.config.Remote.Port
		user = mds.config.Remote.User
		password = mds.config.Remote.Password
	} else {
		host = mds.getDockerHost(mds.config.Local.Host)
		port = mds.config.Local.Port
		user = mds.config.Local.User
		password = mds.config.Local.Password
	}

	// SQL запрос для получения информации о БД
	query := fmt.Sprintf(`
		SELECT 
			COUNT(*) as tables_count,
			COALESCE(SUM(data_length + index_length), 0) as total_size
		FROM information_schema.tables 
		WHERE table_schema = '%s' AND table_type = 'BASE TABLE'
	`, databaseName)

	args := []string{
		"run", "--rm",
		"--network", "host",
		"mysql:8.0",
		"mysql",
		"-h", host,
		"-P", fmt.Sprintf("%d", port),
		"-u", user,
		fmt.Sprintf("-p%s", password),
		"-N", "-s", // No headers, silent
		"-e", query,
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get database info via docker: %w", err)
	}

	// Парсим результат: "tables_count\ttotal_size"
	parts := strings.Fields(strings.TrimSpace(string(output)))
	if len(parts) < 2 {
		return nil, fmt.Errorf("unexpected output format: %s", string(output))
	}

	var tables int
	var size int64
	fmt.Sscanf(parts[0], "%d", &tables)
	fmt.Sscanf(parts[1], "%d", &size)

	return &models.Database{
		Name:   databaseName,
		Tables: tables,
		Size:   size,
	}, nil
}

// DatabaseExistsViaDocker проверяет существование БД через Docker mysql
func (mds *MyDumperService) DatabaseExistsViaDocker(databaseName string, isRemote bool) (bool, error) {
	var host string
	var port int
	var user, password string

	if isRemote {
		host = mds.getDockerHost(mds.config.Remote.Host)
		port = mds.config.Remote.Port
		user = mds.config.Remote.User
		password = mds.config.Remote.Password
	} else {
		host = mds.getDockerHost(mds.config.Local.Host)
		port = mds.config.Local.Port
		user = mds.config.Local.User
		password = mds.config.Local.Password
	}

	query := fmt.Sprintf("SELECT SCHEMA_NAME FROM information_schema.SCHEMATA WHERE SCHEMA_NAME = '%s'", databaseName)

	args := []string{
		"run", "--rm",
		"--network", "host",
		"mysql:8.0",
		"mysql",
		"-h", host,
		"-P", fmt.Sprintf("%d", port),
		"-u", user,
		fmt.Sprintf("-p%s", password),
		"-N", "-s",
		"-e", query,
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.Output()
	if err != nil {
		return false, fmt.Errorf("failed to check database via docker: %w", err)
	}

	return strings.TrimSpace(string(output)) == databaseName, nil
}

// ExecuteMySQLViaDocker выполняет SQL команду через Docker mysql
func (mds *MyDumperService) ExecuteMySQLViaDocker(sql string, isRemote bool) error {
	var host string
	var port int
	var user, password string

	if isRemote {
		host = mds.getDockerHost(mds.config.Remote.Host)
		port = mds.config.Remote.Port
		user = mds.config.Remote.User
		password = mds.config.Remote.Password
	} else {
		host = mds.getDockerHost(mds.config.Local.Host)
		port = mds.config.Local.Port
		user = mds.config.Local.User
		password = mds.config.Local.Password
	}

	args := []string{
		"run", "--rm",
		"--network", "host",
		"mysql:8.0",
		"mysql",
		"-h", host,
		"-P", fmt.Sprintf("%d", port),
		"-u", user,
		fmt.Sprintf("-p%s", password),
		"-e", sql,
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("mysql command failed: %w\nOutput: %s", err, string(output))
	}

	return nil
}

// CreateDump создает дамп удаленной базы данных через mydumper
func (mds *MyDumperService) CreateDump(databaseName string, dryRun bool) (*models.SyncResult, string, error) {
	startTime := time.Now()

	// Получаем информацию о базе данных
	dbInfo, err := mds.dbService.GetDatabaseInfo(databaseName, true)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get database info: %w", err)
	}

	if dryRun {
		result := &models.SyncResult{
			Success:      true,
			DatabaseName: databaseName,
			DumpSize:     dbInfo.Size,
			TablesCount:  dbInfo.Tables,
			StartTime:    time.Now(),
			EndTime:      time.Now(),
			Duration:     0,
		}
		result.Error = fmt.Sprintf("DRY RUN: Would dump database '%s' using mydumper with %d threads",
			databaseName, mds.config.Dump.Threads)
		return result, "", nil
	}

	// Создаём директорию для дампа
	absPath, err := mds.getAbsoluteTempDir()
	if err != nil {
		return nil, "", err
	}

	dumpDir := filepath.Join(absPath, fmt.Sprintf("mydumper_%s_%d", databaseName, time.Now().Unix()))
	if err := os.MkdirAll(dumpDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create dump directory: %w", err)
	}

	// Преобразуем путь для Docker
	dockerDumpDir := mds.convertPathForDocker(dumpDir)

	// Получаем хост для Docker
	remoteHost := mds.getDockerHost(mds.config.Remote.Host)

	// Строим команду mydumper
	args := []string{
		"run", "--rm",
		"--network", "host", // Используем host networking для доступа к MySQL
		"-v", fmt.Sprintf("%s:/dump", dockerDumpDir),
		mds.config.Dump.MyDumperImage,
		"mydumper",
		"--host", remoteHost,
		"--port", fmt.Sprintf("%d", mds.config.Remote.Port),
		"--user", mds.config.Remote.User,
		"--password", mds.config.Remote.Password,
		"--database", databaseName,
		"--outputdir", "/dump",
		"--threads", fmt.Sprintf("%d", mds.config.Dump.Threads),
		"--rows", fmt.Sprintf("%d", mds.config.Dump.ChunkSize),
		"--compress-protocol", // Сжатие при передаче по сети
		"--triggers",
		"--routines",
		"--events",
		"--sync-thread-lock-mode=NO_LOCK", // Без блокировок (для managed MySQL)
		"--skip-constraints",              // FK создаются отдельным файлом
		"--skip-indexes",                  // Индексы создаются отдельным файлом (быстрее восстановление)
		"--verbose", "3",
	}

	// Добавляем сжатие файлов если включено
	if mds.config.Dump.Compress {
		args = append(args, "--compress")
	}

	cmd := exec.Command("docker", args...)

	fmt.Printf("🚀 Starting mydumper with %d threads...\n", mds.config.Dump.Threads)
	fmt.Printf("📦 Dumping database '%s' (%d tables)...\n", databaseName, dbInfo.Tables)

	// Захватываем вывод
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.RemoveAll(dumpDir)
		return nil, "", fmt.Errorf("mydumper failed: %w\nOutput: %s", err, string(output))
	}

	// Подсчитываем размер дампа
	var totalSize int64
	err = filepath.Walk(dumpDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("failed to calculate dump size: %w", err)
	}

	endTime := time.Now()

	fmt.Printf("✅ Dump completed: %s in %v\n", FormatSize(totalSize), endTime.Sub(startTime).Round(time.Second))

	result := &models.SyncResult{
		Success:      true,
		DatabaseName: databaseName,
		Duration:     endTime.Sub(startTime),
		DumpSize:     totalSize,
		TablesCount:  dbInfo.Tables,
		StartTime:    startTime,
		EndTime:      endTime,
	}

	return result, dumpDir, nil
}

// RestoreDump восстанавливает дамп в локальную БД через myloader
func (mds *MyDumperService) RestoreDump(dumpDir string, databaseName string, dryRun bool) error {
	if dryRun {
		if _, err := os.Stat(dumpDir); os.IsNotExist(err) {
			return fmt.Errorf("dump directory does not exist: %s", dumpDir)
		}
		return nil
	}

	// Проверяем что директория дампа существует
	if _, err := os.Stat(dumpDir); os.IsNotExist(err) {
		return fmt.Errorf("dump directory does not exist: %s", dumpDir)
	}

	// Проверяем существует ли локальная база данных перед попыткой удаления
	localExists, err := mds.dbService.DatabaseExists(databaseName, false)
	if err != nil {
		return fmt.Errorf("failed to check if local database exists: %w", err)
	}

	// Используем локальный mysql клиент для drop/create (не Docker)
	if localExists {
		// Сначала убиваем все сессии, подключённые к этой БД (иначе DROP зависнет)
		fmt.Printf("🔪 Killing existing connections to '%s'...\n", databaseName)
		killCmd := exec.Command(
			"mysql",
			"--host="+mds.config.Local.Host,
			"--port="+fmt.Sprintf("%d", mds.config.Local.Port),
			"--user="+mds.config.Local.User,
			"--password="+mds.config.Local.Password,
			"-e", fmt.Sprintf(`
				SELECT CONCAT('KILL ', id, ';') INTO @kills FROM information_schema.processlist 
				WHERE db = '%s' AND id != CONNECTION_ID() LIMIT 1;
				PREPARE stmt FROM @kills;
				EXECUTE stmt;
			`, databaseName),
		)
		// Игнорируем ошибки - может не быть сессий
		killCmd.Run()

		// Более надёжный способ - убить все сессии через цикл
		killAllCmd := exec.Command(
			"mysql",
			"--host="+mds.config.Local.Host,
			"--port="+fmt.Sprintf("%d", mds.config.Local.Port),
			"--user="+mds.config.Local.User,
			"--password="+mds.config.Local.Password,
			"-N", "-e", fmt.Sprintf(`SELECT id FROM information_schema.processlist WHERE db = '%s' AND id != CONNECTION_ID()`, databaseName),
		)
		if output, err := killAllCmd.Output(); err == nil && len(output) > 0 {
			ids := strings.Fields(string(output))
			for _, id := range ids {
				exec.Command(
					"mysql",
					"--host="+mds.config.Local.Host,
					"--port="+fmt.Sprintf("%d", mds.config.Local.Port),
					"--user="+mds.config.Local.User,
					"--password="+mds.config.Local.Password,
					"-e", fmt.Sprintf("KILL %s", id),
				).Run()
			}
			fmt.Printf("   Killed %d connections\n", len(ids))
		}

		fmt.Printf("🗑️  Dropping existing local database '%s'...\n", databaseName)
		dropCmd := exec.Command(
			"mysql",
			"--host="+mds.config.Local.Host,
			"--port="+fmt.Sprintf("%d", mds.config.Local.Port),
			"--user="+mds.config.Local.User,
			"--password="+mds.config.Local.Password,
			"-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", databaseName),
		)

		if output, err := dropCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to drop existing database: %w\nOutput: %s", err, string(output))
		}
	}

	// Создаём новую БД используя локальный mysql клиент
	fmt.Printf("🔨 Creating local database '%s'...\n", databaseName)
	createCmd := exec.Command(
		"mysql",
		"--host="+mds.config.Local.Host,
		"--port="+fmt.Sprintf("%d", mds.config.Local.Port),
		"--user="+mds.config.Local.User,
		"--password="+mds.config.Local.Password,
		"-e", fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", databaseName),
	)

	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create database: %w\nOutput: %s", err, string(output))
	}

	// Преобразуем путь для Docker
	absDumpDir, err := filepath.Abs(dumpDir)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}
	dockerDumpDir := mds.convertPathForDocker(absDumpDir)

	// Получаем хост для Docker (myloader подключается к локальному MySQL)
	localHost := mds.getDockerHost(mds.config.Local.Host)

	// Строим команду myloader с оптимальными настройками
	threads := mds.config.Dump.Threads
	args := []string{
		"run", "--rm",
		"--network", "host",
		"-v", fmt.Sprintf("%s:/dump", dockerDumpDir),
		mds.config.Dump.MyDumperImage,
		"myloader",
		"--host", localHost,
		"--port", fmt.Sprintf("%d", mds.config.Local.Port),
		"--user", mds.config.Local.User,
		"--password", mds.config.Local.Password,
		"--database", databaseName,
		"--directory", "/dump",

		// === THREADS ===
		"--threads", fmt.Sprintf("%d", threads),
		"--max-threads-per-table", fmt.Sprintf("%d", threads), // Параллельный импорт одной таблицы
		"--max-threads-for-schema-creation", "1", // Схемы в 1 поток (FK требуют порядка)
		"--max-threads-for-index-creation", fmt.Sprintf("%d", threads), // Индексы параллельно

		// === SPEED OPTIMIZATIONS ===
		// Индексы и FK создаются ПОСЛЕ загрузки всех данных (т.к. mydumper экспортировал их отдельно)
		"--optimize-keys", // Оптимизация: данные -> потом индексы
		"--skip-post",     // Пропускаем triggers, procedures, events (для dev не нужны)

		// === TRANSACTION TUNING ===
		"--queries-per-transaction", "50000",

		// === OTHER ===
		"--skip-definer",
		"--verbose", "1",
	}

	cmd := exec.Command("docker", args...)

	fmt.Printf("🔄 Restoring dump to local database '%s' with %d threads...\n", databaseName, threads)
	fmt.Printf("   Options: innodb-optimize-keys, skip-post\n")

	// Стримим вывод myloader в реальном времени чтобы видеть прогресс
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("myloader failed: %w", err)
	}

	fmt.Printf("✅ Restore completed in %v\n", time.Since(startTime).Round(time.Second))

	return nil
}

// ExecuteSync выполняет полную синхронизацию базы данных через mydumper/myloader
func (mds *MyDumperService) ExecuteSync(databaseName string) (*models.SyncResult, error) {
	startTime := time.Now()

	// Валидация операции
	if err := mds.ValidateDumpOperation(databaseName); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Создаём дамп
	fmt.Printf("📦 Creating dump of remote database '%s' using mydumper...\n", databaseName)
	dumpStartTime := time.Now()
	dumpResult, dumpDir, err := mds.CreateDump(databaseName, false)
	if err != nil {
		return nil, fmt.Errorf("dump creation failed: %w", err)
	}
	dumpDuration := time.Since(dumpStartTime)

	// Восстанавливаем дамп
	fmt.Printf("🔄 Restoring dump to local database '%s' using myloader...\n", databaseName)
	restoreStartTime := time.Now()
	if err := mds.RestoreDump(dumpDir, databaseName, false); err != nil {
		// Удаляем директорию дампа при ошибке
		os.RemoveAll(dumpDir)
		return nil, fmt.Errorf("dump restoration failed: %w", err)
	}
	restoreDuration := time.Since(restoreStartTime)

	// Очищаем временную директорию
	if err := os.RemoveAll(dumpDir); err != nil {
		fmt.Printf("⚠️  Warning: failed to cleanup dump directory: %v\n", err)
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

// PullDockerImage загружает Docker образ mydumper
func (mds *MyDumperService) PullDockerImage() error {
	fmt.Printf("🐳 Pulling Docker image: %s...\n", mds.config.Dump.MyDumperImage)

	cmd := exec.Command("docker", "pull", mds.config.Dump.MyDumperImage)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to pull Docker image: %w", err)
	}

	fmt.Println("✅ Docker image pulled successfully")
	return nil
}

// GetMethod возвращает название метода дампа
func (mds *MyDumperService) GetMethod() string {
	return "mydumper"
}
