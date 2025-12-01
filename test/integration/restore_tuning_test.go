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

// TestDB для тюнинга restore
const RestoreTestDB = "easy_pushka_biz"

// RestoreTestOptions - опции для тестирования myloader
type RestoreTestOptions struct {
	Name                        string
	Threads                     int
	MaxThreadsForSchemaCreation int
	MaxThreadsForIndexCreation  int
	OptimizeKeys                string
	InnoDBOptimizeKeys          string
	SkipDefiner                 bool
	RetryCount                  int
	Overwrite                   bool   // -o DROP
	PurgeMode                   string // NONE, TRUNCATE, DROP, DELETE
	DisableRedoLog              bool
}

func (o RestoreTestOptions) String() string {
	return fmt.Sprintf("t=%d,schema=%d,idx=%d,opt=%s",
		o.Threads, o.MaxThreadsForSchemaCreation, o.MaxThreadsForIndexCreation, o.OptimizeKeys)
}

// convertPath конвертирует Windows путь для Docker
func convertPath(path string) string {
	if runtime.GOOS == "windows" {
		path = strings.ReplaceAll(path, "\\", "/")
		if len(path) >= 2 && path[1] == ':' {
			path = "/" + strings.ToLower(string(path[0])) + path[2:]
		}
	}
	return path
}

// dockerHost возвращает адрес хоста для Docker
func dockerHost(host string) string {
	if host == "localhost" || host == "127.0.0.1" {
		return "host.docker.internal"
	}
	return host
}

// createDump создаёт один дамп для всех тестов restore
func createDump(cfg *config.Config, dbName, dumpDir string) (int64, error) {
	dockerDumpDir := convertPath(dumpDir)
	remoteHost := dockerHost(cfg.Remote.Host)

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
		"--threads", "8",
		"--rows", "100000",
		"--triggers",
		"--routines",
		"--events",
		"--sync-thread-lock-mode=NO_LOCK",
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("mydumper failed: %w\nOutput: %s", err, string(output))
	}

	// Размер дампа
	var size int64
	filepath.Walk(dumpDir, func(path string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			size += info.Size()
		}
		return nil
	})

	return size, nil
}

// runRestore выполняет myloader с заданными опциями
func runRestore(cfg *config.Config, dbName string, opts RestoreTestOptions, dumpDir string) (time.Duration, error) {
	// Recreate database
	dropCmd := exec.Command(
		cfg.Dump.MysqlPath,
		"--host="+cfg.Local.Host,
		"--port="+fmt.Sprintf("%d", cfg.Local.Port),
		"--user="+cfg.Local.User,
		"--password="+cfg.Local.Password,
		"-e", fmt.Sprintf("DROP DATABASE IF EXISTS `%s`; CREATE DATABASE `%s`;", dbName, dbName),
	)
	if output, err := dropCmd.CombinedOutput(); err != nil {
		return 0, fmt.Errorf("failed to recreate db: %w\nOutput: %s", err, string(output))
	}

	absDumpDir, _ := filepath.Abs(dumpDir)
	dockerDumpDir := convertPath(absDumpDir)
	localHost := dockerHost(cfg.Local.Host)

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
	}

	// Schema creation threads
	if opts.MaxThreadsForSchemaCreation > 0 {
		args = append(args, "--max-threads-for-schema-creation", fmt.Sprintf("%d", opts.MaxThreadsForSchemaCreation))
	}

	// Index creation threads
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

	// Skip definer
	if opts.SkipDefiner {
		args = append(args, "--skip-definer")
	}

	// Retry count
	if opts.RetryCount > 0 {
		args = append(args, "--retry-count", fmt.Sprintf("%d", opts.RetryCount))
	}

	// Overwrite mode
	if opts.Overwrite {
		args = append(args, "-o", "DROP")
	}

	// Purge mode
	if opts.PurgeMode != "" && opts.PurgeMode != "NONE" {
		args = append(args, "--purge-mode", opts.PurgeMode)
	}

	// Disable redo log (MySQL 8.0.21+)
	if opts.DisableRedoLog {
		args = append(args, "--disable-redo-log")
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

// TestE2E_RestoreTuning - основной тест для оптимизации restore
func TestE2E_RestoreTuning(t *testing.T) {
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	dbService := services.NewDatabaseService(cfg)

	// Информация о БД
	dbInfo, err := dbService.GetDatabaseInfo(RestoreTestDB, true)
	if err != nil {
		t.Fatalf("Failed to get database info: %v", err)
	}
	t.Logf("📊 Database: %s (%s, %d tables)", RestoreTestDB, services.FormatSize(dbInfo.Size), dbInfo.Tables)

	// Создаём дамп один раз
	dumpDir := "./tmp/restore_tuning_dump"
	os.RemoveAll(dumpDir)
	os.MkdirAll(dumpDir, 0755)
	defer os.RemoveAll(dumpDir)

	t.Log("⬇️  Creating dump (one time for all restore tests)...")
	dumpStart := time.Now()
	dumpSize, err := createDump(cfg, RestoreTestDB, dumpDir)
	if err != nil {
		t.Fatalf("Dump failed: %v", err)
	}
	t.Logf("✅ Dump created: %s in %v", services.FormatSize(dumpSize), time.Since(dumpStart).Round(time.Second))

	// Комбинации для тестирования RESTORE
	testCases := []RestoreTestOptions{
		// === Базовые тесты потоков ===
		{Name: "1_thread_baseline", Threads: 1, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 1, RetryCount: 10, Overwrite: true},
		{Name: "4_threads", Threads: 4, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 4, RetryCount: 10, Overwrite: true},
		{Name: "8_threads", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, RetryCount: 10, Overwrite: true},
		{Name: "16_threads", Threads: 16, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 16, RetryCount: 10, Overwrite: true},

		// === OptimizeKeys варианты ===
		{Name: "8t_no_optimize", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, RetryCount: 10, Overwrite: true},
		{Name: "8t_optimize_per_table", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, OptimizeKeys: "AFTER_IMPORT_PER_TABLE", RetryCount: 10, Overwrite: true},
		{Name: "8t_optimize_all_tables", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, OptimizeKeys: "AFTER_IMPORT_ALL_TABLES", RetryCount: 10, Overwrite: true},

		// === InnoDB OptimizeKeys ===
		{Name: "8t_innodb_opt_per_table", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, InnoDBOptimizeKeys: "AFTER_IMPORT_PER_TABLE", RetryCount: 10, Overwrite: true},
		{Name: "8t_innodb_opt_all", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, InnoDBOptimizeKeys: "AFTER_IMPORT_ALL_TABLES", RetryCount: 10, Overwrite: true},

		// === Schema creation threads ===
		{Name: "8t_schema_2", Threads: 8, MaxThreadsForSchemaCreation: 2, MaxThreadsForIndexCreation: 8, RetryCount: 10, Overwrite: true},
		{Name: "8t_schema_4", Threads: 8, MaxThreadsForSchemaCreation: 4, MaxThreadsForIndexCreation: 8, RetryCount: 10, Overwrite: true},
		{Name: "8t_schema_8", Threads: 8, MaxThreadsForSchemaCreation: 8, MaxThreadsForIndexCreation: 8, RetryCount: 20, Overwrite: true},

		// === Disable redo log (может ускорить на SSD) ===
		{Name: "8t_disable_redo", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, DisableRedoLog: true, RetryCount: 10, Overwrite: true},

		// === Комбинированные варианты ===
		{Name: "8t_innodb_redo_off", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, InnoDBOptimizeKeys: "AFTER_IMPORT_PER_TABLE", DisableRedoLog: true, RetryCount: 10, Overwrite: true},
		{Name: "16t_innodb_redo_off", Threads: 16, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 16, InnoDBOptimizeKeys: "AFTER_IMPORT_PER_TABLE", DisableRedoLog: true, RetryCount: 10, Overwrite: true},

		// === Skip definer ===
		{Name: "8t_skip_definer", Threads: 8, MaxThreadsForSchemaCreation: 1, MaxThreadsForIndexCreation: 8, SkipDefiner: true, RetryCount: 10, Overwrite: true},
	}

	type Result struct {
		Name     string
		Duration time.Duration
		Success  bool
		Error    string
	}

	results := make([]Result, 0, len(testCases))

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			t.Logf("⬆️  Testing: %s", tc.Name)
			t.Logf("   Options: threads=%d, schema=%d, index=%d, optimize=%s, innodb=%s, redo_off=%v",
				tc.Threads, tc.MaxThreadsForSchemaCreation, tc.MaxThreadsForIndexCreation,
				tc.OptimizeKeys, tc.InnoDBOptimizeKeys, tc.DisableRedoLog)

			duration, err := runRestore(cfg, RestoreTestDB, tc, dumpDir)

			result := Result{Name: tc.Name, Duration: duration}
			if err != nil {
				result.Success = false
				result.Error = err.Error()
				t.Logf("❌ FAILED in %v: %v", duration.Round(time.Second), err)
			} else {
				result.Success = true
				t.Logf("✅ PASSED in %v", duration.Round(time.Second))
			}
			results = append(results, result)
		})
	}

	// Итоговая таблица
	t.Run("Summary", func(t *testing.T) {
		fmt.Println()
		fmt.Println(strings.Repeat("═", 70))
		fmt.Println("              RESTORE TUNING RESULTS")
		fmt.Println(strings.Repeat("═", 70))
		fmt.Printf("%-35s | %-12s | %s\n", "Test Name", "Duration", "Status")
		fmt.Println(strings.Repeat("-", 70))

		var best Result
		for _, r := range results {
			status := "✅ PASS"
			if !r.Success {
				status = "❌ FAIL"
			}

			fmt.Printf("%-35s | %-12v | %s\n", r.Name, r.Duration.Round(time.Second), status)

			if r.Success && (best.Duration == 0 || r.Duration < best.Duration) {
				best = r
			}
		}

		fmt.Println(strings.Repeat("═", 70))
		if best.Name != "" {
			fmt.Printf("🏆 BEST: %s with %v\n", best.Name, best.Duration.Round(time.Second))
		}
		fmt.Println(strings.Repeat("═", 70))
	})
}
