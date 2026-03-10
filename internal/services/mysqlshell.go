package services

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"db-sync-cli/internal/config"
	"db-sync-cli/internal/models"
)

// MySQLShellService предоставляет функции для создания и восстановления дампов через MySQL Shell
type MySQLShellService struct {
	config      *config.Config
	dbService   DatabaseServiceInterface
	mysqlshPath string
	quiet       bool
}

type mysqlShellParsedProgress struct {
	Percent        float64
	BytesCompleted int64
	BytesTotal     int64
	BytesPerSecond float64
	ETA            time.Duration
	Message        string
}

var (
	progressPercentPattern  = regexp.MustCompile(`(?i)(\d{1,3}(?:[\.,]\d+)?)\s*%`)
	progressSizePairPattern = regexp.MustCompile(`(?i)(\d+(?:[\.,]\d+)?)\s*([kmgt]?i?b|bytes?)\s*/\s*(\d+(?:[\.,]\d+)?)\s*([kmgt]?i?b|bytes?)`)
	progressSpeedPattern    = regexp.MustCompile(`(?i)(\d+(?:[\.,]\d+)?)\s*([kmgt]?i?b|bytes?)\s*/\s*(?:s|sec|second)`)
	progressETAPattern      = regexp.MustCompile(`(?i)(?:eta|remaining|left)\s*[:=]?\s*([0-9hms:\s]+)`)
)

// NewMySQLShellService создает новый экземпляр MySQLShellService
func NewMySQLShellService(cfg *config.Config, dbService DatabaseServiceInterface) *MySQLShellService {
	return &MySQLShellService{
		config:    cfg,
		dbService: dbService,
	}
}

// SetQuiet отключает прямой вывод статусов в stdout/stderr для TUI режима.
func (s *MySQLShellService) SetQuiet(quiet bool) {
	s.quiet = quiet
}

func (s *MySQLShellService) printStatusf(format string, args ...any) {
	if s.quiet {
		return
	}
	fmt.Printf(format, args...)
}

type progressCaptureWriter struct {
	buffer bytes.Buffer
	writer io.Writer
}

func (w *progressCaptureWriter) Write(p []byte) (int, error) {
	_, _ = w.buffer.Write(p)
	if w.writer == nil {
		return len(p), nil
	}
	return w.writer.Write(p)
}

func (w *progressCaptureWriter) String() string {
	return strings.TrimSpace(w.buffer.String())
}

// filterMySQLShellOutput фильтрует вывод mysqlsh и по возможности извлекает progress snapshots.
func filterMySQLShellOutput(r io.Reader, phase models.SyncPhase, databaseName string, metricsFn func() models.TrafficMetrics, observer models.ProgressObserver, _ io.Writer) {
	scanner := bufio.NewScanner(r)
	// Увеличиваем буфер для длинных строк
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if observer == nil || line == "" {
			continue
		}
		parsed, ok := parseMySQLShellProgressLine(line)
		snapshot := models.ProgressSnapshot{
			Phase:        phase,
			DatabaseName: databaseName,
			Timestamp:    time.Now(),
		}
		if metricsFn != nil {
			snapshot.Traffic = metricsFn()
		}
		if ok {
			if statusMessage, statusOK := classifyMySQLShellStatusLine(phase, line); statusOK {
				snapshot.Message = statusMessage
			}
			snapshot.Percent = parsed.Percent
			snapshot.BytesCompleted = parsed.BytesCompleted
			snapshot.BytesTotal = parsed.BytesTotal
			snapshot.ETA = parsed.ETA
			if parsed.BytesPerSecond > 0 {
				snapshot.Traffic.CurrentBytesPerSecond = parsed.BytesPerSecond
			}
			observer(snapshot)
			continue
		}

		statusMessage, statusOK := classifyMySQLShellStatusLine(phase, line)
		if !statusOK {
			continue
		}
		snapshot.Message = statusMessage
		observer(snapshot)
	}
}

func parseMySQLShellProgressLine(line string) (mysqlShellParsedProgress, bool) {
	parsed := mysqlShellParsedProgress{Message: strings.TrimSpace(line)}
	matched := false

	if matches := progressPercentPattern.FindStringSubmatch(line); len(matches) == 2 {
		percent, err := strconv.ParseFloat(strings.ReplaceAll(matches[1], ",", "."), 64)
		if err == nil {
			parsed.Percent = percent
			matched = true
		}
	}
	if matches := progressSizePairPattern.FindStringSubmatch(line); len(matches) == 5 {
		completed := parseHumanBytes(matches[1], matches[2])
		total := parseHumanBytes(matches[3], matches[4])
		if completed > 0 {
			parsed.BytesCompleted = completed
			matched = true
		}
		if total > 0 {
			parsed.BytesTotal = total
			matched = true
			if parsed.Percent == 0 && completed > 0 {
				parsed.Percent = float64(completed) / float64(total) * 100
			}
		}
	}
	if matches := progressSpeedPattern.FindStringSubmatch(line); len(matches) == 3 {
		bytesPerSecond := parseHumanBytes(matches[1], matches[2])
		if bytesPerSecond > 0 {
			parsed.BytesPerSecond = float64(bytesPerSecond)
			matched = true
		}
	}
	if matches := progressETAPattern.FindStringSubmatch(line); len(matches) == 2 {
		if eta, ok := parseLooseDuration(matches[1]); ok {
			parsed.ETA = eta
			matched = true
		}
	}

	return parsed, matched
}

func classifyMySQLShellStatusLine(phase models.SyncPhase, line string) (string, bool) {
	normalized := strings.TrimSpace(line)
	if normalized == "" {
		return "", false
	}
	lower := strings.ToLower(normalized)
	if strings.HasPrefix(lower, "warning:") || strings.HasPrefix(lower, "note:") {
		return "", false
	}
	if strings.HasPrefix(lower, "dump duration:") || strings.HasPrefix(lower, "data load duration:") || strings.HasPrefix(lower, "total duration:") || strings.HasPrefix(lower, "schemas dumped:") || strings.HasPrefix(lower, "tables dumped:") || strings.HasPrefix(lower, "rows written:") || strings.HasPrefix(lower, "bytes written:") || strings.HasPrefix(lower, "uncompressed data size:") || strings.HasPrefix(lower, "compressed data size:") || strings.HasPrefix(lower, "compression ratio:") || strings.HasPrefix(lower, "average uncompressed throughput:") || strings.HasPrefix(lower, "average compressed throughput:") || strings.Contains(lower, " chunks (") || strings.Contains(lower, " ddl files were executed") || strings.Contains(lower, " indexes were built") || strings.Contains(lower, " warnings were reported") {
		return "", false
	}
	if phase == models.SyncPhaseRestore {
		switch {
		case strings.Contains(lower, "loading ddl and data") || strings.Contains(lower, "opening dump") || strings.Contains(lower, "dump is complete") || strings.Contains(lower, "target is mysql") || strings.Contains(lower, "scanning metadata") || strings.Contains(lower, "checking for pre-existing objects") || strings.Contains(lower, "prepar") || strings.Contains(lower, "open") || strings.Contains(lower, "validat"):
			return "Preparing local restore", true
		case strings.Contains(lower, "postamble"):
			return "Finalizing restore", true
		case strings.Contains(lower, "schema") || strings.Contains(lower, "table ddl") || strings.Contains(lower, "view ddl") || strings.Contains(lower, "common preamble") || strings.Contains(lower, "ddl") || strings.Contains(lower, "metadata"):
			return "Applying schema metadata", true
		case strings.Contains(lower, "starting data load") || strings.Contains(lower, "load") || strings.Contains(lower, "import") || strings.Contains(lower, "chunk") || strings.Contains(lower, "rows") || strings.Contains(lower, "data"):
			return "Loading table data", true
		case strings.Contains(lower, "building indexes") || strings.Contains(lower, "indexing") || strings.Contains(lower, "index") || strings.Contains(lower, "analy") || strings.Contains(lower, "constraint"):
			return "Rebuilding indexes", true
		case strings.Contains(lower, "final") || strings.Contains(lower, "finish") || strings.Contains(lower, "complete"):
			return "Finalizing restore", true
		default:
			return "", false
		}
	}
	switch {
	case strings.Contains(lower, "initializ") || strings.Contains(lower, "schemas will be dumped") || strings.Contains(lower, "gather") || strings.Contains(lower, "analy") || strings.Contains(lower, "check") || strings.Contains(lower, "discover") || strings.Contains(lower, "prepar"):
		return "Preparing dump metadata", true
	case strings.Contains(lower, "table metadata"):
		return "Writing table metadata", true
	case strings.Contains(lower, "global ddl") || strings.Contains(lower, "writing ddl") || strings.Contains(lower, "schema") || strings.Contains(lower, "metadata"):
		return "Writing schema metadata", true
	case strings.Contains(lower, "running data dump") || strings.Contains(lower, "starting data dump") || strings.Contains(lower, "dumping") || strings.Contains(lower, "chunk") || strings.Contains(lower, "writing data") || strings.Contains(lower, "rows"):
		return "Streaming table data", true
	case strings.Contains(lower, "final") || strings.Contains(lower, "compress") || strings.Contains(lower, "finish") || strings.Contains(lower, "complete"):
		return "Finalizing dump files", true
	default:
		return "", false
	}
}

func parseHumanBytes(value string, unit string) int64 {
	normalizedValue := strings.ReplaceAll(strings.TrimSpace(value), ",", ".")
	number, err := strconv.ParseFloat(normalizedValue, 64)
	if err != nil {
		return 0
	}
	switch strings.ToUpper(strings.TrimSpace(unit)) {
	case "B", "BYTE", "BYTES":
		return int64(number)
	case "KB", "KIB":
		return int64(number * 1024)
	case "MB", "MIB":
		return int64(number * 1024 * 1024)
	case "GB", "GIB":
		return int64(number * 1024 * 1024 * 1024)
	case "TB", "TIB":
		return int64(number * 1024 * 1024 * 1024 * 1024)
	default:
		return 0
	}
}

func parseLooseDuration(value string) (time.Duration, bool) {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == "" {
		return 0, false
	}
	if strings.ContainsAny(trimmed, "hms") {
		duration, err := time.ParseDuration(strings.ReplaceAll(trimmed, " ", ""))
		if err == nil {
			return duration, true
		}
	}
	parts := strings.Split(trimmed, ":")
	if len(parts) == 2 || len(parts) == 3 {
		multiplier := []time.Duration{time.Second, time.Minute, time.Hour}
		var total time.Duration
		for index := 0; index < len(parts); index++ {
			partValue, err := strconv.Atoi(strings.TrimSpace(parts[len(parts)-1-index]))
			if err != nil {
				return 0, false
			}
			total += time.Duration(partValue) * multiplier[index]
		}
		return total, true
	}
	return 0, false
}

// findMySQLShell ищет путь к mysqlsh
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
	return s.buildURI(s.config.Remote, s.config.Remote.Host, s.config.Remote.Port)
}

// buildLocalURI создает URI для подключения к локальному серверу (без пароля)
func (s *MySQLShellService) buildLocalURI() string {
	return s.buildURI(s.config.Local, s.config.Local.Host, s.config.Local.Port)
}

func (s *MySQLShellService) buildURI(mysqlConfig config.MySQLConfig, host string, port int) string {
	return fmt.Sprintf("mysql://%s@%s:%d", mysqlConfig.User, host, port)
}

func (s *MySQLShellService) transportMode() models.TransportMode {
	if s.config.Remote.HasProxy() {
		return models.TransportModeProxy
	}
	return models.TransportModeDirect
}

func (s *MySQLShellService) remoteDumpURI() (string, *proxyTunnel, func(), error) {
	tunnel, err := newProxyTunnel(s.config.Remote)
	if err != nil {
		return "", nil, nil, fmt.Errorf("failed to start proxy tunnel: %w", err)
	}

	cleanup := func() {
		_ = tunnel.Close()
	}

	return s.buildURI(s.config.Remote, tunnel.Host(), tunnel.Port()), tunnel, cleanup, nil
}

func (s *MySQLShellService) effectiveDumpThreads(logicalSize int64) int {
	threads := s.config.Dump.Threads
	if threads <= 0 {
		threads = 1
	}

	// Small schemas pay more for extra worker coordination and proxy round-trips than they gain.
	if logicalSize > 0 && logicalSize <= 64*1024*1024 && threads > 2 {
		return 2
	}
	if logicalSize > 0 && logicalSize <= 256*1024*1024 && threads > 4 {
		return 4
	}

	return threads
}

func (s *MySQLShellService) dumpCompressionArg() string {
	if !s.config.Dump.Compress {
		return "--compression=none"
	}
	return "--compression=zstd"
}

func (s *MySQLShellService) transportCompressionArgs() []string {
	if !s.config.Dump.NetworkCompress {
		return nil
	}

	level := s.config.Dump.NetworkZstdLevel
	if level <= 0 {
		level = 7
	}

	return []string{
		"--compress=REQUIRED",
		"--compression-algorithms=zstd,zlib",
		fmt.Sprintf("--zstd-compression-level=%d", level),
	}
}

func (s *MySQLShellService) buildDumpArgs(remoteURI string, databaseName string, dumpDir string, logicalSize int64, effectiveTables []string) []string {
	args := append([]string{
		"--uri", remoteURI,
		fmt.Sprintf("--password=%s", s.config.Remote.Password),
	}, s.transportCompressionArgs()...)
	args = append(args,
		"--", "util", "dump-schemas", databaseName,
		fmt.Sprintf("--outputUrl=%s", dumpDir),
		fmt.Sprintf("--threads=%d", s.effectiveDumpThreads(logicalSize)),
		"--consistent=false",
		"--skipConsistencyChecks",
		s.dumpCompressionArg(),
	)
	if len(effectiveTables) > 0 {
		qualified := make([]string, 0, len(effectiveTables))
		for _, tableName := range effectiveTables {
			qualified = append(qualified, fmt.Sprintf("%s.%s", databaseName, tableName))
		}
		args = append(args, "--includeTables="+strings.Join(qualified, ","))
	}
	return args
}

// CreateDump создает дамп удаленной базы данных через MySQL Shell
func (s *MySQLShellService) CreateDump(databaseName string, dryRun bool) (*models.SyncResult, string, error) {
	return s.CreateDumpTargetWithObserver(models.SyncTarget{DatabaseName: databaseName, ReplaceEntireDatabase: true}, dryRun, nil)
}

// CreateDumpTarget создает дамп удаленной базы данных или выбранного набора таблиц.
func (s *MySQLShellService) CreateDumpTarget(target models.SyncTarget, dryRun bool) (*models.SyncResult, string, error) {
	return s.CreateDumpTargetWithObserver(target, dryRun, nil)
}

// CreateDumpTargetWithObserver создает дамп и отправляет progress snapshots в observer.
func (s *MySQLShellService) CreateDumpTargetWithObserver(target models.SyncTarget, dryRun bool, observer models.ProgressObserver) (*models.SyncResult, string, error) {
	startTime := time.Now()
	databaseName := target.DatabaseName
	effectiveTables := target.EffectiveTables()
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhasePlanning, DatabaseName: databaseName, Message: "Preparing remote dump", Timestamp: startTime})
	}

	// Получаем информацию о базе данных
	dbInfo, err := s.dbService.GetDatabaseInfo(databaseName, true)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get database info: %w", err)
	}

	logicalSize := dbInfo.DataSize
	indexSize := dbInfo.IndexSize
	tablesCount := dbInfo.Tables
	if len(effectiveTables) > 0 {
		logicalSize, indexSize, tablesCount, err = s.selectedTableStats(databaseName, effectiveTables)
		if err != nil {
			return nil, "", fmt.Errorf("failed to calculate selected table stats: %w", err)
		}
	}

	if dryRun {
		result := &models.SyncResult{
			Success:            true,
			DatabaseName:       databaseName,
			DumpSize:           logicalSize,
			DumpSizeOnDisk:     logicalSize,
			LogicalSize:        logicalSize,
			IndexSize:          indexSize,
			TablesCount:        tablesCount,
			SelectedTables:     append([]string(nil), target.SelectedTables...),
			AutoIncludedTables: append([]string(nil), target.AutoIncludedTables...),
			TransportMode:      s.transportMode(),
			StartTime:          time.Now(),
			EndTime:            time.Now(),
			Duration:           0,
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

	remoteURI, tunnel, cleanup, err := s.remoteDumpURI()
	if err != nil {
		os.RemoveAll(dumpDir)
		return nil, "", err
	}
	defer cleanup()

	// Строим команду mysqlsh для дампа
	args := s.buildDumpArgs(remoteURI, databaseName, dumpDir, logicalSize, effectiveTables)

	cmd := exec.Command(mysqlshPath, args...)
	cmd.Env = append(os.Environ(), "MYSQLSH_TERM_COLOR_MODE=nocolor")

	// Показываем статус в одной строке (будет перезаписана)
	s.printStatusf("📦 Dumping %s (%d tables)...", databaseName, tablesCount)

	// Создаём pipe для фильтрации вывода
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		os.RemoveAll(dumpDir)
		return nil, "", fmt.Errorf("failed to start mysqlsh: %w", err)
	}
	if observer != nil {
		observer(models.ProgressSnapshot{
			Phase:          models.SyncPhaseDump,
			DatabaseName:   databaseName,
			Message:        "Streaming remote dump",
			BytesCompleted: 0,
			BytesTotal:     logicalSize,
			Traffic:        tunnel.Metrics(),
			Timestamp:      time.Now(),
		})
	}

	stdoutCapture := &progressCaptureWriter{}
	stderrCapture := &progressCaptureWriter{}
	if !s.quiet {
		stdoutCapture.writer = os.Stdout
		stderrCapture.writer = os.Stderr
	}

	stopLiveProgress := make(chan struct{})
	defer close(stopLiveProgress)
	if observer != nil {
		go emitTrafficSnapshots(stopLiveProgress, 250*time.Millisecond, databaseName, logicalSize, tunnel.Metrics, observer)
	}

	var streamWG sync.WaitGroup
	streamWG.Add(2)
	go func() {
		defer streamWG.Done()
		filterMySQLShellOutput(io.TeeReader(stdoutPipe, stdoutCapture), models.SyncPhaseDump, databaseName, tunnel.Metrics, observer, nil)
	}()
	go func() {
		defer streamWG.Done()
		filterMySQLShellOutput(io.TeeReader(stderrPipe, stderrCapture), models.SyncPhaseDump, databaseName, tunnel.Metrics, observer, nil)
	}()

	err = cmd.Wait()
	streamWG.Wait()
	if err != nil {
		os.RemoveAll(dumpDir)
		return nil, "", formatMySQLShellError("dump", err, stdoutCapture.String(), stderrCapture.String())
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
	s.printStatusf("\r✅ Dumped %s (%d tables) → %s in %v\n", databaseName, tablesCount, FormatSize(totalSize), endTime.Sub(startTime).Round(time.Second))

	result := &models.SyncResult{
		Success:            true,
		DatabaseName:       databaseName,
		Duration:           endTime.Sub(startTime),
		DumpSize:           totalSize,
		DumpSizeOnDisk:     totalSize,
		LogicalSize:        logicalSize,
		IndexSize:          indexSize,
		TablesCount:        tablesCount,
		SelectedTables:     append([]string(nil), target.SelectedTables...),
		AutoIncludedTables: append([]string(nil), target.AutoIncludedTables...),
		TransportMode:      tunnel.TransportMode(),
		Traffic:            tunnel.Metrics(),
		StartTime:          startTime,
		EndTime:            endTime,
	}
	if logicalSize > 0 {
		result.CompressionRatio = float64(totalSize) / float64(logicalSize)
	}
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhaseDump, DatabaseName: databaseName, Message: "Dump complete", Percent: 100, BytesCompleted: totalSize, BytesTotal: totalSize, Traffic: result.Traffic, Timestamp: endTime})
	}

	return result, dumpDir, nil
}

func (s *MySQLShellService) selectedTableStats(databaseName string, selectedTables []string) (int64, int64, int, error) {
	tables, err := s.dbService.ListTables(databaseName, true)
	if err != nil {
		return 0, 0, 0, err
	}
	selected := make(map[string]struct{}, len(selectedTables))
	for _, tableName := range selectedTables {
		selected[tableName] = struct{}{}
	}
	var logicalSize int64
	var indexSize int64
	var count int
	for _, table := range tables {
		if _, ok := selected[table.Name]; !ok {
			continue
		}
		logicalSize += table.DataSize
		indexSize += table.IndexSize
		count++
	}
	return logicalSize, indexSize, count, nil
}

// RestoreDump восстанавливает дамп в локальную БД через MySQL Shell
func (s *MySQLShellService) RestoreDump(dumpDir string, databaseName string, dryRun bool) error {
	return s.RestoreDumpWithObserver(dumpDir, databaseName, dryRun, nil)
}

// RestoreDumpWithObserver восстанавливает дамп и отправляет progress snapshots в observer.
func (s *MySQLShellService) RestoreDumpWithObserver(dumpDir string, databaseName string, dryRun bool, observer models.ProgressObserver) error {
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
	s.printStatusf("🔄 Restoring %s...", databaseName)

	// Создаём pipe для фильтрации вывода
	stdoutPipe, _ := cmd.StdoutPipe()
	stderrPipe, _ := cmd.StderrPipe()

	startTime := time.Now()
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhaseRestore, DatabaseName: databaseName, Message: "Preparing local restore", Timestamp: startTime})
	}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start mysqlsh: %w", err)
	}
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhaseRestore, DatabaseName: databaseName, Message: "Loading dump into local MySQL", Timestamp: time.Now()})
	}

	stdoutCapture := &progressCaptureWriter{}
	stderrCapture := &progressCaptureWriter{}
	if !s.quiet {
		stdoutCapture.writer = os.Stdout
		stderrCapture.writer = os.Stderr
	}

	var streamWG sync.WaitGroup
	streamWG.Add(2)
	go func() {
		defer streamWG.Done()
		filterMySQLShellOutput(io.TeeReader(stdoutPipe, stdoutCapture), models.SyncPhaseRestore, databaseName, nil, observer, nil)
	}()
	go func() {
		defer streamWG.Done()
		filterMySQLShellOutput(io.TeeReader(stderrPipe, stderrCapture), models.SyncPhaseRestore, databaseName, nil, observer, nil)
	}()

	err = cmd.Wait()
	streamWG.Wait()
	if err != nil {
		return formatMySQLShellError("load", err, stdoutCapture.String(), stderrCapture.String())
	}

	// Перезаписываем строку с результатом
	s.printStatusf("\r✅ Restored %s in %v                    \n", databaseName, time.Since(startTime).Round(time.Second))
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhaseRestore, DatabaseName: databaseName, Message: "Restore complete", Percent: 100, Timestamp: time.Now()})
	}

	return nil
}

// ExecuteSync выполняет полную синхронизацию базы данных через MySQL Shell
func (s *MySQLShellService) ExecuteSync(databaseName string) (*models.SyncResult, error) {
	return s.ExecuteTargetWithObserver(models.SyncTarget{DatabaseName: databaseName, ReplaceEntireDatabase: true}, nil)
}

// ExecuteTarget выполняет полную синхронизацию указанной цели, включая partial table sync.
func (s *MySQLShellService) ExecuteTarget(target models.SyncTarget) (*models.SyncResult, error) {
	return s.ExecuteTargetWithObserver(target, nil)
}

// ExecuteTargetWithObserver выполняет синхронизацию одной цели с progress observer.
func (s *MySQLShellService) ExecuteTargetWithObserver(target models.SyncTarget, observer models.ProgressObserver) (*models.SyncResult, error) {
	startTime := time.Now()
	databaseName := target.DatabaseName
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhaseValidation, DatabaseName: databaseName, Message: "Validating connections and prerequisites", Timestamp: startTime})
	}

	// Валидация операции
	if err := s.ValidateDumpOperation(databaseName); err != nil {
		if observer != nil {
			observer(models.ProgressSnapshot{Phase: models.SyncPhaseFailed, DatabaseName: databaseName, Message: err.Error(), Timestamp: time.Now()})
		}
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Создаем дамп
	dumpResult, dumpDir, err := s.CreateDumpTargetWithObserver(target, false, observer)
	if err != nil {
		if observer != nil {
			observer(models.ProgressSnapshot{Phase: models.SyncPhaseFailed, DatabaseName: databaseName, Message: err.Error(), Timestamp: time.Now()})
		}
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
	if err := s.RestoreDumpWithObserver(dumpDir, databaseName, false, observer); err != nil {
		if observer != nil {
			observer(models.ProgressSnapshot{Phase: models.SyncPhaseFailed, DatabaseName: databaseName, Message: err.Error(), Timestamp: time.Now()})
		}
		return nil, fmt.Errorf("restore failed: %w", err)
	}
	restoreDuration := time.Since(restoreStart)

	endTime := time.Now()

	result := &models.SyncResult{
		Success:            true,
		DatabaseName:       databaseName,
		Duration:           endTime.Sub(startTime),
		DumpDuration:       dumpResult.Duration,
		RestoreDuration:    restoreDuration,
		DumpSize:           dumpResult.DumpSize,
		DumpSizeOnDisk:     dumpResult.DumpSizeOnDisk,
		LogicalSize:        dumpResult.LogicalSize,
		IndexSize:          dumpResult.IndexSize,
		TablesCount:        dumpResult.TablesCount,
		SelectedTables:     append([]string(nil), dumpResult.SelectedTables...),
		AutoIncludedTables: append([]string(nil), dumpResult.AutoIncludedTables...),
		TransportMode:      dumpResult.TransportMode,
		CompressionRatio:   dumpResult.CompressionRatio,
		Traffic:            dumpResult.Traffic,
		StartTime:          startTime,
		EndTime:            endTime,
	}
	if observer != nil {
		observer(models.ProgressSnapshot{Phase: models.SyncPhaseDone, DatabaseName: databaseName, Message: "Sync complete", Percent: 100, BytesCompleted: result.Traffic.TotalBytes(), BytesTotal: result.Traffic.TotalBytes(), Traffic: result.Traffic, Timestamp: endTime})
	}

	return result, nil
}

// ExecutePlan выполняет план синхронизации последовательно и стримит progress snapshots.
func (s *MySQLShellService) ExecutePlan(plan *models.SyncPlan, runtime models.RuntimeOptions, observer models.ProgressObserver) ([]models.SyncResult, error) {
	if plan == nil {
		return nil, fmt.Errorf("sync plan is nil")
	}
	results := make([]models.SyncResult, 0, len(plan.Targets))
	for _, target := range plan.Targets {
		result, err := s.ExecuteTargetWithObserver(target, observer)
		if err != nil {
			failed := models.SyncResult{DatabaseName: target.DatabaseName, Success: false, Error: err.Error(), StartTime: time.Now(), EndTime: time.Now()}
			results = append(results, failed)
			return results, err
		}
		results = append(results, *result)
	}
	_ = runtime
	return results, nil
}

// Cleanup удаляет временные файлы
func (s *MySQLShellService) Cleanup(dumpDir string) error {
	if dumpDir != "" {
		return os.RemoveAll(dumpDir)
	}
	return nil
}

func formatMySQLShellError(operation string, commandErr error, stdout string, stderr string) error {
	parts := []string{fmt.Sprintf("mysqlsh %s failed: %v", operation, commandErr)}
	if stderr != "" {
		parts = append(parts, "stderr: "+stderr)
	}
	if stdout != "" {
		parts = append(parts, "stdout: "+stdout)
	}
	return errors.New(strings.Join(parts, "\n"))
}

func emitTrafficSnapshots(stop <-chan struct{}, interval time.Duration, databaseName string, bytesTotal int64, metricsFn func() models.TrafficMetrics, observer models.ProgressObserver) {
	if metricsFn == nil || observer == nil {
		return
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	previousTotal := int64(0)
	previousAt := time.Now()

	for {
		select {
		case <-stop:
			return
		case now := <-ticker.C:
			metrics := metricsFn()
			downloaded := metrics.DownloadedBytes()
			percent := float64(0)
			if bytesTotal > 0 {
				percent = float64(downloaded) / float64(bytesTotal) * 100
				if percent > 99 {
					percent = 99
				}
			}
			elapsed := now.Sub(previousAt)
			if elapsed > 0 {
				delta := downloaded - previousTotal
				if delta >= 0 {
					metrics.CurrentBytesPerSecond = float64(delta) / elapsed.Seconds()
				}
			}
			if metrics.SampleWindow > 0 {
				metrics.AverageBytesPerSecond = float64(downloaded) / metrics.SampleWindow.Seconds()
			}
			previousTotal = downloaded
			previousAt = now

			observer(models.ProgressSnapshot{
				Phase:          models.SyncPhaseDump,
				DatabaseName:   databaseName,
				Message:        "Streaming remote dump",
				Percent:        percent,
				BytesCompleted: downloaded,
				BytesTotal:     bytesTotal,
				Traffic:        metrics,
				Timestamp:      now,
			})
		}
	}
}
