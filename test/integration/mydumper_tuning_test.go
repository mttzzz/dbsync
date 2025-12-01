//go:build e2e
// +build e2e

package integration

import (
	"db-sync-cli/internal/config"
	"db-sync-cli/internal/services"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestDatabase для тюнинга - средняя БД для быстрых тестов
const TuningDatabaseName = "easy_pushka_biz"

// DumpOptions содержит опции для mydumper
type DumpOptions struct {
	Threads        int
	Rows           int // chunk size
	Compress       bool
	LongQueryGuard int // секунды
	StatementSize  int // bytes
	ChunkFilesize  int // MB, 0 = disabled
}

// RestoreOptions содержит опции для myloader
type RestoreOptions struct {
	Threads                     int
	MaxThreadsForSchemaCreation int
	MaxThreadsForIndexCreation  int
	OptimizeKeys                string // AFTER_IMPORT_PER_TABLE, AFTER_IMPORT_ALL_TABLES, или пусто
	EnableBinlog                bool
	InnoDBOptimizeKeys          string // NONE, AFTER_IMPORT_PER_TABLE, AFTER_IMPORT_ALL_TABLES
	SkipDefiner                 bool
	AppendIfNotExist            bool
	SerialTblCreation           bool // --serialized-table-creation (deprecated но всё ещё работает)
	RetryCount                  int
}

// TuningResult хранит результат одного теста
type TuningResult struct {
	Name            string
	DumpOptions     DumpOptions
	RestoreOptions  RestoreOptions
	DumpDuration    time.Duration
	RestoreDuration time.Duration
	TotalDuration   time.Duration
	DumpSize        int64
	Success         bool
	Error           string
}

// convertPathForDocker конвертирует Windows путь для Docker
func convertPathForDocker(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
		if len(path) >= 2 && path[1] == ':' {
			path = "/" + strings.ToLower(string(path[0])) + path[2:]
		}
	}
	return path
}

// getDockerHost возвращает адрес хоста для Docker
func getDockerHost(originalHost string) string {
	if originalHost == "localhost" || originalHost == "127.0.0.1" {
		return "host.docker.internal"
	}
	return originalHost
}

// runDumpWithOptions выполняет mydumper с заданными опциями
func runDumpWithOptions(cfg *config.Config, dbName string, opts DumpOptions, dumpDir string) (time.Duration, int64, error) {
	dockerDumpDir := convertPathForDocker(dumpDir)
	remoteHost := getDockerHost(cfg.Remote.Host)

	args := []string{
		"run", "--rm",
		"--network", "host",
		"-v", fmt.Sprintf("%s:/dump", dockerDumpDir),
		cfg.Dump.MyDumperImage,
		"mydumper",
		"--host", remoteHost,
		"--port", fmt.Sprintf("%d", cfg.Remote.Port),
		"--user", cfg.Remote.User,
		"--password", cfg.Remote.Password,
		"--database", dbName,
		"--outputdir", "/dump",
		"--threads", fmt.Sprintf("%d", opts.Threads),
		"--triggers",
		"--routines",
		"--events",
		"--sync-thread-lock-mode=NO_LOCK",
	}

	// Rows (chunk size)
	if opts.Rows > 0 {
		args = append(args, "--rows", fmt.Sprintf("%d", opts.Rows))
	}

	// Compress
	if opts.Compress {
		args = append(args, "--compress")
	}

	// Long query guard
	if opts.LongQueryGuard > 0 {
		args = append(args, "--long-query-guard", fmt.Sprintf("%d", opts.LongQueryGuard))
	}

	// Statement size
	if opts.StatementSize > 0 {
		args = append(args, "--statement-size", fmt.Sprintf("%d", opts.StatementSize))
	}

	// Chunk filesize
	if opts.ChunkFilesize > 0 {
		args = append(args, "--chunk-filesize", fmt.Sprintf("%d", opts.ChunkFilesize))
	}

	cmd := exec.Command("docker", args...)
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		return duration, 0, fmt.Errorf("mydumper failed: %w\nOutput: %s", err, string(output))
	}

	// Подсчитываем размер
	var totalSize int64
	filepath.Walk(dumpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	return duration, totalSize, nil
}

// runRestoreWithOptions выполняет myloader с заданными опциями
func runRestoreWithOptions(cfg *config.Config, dbName string, opts RestoreOptions, dumpDir string) (time.Duration, error) {
	// Drop и create database
	dropCmd := exec.Command(
		cfg.Dump.MysqlPath,
		"--host="+cfg.Local.Host,
		"--port="+fmt.Sprintf("%d", cfg.Local.Port),
		"--user="+cfg.Local.User,
		"--password="+cfg.Local.Password,
		"-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`; CREATE DATABASE `%s`;", dbName, dbName),
	)
	if output, err := dropCmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("failed to recreate database: %w\nOutput: %s", err, string(output))
	}

	absDumpDir, _ := filepath.Abs(dumpDir)
	dockerDumpDir := convertPathForDocker(absDumpDir)
	localHost := getDockerHost(cfg.Local.Host)

	args := []string{
		"run", "--rm",
		"--network", "host",
		"-v", fmt.Sprintf("%s:/dump", dockerDumpDir),
		cfg.Dump.MyDumperImage,
		"myloader",
		"--host", localHost,
		"--port", fmt.Sprintf("%d", cfg.Local.Port),
		"--user", cfg.Local.User,
		"--password", cfg.Local.Password,
		"--database", dbName,
		"--directory", "/dump",
		"--threads", fmt.Sprintf("%d", opts.Threads),
		"-o", "DROP", // Overwrite tables
	}

	// Max threads for schema creation
	if opts.MaxThreadsForSchemaCreation > 0 {
		args = append(args, "--max-threads-for-schema-creation", fmt.Sprintf("%d", opts.MaxThreadsForSchemaCreation))
	}

	// Max threads for index creation
	if opts.MaxThreadsForIndexCreation > 0 {
		args = append(args, "--max-threads-for-index-creation", fmt.Sprintf("%d", opts.MaxThreadsForIndexCreation))
	}

	// Optimize keys
	if opts.OptimizeKeys != "" {
		args = append(args, "--optimize-keys", opts.OptimizeKeys)
	}

	// InnoDB optimize keys
	if opts.InnoDBOptimizeKeys != "" {
		args = append(args, "--innodb-optimize-keys", opts.InnoDBOptimizeKeys)
	}

	// Enable binlog
	if opts.EnableBinlog {
		args = append(args, "--enable-binlog")
	}

	// Skip definer
	if opts.SkipDefiner {
		args = append(args, "--skip-definer")
	}

	// Retry count
	if opts.RetryCount > 0 {
		args = append(args, "--retry-count", fmt.Sprintf("%d", opts.RetryCount))
	}

	cmd := exec.Command("docker", args...)
	startTime := time.Now()
	output, err := cmd.CombinedOutput()
	duration := time.Since(startTime)

	if err != nil {
		return duration, fmt.Errorf("myloader failed: %w\nOutput: %s", err, string(output))
	}

	return duration, nil
}

// TestE2E_MydumperTuning тестирует разные комбинации опций mydumper/myloader
func TestE2E_MydumperTuning(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	dbService := services.NewDatabaseService(cfg)

	// Проверяем доступность БД
	dbInfo, err := dbService.GetDatabaseInfo(TuningDatabaseName, true)
	if err != nil {
		t.Fatalf("Failed to get database info: %v", err)
	}
	t.Logf("📊 Database: %s (%s, %d tables)", TuningDatabaseName, services.FormatSize(dbInfo.Size), dbInfo.Tables)

	// Базовые опции дампа (8 потоков, оптимальные настройки)
	baseDumpOpts := DumpOptions{
		Threads:  8,
		Rows:     100000,
		Compress: false,
	}

	// Комбинации опций для restore
	testCases := []struct {
		name        string
		dumpOpts    DumpOptions
		restoreOpts RestoreOptions
	}{
		// === ТЕСТ 1: Базовый (текущие настройки) ===
		{
			name:     "Baseline",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 2: Без OptimizeKeys ===
		{
			name:     "No_OptimizeKeys",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 3: OptimizeKeys AFTER_IMPORT_ALL_TABLES ===
		{
			name:     "OptimizeKeys_AllTables",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_ALL_TABLES",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 4: Больше потоков на schema creation ===
		{
			name:     "Schema_4_Threads",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 4,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 5: InnoDB Optimize Keys ===
		{
			name:     "InnoDB_OptimizeKeys",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				InnoDBOptimizeKeys:          "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 6: Skip Definer ===
		{
			name:     "Skip_Definer",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				SkipDefiner:                 true,
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 7: Dump с Compress ===
		{
			name: "Compressed_Dump",
			dumpOpts: DumpOptions{
				Threads:  8,
				Rows:     100000,
				Compress: true,
			},
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 8: Меньший chunk size ===
		{
			name: "Small_Chunks_50k",
			dumpOpts: DumpOptions{
				Threads:  8,
				Rows:     50000,
				Compress: false,
			},
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 9: Большой chunk size ===
		{
			name: "Large_Chunks_500k",
			dumpOpts: DumpOptions{
				Threads:  8,
				Rows:     500000,
				Compress: false,
			},
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 10: Chunk по размеру файла ===
		{
			name: "ChunkFilesize_64MB",
			dumpOpts: DumpOptions{
				Threads:       8,
				ChunkFilesize: 64, // 64MB файлы
				Compress:      false,
			},
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 11: Максимум параллельности ===
		{
			name:     "Max_Parallel",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 8, // Всё параллельно
				MaxThreadsForIndexCreation:  8,
				RetryCount:                  20, // Больше retry на случай FK
			},
		},

		// === ТЕСТ 12: Минимум параллельности на restore ===
		{
			name:     "Min_Parallel_Restore",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     1,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  1,
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 13: Средняя параллельность ===
		{
			name:     "Medium_Parallel_4",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     4,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  4,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 14: 16 потоков на restore ===
		{
			name:     "Threads_16_Restore",
			dumpOpts: baseDumpOpts,
			restoreOpts: RestoreOptions{
				Threads:                     16,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  16,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},

		// === ТЕСТ 15: Statement size увеличенный ===
		{
			name: "LargeStatements_10MB",
			dumpOpts: DumpOptions{
				Threads:       8,
				Rows:          100000,
				StatementSize: 10000000, // 10MB statements
			},
			restoreOpts: RestoreOptions{
				Threads:                     8,
				MaxThreadsForSchemaCreation: 1,
				MaxThreadsForIndexCreation:  8,
				OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
				RetryCount:                  10,
			},
		},
	}

	results := make([]TuningResult, 0, len(testCases))

	// Создаём базовую директорию для дампов
	tempDir := "./tmp/tuning_tests"
	os.MkdirAll(tempDir, 0755)
	defer os.RemoveAll(tempDir)

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := TuningResult{
				Name:           tc.name,
				DumpOptions:    tc.dumpOpts,
				RestoreOptions: tc.restoreOpts,
			}

			// Директория для этого теста
			dumpDir := filepath.Join(tempDir, fmt.Sprintf("dump_%s_%d", tc.name, time.Now().UnixNano()))
			os.MkdirAll(dumpDir, 0755)
			defer os.RemoveAll(dumpDir)

			// === DUMP ===
			t.Logf("⬇️  Dumping with: threads=%d, rows=%d, compress=%v",
				tc.dumpOpts.Threads, tc.dumpOpts.Rows, tc.dumpOpts.Compress)

			dumpDuration, dumpSize, err := runDumpWithOptions(cfg, TuningDatabaseName, tc.dumpOpts, dumpDir)
			result.DumpDuration = dumpDuration
			result.DumpSize = dumpSize

			if err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("Dump failed: %v", err)
				t.Logf("❌ %s", result.Error)
				results = append(results, result)
				return
			}

			t.Logf("✅ Dump: %s in %v", services.FormatSize(dumpSize), dumpDuration.Round(time.Second))

			// === RESTORE ===
			t.Logf("⬆️  Restoring with: threads=%d, schemaThreads=%d, indexThreads=%d, optimize=%s",
				tc.restoreOpts.Threads,
				tc.restoreOpts.MaxThreadsForSchemaCreation,
				tc.restoreOpts.MaxThreadsForIndexCreation,
				tc.restoreOpts.OptimizeKeys)

			restoreDuration, err := runRestoreWithOptions(cfg, TuningDatabaseName, tc.restoreOpts, dumpDir)
			result.RestoreDuration = restoreDuration

			if err != nil {
				result.Success = false
				result.Error = fmt.Sprintf("Restore failed: %v", err)
				t.Logf("❌ %s", result.Error)
				results = append(results, result)
				return
			}

			t.Logf("✅ Restore: %v", restoreDuration.Round(time.Second))

			result.TotalDuration = dumpDuration + restoreDuration
			result.Success = true

			t.Logf("📊 TOTAL: %v (dump: %v, restore: %v)",
				result.TotalDuration.Round(time.Second),
				result.DumpDuration.Round(time.Second),
				result.RestoreDuration.Round(time.Second))

			results = append(results, result)
		})
	}

	// Итоговая таблица
	t.Run("Summary", func(t *testing.T) {
		fmt.Println("\n" + strings.Repeat("═", 80))
		fmt.Println("                    TUNING RESULTS SUMMARY")
		fmt.Println(strings.Repeat("═", 80))
		fmt.Printf("%-30s | %-10s | %-10s | %-10s | %s\n", "Test Name", "Dump", "Restore", "TOTAL", "Status")
		fmt.Println(strings.Repeat("-", 80))

		var bestResult TuningResult
		for _, r := range results {
			status := "✅ PASS"
			if !r.Success {
				status = "❌ FAIL"
			}

			if r.Success && (bestResult.TotalDuration == 0 || r.TotalDuration < bestResult.TotalDuration) {
				bestResult = r
			}

			fmt.Printf("%-30s | %-10v | %-10v | %-10v | %s\n",
				r.Name,
				r.DumpDuration.Round(time.Second),
				r.RestoreDuration.Round(time.Second),
				r.TotalDuration.Round(time.Second),
				status)
		}

		fmt.Println(strings.Repeat("═", 80))
		if bestResult.Name != "" {
			fmt.Printf("🏆 BEST: %s with %v total\n", bestResult.Name, bestResult.TotalDuration.Round(time.Second))
			fmt.Printf("   Dump options: threads=%d, rows=%d, compress=%v\n",
				bestResult.DumpOptions.Threads, bestResult.DumpOptions.Rows, bestResult.DumpOptions.Compress)
			fmt.Printf("   Restore options: threads=%d, schema=%d, index=%d, optimize=%s\n",
				bestResult.RestoreOptions.Threads,
				bestResult.RestoreOptions.MaxThreadsForSchemaCreation,
				bestResult.RestoreOptions.MaxThreadsForIndexCreation,
				bestResult.RestoreOptions.OptimizeKeys)
		}
		fmt.Println(strings.Repeat("═", 80))
	})
}

// TestE2E_QuickTuning - быстрый тест для проверки работоспособности
func TestE2E_QuickTuning(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	dbService := services.NewDatabaseService(cfg)

	// Проверяем доступность большой БД
	dbInfo, err := dbService.GetDatabaseInfo(TuningDatabaseName, true)
	if err != nil {
		t.Fatalf("Failed to get database info for %s: %v", TuningDatabaseName, err)
	}
	t.Logf("📊 Database: %s (%s, %d tables)", TuningDatabaseName, services.FormatSize(dbInfo.Size), dbInfo.Tables)

	// Один тест с оптимальными настройками
	dumpDir := "./tmp/quick_tuning"
	os.MkdirAll(dumpDir, 0755)
	defer os.RemoveAll(dumpDir)

	dumpOpts := DumpOptions{
		Threads:  8,
		Rows:     100000,
		Compress: false,
	}

	restoreOpts := RestoreOptions{
		Threads:                     8,
		MaxThreadsForSchemaCreation: 1,
		MaxThreadsForIndexCreation:  8,
		OptimizeKeys:                "AFTER_IMPORT_PER_TABLE",
		RetryCount:                  10,
	}

	// Dump
	t.Log("⬇️  Starting dump...")
	dumpDuration, dumpSize, err := runDumpWithOptions(cfg, TuningDatabaseName, dumpOpts, dumpDir)
	if err != nil {
		t.Fatalf("Dump failed: %v", err)
	}
	t.Logf("✅ Dump: %s in %v", services.FormatSize(dumpSize), dumpDuration.Round(time.Second))

	// Restore
	t.Log("⬆️  Starting restore...")
	restoreDuration, err := runRestoreWithOptions(cfg, TuningDatabaseName, restoreOpts, dumpDir)
	if err != nil {
		t.Fatalf("Restore failed: %v", err)
	}
	t.Logf("✅ Restore: %v", restoreDuration.Round(time.Second))

	total := dumpDuration + restoreDuration
	t.Logf("📊 TOTAL: %v", total.Round(time.Second))
}
