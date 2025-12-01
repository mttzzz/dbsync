package services

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

// MySQLShellService предоставляет функции для создания и восстановления дампов через MySQL Shell
type MySQLShellService struct {
	config      *config.Config
	dbService   DatabaseServiceInterface
	mysqlshPath string
}

// NewMySQLShellService создает новый экземпляр MySQLShellService
func NewMySQLShellService(cfg *config.Config, dbService DatabaseServiceInterface) *MySQLShellService {
	return &MySQLShellService{
		config:    cfg,
		dbService: dbService,
	}
}

// filterMySQLShellOutput фильтрует вывод mysqlsh, показывая только важную информацию
func filterMySQLShellOutput(r io.Reader, _ io.Writer) {
	scanner := bufio.NewScanner(r)
	// Увеличиваем буфер для длинных строк
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	// Просто читаем и отбрасываем вывод - итоги показываем сами
	for scanner.Scan() {
		// Игнорируем весь вывод - результаты покажем в конце
	}
} // findMySQLShell ищет путь к mysqlsh
func (s *MySQLShellService) findMySQLShell() (string, error) {
	// Если уже найден
	if s.mysqlshPath != "" {
		return s.mysqlshPath, nil
	}

	// Пробуем найти в PATH
	if path, err := exec.LookPath("mysqlsh"); err == nil {
		s.mysqlshPath = path
		return path, nil
	}

	// Стандартные пути установки
	var paths []string
	if runtime.GOOS == "windows" {
		paths = []string{
			`C:\Program Files\MySQL\MySQL Shell 8.4\bin\mysqlsh.exe`,
			`C:\Program Files\MySQL\MySQL Shell 8.0\bin\mysqlsh.exe`,
			`C:\Program Files (x86)\MySQL\MySQL Shell 8.4\bin\mysqlsh.exe`,
			`C:\Program Files (x86)\MySQL\MySQL Shell 8.0\bin\mysqlsh.exe`,
		}
	} else {
		paths = []string{
			"/usr/bin/mysqlsh",
			"/usr/local/bin/mysqlsh",
			"/opt/mysql-shell/bin/mysqlsh",
		}
	}

	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			s.mysqlshPath = p
			return p, nil
		}
	}

	return "", fmt.Errorf("mysqlsh not found. Please install MySQL Shell: https://dev.mysql.com/downloads/shell/")
}

// ValidateDumpOperation проверяет возможность выполнения операции дампа
func (s *MySQLShellService) ValidateDumpOperation(databaseName string) error {
	// Валидация имени базы данных
	if err := s.dbService.ValidateDatabaseName(databaseName); err != nil {
		return fmt.Errorf("invalid database name: %w", err)
	}

	// Проверяем что удаленная БД существует
	exists, err := s.dbService.DatabaseExists(databaseName, true)
	if err != nil {
		return fmt.Errorf("failed to check remote database: %w", err)
	}
	if !exists {
		return fmt.Errorf("database '%s' not found on remote server", databaseName)
	}

	// Проверяем подключение к удаленному серверу
	remoteConn, err := s.dbService.TestConnection(true)
	if err != nil || !remoteConn.Connected {
		return fmt.Errorf("cannot connect to remote server: %s", remoteConn.Error)
	}

	// Проверяем подключение к локальному серверу
	localConn, err := s.dbService.TestConnection(false)
	if err != nil || !localConn.Connected {
		return fmt.Errorf("cannot connect to local server: %s", localConn.Error)
	}

	// Проверяем что MySQL Shell доступен
	_, err = s.findMySQLShell()
	if err != nil {
		return err
	}

	return nil
}

// buildRemoteURI создает URI для подключения к удаленному серверу (без пароля)
func (s *MySQLShellService) buildRemoteURI() string {
	return fmt.Sprintf("mysql://%s@%s:%d",
		s.config.Remote.User,
		s.config.Remote.Host,
		s.config.Remote.Port,
	)
}

// buildLocalURI создает URI для подключения к локальному серверу (без пароля)
func (s *MySQLShellService) buildLocalURI() string {
	return fmt.Sprintf("mysql://%s@%s:%d",
		s.config.Local.User,
		s.config.Local.Host,
		s.config.Local.Port,
	)
}

// CreateDump создает дамп удаленной базы данных через MySQL Shell
func (s *MySQLShellService) CreateDump(databaseName string, dryRun bool) (*models.SyncResult, string, error) {
	startTime := time.Now()

	// Получаем информацию о базе данных
	dbInfo, err := s.dbService.GetDatabaseInfo(databaseName, true)
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
		result.Error = fmt.Sprintf("DRY RUN: Would dump database '%s' using MySQL Shell with %d threads",
			databaseName, s.config.Dump.Threads)
		return result, "", nil
	}

	// Создаём директорию для дампа
	tempDir := os.TempDir()
	dumpDir := filepath.Join(tempDir, fmt.Sprintf("mysqlsh_%s_%d", databaseName, time.Now().Unix()))
	if err := os.MkdirAll(dumpDir, 0755); err != nil {
		return nil, "", fmt.Errorf("failed to create dump directory: %w", err)
	}

	mysqlshPath, err := s.findMySQLShell()
	if err != nil {
		return nil, "", err
	}

	// Строим команду mysqlsh для дампа
	args := []string{
		"--uri", s.buildRemoteURI(),
		fmt.Sprintf("--password=%s", s.config.Remote.Password),
		"--", "util", "dump-schemas", databaseName,
		fmt.Sprintf("--outputUrl=%s", dumpDir),
		fmt.Sprintf("--threads=%d", s.config.Dump.Threads),
		"--consistent=false",      // Без блокировок для managed MySQL
		"--skipConsistencyChecks", // Пропускаем проверку GTID
		"--compression=zstd",
	}

	cmd := exec.Command(mysqlshPath, args...)
	cmd.Env = append(os.Environ(), "MYSQLSH_TERM_COLOR_MODE=nocolor")

	// Показываем статус в одной строке (будет перезаписана)
	fmt.Printf("📦 Dumping %s (%d tables)...", databaseName, dbInfo.Tables)

	// Создаём pipe для фильтрации вывода
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		os.RemoveAll(dumpDir)
		return nil, "", fmt.Errorf("failed to start mysqlsh: %w", err)
	}

	// Фильтруем вывод в отдельных горутинах
	go filterMySQLShellOutput(stdoutPipe, os.Stdout)
	go filterMySQLShellOutput(stderrPipe, os.Stderr)

	err = cmd.Wait()
	if err != nil {
		os.RemoveAll(dumpDir)
		return nil, "", fmt.Errorf("mysqlsh dump failed: %w", err)
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

	// Перезаписываем строку с результатом
	fmt.Printf("\r✅ Dumped %s (%d tables) → %s in %v\n", databaseName, dbInfo.Tables, FormatSize(totalSize), endTime.Sub(startTime).Round(time.Second))

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

// RestoreDump восстанавливает дамп в локальную БД через MySQL Shell
func (s *MySQLShellService) RestoreDump(dumpDir string, databaseName string, dryRun bool) error {
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

	// Включаем local_infile на локальном сервере (требуется для MySQL Shell)
	enableLocalInfile := exec.Command(
		"mysql",
		"--host="+s.config.Local.Host,
		"--port="+fmt.Sprintf("%d", s.config.Local.Port),
		"--user="+s.config.Local.User,
		"--password="+s.config.Local.Password,
		"-e", "SET GLOBAL local_infile = 1",
	)
	if output, err := enableLocalInfile.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to enable local_infile: %w\nOutput: %s", err, string(output))
	}

	// Проверяем существует ли локальная база данных
	localExists, err := s.dbService.DatabaseExists(databaseName, false)
	if err != nil {
		return fmt.Errorf("failed to check if local database exists: %w", err)
	}

	if localExists {
		// Убиваем все сессии, подключённые к этой БД
		killQuery := fmt.Sprintf(`SELECT GROUP_CONCAT(id) FROM information_schema.processlist WHERE db = '%s' AND id != CONNECTION_ID()`, databaseName)
		killCmd := exec.Command(
			"mysql",
			"--host="+s.config.Local.Host,
			"--port="+fmt.Sprintf("%d", s.config.Local.Port),
			"--user="+s.config.Local.User,
			"--password="+s.config.Local.Password,
			"-N", "-s",
			"-e", killQuery,
		)
		if output, err := killCmd.Output(); err == nil {
			ids := strings.Split(strings.TrimSpace(string(output)), ",")
			for _, id := range ids {
				if id != "" && id != "NULL" {
					exec.Command(
						"mysql",
						"--host="+s.config.Local.Host,
						"--port="+fmt.Sprintf("%d", s.config.Local.Port),
						"--user="+s.config.Local.User,
						"--password="+s.config.Local.Password,
						"-e", fmt.Sprintf("KILL %s", id),
					).Run()
				}
			}
		}

		dropCmd := exec.Command(
			"mysql",
			"--host="+s.config.Local.Host,
			"--port="+fmt.Sprintf("%d", s.config.Local.Port),
			"--user="+s.config.Local.User,
			"--password="+s.config.Local.Password,
			"-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`", databaseName),
		)

		if output, err := dropCmd.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to drop existing database: %w\nOutput: %s", err, string(output))
		}
	}

	// Создаём новую БД
	createCmd := exec.Command(
		"mysql",
		"--host="+s.config.Local.Host,
		"--port="+fmt.Sprintf("%d", s.config.Local.Port),
		"--user="+s.config.Local.User,
		"--password="+s.config.Local.Password,
		"-e", fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s`", databaseName),
	)

	if output, err := createCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to create database: %w\nOutput: %s", err, string(output))
	}

	mysqlshPath, err := s.findMySQLShell()
	if err != nil {
		return err
	}

	// Строим команду mysqlsh для загрузки
	threads := s.config.Dump.Threads
	args := []string{
		"--uri", s.buildLocalURI(),
		fmt.Sprintf("--password=%s", s.config.Local.Password),
		"--", "util", "load-dump", dumpDir,
		fmt.Sprintf("--threads=%d", threads),
		"--deferTableIndexes=all", // Создаём индексы после данных
		"--resetProgress",         // Сбрасываем прогресс предыдущих попыток
		"--ignoreVersion",         // Игнорируем разницу версий MySQL
		"--skipBinlog=true",       // Пропускаем запись в binlog
	}

	cmd := exec.Command(mysqlshPath, args...)
	cmd.Env = append(os.Environ(), "MYSQLSH_TERM_COLOR_MODE=nocolor")

	// Показываем статус в одной строке (будет перезаписана)
	fmt.Printf("🔄 Restoring %s...", databaseName)

	// Создаём pipe для фильтрации вывода
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	startTime := time.Now()
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mysqlsh: %w", err)
	}

	// Фильтруем вывод в отдельных горутинах
	go filterMySQLShellOutput(stdoutPipe, os.Stdout)
	go filterMySQLShellOutput(stderrPipe, os.Stderr)

	err = cmd.Wait()
	if err != nil {
		return fmt.Errorf("mysqlsh load failed: %w", err)
	}

	// Перезаписываем строку с результатом
	fmt.Printf("\r✅ Restored %s in %v                    \n", databaseName, time.Since(startTime).Round(time.Second))

	return nil
}

// ExecuteSync выполняет полную синхронизацию базы данных через MySQL Shell
func (s *MySQLShellService) ExecuteSync(databaseName string) (*models.SyncResult, error) {
	startTime := time.Now()

	// Валидация операции
	if err := s.ValidateDumpOperation(databaseName); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Создаем дамп
	dumpResult, dumpDir, err := s.CreateDump(databaseName, false)
	if err != nil {
		return nil, fmt.Errorf("dump creation failed: %w", err)
	}

	// Очищаем директорию дампа после завершения
	defer func() {
		if dumpDir != "" {
			os.RemoveAll(dumpDir)
		}
	}()

	// Восстанавливаем дамп
	restoreStart := time.Now()
	if err := s.RestoreDump(dumpDir, databaseName, false); err != nil {
		return nil, fmt.Errorf("restore failed: %w", err)
	}
	restoreDuration := time.Since(restoreStart)

	endTime := time.Now()

	result := &models.SyncResult{
		Success:         true,
		DatabaseName:    databaseName,
		Duration:        endTime.Sub(startTime),
		DumpDuration:    dumpResult.Duration,
		RestoreDuration: restoreDuration,
		DumpSize:        dumpResult.DumpSize,
		TablesCount:     dumpResult.TablesCount,
		StartTime:       startTime,
		EndTime:         endTime,
	}

	return result, nil
}

// Cleanup удаляет временные файлы
func (s *MySQLShellService) Cleanup(dumpDir string) error {
	if dumpDir != "" {
		return os.RemoveAll(dumpDir)
	}
	return nil
}
